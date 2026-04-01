package api

import (
	"encoding/json"
	"net/http"

	"github.com/madhavanp/universalcrawl/internal/extract"
	"github.com/madhavanp/universalcrawl/internal/models"
)

type extractHandler struct {
	extractor *extract.Extractor
}

// HandleExtract processes POST /v1/extract — scrape + LLM extraction.
func (h *extractHandler) HandleExtract(w http.ResponseWriter, r *http.Request) {
	if h.extractor == nil {
		writeError(w, http.StatusServiceUnavailable, "extract requires an LLM provider (set ANTHROPIC_API_KEY or OPENAI_API_KEY)")
		return
	}

	var req models.ExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if len(req.URLs) == 0 {
		writeError(w, http.StatusBadRequest, "urls is required")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	data, err := h.extractor.Extract(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, data)
}
