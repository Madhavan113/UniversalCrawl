# UniversalCrawl — The Definitive Plan

## 1. What This Is

UniversalCrawl is an open-source Go binary that gives any AI agent the ability to observe, understand, and interact with the web. It ships as a single binary with zero external dependencies — no Redis, no Postgres, no Docker Compose, no separate browser microservice. One binary, one process, one REST API.

The system does three things, in strict order:

1. **Observe** — load pages in a real browser, record everything that happens (network calls, DOM state, rendered content, links, forms)
2. **Normalize** — turn raw observations into clean, structured, LLM-ready data (markdown, metadata, links, structured JSON)
3. **Serve** — expose all of this through a simple REST API that any agent framework can call

This is not a research project. This is not a browser agent. This is infrastructure — the web data layer that agents are missing.

---

## 2. Why This Matters

Every agent framework today has the same problem: the web is not designed for machines. When an agent needs information from a website, it has three bad options:

- **Raw HTTP** — misses JavaScript-rendered content, breaks on SPAs, can't handle dynamic state
- **Browser-use** — puts a model in front of pixels and hopes for the best, burns tokens on navigation, brittle across sites
- **Firecrawl/paid APIs** — works but costs money per page, vendor lock-in, TypeScript monolith with 5 Docker containers

UniversalCrawl replaces all three with a local-first, agent-native web data API. The agent says "scrape this URL" or "crawl this site" and gets back clean markdown, structured data, and discovered URLs. No guessing, no pixel parsing, no vendor dependency.

---

## 3. Core Principles

These principles are load-bearing. Every design decision flows from them. When vibe coding creates a question about how to do something, these principles answer it.

### P1: Observation before inference

Raw HTML, network state, and page content are ground truth. Everything derived (markdown, metadata, capabilities) must be traceable back to what was actually observed. Never generate data the browser didn't witness.

### P2: Separation of concerns is the anti-slop mechanism

The system has exactly four layers. Each layer has a clear boundary. Code that crosses layers is a bug, not a feature.

```
Browser Pool → Scrape Engines → Transform Pipeline → API Handlers
     ↑              ↑                   ↑                 ↑
  manages       fetches raw         converts to       serves to
  Chrome        HTML/state         markdown/JSON       agents
  instances     from the web       from raw HTML      via REST
```

### P3: One file, one job

Every `.go` file does exactly one thing. If you can't describe what a file does in one sentence, it's doing too much. This is the single most important rule for keeping a vibe-coded codebase from becoming slop.

### P4: Interfaces at boundaries, concrete types inside

The scrape engine is an interface. The transform steps are interfaces. The storage layer is an interface. Everything else is concrete. Don't abstract until there are two implementations.

### P5: Errors are data, not strings

Every operation that can fail returns typed errors with enough context to debug. No `fmt.Errorf("something went wrong")`. Every error should answer: what happened, to what URL, at what stage, and is it retryable?

### P6: No dead code, no speculative features

If it's not called from a handler or a test, it doesn't exist. Don't build the execution layer, the policy layer, or the workflow engine until the discovery layer ships and proves value. The Codex thesis about observation → capability → execution → policy is correct in ordering. We build in that order, not all at once.

---

## 4. Anti-Slop Rules for Vibe Coding

This codebase will be built with AI assistance. That's fine. But AI-assisted codebases rot fast unless you enforce structure. These rules exist to prevent that rot.

### Rule 1: No god files
No file exceeds 300 lines. If it does, split it. The moment a file starts accumulating unrelated functions, the codebase is dying.

### Rule 2: No global state
No package-level `var` that holds mutable state. Browser pool, storage, config — all passed explicitly via struct fields or function parameters. Global state is where vibe-coded bugs hide.

### Rule 3: Every public function has a doc comment
One sentence. What does it do, what does it return. If the AI generates a public function without a doc comment, add one before moving on.

### Rule 4: Tests exist for the transform pipeline
The scrape engines depend on external state (the web) and are hard to test. The transform pipeline is pure functions: HTML in, markdown out. These functions MUST have tests. They are the core value of the product.

### Rule 5: One error type per package
Each package defines its own error types. `scraper.ErrTimeout`, `scraper.ErrDNS`, `storage.ErrNotFound`. Never return bare `error` from a package boundary.

### Rule 6: No premature generics
Don't build `Provider[T]` abstractions. Don't build plugin systems. Don't build middleware chains longer than 3 layers. The system is small enough to be explicit.

