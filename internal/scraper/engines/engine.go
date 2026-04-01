package engines

import "context"

// RawResult holds the raw output from a scrape engine before transformation.
type RawResult struct {
	HTML       string
	StatusCode int
	Headers    map[string]string
	URL        string
}

// Engine defines the interface for fetching raw HTML from a URL.
type Engine interface {
	// Name returns the engine identifier for logging.
	Name() string
	// Fetch retrieves raw HTML and metadata from the given URL.
	Fetch(ctx context.Context, url string, opts FetchOptions) (*RawResult, error)
}

// FetchOptions controls engine behavior for a single fetch.
type FetchOptions struct {
	WaitFor     int               // milliseconds to wait after page load
	Timeout     int               // milliseconds before aborting
	Headers     map[string]string // custom request headers
	Mobile      bool              // emulate mobile viewport
}
