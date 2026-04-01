package engines

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
	"github.com/madhavanp/universalcrawl/internal/browser"
)

// RodEngine scrapes pages using headless Chrome via the Rod library.
type RodEngine struct {
	pool *browser.Pool
}

// NewRodEngine creates a Chrome-based scrape engine backed by the given pool.
func NewRodEngine(pool *browser.Pool) *RodEngine {
	return &RodEngine{pool: pool}
}

// Name returns the engine identifier.
func (e *RodEngine) Name() string { return "rod" }

// Fetch navigates to a URL in headless Chrome and returns the rendered HTML.
func (e *RodEngine) Fetch(ctx context.Context, url string, opts FetchOptions) (*RawResult, error) {
	timeout := 30 * time.Second
	if opts.Timeout > 0 {
		timeout = time.Duration(opts.Timeout) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	b, err := e.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire browser: %w", err)
	}
	defer e.pool.Release(b)

	page, err := b.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	if opts.Mobile {
		err = page.Emulate(devices.IPhoneX)
		if err != nil {
			return nil, fmt.Errorf("emulate mobile: %w", err)
		}
	}

	if opts.Headers != nil {
		headers := make([]string, 0, len(opts.Headers)*2)
		for k, v := range opts.Headers {
			headers = append(headers, k, v)
		}
		_, err = page.SetExtraHeaders(headers)
		if err != nil {
			return nil, fmt.Errorf("set headers: %w", err)
		}
	}

	err = page.Context(ctx).Navigate(url)
	if err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	err = page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("wait load: %w", err)
	}

	if opts.WaitFor > 0 {
		time.Sleep(time.Duration(opts.WaitFor) * time.Millisecond)
	}

	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("get html: %w", err)
	}

	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("get page info: %w", err)
	}

	return &RawResult{
		HTML:       html,
		StatusCode: 200,
		Headers:    nil,
		URL:        info.URL,
	}, nil
}