### Rule 7: Format on save, lint on commit
`gofmt` and `go vet` are non-negotiable. Run them always. If the AI generates code that doesn't pass `go vet`, fix it before moving on.

### Rule 8: README stays current
Every time a new endpoint ships, the README gets updated in the same commit. Documentation that trails code by a week is documentation that never catches up.

---

## 5. API Surface

Six endpoints. Modeled after Firecrawl's v2 API for ecosystem compatibility but served from a single Go binary.

### POST /v1/scrape

Scrape a single URL. Returns content in requested formats. Synchronous — blocks until done.

Request:
```json
{
  "url": "https://example.com",
  "formats": ["markdown", "html", "links", "screenshot"],
  "onlyMainContent": true,
  "waitFor": 2000,
  "timeout": 30000,
  "headers": { "Accept-Language": "en-US" },
  "actions": [
    { "type": "click", "selector": "button.load-more" },
    { "type": "wait", "milliseconds": 1000 }
  ],
  "mobile": false,
  "includeTags": [],
  "excludeTags": ["nav", "footer"]
}
```

Response:
```json
{
  "success": true,
  "data": {
    "url": "https://example.com",
    "markdown": "# Example...",
    "html": "<main>...</main>",
    "links": ["https://example.com/about", "..."],
    "screenshot": "data:image/png;base64,...",
    "metadata": {
      "title": "Example",
      "description": "...",
      "language": "en",
      "ogImage": "https://..."
    }
  }
}
```

### POST /v1/crawl

Start an async crawl job. Discovers pages, scrapes each one.

Request:
```json
{
  "url": "https://example.com",
  "limit": 100,
  "maxDepth": 5,
  "formats": ["markdown", "links"],
  "onlyMainContent": true,
  "includePaths": ["/blog/*", "/docs/*"],
  "excludePaths": ["/admin/*"],
  "allowSubdomains": false,
  "ignoreSitemap": false,
  "delay": 200
}
```

Response:
```json
{
  "success": true,
  "id": "crawl_abc123",
  "url": "https://api.universalcrawl.local/v1/crawl/crawl_abc123"
}
```

### GET /v1/crawl/{id}

Poll crawl status. Returns paginated results.

Response:
```json
{
  "success": true,
  "status": "scraping",
  "total": 47,
  "completed": 23,
  "expiresAt": "2026-04-02T00:00:00Z",
  "data": [
    { "url": "...", "markdown": "...", "metadata": { "..." } }
  ],
  "next": "cursor_token"
}
```

### POST /v1/map

Discover all URLs on a site. Fast — no content extraction, just URL collection.

Request:
```json
{
  "url": "https://example.com",
  "search": "pricing",
  "includeSubdomains": false,
  "limit": 5000,
  "sitemapOnly": false
}
```

Response:
```json
{
  "success": true,
  "links": [
    "https://example.com/pricing",
    "https://example.com/pricing/enterprise",
    "..."
  ]
}
```

### POST /v1/extract

Scrape one or more URLs and use an LLM to extract structured data.

Request:
```json
{
  "urls": ["https://example.com/pricing"],
  "prompt": "Extract all pricing tiers with name, price, and features",
  "schema": {
    "type": "array",
    "items": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "price": { "type": "string" },
        "features": { "type": "array", "items": { "type": "string" } }
      }
    }
  }
}
```

Response:
```json
{
  "success": true,
  "data": [
    { "name": "Starter", "price": "$9/mo", "features": ["5 users", "10GB"] },
    { "name": "Pro", "price": "$29/mo", "features": ["Unlimited users", "100GB"] }
  ]
}
```

### POST /v1/search

Search the web, scrape top results, return LLM-ready content.

Request:
```json
{
  "query": "best go web frameworks 2026",
  "limit": 5,
  "formats": ["markdown"],
  "lang": "en"
}
```

Response:
```json
{
  "success": true,
  "data": [
    { "url": "https://...", "markdown": "...", "metadata": { "..." } }
  ]
}
```

---

## 6. Architecture

### System Diagram

