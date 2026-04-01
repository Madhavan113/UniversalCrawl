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
	"github.com/madhavanp/universalcrawl/internal/scraper"
	"github.com/madhavanp/universalcrawl/internal/scraper/engines"
)

func main() {
	port := flag.Int("port", envInt("PORT", 3000), "API server port")
	apiKey := flag.String("api-key", os.Getenv("API_KEY"), "Bearer token for API auth")
	poolSize := flag.Int("pool-size", envInt("BROWSER_POOL_SIZE", 5), "Headless Chrome pool size")
	logLevel := flag.String("log-level", envOr("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	flag.Parse()

	setupLogging(*logLevel)

	slog.Info("starting universalcrawl",
		"port", *port,
		"pool_size", *poolSize,
	)

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

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      api.NewServer(api.Config{APIKey: *apiKey}, orch),
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
