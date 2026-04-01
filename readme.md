# UniversalCrawl

A single-binary web scraping tool for AI agents. Fetches, renders, and normalizes web pages into LLM-ready markdown through a Firecrawl-compatible REST API. Zero external dependencies -- no Redis, no Postgres, no Docker. One Go binary, one process.

UniversalCrawl loads pages in headless Chrome (via Rod/CDP), falls back to plain HTTP for static sites, and runs a transform pipeline that cleans HTML, extracts main content, converts to markdown, and pulls metadata and links.

## Quick Start

```bash
go build -o universalcrawl .
./universalcrawl
```

The server starts on port 3000. On first run, Rod downloads Chromium automatically. If Chrome can't launch, the server runs in fetch-only mode.

### Scrape a page

```bash
curl -X POST http://localhost:3000/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "formats": ["markdown"]}'
```

### Crawl a site

```bash
# Start async crawl
curl -X POST http://localhost:3000/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "limit": 50, "formats": ["markdown"]}'

# Poll status
curl http://localhost:3000/v1/crawl/crawl_abc123
```

### Discover URLs

```bash
curl -X POST http://localhost:3000/v1/map \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "limit": 1000}'
```

### Extract structured data (requires LLM key)

```bash
ANTHROPIC_API_KEY=sk-... ./universalcrawl

curl -X POST http://localhost:3000/v1/extract \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com/pricing"], "prompt": "Extract pricing tiers"}'
```

## API Reference

### POST /v1/scrape

Scrape a single URL. Synchronous.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | string | (required) | URL to scrape |
| `formats` | string[] | `["markdown"]` | `markdown`, `html`, `rawHtml`, `links`, `screenshot` |
| `onlyMainContent` | bool | `false` | Extract main content via readability |
| `waitFor` | int | `0` | ms to wait after page load |
| `timeout` | int | `30000` | Request timeout (ms) |
| `headers` | object | `{}` | Custom HTTP headers |
| `actions` | object[] | `[]` | Browser actions: click, type, scroll, wait |
| `mobile` | bool | `false` | Emulate mobile viewport |
| `includeTags` | string[] | `[]` | Keep only these elements |
| `excludeTags` | string[] | `[]` | Remove these elements |

### POST /v1/crawl

Start async multi-page crawl. Returns job ID for polling.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | string | (required) | Start URL |
| `limit` | int | `100` | Max pages to scrape |
| `maxDepth` | int | `0` | Max link depth (0 = unlimited) |
| `formats` | string[] | `["markdown"]` | Output formats |
| `includePaths` | string[] | `[]` | Glob patterns to include |
| `excludePaths` | string[] | `[]` | Glob patterns to exclude |
| `allowSubdomains` | bool | `false` | Crawl subdomains |
| `ignoreSitemap` | bool | `false` | Skip sitemap discovery |
| `delay` | int | `0` | Politeness delay (ms) between requests |

### GET /v1/crawl/{id}

Poll crawl status. Supports `?cursor=N&limit=N` for pagination.

### POST /v1/map

Discover URLs on a site without scraping content.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | string | (required) | Site URL |
| `search` | string | `""` | Filter URLs containing this string |
| `includeSubdomains` | bool | `false` | Include subdomain URLs |
| `limit` | int | `5000` | Max URLs to return |
| `sitemapOnly` | bool | `false` | Only use sitemap, don't crawl |

### POST /v1/extract

Scrape URLs + LLM structured extraction. Requires `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, or `OLLAMA_BASE_URL`.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `urls` | string[] | (required) | URLs to scrape |
| `prompt` | string | (required) | Extraction instruction |
| `schema` | object | `null` | JSON schema for output |

### POST /v1/search

Web search + scrape results. Requires `SEARXNG_ENDPOINT`.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `query` | string | (required) | Search query |
| `limit` | int | `5` | Number of results |
| `formats` | string[] | `["markdown"]` | Output formats |

### GET /health

Returns `{"success": true, "data": {"status": "ok"}}`.

## Configuration

| Env Var | CLI Flag | Default | Description |
|---------|----------|---------|-------------|
| `PORT` | `-port` | `3000` | Server port |
| `API_KEY` | `-api-key` | (empty) | Bearer auth token |
| `BROWSER_POOL_SIZE` | `-pool-size` | `5` | Chrome instances |
| `WORKERS` | `-workers` | `4` | Background goroutines |
| `DATA_DIR` | `-data-dir` | `./data` | Database directory |
| `LOG_LEVEL` | `-log-level` | `info` | Log verbosity |
| `ANTHROPIC_API_KEY` | -- | -- | Claude API (for extract) |
| `OPENAI_API_KEY` | -- | -- | OpenAI (for extract) |
| `OLLAMA_BASE_URL` | -- | -- | Local LLM (for extract) |
| `SEARXNG_ENDPOINT` | -- | -- | Search backend (for search) |

## Architecture

```
                ┌─────────────────────────────────────┐
                │           API Server (chi)           │
                │  /v1/scrape  /v1/crawl  /v1/map     │
                │  /v1/extract /v1/search              │
                └──────┬──────┬──────┬────────────────┘
                       │      │      │
          ┌────────────┘      │      └────────────┐
          v                   v                    v
 ┌────────────────┐  ┌───────────────┐  ┌─────────────────┐
 │ Scrape Engine  │  │   Crawler     │  │   LLM Layer     │
 │ (orchestrator) │  │ (multi-page)  │  │ (extract/search)│
 └───────┬────────┘  └──────┬────────┘  └────────┬────────┘
         │                  │                     │
 ┌───────┴────────┐        │              ┌──────┴───────┐
 │                │        │              │              │
┌┴──────┐   ┌────┴──┐     │         ┌────┴────┐  ┌─────┴────┐
│  Rod  │   │ Fetch │     │         │Anthropic│  │  OpenAI  │
│Engine │   │Engine │     │         │Provider │  │ Provider │
└───┬───┘   └───┬───┘     │         └─────────┘  └──────────┘
    │           │          │
    v           v          v
┌──────────────────────────────────┐
│       Transform Pipeline         │
│  clean → readability → markdown  │
└──────────────────────────────────┘
         │
         v
┌──────────────────────┐
│   bbolt Storage      │
│  jobs │ results │    │
│  cache │ config  │   │
└──────────────────────┘
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/go-rod/rod` | Headless Chrome via CDP |
| `github.com/go-chi/chi/v5` | HTTP router |
| `go.etcd.io/bbolt` | Embedded KV store |
| `github.com/go-shiori/go-readability` | Content extraction |
| `github.com/JohannesKaufmann/html-to-markdown` | HTML to Markdown |
| `github.com/PuerkitoBio/goquery` | HTML DOM manipulation |