```
                    ┌─────────────────────────────────────┐
                    │           API Server (chi)           │
                    │  /v1/scrape  /v1/crawl  /v1/map     │
                    │  /v1/extract /v1/search              │
                    └──────┬──────┬──────┬────────────────┘
                           │      │      │
              ┌────────────┘      │      └────────────┐
              ▼                   ▼                    ▼
     ┌────────────────┐  ┌───────────────┐  ┌─────────────────┐
     │ Scrape Engine  │  │   Crawler     │  │   LLM Layer     │
     │ (orchestrator) │  │ (multi-page)  │  │ (extract/search)│
     └───────┬────────┘  └──────┬────────┘  └────────┬────────┘
             │                  │                     │
     ┌───────┴────────┐        │              ┌──────┴───────┐
     │                │        │              │              │
 ┌───┴───┐      ┌────┴───┐    │         ┌────┴────┐  ┌─────┴────┐
 │  Rod  │      │ Fetch  │    │         │Anthropic│  │  OpenAI  │
 │Engine │      │Engine  │    │         │Provider │  │ Provider │
 └───┬───┘      └────┬───┘    │         └─────────┘  └──────────┘
     │               │        │
     ▼               ▼        ▼
 ┌──────────────────────────────────┐
 │       Transform Pipeline         │
 │  clean → readability → markdown  │
 └──────────────────────────────────┘
             │
             ▼
 ┌──────────────────────┐
 │   bbolt Storage      │
 │  jobs │ results │    │
 │  cache │ config  │   │
 └──────────────────────┘
```

### Layer Responsibilities

**API Layer** (`internal/api/`):
- HTTP routing via chi
- Request validation and response formatting
- Auth middleware (optional API key)
- Rate limiting (token bucket per IP)
- All handlers are thin — validate input, call service layer, format output

**Scrape Engine** (`internal/scraper/`):
- Orchestrator selects engine (Rod for JS-heavy, Fetch for static)
- Rod Engine: headless Chrome via go-rod/rod, pulls browser from pool, navigates, waits, captures HTML
- Fetch Engine: plain net/http GET, for static pages and fallback
- PDF Engine: downloads and extracts text from PDFs
- Each engine returns raw HTML + status code + response headers

**Transform Pipeline** (`internal/scraper/transform/`):
- HTML Cleaner: removes script, style, nav, footer, ads (goquery)
- Readability: extracts main content (go-readability)
- Markdown: converts HTML to markdown (html-to-markdown)
- Metadata: extracts title, description, OG tags, language
- Link Extractor: collects all href links from the page
- Screenshot: captures page screenshot via Rod
- Each step is a pure function: input → output, independently testable

**Browser Pool** (`internal/browser/`):
- Manages N headless Chrome instances
- Goroutine-safe acquire/release
- Health checks and crash recovery
- Configurable pool size

**Crawler** (`internal/crawler/`):
- Drives multi-page crawling for /v1/crawl and /v1/map
- URL discovery: sitemap.xml parsing, robots.txt parsing, link extraction
- Link filtering: same-origin, max depth, include/exclude globs, dedup
- Politeness: configurable delay, concurrent request cap
- State management: tracks visited URLs, queued URLs, results per job

**LLM Layer** (`internal/llm/`):
- Provider interface with Anthropic, OpenAI, and Ollama implementations
- Used by Extract (scrape + LLM structured extraction) and Search (web search + scrape)
- JSON mode support for structured output

**Storage** (`internal/storage/`):
- bbolt embedded KV store
- Buckets: jobs, job_results, cache, config
- Cache with configurable TTL
- All operations go through a Store interface for testability

**Job Queue** (`internal/jobs/`):
- In-process channel-based queue
- Configurable goroutine worker pool
- Used for async crawl and extract jobs
- No external dependencies

---

## 7. Data Types

These are the canonical types. They live in `internal/models/` and are the shared language across all packages.

