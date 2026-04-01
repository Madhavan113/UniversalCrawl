package api

import (
	"encoding/json"
	"net/http"

	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/search"
)

type searchHandler struct {
	searcher *search.Searcher
}

// HandleSearch processes POST /v1/search — web search + scrape.
func (h *searchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if h.searcher == nil {
		writeError(w, http.StatusServiceUnavailable, "search requires SEARXNG_ENDPOINT to be configured")
		return
	}

	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	results, err := h.searcher.Search(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, results)
}
