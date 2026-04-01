package agentmap

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/madhavanp/universalcrawl/internal/llm"
	"github.com/madhavanp/universalcrawl/internal/models"
)

const maxContentLen = 2000
const batchSize = 5

const judgeSystemPrompt = `You are a website content analyst. For each page provided, return a JSON array where each element has:
- "url": the page URL (exactly as provided)
- "summary": 1-2 sentence description of what this page contains
- "contentType": one of "documentation", "api-reference", "blog", "landing-page", "about", "pricing", "legal", "support", "changelog", "other"
- "relevance": integer 1-10 (10 = most useful for an AI agent trying to understand and use this site)
- "keyTopics": array of 3-5 keywords

Return ONLY the JSON array, no other text.`

type pageInput struct {
	URL      string
	Title    string
	Markdown string
}

// judgeBatch sends a batch of pages to the LLM for quality assessment.
func judgeBatch(ctx context.Context, provider llm.Provider, pages []pageInput) ([]models.PageSummary, error) {
	var sb strings.Builder
	for i, p := range pages {
		content := p.Markdown
		if len(content) > maxContentLen {
			content = content[:maxContentLen] + "\n[truncated]"
		}
		fmt.Fprintf(&sb, "--- PAGE %d ---\nURL: %s\nTitle: %s\nContent:\n%s\n\n", i+1, p.URL, p.Title, content)
	}

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: judgeSystemPrompt,
		UserPrompt:   sb.String(),
		JSONMode:     true,
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("llm judge call: %w", err)
	}

	// Strip markdown code fences if present
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		if idx := strings.Index(resp[3:], "\n"); idx >= 0 {
			resp = resp[3+idx+1:]
		}
		if strings.HasSuffix(resp, "```") {
			resp = resp[:len(resp)-3]
		}
		resp = strings.TrimSpace(resp)
	}

	var summaries []models.PageSummary
	if err := json.Unmarshal([]byte(resp), &summaries); err != nil {
		return nil, fmt.Errorf("parse llm judge response: %w", err)
	}

	return summaries, nil
}

// buildSiteMapPrompt assembles the structured prompt from page summaries.
func buildSiteMapPrompt(siteURL string, pages []models.PageSummary) string {
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Relevance > pages[j].Relevance
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Site Map: %s\n", siteURL)
	fmt.Fprintf(&sb, "## %d pages discovered\n\n", len(pages))

	type tier struct {
		name  string
		min   int
		max   int
		pages []models.PageSummary
	}

	tiers := []tier{
		{name: "High Relevance", min: 8, max: 10},
		{name: "Medium Relevance", min: 5, max: 7},
		{name: "Low Relevance", min: 1, max: 4},
	}

	for i := range tiers {
		for _, p := range pages {
			if p.Relevance >= tiers[i].min && p.Relevance <= tiers[i].max {
				tiers[i].pages = append(tiers[i].pages, p)
			}
		}
	}

	for _, t := range tiers {
		if len(t.pages) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "### %s\n", t.name)
		for _, p := range t.pages {
			topics := strings.Join(p.KeyTopics, ", ")
			fmt.Fprintf(&sb, "- [%s](%s) — %s Topics: %s\n", p.Title, p.URL, p.Summary, topics)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