```go
// --- Scrape ---

type ScrapeRequest struct {
    URL             string            `json:"url"`
    Formats         []string          `json:"formats"`
    OnlyMainContent bool              `json:"onlyMainContent"`
    WaitFor         int               `json:"waitFor"`
    Timeout         int               `json:"timeout"`
    Headers         map[string]string `json:"headers"`
    Actions         []BrowserAction   `json:"actions"`
    Mobile          bool              `json:"mobile"`
    IncludeTags     []string          `json:"includeTags"`
    ExcludeTags     []string          `json:"excludeTags"`
}

type BrowserAction struct {
    Type         string `json:"type"`         // click, scroll, type, wait, screenshot
    Selector     string `json:"selector"`     // CSS selector (for click, type)
    Text         string `json:"text"`         // text to type
    Milliseconds int    `json:"milliseconds"` // for wait actions
    Direction    string `json:"direction"`    // for scroll: up, down
    Amount       int    `json:"amount"`       // scroll pixels
}

type ScrapeResult struct {
    URL        string            `json:"url"`
    Markdown   string            `json:"markdown,omitempty"`
    HTML       string            `json:"html,omitempty"`
    RawHTML    string            `json:"rawHtml,omitempty"`
    Screenshot string            `json:"screenshot,omitempty"`
    Links      []string          `json:"links,omitempty"`
    Metadata   PageMetadata      `json:"metadata"`
}

type PageMetadata struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    Language    string `json:"language"`
    OGImage     string `json:"ogImage,omitempty"`
    Canonical   string `json:"canonical,omitempty"`
    StatusCode  int    `json:"statusCode"`
}

// --- Crawl ---

type CrawlRequest struct {
    URL              string   `json:"url"`
    Limit            int      `json:"limit"`
    MaxDepth         int      `json:"maxDepth"`
    Formats          []string `json:"formats"`
    OnlyMainContent  bool     `json:"onlyMainContent"`
    IncludePaths     []string `json:"includePaths"`
    ExcludePaths     []string `json:"excludePaths"`
    AllowSubdomains  bool     `json:"allowSubdomains"`
    IgnoreSitemap    bool     `json:"ignoreSitemap"`
    Delay            int      `json:"delay"`
}

type CrawlJob struct {
    ID        string    `json:"id"`
    Status    string    `json:"status"` // scraping, completed, failed, cancelled
    URL       string    `json:"url"`
    Total     int       `json:"total"`
    Completed int       `json:"completed"`
    CreatedAt time.Time `json:"createdAt"`
    ExpiresAt time.Time `json:"expiresAt"`
}

// --- Map ---

type MapRequest struct {
    URL               string `json:"url"`
    Search            string `json:"search"`
    IncludeSubdomains bool   `json:"includeSubdomains"`
    Limit             int    `json:"limit"`
    SitemapOnly       bool   `json:"sitemapOnly"`
}

// --- Extract ---

type ExtractRequest struct {
    URLs   []string         `json:"urls"`
    Prompt string           `json:"prompt"`
    Schema *json.RawMessage `json:"schema"`
}

// --- Search ---

type SearchRequest struct {
    Query           string   `json:"query"`
    Limit           int      `json:"limit"`
    Formats         []string `json:"formats"`
    OnlyMainContent bool     `json:"onlyMainContent"`
    Lang            string   `json:"lang"`
    Country         string   `json:"country"`
}
```

---

## 8. Directory Structure

```
universalcrawl/
├── main.go                              # entrypoint: CLI flags, server startup
├── go.mod
├── go.sum
├── README.md
│
├── internal/
│   ├── api/
│   │   ├── server.go                    # chi router setup, middleware registration
│   │   ├── handlers_scrape.go           # POST /v1/scrape
│   │   ├── handlers_crawl.go            # POST /v1/crawl, GET /v1/crawl/{id}
│   │   ├── handlers_map.go             # POST /v1/map
│   │   ├── handlers_extract.go          # POST /v1/extract
│   │   ├── handlers_search.go           # POST /v1/search
│   │   ├── middleware.go                # auth, rate limit, logging, recovery
│   │   └── response.go                 # standard JSON response helpers
│   │
│   ├── scraper/
│   │   ├── orchestrator.go              # engine selection + fallback chain
│   │   ├── result.go                    # ScrapeResult type + helpers
│   │   ├── engines/
│   │   │   ├── engine.go                # Engine interface definition
│   │   │   ├── rod.go                   # Rod/CDP headless Chrome
│   │   │   ├── fetch.go                 # Plain HTTP GET
│   │   │   └── pdf.go                   # PDF text extraction
│   │   └── transform/
│   │       ├── pipeline.go              # composes all transform steps
│   │       ├── clean.go                 # HTML cleaning (remove boilerplate)
│   │       ├── clean_test.go            # REQUIRED: tests for cleaning
│   │       ├── readability.go           # main content extraction
│   │       ├── readability_test.go      # REQUIRED: tests for readability
│   │       ├── markdown.go              # HTML → Markdown conversion
│   │       ├── markdown_test.go         # REQUIRED: tests for markdown
│   │       ├── metadata.go             # title, OG tags, description extraction
│   │       ├── links.go                 # link extraction from HTML
│   │       └── screenshot.go            # screenshot capture via Rod
│   │
│   ├── browser/
│   │   ├── pool.go                      # browser instance pool
│   │   └── actions.go                   # click, scroll, type, wait implementations
│   │
│   ├── crawler/
│   │   ├── crawler.go                   # WebCrawler orchestration
│   │   ├── discovery.go                 # sitemap + robots.txt + link extraction
│   │   ├── filter.go                    # URL filtering (origin, depth, globs)
│   │   └── state.go                     # crawl state tracking (visited, queued)
│   │
│   ├── llm/
│   │   ├── provider.go                  # LLMProvider interface
│   │   ├── anthropic.go                 # Claude API client
│   │   ├── openai.go                    # OpenAI API client
│   │   └── ollama.go                    # Ollama local client
│   │
│   ├── extract/
│   │   └── extract.go                   # scrape + LLM structured extraction
│   │
│   ├── search/
│   │   └── search.go                    # web search + scrape results
│   │
│   ├── storage/
│   │   ├── store.go                     # Store interface + bbolt implementation
│   │   ├── jobs.go                      # crawl job CRUD
│   │   └── cache.go                     # scrape result caching
│   │
│   ├── jobs/
│   │   ├── queue.go                     # in-process job queue
│   │   └── worker.go                    # goroutine worker pool
│   │
│   └── models/
│       ├── scrape.go                    # ScrapeRequest, ScrapeResult, etc.
│       ├── crawl.go                     # CrawlRequest, CrawlJob
│       ├── map.go                       # MapRequest
│       ├── extract.go                   # ExtractRequest
│       ├── search.go                    # SearchRequest
│       └── errors.go                    # typed error definitions
│
└── plan.md                              # this document
```

