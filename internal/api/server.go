package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/madhavanp/universalcrawl/internal/agentmap"
	"github.com/madhavanp/universalcrawl/internal/crawler"
	"github.com/madhavanp/universalcrawl/internal/extract"
	"github.com/madhavanp/universalcrawl/internal/jobs"
	"github.com/madhavanp/universalcrawl/internal/scraper"
	"github.com/madhavanp/universalcrawl/internal/search"
	"github.com/madhavanp/universalcrawl/internal/storage"
)

// Config holds API server configuration.
type Config struct {
	APIKey string
}

// Deps holds service-layer dependencies injected into handlers.
type Deps struct {
	Orchestrator *scraper.Orchestrator
	Crawler      *crawler.WebCrawler
	Store        storage.Store
	Queue        *jobs.Queue
	Extractor    *extract.Extractor
	Searcher     *search.Searcher
	AgentMap     *agentmap.Service
}

// NewServer creates and configures the HTTP router.
func NewServer(cfg Config, deps Deps) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(recoveryMiddleware)
	r.Use(loggingMiddleware)

	// Health check (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, map[string]string{"status": "ok"})
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware(cfg.APIKey))
	sh := &scrapeHandler{orchestrator: deps.Orchestrator}
	r.Post("/v1/scrape", sh.HandleScrape)

	// Crawl
	ch := &crawlHandler{
		crawler: deps.Crawler,
		store:   deps.Store,
		queue:   deps.Queue,
	}
	r.Post("/v1/crawl", ch.HandleCrawlStart)
	r.Get("/v1/crawl/{id}", ch.HandleCrawlStatus)

	// Map
	mh := &mapHandler{crawler: deps.Crawler}
	r.Post("/v1/map", mh.HandleMap)

	// Extract
	eh := &extractHandler{extractor: deps.Extractor}
	r.Post("/v1/extract", eh.HandleExtract)

	// Search
	srch := &searchHandler{searcher: deps.Searcher}
	r.Post("/v1/search", srch.HandleSearch)

	// Agent Map
	amh := &agentMapHandler{service: deps.AgentMap}
	r.Post("/v1/agent-map", amh.HandleAgentMap)
	})

	return r
}
