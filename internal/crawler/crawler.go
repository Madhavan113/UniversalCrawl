package crawler

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/scraper"
	"github.com/madhavanp/universalcrawl/internal/scraper/transform"
)

// WebCrawler drives multi-page crawling using the scrape engine.
type WebCrawler struct {
	orchestrator *scraper.Orchestrator
	concurrency  int
}

// NewWebCrawler creates a crawler with the given scrape orchestrator.
func NewWebCrawler(orch *scraper.Orchestrator, concurrency int) *WebCrawler {
	if concurrency < 1 {
		concurrency = 4
	}
	return &WebCrawler{
		orchestrator: orch,
		concurrency:  concurrency,
	}
}

// Crawl executes a multi-page crawl. For each scraped page it calls onResult
// with the scrape result and the current total number of discovered URLs.
func (c *WebCrawler) Crawl(ctx context.Context, req *models.CrawlRequest, onResult func(*models.ScrapeResult, int)) error {
	origin, err := url.Parse(req.URL)
	if err != nil {
		return err
	}

	filter := &FilterConfig{
		Origin:          origin,
		MaxDepth:        req.MaxDepth,
		IncludePaths:    req.IncludePaths,
		ExcludePaths:    req.ExcludePaths,
		AllowSubdomains: req.AllowSubdomains,
		Limit:           req.Limit,
	}

	state := NewState()

	// Seed with discovered URLs
	if !req.IgnoreSitemap {
		discovered, err := DiscoverURLs(ctx, req.URL)
		if err != nil {
			slog.Debug("discovery failed, will crawl from seed URL", "error", err)
		}
		for _, u := range discovered {
			if filter.Accept(u, 0) {
				state.Enqueue(u, 0)
			}
		}
	}

	// Always seed the start URL
	state.Enqueue(req.URL, 0)

	delay := time.Duration(req.Delay) * time.Millisecond
	sem := make(chan struct{}, c.concurrency)

	for {
		pageURL, depth, ok := state.Dequeue()
		if !ok {
			break
		}

		if req.Limit > 0 && state.Stats() > req.Limit {
			state.Done()
			break
		}

		sem <- struct{}{}

		go func(pageURL string, depth int) {
			defer func() { <-sem; state.Done() }()

			if delay > 0 {
				time.Sleep(delay)
			}

			scrapeReq := &models.ScrapeRequest{
				URL:             pageURL,
				Formats:         req.Formats,
				OnlyMainContent: req.OnlyMainContent,
				Timeout:         30000,
			}

			result, err := c.orchestrator.Scrape(ctx, scrapeReq)
			if err != nil {
				slog.Warn("crawl scrape failed", "url", pageURL, "error", err)
				return
			}

			links, _ := transform.ExtractLinks(result.HTML, pageURL)
			if result.HTML == "" && result.Markdown != "" {
				links, _ = transform.ExtractLinks(result.RawHTML, pageURL)
			}

			for _, link := range links {
				if filter.Accept(link, depth+1) {
					state.Enqueue(link, depth+1)
				}
			}

			if onResult != nil {
				onResult(result, state.Stats())
			}
		}(pageURL, depth)
	}

	return nil
}

// Map discovers URLs on a site without scraping content.
func (c *WebCrawler) Map(ctx context.Context, req *models.MapRequest) ([]string, error) {
	origin, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	var urls []string

	// Sitemap-based discovery
	discovered, err := DiscoverURLs(ctx, req.URL)
	if err != nil {
		slog.Debug("sitemap discovery failed", "error", err)
	}
	urls = append(urls, discovered...)

	// If sitemap-only, return now
	if req.SitemapOnly {
		return filterAndLimit(urls, origin, req), nil
	}

	// Also crawl links from the seed page
	scrapeReq := &models.ScrapeRequest{
		URL:     req.URL,
		Formats: []string{"links"},
		Timeout: 15000,
	}
	result, err := c.orchestrator.Scrape(ctx, scrapeReq)
	if err == nil && result != nil {
		urls = append(urls, result.Links...)
	}

	return filterAndLimit(urls, origin, req), nil
}

func filterAndLimit(urls []string, origin *url.URL, req *models.MapRequest) []string {
	seen := make(map[string]struct{})
	var filtered []string

	for _, u := range urls {
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}

		parsed, err := url.Parse(u)
		if err != nil {
			continue
		}

		// Origin check
		if !req.IncludeSubdomains && parsed.Hostname() != origin.Hostname() {
			continue
		}

		// Search filter
		if req.Search != "" {
			if !containsIgnoreCase(u, req.Search) {
				continue
			}
		}

		filtered = append(filtered, u)
		if req.Limit > 0 && len(filtered) >= req.Limit {
			break
		}
	}
	return filtered
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		contains(toLower(s), toLower(substr))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	return len(substr) == 0 || indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
