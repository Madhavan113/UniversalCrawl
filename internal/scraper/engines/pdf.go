package engines

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PDFEngine downloads PDFs and wraps the raw bytes as HTML-like content.
type PDFEngine struct {
	client *http.Client
}

// NewPDFEngine creates a PDF download engine.
func NewPDFEngine() *PDFEngine {
	return &PDFEngine{
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Name returns the engine identifier.
func (e *PDFEngine) Name() string { return "pdf" }

// Fetch downloads a PDF and returns its content as a simple HTML wrapper.
func (e *PDFEngine) Fetch(ctx context.Context, url string, opts FetchOptions) (*RawResult, error) {
	if !strings.HasSuffix(strings.ToLower(url), ".pdf") {
		return nil, fmt.Errorf("not a PDF URL")
	}

	timeout := 60 * time.Second
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

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download pdf: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20)) // 50MB limit
	if err != nil {
		return nil, fmt.Errorf("read pdf: %w", err)
	}

	// Wrap PDF content indication in HTML for the transform pipeline
	html := fmt.Sprintf("<html><body><p>[PDF document: %s, %d bytes]</p></body></html>", url, len(body))

	return &RawResult{
		HTML:       html,
		StatusCode: resp.StatusCode,
		URL:        url,
	}, nil
}