---

## 9. Dependencies

Minimal, well-maintained, Go-native:

```
github.com/go-rod/rod                        # headless Chrome via CDP
github.com/go-chi/chi/v5                     # HTTP router
go.etcd.io/bbolt                             # embedded KV store
github.com/go-shiori/go-readability          # content extraction
github.com/JohannesKaufmann/html-to-markdown # HTML → Markdown
github.com/PuerkitoBio/goquery              # HTML DOM manipulation
```

That's six dependencies. No ORMs, no DI frameworks, no config libraries, no logging frameworks. Use `log/slog` from stdlib for structured logging. Use `encoding/json` from stdlib for JSON. Use `net/http` from stdlib for the fetch engine and LLM API clients.

---

## 10. Configuration

All config via environment variables. CLI flags override for convenience.

| Env Var | Default | Purpose |
|---------|---------|---------|
| `PORT` | `3000` | API server port |
| `API_KEY` | (empty = auth disabled) | Bearer token for API auth |
| `BROWSER_POOL_SIZE` | `5` | Headless Chrome instances |
| `WORKERS` | `4` | Background job goroutines |
| `DATA_DIR` | `./data` | bbolt database directory |
| `CACHE_TTL` | `1h` | Scrape result cache TTL |
| `ANTHROPIC_API_KEY` | — | For extract/search |
| `OPENAI_API_KEY` | — | Alternative LLM |
| `OLLAMA_BASE_URL` | — | Local LLM |
| `SEARXNG_ENDPOINT` | — | Search backend |
| `LOG_LEVEL` | `info` | Logging verbosity |

---

## 11. Build Phases

Each phase has a clear exit criteria. Don't start the next phase until the current one ships.

### Phase 1 — Scrape Engine

The foundation. Everything else depends on this being solid.

Build:
1. `internal/models/` — all data types
2. `internal/browser/pool.go` — Rod browser pool (acquire/release/health)
3. `internal/scraper/engines/rod.go` — Rod engine (navigate, wait, return HTML)
4. `internal/scraper/engines/fetch.go` — HTTP GET engine (static pages)
5. `internal/scraper/engines/engine.go` — Engine interface
6. `internal/scraper/orchestrator.go` — tries Rod, falls back to Fetch
7. `internal/scraper/transform/` — clean, readability, markdown, metadata, links
8. `internal/scraper/transform/*_test.go` — tests for each transform step
9. `internal/storage/store.go` — bbolt setup + cache
10. `internal/api/server.go` — chi router + middleware
11. `internal/api/handlers_scrape.go` — POST /v1/scrape
12. `main.go` — server startup with flags

Exit criteria:
- `curl -X POST localhost:3000/v1/scrape -d '{"url":"https://example.com","formats":["markdown"]}'` returns clean markdown
- Transform tests pass
- Rod engine handles JS-heavy SPAs
- Fetch engine handles static pages
- Engine fallback works (Rod timeout → Fetch)

### Phase 2 — Crawl + Map

Multi-page crawling and URL discovery.

