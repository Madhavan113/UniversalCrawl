package models

// AgentMapRequest defines parameters for the unified agent-map endpoint.
type AgentMapRequest struct {
	URL             string   `json:"url"`
	LLMProvider     string   `json:"llmProvider"`
	LLMAPIKey       string   `json:"llmApiKey"`
	OllamaBaseURL   string   `json:"ollamaBaseUrl,omitempty"`
	Limit           int      `json:"limit"`
	MaxDepth        int      `json:"maxDepth"`
	IncludePaths    []string `json:"includePaths,omitempty"`
	ExcludePaths    []string `json:"excludePaths,omitempty"`
	AllowSubdomains bool     `json:"allowSubdomains"`
}

// AgentMapResponse is the top-level response for the agent-map endpoint.
type AgentMapResponse struct {
	URL           string        `json:"url"`
	TotalPages    int           `json:"totalPages"`
	SiteMapPrompt string        `json:"siteMapPrompt"`
	Pages         []PageSummary `json:"pages"`
}

// PageSummary is the LLM-judged summary of a single scraped page.
type PageSummary struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	ContentType string   `json:"contentType"`
	Relevance   int      `json:"relevance"`
	KeyTopics   []string `json:"keyTopics"`
}
