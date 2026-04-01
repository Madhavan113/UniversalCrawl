package engines

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FetchEngine scrapes pages using plain HTTP GET requests.
type FetchEngine struct {
	client *http.Client
}

// NewFetchEngine creates an HTTP-based scrape engine.
func NewFetchEngine() *FetchEngine {
	return &FetchEngine{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the engine identifier.
func (e *FetchEngine) Name() string { return "fetch" }

// Fetch performs an HTTP GET and returns the raw HTML.
func (e *FetchEngine) Fetch(ctx context.Context, url string, opts FetchOptions) (*RawResult, error) {
	timeout := 30 * time.Second
	if opts.Timeout > 0 {
		timeout = time.Duration(opts.Timeout) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "UniversalCrawl/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	headers := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	return &RawResult{
		HTML:       string(body),
		StatusCode: resp.StatusCode,
		Headers:    headers,
		URL:        resp.Request.URL.String(),
	}, nil
}
