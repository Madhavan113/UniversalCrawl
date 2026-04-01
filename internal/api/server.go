package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/madhavanp/universalcrawl/internal/scraper"
)

// Config holds API server configuration.
type Config struct {
	APIKey string
}

// NewServer creates and configures the HTTP router.
func NewServer(cfg Config, orch *scraper.Orchestrator) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(recoveryMiddleware)
	r.Use(loggingMiddleware)
	r.Use(authMiddleware(cfg.APIKey))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, map[string]string{"status": "ok"})
	})

	// Scrape handler
	sh := &scrapeHandler{orchestrator: orch}
	r.Post("/v1/scrape", sh.HandleScrape)

	return r
}
