package crawler

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DiscoverURLs finds URLs on a site via sitemap.xml and robots.txt.
func DiscoverURLs(ctx context.Context, siteURL string) ([]string, error) {
	u, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	var urls []string

	// Try sitemap.xml first
	sitemapURL := fmt.Sprintf("%s://%s/sitemap.xml", u.Scheme, u.Host)
	sitemapURLs, err := parseSitemap(ctx, sitemapURL)
	if err == nil {
		urls = append(urls, sitemapURLs...)
	}

	// Check robots.txt for additional sitemaps
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", u.Scheme, u.Host)
	robotsSitemaps, err := parseSitemapsFromRobots(ctx, robotsURL)
	if err == nil {
		for _, sm := range robotsSitemaps {
			smURLs, err := parseSitemap(ctx, sm)
			if err == nil {
				urls = append(urls, smURLs...)
			}
		}
	}

	return dedup(urls), nil
}

type sitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc string `xml:"loc"`
}

type urlSet struct {
	XMLName xml.Name   `xml:"urlset"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc string `xml:"loc"`
}

func parseSitemap(ctx context.Context, sitemapURL string) ([]string, error) {
	body, err := httpGet(ctx, sitemapURL)
	if err != nil {
		return nil, err
	}

	// Try as sitemap index first
	var idx sitemapIndex
	if err := xml.Unmarshal(body, &idx); err == nil && len(idx.Sitemaps) > 0 {
		var urls []string
		for _, sm := range idx.Sitemaps {
			sub, err := parseSitemap(ctx, sm.Loc)
			if err == nil {
				urls = append(urls, sub...)
			}
		}
		return urls, nil
	}

	// Try as urlset
	var us urlSet
	if err := xml.Unmarshal(body, &us); err != nil {
		return nil, fmt.Errorf("parse sitemap xml: %w", err)
	}

	urls := make([]string, 0, len(us.URLs))
	for _, u := range us.URLs {
		if u.Loc != "" {
			urls = append(urls, u.Loc)
		}
	}
	return urls, nil
}

func parseSitemapsFromRobots(ctx context.Context, robotsURL string) ([]string, error) {
	body, err := httpGet(ctx, robotsURL)
	if err != nil {
		return nil, err
	}

	var sitemaps []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "sitemap:") {
			sm := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			if sm != "" {
				sitemaps = append(sitemaps, sm)
			}
		}
	}
	return sitemaps, nil
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "UniversalCrawl/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, url)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}

func dedup(urls []string) []string {
	seen := make(map[string]struct{}, len(urls))
	result := make([]string, 0, len(urls))
	for _, u := range urls {
		if _, ok := seen[u]; !ok {
			seen[u] = struct{}{}
			result = append(result, u)
		}
	}
	return result
}
