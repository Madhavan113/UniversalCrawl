# CLAUDE.md -- UniversalCrawl

## Project Overview

UniversalCrawl is a single-binary Go web scraping tool that exposes a Firecrawl-compatible REST API. It gives AI agents the ability to fetch, render, and normalize web content into LLM-ready markdown, structured metadata, and link graphs. Zero external dependencies beyond the binary itself -- no Redis, no Postgres, no Docker Compose. One binary, one process, one API.

The module path is `github.com/madhavanp/universalcrawl`. The binary entrypoint is `main.go` at the project root.

## Build and Run

```
go build -o universalcrawl .
./universalcrawl
./universalcrawl -port 8080 -api-key mysecret -pool-size 3 -workers 8 -log-level debug
```

Chrome is auto-downloaded by Rod on first run. If Chrome cannot be launched, the server starts in fetch-only mode and logs a warning.

## Configuration

| Flag | Env Var | Default | Purpose |
|------|---------|---------|---------|
| `-port` | `PORT` | `3000` | API server port |
| `-api-key` | `API_KEY` | (empty = no auth) | Bearer token |
| `-pool-size` | `BROWSER_POOL_SIZE` | `5` | Chrome instances |
| `-workers` | `WORKERS` | `4` | Background goroutines |
| `-data-dir` | `DATA_DIR` | `./data` | bbolt database path |
| `-log-level` | `LOG_LEVEL` | `info` | debug/info/warn/error |

Optional: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `OLLAMA_BASE_URL` (for /v1/extract), `SEARXNG_ENDPOINT` (for /v1/search). Note: `/v1/agent-map` accepts LLM keys per-request, no server env vars needed.

## Architecture

```
Browser Pool -> Scrape Engines -> Transform Pipeline -> API Handlers
                    ↑                                       ↑
               Rod | Fetch | PDF                     chi router + middleware
```

### Package Map

```
main.go                              -- entrypoint, CLI flags, wiring, graceful shutdown
internal/
  models/                            -- shared types: ScrapeRequest, CrawlJob, typed errors
  storage/store.go                   -- Store interface + BoltStore (bbolt), cache, job CRUD
  browser/
    pool.go                          -- channel-based Rod browser pool (acquire/release)
    actions.go                       -- click, type, scroll, wait, press actions on Rod pages
  scraper/
    orchestrator.go                  -- engine fallback chain (Rod -> Fetch) + transform
    engines/
      engine.go                      -- Engine interface, RawResult, FetchOptions
      rod.go                         -- headless Chrome via go-rod, actions, screenshots
      fetch.go                       -- plain net/http GET, 10MB limit
      pdf.go                         -- PDF download engine
    transform/
      pipeline.go                    -- composes steps based on requested formats
      clean.go + clean_test.go       -- HTML cleaning (script/style/nav removal)
      readability.go + _test.go      -- main content extraction (go-readability)
      markdown.go + _test.go         -- HTML -> Markdown conversion
      metadata.go                    -- title, description, OG tags, language
      links.go                       -- href extraction with URL resolution
  crawler/
    crawler.go                       -- WebCrawler: multi-page crawl + Map
    discovery.go                     -- sitemap.xml + robots.txt parsing
    filter.go                        -- URL filtering (origin, depth, globs)
    state.go                         -- crawl state (visited/queued with sync.Cond)
  jobs/queue.go                      -- in-process channel-based job queue
  llm/
    provider.go                      -- Provider interface
    factory.go                       -- per-request provider creation from API key
    anthropic.go                     -- Claude API client
    openai.go                        -- OpenAI client
    ollama.go                        -- Ollama local client
  extract/extract.go                 -- scrape + LLM structured extraction
  search/search.go                   -- SearXNG search + scrape results
  agentmap/
    agentmap.go                      -- unified agent-map: crawl + scrape + judge + prompt
    judge.go                         -- LLM batch judging + site map prompt assembly
  api/
    server.go                        -- chi router, middleware, route registration
    handlers_scrape.go               -- POST /v1/scrape
    handlers_crawl.go                -- POST /v1/crawl, GET /v1/crawl/{id}
    handlers_map.go                  -- POST /v1/map
    handlers_extract.go              -- POST /v1/extract
    handlers_search.go               -- POST /v1/search
    handlers_agentmap.go             -- POST /v1/agent-map
    middleware.go                     -- auth, logging, recovery
    response.go                      -- JSON response helpers
```

