package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CrawledPage struct {
	URL      string
	Markdown string
}

// crawlSite starts a Cloudflare /crawl job, polls until done, and returns markdown for each page.
func crawlSite(ctx context.Context, apiToken, accountID, url string, maxPages int) ([]CrawledPage, error) {
	base := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering/crawl", accountID)

	// Start the crawl
	reqBody, _ := json.Marshal(map[string]any{
		"url":     url,
		"limit":   maxPages,
		"formats": []string{"markdown"},
		"rejectResourceTypes": []string{"image", "media", "font", "stylesheet"},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", base, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("crawl start failed (%d): %s", resp.StatusCode, body)
	}

	var startResp struct {
		Success bool   `json:"success"`
		Result  string `json:"result"`
	}
	if err := json.Unmarshal(body, &startResp); err != nil {
		return nil, fmt.Errorf("parse start response: %w", err)
	}
	if !startResp.Success || startResp.Result == "" {
		return nil, fmt.Errorf("crawl start unsuccessful: %s", body)
	}

	jobID := startResp.Result
	// Poll until done
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}

		pages, done, err := pollCrawl(ctx, apiToken, base, jobID)
		if err != nil {
			return nil, err
		}
		if done {
			return pages, nil
		}
		_ = pages
	}
}

// pollCrawl fetches the current state of a crawl job. Returns all completed pages and whether the job is done.
func pollCrawl(ctx context.Context, apiToken, base, jobID string) ([]CrawledPage, bool, error) {
	url := fmt.Sprintf("%s/%s", base, jobID)

	var allPages []CrawledPage
	cursor := ""
	done := false

	for {
		reqURL := url
		sep := "?"
		if cursor != "" {
			reqURL += sep + "cursor=" + cursor
			sep = "&"
		}
		reqURL += sep + "limit=50"

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, false, err
		}
		req.Header.Set("Authorization", "Bearer "+apiToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, false, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, false, fmt.Errorf("poll failed (%d): %s", resp.StatusCode, body)
		}

		var pollResp struct {
			Success bool `json:"success"`
			Result  struct {
				ID      string `json:"id"`
				Status  string `json:"status"`
				Total   int    `json:"total"`
				Records []struct {
					URL      string `json:"url"`
					Status   string `json:"status"`
					Markdown string `json:"markdown"`
				} `json:"records"`
				Cursor any `json:"cursor"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &pollResp); err != nil {
			return nil, false, fmt.Errorf("parse poll response: %w", err)
		}

		for _, r := range pollResp.Result.Records {
			if r.Status == "completed" && r.Markdown != "" {
				allPages = append(allPages, CrawledPage{URL: r.URL, Markdown: r.Markdown})
			}
		}

		status := pollResp.Result.Status
		done = status != "running"

		// Handle pagination cursor
		switch v := pollResp.Result.Cursor.(type) {
		case float64:
			if int(v) > 0 && !done {
				cursor = fmt.Sprintf("%d", int(v))
				continue
			}
		case string:
			if v != "" && v != "0" {
				cursor = v
				continue
			}
		}
		break
	}

	return allPages, done, nil
}
