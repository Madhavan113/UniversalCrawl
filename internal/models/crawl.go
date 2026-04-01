package models

import "time"

// CrawlRequest defines parameters for a multi-page crawl job.
type CrawlRequest struct {
	URL             string   `json:"url"`
	Limit           int      `json:"limit"`
	MaxDepth        int      `json:"maxDepth"`
	Formats         []string `json:"formats"`
	OnlyMainContent bool     `json:"onlyMainContent"`
	IncludePaths    []string `json:"includePaths"`
	ExcludePaths    []string `json:"excludePaths"`
	AllowSubdomains bool     `json:"allowSubdomains"`
	IgnoreSitemap   bool     `json:"ignoreSitemap"`
	Delay           int      `json:"delay"`
}

// CrawlJob tracks the state of an async crawl operation.
type CrawlJob struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	URL       string    `json:"url"`
	Total     int       `json:"total"`
	Completed int       `json:"completed"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Crawl job status constants.
const (
	CrawlStatusScraping  = "scraping"
	CrawlStatusCompleted = "completed"
	CrawlStatusFailed    = "failed"
	CrawlStatusCancelled = "cancelled"
)
