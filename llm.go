package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"

// refine uses Haiku to distill crawled markdown into content relevant to the user's query.
func refine(ctx context.Context, apiKey, crawledContent, query string) (string, error) {
	prompt := fmt.Sprintf(`You are a content distiller. You have been given raw markdown crawled from a website.
The user's query is: %q

Your job:
1. Remove navigation, boilerplate, ads, and irrelevant content
2. Keep only the sections and information relevant to the user's query
3. Preserve important details, code examples, URLs, and data
4. Output clean, structured markdown that a more capable model can use to answer the query

Crawled content:
%s`, query, crawledContent)

	return callAnthropic(ctx, apiKey, "claude-haiku-4-5-20251001", prompt, 4096)
}

// answer uses the specified model to produce a final answer from refined content.
func answer(ctx context.Context, apiKey, model, refinedContent, query string) (string, error) {
	prompt := fmt.Sprintf(`You have been given curated content from a website crawl. Use it to answer the user's query thoroughly and accurately.

User's query: %q

Website content:
%s`, query, refinedContent)

	return callAnthropic(ctx, apiKey, model, prompt, 8192)
}

func callAnthropic(ctx context.Context, apiKey, model, userMessage string, maxTokens int) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": userMessage},
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPI, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anthropic API error (%d): %s", resp.StatusCode, body)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	return result.Content[0].Text, nil
}
