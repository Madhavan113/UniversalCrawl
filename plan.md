# UniversalCrawl — Architecture & Implementation Plan

## Problem

Autonomous agents need to act on the web. Current approaches fail in one of two ways:

1. **Raw browser use** (Playwright MCP, browser-use) — dumps HTML, screenshots, or DOM trees directly into the agent's context window. Expensive, noisy, and fundamentally unscalable.
2. **Hand-coded integrations** (Zapier, custom APIs) — require human authorship per site. Not autonomous.

Neither gives agents a clean, minimal, self-describing interface to arbitrary websites without human intervention.

## Vision

A Go-based infrastructure server that autonomously crawls any website, extracts its capabilities (actions, forms, endpoints), caches a structured capability map, and exposes a clean REST API. Agents discover and execute actions through a single, lightweight interface — they never see HTML, selectors, or DOM state.

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                 universalcrawl server                 │
│                     (Go binary)                       │
│                                                       │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────┐  │
│  │ Crawler  │→ │ Extractor │→ │    Registry       │  │
│  │  (Rod)   │  │(heuristic │  │  (bbolt / SQLite) │  │
│  │          │  │  + LLM)   │  │  capability cache │  │
│  └──────────┘  └───────────┘  └──────────────────┘  │
│                                        │              │
│                               ┌────────▼───────┐     │
│                               │    Executor    │     │
│                               │  (headless     │     │
│                               │   Chrome/Rod)  │     │
│                               └────────┬───────┘     │
│                                        │              │
│                          ┌─────────────▼──────────┐  │
│                          │     HTTP API (chi)      │  │
│                          └─────────────────────────┘  │
└──────────────────────────────────────────────────────┘
                 │ HTTP / Unix socket
     ┌───────────┼────────────────┐
     ▼           ▼                ▼
  Python      TypeScript       Raw HTTP
  thin SDK    thin SDK         (any agent)
```

---

## Core Data Model

```go
// A site's full capability map
type SiteCapability struct {
    Domain     string
    Actions    []Action
    AuthType   AuthType   // None, BasicAuth, FormLogin, OAuth, APIKey
    CrawledAt  time.Time
    TTL        time.Duration
}

// A single discoverable action on a site
type Action struct {
    Name        string            // semantic name: "search_products"
    Description string            // human/agent-readable description
    Method      string            // GET, POST, PUT, DELETE
    Endpoint    string            // resolved URL pattern
    Params      []Param           // typed parameters with validation
    Returns     Schema            // JSON schema of response shape
    AuthRequired bool
}

type Param struct {
    Name     string
    Type     string   // string, int, bool, enum
    Required bool
    Enum     []string // if applicable
    Default  any
}
```

---

## Components

### 1. Crawler (`internal/crawler`)

Responsible for visiting a site and collecting raw signals.

**Strategy (in order of priority):**
1. Check for OpenAPI/Swagger spec at common paths (`/openapi.json`, `/swagger.json`, `/api-docs`)
2. Intercept XHR/fetch network requests via Rod's network event hooks — captures real API calls the page makes
3. Parse HTML forms (action URL, method, input names/types)
4. Parse `<a>` navigation structure for URL patterns
5. Execute JS-heavy SPAs in headless Chrome, wait for network idle, then re-collect

**Key library:** `github.com/go-rod/rod` — Chrome DevTools Protocol, full JS execution, network interception

```go
type Crawler struct {
    browser   *rod.Browser
    extractor *Extractor
    registry  *Registry
}

func (c *Crawler) Crawl(domain string) (*SiteCapability, error)
func (c *Crawler) CrawlWithAuth(domain string, creds Credentials) (*SiteCapability, error)
```

### 2. Extractor (`internal/extractor`)

Converts raw crawler signals into semantic, named Actions.

**Two-pass approach:**

**Pass 1 — Heuristic:** Pattern-match known signal types
- `/search?q=` → `search(query string)`
- `POST /login` with `username`+`password` fields → `login(username, password)`
- `POST /cart/add` with `product_id` → `add_to_cart(product_id)`
- REST URL patterns (`/users/{id}`) → typed CRUD actions

**Pass 2 — LLM-assisted (optional, configurable):** Send a compact signal summary to Claude API
- Input: list of endpoints, form fields, URL patterns (no HTML)
- Output: semantic action names, descriptions, param types
- Cached — only runs once per crawl, not per execution

```go
type Extractor struct {
    llmClient *anthropic.Client // optional
}

func (e *Extractor) Extract(signals *CrawlSignals) ([]Action, error)
```

### 3. Registry (`internal/registry`)

Persistent, embedded capability store.

- **Storage:** bbolt (pure Go, embedded, zero dependencies) or SQLite via `modernc.org/sqlite`
- **TTL:** configurable per-site (default 24h for static sites, 1h for dynamic)
- **Invalidation:** manual via API or automatic on TTL expiry
- **Index:** domain → SiteCapability (msgpack serialized)

```go
type Registry struct {
    db *bbolt.DB
}

func (r *Registry) Get(domain string) (*SiteCapability, error)
func (r *Registry) Put(domain string, cap *SiteCapability) error
func (r *Registry) Invalidate(domain string) error
func (r *Registry) List() ([]string, error)
```

### 4. Executor (`internal/executor`)

Executes an action against a live site.

- Resolves action definition from registry
- Validates params against action schema
- Executes via Rod (form fill + submit, XHR intercept, navigation)
- Extracts structured result — **never returns raw HTML**, always returns JSON-shaped data
- Session management: per-domain cookie jar, auto-rehydration

```go
type Executor struct {
    browser  *rod.Browser
    sessions *SessionStore
}

func (e *Executor) Execute(domain, action string, params map[string]any) (*Result, error)

