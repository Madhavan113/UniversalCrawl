package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/madhavanp/universalcrawl/internal/llm"
	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/scraper"
)

// Extractor scrapes URLs and uses an LLM to extract structured data.
type Extractor struct {
	orchestrator *scraper.Orchestrator
	provider     llm.Provider
}

// NewExtractor creates an LLM-powered data extractor.
func NewExtractor(orch *scraper.Orchestrator, provider llm.Provider) *Extractor {
	return &Extractor{
		orchestrator: orch,
		provider:     provider,
	}
}

// Extract scrapes the given URLs and uses the LLM to extract structured data.
func (e *Extractor) Extract(ctx context.Context, req *models.ExtractRequest) (json.RawMessage, error) {
	// Scrape all URLs
	var contents []string
	for _, u := range req.URLs {
		result, err := e.orchestrator.Scrape(ctx, &models.ScrapeRequest{
			URL:             u,
			Formats:         []string{"markdown"},
			OnlyMainContent: true,
			Timeout:         30000,
		})
		if err != nil {
			continue
		}
		if result.Markdown != "" {
			contents = append(contents, fmt.Sprintf("--- Content from %s ---\n%s", u, result.Markdown))
		}
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("failed to scrape any of the provided URLs")
	}

	// Build LLM prompt
	systemPrompt := "You are a data extraction assistant. Extract structured data from the provided web content. Return ONLY valid JSON, no other text."
	if req.Schema != nil {
		systemPrompt += fmt.Sprintf("\n\nThe output must conform to this JSON schema:\n%s", string(*req.Schema))
	}

	userPrompt := req.Prompt + "\n\nWeb content:\n" + strings.Join(contents, "\n\n")

	response, err := e.provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		JSONMode:     true,
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("llm extraction: %w", err)
	}

	// Validate JSON
	response = strings.TrimSpace(response)
	if !json.Valid([]byte(response)) {
		return nil, fmt.Errorf("llm returned invalid JSON")
	}

	return json.RawMessage(response), nil
}
