package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/madhavanp/universalcrawl/internal/api"
	"github.com/madhavanp/universalcrawl/internal/browser"
	"github.com/madhavanp/universalcrawl/internal/crawler"
	"github.com/madhavanp/universalcrawl/internal/extract"
	"github.com/madhavanp/universalcrawl/internal/jobs"
	"github.com/madhavanp/universalcrawl/internal/llm"
	"github.com/madhavanp/universalcrawl/internal/scraper"
	"github.com/madhavanp/universalcrawl/internal/scraper/engines"
	"github.com/madhavanp/universalcrawl/internal/search"
	"github.com/madhavanp/universalcrawl/internal/storage"
)

func main() {
	port := flag.Int("port", envInt("PORT", 3000), "API server port")
	apiKey := flag.String("api-key", os.Getenv("API_KEY"), "Bearer token for API auth")
	poolSize := flag.Int("pool-size", envInt("BROWSER_POOL_SIZE", 5), "Headless Chrome pool size")
	workers := flag.Int("workers", envInt("WORKERS", 4), "Background job goroutines")
	dataDir := flag.String("data-dir", envOr("DATA_DIR", "./data"), "Database directory")
	logLevel := flag.String("log-level", envOr("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	flag.Parse()

	setupLogging(*logLevel)

	slog.Info("starting universalcrawl",
		"port", *port,
		"pool_size", *poolSize,
		"workers", *workers,
	)

	// Storage
	store, err := storage.NewBoltStore(*dataDir)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Build engine chain: Rod (headless Chrome) -> Fetch (HTTP fallback)
	var engs []engines.Engine

	pool, err := browser.NewPool(*poolSize)
	if err != nil {
		slog.Warn("browser pool failed, using fetch-only mode", "error", err)
	} else {
		engs = append(engs, engines.NewRodEngine(pool))
		defer pool.Close()
	}
	engs = append(engs, engines.NewFetchEngine())

	orch := scraper.NewOrchestrator(engs...)

	// Crawler
	webCrawler := crawler.NewWebCrawler(orch, *workers)

	// Job queue
	queue := jobs.NewQueue(100)
	queue.Start(*workers)
	defer queue.Stop()

	// LLM provider (optional, for extract/search)
	var llmProvider llm.Provider
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		llmProvider = llm.NewAnthropicProvider(key)
		slog.Info("llm provider configured", "provider", "anthropic")
	} else if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		llmProvider = llm.NewOpenAIProvider(key)
		slog.Info("llm provider configured", "provider", "openai")
	} else if base := os.Getenv("OLLAMA_BASE_URL"); base != "" {
		llmProvider = llm.NewOllamaProvider(base)
		slog.Info("llm provider configured", "provider", "ollama")
	}

	var extractor *extract.Extractor
	if llmProvider != nil {
		extractor = extract.NewExtractor(orch, llmProvider)
	}

	var searcher *search.Searcher
	if endpoint := os.Getenv("SEARXNG_ENDPOINT"); endpoint != "" {
		searcher = search.NewSearcher(orch, endpoint)
		slog.Info("search configured", "endpoint", endpoint)
	}

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", *port),
		Handler: api.NewServer(api.Config{APIKey: *apiKey}, api.Deps{
			Orchestrator: orch,
			Crawler:      webCrawler,
			Store:        store,
			Queue:        queue,
			Extractor:    extractor,
			Searcher:     searcher,
		}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func setupLogging(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	if n == 0 {
		return fallback
	}
	return n
}