type Result struct {
    Data     any               // structured response
    Meta     map[string]string // pagination cursors, status codes, etc.
    RawType  string            // "json" | "html_table" | "list" | "text"
}
```

### 5. API Server (`internal/api`)

HTTP server exposing all capabilities to agents.

**Router:** `github.com/go-chi/chi`

```
GET  /v1/sites/{domain}/capabilities     → Get capability map (triggers crawl if uncached)
POST /v1/sites/{domain}/execute          → Execute an action
POST /v1/sites/{domain}/crawl            → Force re-crawl
GET  /v1/sites/{domain}/status           → Crawl status + cache age
DELETE /v1/sites/{domain}/cache          → Invalidate cache
POST /v1/auth/{domain}                   → Store credentials (written to vault, never returned)
GET  /v1/sites                           → List all known sites
```

**Execute request:**
```json
{
  "action": "search_products",
  "params": { "query": "mechanical keyboard", "max_results": 10 }
}
```

**Execute response:**
```json
{
  "data": [ { "name": "...", "price": 129.99, "url": "..." } ],
  "meta": { "total": 284, "next_cursor": "eyJwIjoxfQ==" }
}
```

---

## The Auto Skill

The agent-facing interface. Instead of MCP (which injects tool definitions into context), this is a **single, stable tool call** that routes to the local server:

```python
# Python SDK — entire interface surface for an agent
from universalcrawl import web

# Discover what a site can do
caps = web.capabilities("amazon.com")
# → ["search_products", "get_product", "add_to_cart", "checkout"]

# Execute
results = web.execute("amazon.com", "search_products", {"query": "keyboard"})
```

For Claude Code specifically — a skill definition (`skills/claude/web.md`) that wraps these calls. The agent uses `/web` as a skill, which resolves to the local server. **Zero new tool definitions added to context.**

For other agent frameworks (LangChain, AutoGen, CrewAI) — single `WebTool` class, one tool, one description. The tool's schema never changes regardless of how many sites are known.

---

## Directory Structure

```
universalcrawl/
├── cmd/
│   └── server/
│       └── main.go              # Binary entrypoint, config, server startup
├── internal/
│   ├── crawler/
│   │   ├── crawler.go           # Crawl orchestration
│   │   ├── rod.go               # Rod browser wrapper
│   │   └── signals.go           # CrawlSignals type + collectors
│   ├── extractor/
│   │   ├── extractor.go         # Orchestrates heuristic + LLM passes
│   │   ├── heuristic.go         # Pattern-based extraction
│   │   └── llm.go               # Claude API extraction (optional)
│   ├── executor/
│   │   ├── executor.go          # Action execution
│   │   └── session.go           # Cookie/session management
│   ├── registry/
│   │   ├── registry.go          # bbolt-backed store
│   │   └── models.go            # SiteCapability, Action, Param types
│   └── api/
│       ├── server.go            # chi router setup
│       ├── handlers.go          # HTTP handlers
│       └── middleware.go        # Auth, logging, rate limiting
├── pkg/
│   └── client/
│       └── client.go            # Go client library (for embedding)
├── sdks/
│   ├── python/
│   │   ├── universalcrawl/
│   │   │   ├── __init__.py
│   │   │   └── client.py        # Thin HTTP wrapper
│   │   └── pyproject.toml
│   └── typescript/
│       ├── src/
│       │   └── index.ts
│       └── package.json
├── skills/
│   └── claude/
│       └── web.md               # Claude Code auto skill definition
├── go.mod
├── go.sum
└── plan.md
```

---

## Build Phases

### Phase 1 — Foundation
- [ ] `go.mod` init, dependency resolution
- [ ] Core types (`registry/models.go`)
- [ ] Registry with bbolt
- [ ] Basic HTTP crawler (no JS) + heuristic extractor
- [ ] API server skeleton (chi, health endpoint)

### Phase 2 — Browser Automation
- [ ] Rod integration + browser pool
- [ ] Network interception for XHR capture
- [ ] JS-heavy SPA crawling
- [ ] Form detection + submission executor

### Phase 3 — Intelligence Layer
- [ ] LLM-assisted extraction (Claude API, optional/configurable)
- [ ] Semantic action naming pipeline
- [ ] Result normalization (HTML table → JSON, pagination handling)

### Phase 4 — Auth & Sessions
- [ ] Credential vault (encrypted, local)
- [ ] Session hydration / cookie persistence
- [ ] Auto re-auth on session expiry

### Phase 5 — Agent SDKs & Skill
- [ ] Python SDK (`pip install universalcrawl`)
- [ ] TypeScript SDK (`npm install universalcrawl`)
- [ ] Claude Code auto skill (`/web`)
- [ ] LangChain / AutoGen tool wrappers

---

## Key Technical Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Language | Go | Goroutine-based concurrency for parallel crawling, single binary, low memory |
| Browser automation | Rod (`go-rod/rod`) | Native Go, CDP, network interception, active maintenance |
| HTTP router | chi | Lightweight, idiomatic, middleware-composable |
| Capability store | bbolt | Embedded, zero-config, pure Go, battle-tested |
| LLM extraction | Claude API (optional) | Can run fully heuristic-only; LLM pass is additive |
| Serialization | msgpack (registry) + JSON (API) | msgpack for compact storage, JSON for wire format |
| Auth storage | AES-256-GCM encrypted bbolt bucket | Credentials never leave the local machine |

---

## Non-Goals (for now)

- No cloud sync / shared capability registry (Phase 2 product concern)
- No visual/screenshot-based understanding (adds complexity, rarely needed)
- No JavaScript execution sandbox (Rod runs real Chrome)
- No rate limiting enforcement on behalf of crawled sites (caller's responsibility)