Build:
1. `internal/crawler/discovery.go` — sitemap.xml parser, robots.txt parser, link extractor
2. `internal/crawler/filter.go` — same-origin, depth, glob patterns, dedup
3. `internal/crawler/state.go` — tracks visited/queued/completed per job
4. `internal/crawler/crawler.go` — WebCrawler that ties it all together
5. `internal/jobs/queue.go` + `worker.go` — in-process job queue
6. `internal/storage/jobs.go` — job CRUD in bbolt
7. `internal/api/handlers_crawl.go` — POST /v1/crawl, GET /v1/crawl/{id}
8. `internal/api/handlers_map.go` — POST /v1/map

Exit criteria:
- Crawl job starts async, returns job ID
- Polling returns paginated results as pages complete
- Map returns discovered URLs in under 5 seconds for most sites
- robots.txt and sitemaps are respected
- Include/exclude path globs work
- Politeness delay works

### Phase 3 — Extract + Search + LLM

LLM-powered features.

Build:
1. `internal/llm/provider.go` — LLMProvider interface
2. `internal/llm/anthropic.go` — Anthropic Claude client
3. `internal/llm/openai.go` — OpenAI client
4. `internal/llm/ollama.go` — Ollama client
5. `internal/extract/extract.go` — scrape URLs + LLM extraction with JSON schema
6. `internal/search/search.go` — SearXNG/Google search + scrape results
7. `internal/api/handlers_extract.go` — POST /v1/extract
8. `internal/api/handlers_search.go` — POST /v1/search

Exit criteria:
- Extract returns structured JSON matching provided schema
- Search returns scraped markdown from web search results
- Works with at least Anthropic and one other provider

### Phase 4 — Browser Actions + Polish

Advanced scraping features and production readiness.

Build:
1. `internal/browser/actions.go` — click, scroll, type, wait, screenshot actions
2. `internal/scraper/engines/pdf.go` — PDF extraction
3. `internal/scraper/transform/screenshot.go` — screenshot capture
4. CLI mode (scrape/crawl/map one-shot commands in `main.go`)
5. README with full API documentation
6. Error handling hardening across all engines
7. Graceful shutdown (SIGTERM handler)
8. Request timeout enforcement

Exit criteria:
- Actions work (click a "load more" button, then scrape the expanded page)
- Screenshots return base64 PNG
- PDF extraction returns markdown
- CLI mode works for quick testing
- README is complete and accurate
- Server handles 50+ concurrent scrape requests without crashing

---

## 12. What This Is Not

- Not a browser agent. It does not make decisions about what to click or where to navigate. Agents do that. This provides the data.
- Not a Firecrawl fork. It's a clean-room implementation in Go with a compatible API shape.
- Not a research project. Each phase ships working software.
- Not a distributed system. Single binary, single process. Scale vertically by running a bigger machine or horizontally by running multiple instances behind a load balancer.
- Not complete on day one. Phase 1 must work perfectly before Phase 2 starts. Scope discipline is how this ships.

---

## 13. Future Considerations (Post-Phase 4)

These ideas come from the Codex strategic thesis. They are worth building eventually but are explicitly deferred until the core scrape/crawl/map/extract/search loop is solid.

- **Capability Graph**: per-site structured model of what a site can do, derived from crawl observations. The Codex concept of Observation → Capability → Workflow → ExecutionRecipe → PolicyVerdict is the right long-term architecture. But the capability graph is useful only after the observation layer (scrape engine) is reliable. Build it when Phase 2 is solid.

- **Network Interception**: record XHR/fetch calls during page loads via CDP to discover API endpoints. This was the original plan.md vision. It becomes a Rod engine feature after Phase 1.

- **Execution Recipes**: replay discovered API calls with proper headers/tokens. Only after capability discovery proves useful.

- **Policy Layer**: read_safe, auth_required, write_guarded, high_risk classifications for discovered capabilities. Only after execution recipes exist.

- **Evaluation Harness**: benchmark suite measuring discovery quality across site archetypes. Important but requires the core system to exist first.

- **MCP Server**: Model Context Protocol integration so Claude/Cursor/etc. can call UniversalCrawl as a tool directly.

---

## 14. Success Metrics

Phase 1 is successful when:
- Any LLM agent can call one HTTP endpoint and get clean markdown from any URL on the web
- JS-heavy SPAs render correctly
- Static sites scrape in under 2 seconds

The full system is successful when:
- An agent framework can crawl a site, discover all URLs, extract structured data, and search the web through a single local binary
- No paid API keys required for core scrape/crawl/map functionality
- The binary runs on a $5/month VPS
