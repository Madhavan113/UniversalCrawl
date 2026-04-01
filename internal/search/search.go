package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/scraper"
)

// Searcher performs web searches and scrapes the results.
type Searcher struct {
	orchestrator   *scraper.Orchestrator
	searxngBaseURL string
	client         *http.Client
}

// NewSearcher creates a web search + scrape service.
func NewSearcher(orch *scraper.Orchestrator, searxngURL string) *Searcher {
	return &Searcher{
		orchestrator:   orch,
		searxngBaseURL: searxngURL,
		client:         &http.Client{Timeout: 15 * time.Second},
	}
}

// Search queries the web and scrapes top results.
func (s *Searcher) Search(ctx context.Context, req *models.SearchRequest) ([]*models.ScrapeResult, error) {
	if s.searxngBaseURL == "" {
		return nil, fmt.Errorf("search requires SEARXNG_ENDPOINT to be configured")
	}

	limit := req.Limit
	if limit == 0 {
		limit = 5
	}

	urls, err := s.searchURLs(ctx, req.Query, req.Lang, limit)
	if err != nil {
		return nil, fmt.Errorf("web search: %w", err)
	}

	formats := req.Formats
	if len(formats) == 0 {
		formats = []string{"markdown"}
	}

	var results []*models.ScrapeResult
	for _, u := range urls {
		result, err := s.orchestrator.Scrape(ctx, &models.ScrapeRequest{
			URL:             u,
			Formats:         formats,
			OnlyMainContent: req.OnlyMainContent,
			Timeout:         15000,
		})
		if err != nil {
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

func (s *Searcher) searchURLs(ctx context.Context, query string, lang string, limit int) ([]string, error) {
	params := url.Values{
		"q":      {query},
		"format": {"json"},
	}
	if lang != "" {
		params.Set("language", lang)
	}

	searchURL := s.searxngBaseURL + "/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "UniversalCrawl/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Results []struct {
			URL string `json:"url"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse search results: %w", err)
	}

	var urls []string
	for _, r := range result.Results {
		if r.URL != "" {
			urls = append(urls, r.URL)
			if len(urls) >= limit {
				break
			}
		}
	}

	return urls, nil
}
