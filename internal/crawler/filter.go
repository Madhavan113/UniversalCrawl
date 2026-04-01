package crawler

import (
	"net/url"
	"path"
	"strings"
)

// FilterConfig controls which URLs pass through the crawler's filter.
type FilterConfig struct {
	Origin          *url.URL
	MaxDepth        int
	IncludePaths    []string
	ExcludePaths    []string
	AllowSubdomains bool
	Limit           int
}

// Accept returns true if the URL passes all filter criteria.
func (f *FilterConfig) Accept(rawURL string, depth int) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Scheme check
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Origin check
	if !f.sameOrigin(u) {
		return false
	}

	// Depth check
	if f.MaxDepth > 0 && depth > f.MaxDepth {
		return false
	}

	// Include paths
	if len(f.IncludePaths) > 0 {
		matched := false
		for _, pattern := range f.IncludePaths {
			if matchGlob(pattern, u.Path) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Exclude paths
	for _, pattern := range f.ExcludePaths {
		if matchGlob(pattern, u.Path) {
			return false
		}
	}

	return true
}

func (f *FilterConfig) sameOrigin(u *url.URL) bool {
	if f.AllowSubdomains {
		return u.Hostname() == f.Origin.Hostname() ||
			strings.HasSuffix(u.Hostname(), "."+f.Origin.Hostname())
	}
	return u.Hostname() == f.Origin.Hostname()
}

// matchGlob does simple path glob matching supporting * wildcards.
func matchGlob(pattern, urlPath string) bool {
	matched, _ := path.Match(pattern, urlPath)
	if matched {
		return true
	}
	// Also try with trailing wildcard for directory patterns
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if strings.HasPrefix(urlPath, prefix) {
			return true
		}
	}
	return false
}