### Data Flow (POST /v1/scrape)

1. Handler validates JSON, sets defaults (formats=["markdown"], timeout=30000)
2. Orchestrator tries Rod engine, falls back to Fetch on failure
3. Rod executes browser actions if specified, captures screenshot if requested
4. Engine returns RawResult (HTML, status code, headers, URL, screenshot bytes)
5. Transform pipeline runs: clean -> readability -> markdown -> metadata -> links
6. Handler wraps in `{"success": true, "data": {...}}` envelope

### Key Interfaces

- `engines.Engine` -- `Name()` + `Fetch(ctx, url, opts)`. Implementations: Rod, Fetch, PDF
- `storage.Store` -- job CRUD + cache. Implementation: BoltStore (bbolt)
- `llm.Provider` -- `Complete(ctx, req)`. Implementations: Anthropic, OpenAI, Ollama

## API Endpoints

- `GET /health` -- health check
- `POST /v1/scrape` -- single URL scrape (sync)
- `POST /v1/crawl` -- async multi-page crawl, returns job ID
- `GET /v1/crawl/{id}` -- poll crawl status + paginated results
- `POST /v1/map` -- URL discovery (sitemap + links)
- `POST /v1/extract` -- scrape + LLM extraction (requires LLM key)
- `POST /v1/search` -- web search + scrape (requires SEARXNG_ENDPOINT)
- `POST /v1/agent-map` -- unified crawl + LLM judge + site map (bring your own key)

## Anti-Slop Rules (ENFORCED)

1. **No god files.** No file exceeds 300 lines. Split if it does.
2. **No global state.** No package-level `var` holding mutable state.
3. **Every public function has a doc comment.** One sentence.
4. **Tests for transform pipeline.** Pure functions must have tests.
5. **One error type per package.** Never return bare `error` from a package boundary.
6. **No premature generics.** Be explicit.
7. **`go vet` must pass.** Always run before committing.
8. **README stays current.** New endpoints ship with README updates.
9. **One file, one job.** If you can't describe a file in one sentence, split it.
10. **Interfaces at boundaries, concrete types inside.**
11. **Errors are data, not strings.** What happened, to what URL, at what stage, retryable?
12. **No dead code, no speculative features.**

## Testing

```
go test ./...                                    # all tests
go test ./internal/scraper/transform/... -v      # transform tests (core value)
go vet ./...                                     # lint
```

## Dependencies (go.mod)

Six direct dependencies: go-rod/rod, go-chi/chi, bbolt, go-readability, html-to-markdown, goquery.
No ORMs, no DI, no config libs. stdlib for logging (slog), JSON, HTTP.

## Current State

All five build phases are complete:
- Phase 1: Scrape engine (Rod + Fetch + transform pipeline + /v1/scrape)
- Phase 2: Crawl + Map (crawler, discovery, job queue, /v1/crawl, /v1/map)
- Phase 3: LLM layer (Anthropic/OpenAI/Ollama + /v1/extract + /v1/search)
- Phase 4: Browser actions, screenshots, PDF engine
- Phase 5: Agent API (/v1/agent-map -- unified crawl + LLM judge + site map prompt)

## Skill routing                                                                     
                                                                      
  When the user's request matches an available skill, ALWAYS invoke it using the Skill   
  tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.       
  The skill has specialized workflows that produce better results than ad-hoc answers.  
                                                                                         
  Key routing rules:
  - QA, test the site, find bugs, dogfood → invoke gstack                                
  - Code quality, simplify, clean up → invoke simplify                                   
  - Run something on a recurring interval, poll status → invoke loop                    
  - Schedule a recurring remote agent, cron job → invoke schedule                        
  - Build with Claude API, Anthropic SDK, Agent SDK → invoke claude-api                  
  - Create slides, presentations → invoke claude-api:pptx                                
  - Create/edit Word docs → invoke claude-api:docx                                       
  - Create/edit spreadsheets, CSV → invoke claude-api:xlsx                               
  - Create/edit PDFs → invoke claude-api:pdf                                             
  - Build frontend UI, landing pages, dashboards → invoke claude-api:frontend-design    
  - Build MCP servers → invoke claude-api:mcp-builder                                    
  - Co-author documentation, specs, proposals → invoke claude-api:doc-coauthoring        
  - Configure Claude Code settings, hooks → invoke update-config                         
  - Customize keybindings → invoke keybindings-help      