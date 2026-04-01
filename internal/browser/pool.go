package browser

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Pool manages a fixed set of headless Chrome browser instances.
type Pool struct {
	mu       sync.Mutex
	browsers chan *rod.Browser
	size     int
	launchURL string
}

// NewPool creates a browser pool with the given number of Chrome instances.
func NewPool(size int) (*Pool, error) {
	if size < 1 {
		size = 1
	}

	l, err := launcher.New().Headless(true).Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	p := &Pool{
		browsers:  make(chan *rod.Browser, size),
		size:      size,
		launchURL: l,
	}

	for i := 0; i < size; i++ {
		b := rod.New().ControlURL(l)
		if err := b.Connect(); err != nil {
			p.Close()
			return nil, fmt.Errorf("connect browser %d: %w", i, err)
		}
		p.browsers <- b
	}

	slog.Info("browser pool started", "size", size)
	return p, nil
}

// Acquire returns a browser from the pool, blocking until one is available.
func (p *Pool) Acquire(ctx context.Context) (*rod.Browser, error) {
	select {
	case b := <-p.browsers:
		return b, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns a browser to the pool.
func (p *Pool) Release(b *rod.Browser) {
	p.browsers <- b
}

// Close shuts down all browsers in the pool.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.browsers)
	for b := range p.browsers {
		if err := b.Close(); err != nil {
			slog.Warn("error closing browser", "error", err)
		}
	}
}
