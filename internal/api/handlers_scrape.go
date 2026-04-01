package api

import (
	"encoding/json"
	"net/http"

	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/scraper"
)

type scrapeHandler struct {
	orchestrator *scraper.Orchestrator
}

// HandleScrape processes POST /v1/scrape requests.
func (h *scrapeHandler) HandleScrape(w http.ResponseWriter, r *http.Request) {
	var req models.ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	if len(req.Formats) == 0 {
		req.Formats = []string{"markdown"}
	}

	if req.Timeout == 0 {
		req.Timeout = 30000
	}

	result, err := h.orchestrator.Scrape(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, result)
}
