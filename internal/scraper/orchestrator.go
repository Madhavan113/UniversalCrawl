package scraper

import (
	"context"
	"log/slog"

	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/scraper/engines"
	"github.com/madhavanp/universalcrawl/internal/scraper/transform"
)

// Orchestrator coordinates scrape engines and the transform pipeline.
type Orchestrator struct {
	engines []engines.Engine
}

// NewOrchestrator creates a scraper with the given engines tried in order.
func NewOrchestrator(engs ...engines.Engine) *Orchestrator {
	return &Orchestrator{engines: engs}
}

// Scrape fetches a URL using the engine chain and transforms the result.
func (o *Orchestrator) Scrape(ctx context.Context, req *models.ScrapeRequest) (*models.ScrapeResult, error) {
	// Check if screenshot format is requested
	wantScreenshot := false
	for _, f := range req.Formats {
		if f == "screenshot" {
			wantScreenshot = true
			break
		}
	}

	opts := engines.FetchOptions{
		WaitFor:    req.WaitFor,
		Timeout:    req.Timeout,
		Headers:    req.Headers,
		Mobile:     req.Mobile,
		Actions:    req.Actions,
		Screenshot: wantScreenshot,
	}

	var raw *engines.RawResult
	var lastErr error

	for _, eng := range o.engines {
		var err error
		raw, err = eng.Fetch(ctx, req.URL, opts)
		if err != nil {
			slog.Warn("engine failed, trying next",
				"engine", eng.Name(),
				"url", req.URL,
				"error", err,
			)
			lastErr = err
			continue
		}
		slog.Info("engine succeeded", "engine", eng.Name(), "url", req.URL)
		break
	}

	if raw == nil {
		return nil, &models.ScrapeError{
			URL:       req.URL,
			Stage:     "fetch",
			Cause:     lastErr,
			Retryable: true,
		}
	}

	transformOpts := transform.Options{
		Formats:         req.Formats,
		OnlyMainContent: req.OnlyMainContent,
		ExcludeTags:     req.ExcludeTags,
		IncludeTags:     req.IncludeTags,
	}

	result, err := transform.Run(raw, transformOpts)
	if err != nil {
		return nil, &models.ScrapeError{
			URL:       req.URL,
			Stage:     "transform",
			Cause:     err,
			Retryable: false,
		}
	}

	return result, nil
}
