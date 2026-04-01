package api

import (
	"encoding/json"
	"net/http"

	"github.com/madhavanp/universalcrawl/internal/agentmap"
	"github.com/madhavanp/universalcrawl/internal/llm"
	"github.com/madhavanp/universalcrawl/internal/models"
)

type agentMapHandler struct {
	service *agentmap.Service
}

// HandleAgentMap processes POST /v1/agent-map — unified crawl + judge + site map.
func (h *agentMapHandler) HandleAgentMap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL             string   `json:"url"`
		LLMProvider     string   `json:"llmProvider"`
		LLMAPIKey       string   `json:"llmApiKey"`
		OllamaBaseURL   string   `json:"ollamaBaseUrl"`
		Limit           int      `json:"limit"`
		MaxDepth        int      `json:"maxDepth"`
		IncludePaths    []string `json:"includePaths"`
		ExcludePaths    []string `json:"excludePaths"`
		AllowSubdomains bool     `json:"allowSubdomains"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if req.LLMProvider == "" {
		writeError(w, http.StatusBadRequest, "llmProvider is required (anthropic, openai, or ollama)")
		return
	}

	provider, err := llm.NewProviderFromRequest(req.LLMProvider, req.LLMAPIKey, req.OllamaBaseURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Defaults
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}
	maxDepth := req.MaxDepth
	if maxDepth == 0 {
		maxDepth = 3
	}

	modelReq := toAgentMapRequest(req.URL, req.LLMProvider, req.LLMAPIKey, req.OllamaBaseURL,
		limit, maxDepth, req.IncludePaths, req.ExcludePaths, req.AllowSubdomains)

	result, err := h.service.Execute(r.Context(), modelReq, provider)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, result)
}

func toAgentMapRequest(url, provider, apiKey, ollamaURL string, limit, maxDepth int,
	include, exclude []string, allowSub bool) *models.AgentMapRequest {
	return &models.AgentMapRequest{
		URL:             url,
		LLMProvider:     provider,
		LLMAPIKey:       apiKey,
		OllamaBaseURL:   ollamaURL,
		Limit:           limit,
		MaxDepth:        maxDepth,
		IncludePaths:    include,
		ExcludePaths:    exclude,
		AllowSubdomains: allowSub,
	}
}
