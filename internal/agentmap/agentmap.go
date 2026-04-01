package agentmap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/madhavanp/universalcrawl/internal/crawler"
	"github.com/madhavanp/universalcrawl/internal/llm"
	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/scraper"
)

// Service orchestrates the unified agent-map workflow.
type Service struct {
	crawler      *crawler.WebCrawler
	orchestrator *scraper.Orchestrator
}

// NewService creates a new agent-map service.
func NewService(crawler *crawler.WebCrawler, orch *scraper.Orchestrator) *Service {
	return &Service{
		crawler:      crawler,
		orchestrator: orch,
	}
}

// Execute runs the full agent-map pipeline: crawl, scrape, judge, assemble.
func (s *Service) Execute(ctx context.Context, req *models.AgentMapRequest, provider llm.Provider) (*models.AgentMapResponse, error) {
	// Phase 1: crawl and collect scraped pages
	pages, err := s.crawlAndScrape(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("crawl phase: %w", err)
	}

	if len(pages) == 0 {
		return &models.AgentMapResponse{
			URL:           req.URL,
			TotalPages:    0,
			SiteMapPrompt: fmt.Sprintf("# Site Map: %s\n## 0 pages discovered\n\nNo pages could be scraped from this site.\n", req.URL),
			Pages:         []models.PageSummary{},
		}, nil
	}

	// Phase 2: LLM judge in batches
	summaries, err := s.judgePages(ctx, provider, pages)
	if err != nil {
		return nil, fmt.Errorf("judge phase: %w", err)
	}

	// Phase 3: assemble response
	prompt := buildSiteMapPrompt(req.URL, summaries)

	return &models.AgentMapResponse{
		URL:           req.URL,
		TotalPages:    len(summaries),
		SiteMapPrompt: prompt,
		Pages:         summaries,
	}, nil
}

func (s *Service) crawlAndScrape(ctx context.Context, req *models.AgentMapRequest) ([]pageInput, error) {
	crawlReq := &models.CrawlRequest{
		URL:             req.URL,
		Limit:           req.Limit,
		MaxDepth:        req.MaxDepth,
		Formats:         []string{"markdown"},
		OnlyMainContent: true,
		IncludePaths:    req.IncludePaths,
		ExcludePaths:    req.ExcludePaths,
		AllowSubdomains: req.AllowSubdomains,
	}

	var mu sync.Mutex
	var pages []pageInput

	err := s.crawler.Crawl(ctx, crawlReq, func(result *models.ScrapeResult, total int) {
		if result == nil || result.Markdown == "" {
			return
		}
		mu.Lock()
		pages = append(pages, pageInput{
			URL:      result.URL,
			Title:    result.Metadata.Title,
			Markdown: result.Markdown,
		})
		mu.Unlock()
		slog.Debug("agent-map scraped page", "url", result.URL, "total_discovered", total)
	})

	return pages, err
}

func (s *Service) judgePages(ctx context.Context, provider llm.Provider, pages []pageInput) ([]models.PageSummary, error) {
	var allSummaries []models.PageSummary

	for i := 0; i < len(pages); i += batchSize {
		end := i + batchSize
		if end > len(pages) {
			end = len(pages)
		}
		batch := pages[i:end]

		slog.Debug("agent-map judging batch", "batch", i/batchSize+1, "pages", len(batch))

		summaries, err := judgeBatch(ctx, provider, batch)
		if err != nil {
			slog.Warn("agent-map judge batch failed, using fallbacks",
				"batch", i/batchSize+1, "error", err)
			for _, p := range batch {
				allSummaries = append(allSummaries, models.PageSummary{
					URL:         p.URL,
					Title:       p.Title,
					Summary:     "Summary unavailable (LLM judge failed).",
					ContentType: "other",
					Relevance:   5,
					KeyTopics:   []string{},
				})
			}
			continue
		}

		allSummaries = append(allSummaries, summaries...)
	}

	return allSummaries, nil
}
