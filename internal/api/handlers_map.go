package api

import (
	"encoding/json"
	"net/http"

	"github.com/madhavanp/universalcrawl/internal/crawler"
	"github.com/madhavanp/universalcrawl/internal/models"
)

type mapHandler struct {
	crawler *crawler.WebCrawler
}

// HandleMap processes POST /v1/map — discovers all URLs on a site.
func (h *mapHandler) HandleMap(w http.ResponseWriter, r *http.Request) {
	var req models.MapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if req.Limit == 0 {
		req.Limit = 5000
	}

	links, err := h.crawler.Map(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"links":   links,
	})
}
