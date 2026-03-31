package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	url := flag.String("url", "", "URL to crawl")
	query := flag.String("query", "", "What you want to know about the crawled content")
	maxPages := flag.Int("pages", 10, "Max pages to crawl")
	model := flag.String("model", "claude-haiku-4-5-20251001", "Anthropic model for final answer")
	flag.Parse()

	if *url == "" || *query == "" {
		fmt.Fprintf(os.Stderr, "Usage: universalcrawl -url <URL> -query <QUERY>\n")
		os.Exit(1)
	}

	cfToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	cfAccount := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")

	if cfToken == "" || cfAccount == "" {
		fmt.Fprintf(os.Stderr, "Set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID\n")
		os.Exit(1)
	}
	if anthropicKey == "" {
		fmt.Fprintf(os.Stderr, "Set ANTHROPIC_API_KEY\n")
		os.Exit(1)
	}

	ctx := context.Background()

	// Step 1: Crawl
	fmt.Fprintf(os.Stderr, "Crawling %s (max %d pages)...\n", *url, *maxPages)
	pages, err := crawlSite(ctx, cfToken, cfAccount, *url, *maxPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Crawl failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Got %d pages of markdown\n", len(pages))

	// Step 2: Combine markdown
	var combined strings.Builder
	for _, p := range pages {
		combined.WriteString(fmt.Sprintf("## Source: %s\n\n%s\n\n---\n\n", p.URL, p.Markdown))
	}

	// Step 3: Refine with cheap model — distill the crawled content into what's relevant
	fmt.Fprintf(os.Stderr, "Refining content with Haiku...\n")
	refined, err := refine(ctx, anthropicKey, combined.String(), *query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Refine failed: %v\n", err)
		os.Exit(1)
	}

	// Step 4: Answer with the chosen model
	fmt.Fprintf(os.Stderr, "Generating answer with %s...\n", *model)
	answer, err := answer(ctx, anthropicKey, *model, refined, *query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Answer failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(answer)
}
