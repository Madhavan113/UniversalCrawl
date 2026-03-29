# UniversalCrawl — Architecture & Implementation Plan (Revised)

## Problem

Agents need to call APIs on arbitrary websites without hand-coded integrations. The missing piece is **automatic discovery of the actual API surface a site exposes to its own frontend**.

Modern web apps are built API-first. Every user action — search, filter, add to cart, submit a form — triggers a real HTTP request from the browser to a backend. Those requests are the ground truth of what the site can do, because they're what the product itself uses. They have real endpoints, real schemas, and real auth flows.

The goal is to capture that traffic automatically and expose it as a structured, queryable endpoint map.

---

## Core Insight

When a headless browser loads a page and interacts with it, the browser emits network events for every XHR/fetch call the page makes. These events contain:

- the exact URL called (including path params and query strings)
- the HTTP method
- the request headers and body
- the response status, headers, and body

That's not inference. That's observation of real, working API calls. No heuristic extraction. No LLM guessing. Just a faithful recording of what the site actually does when used.

This is the approach. Everything else — normalization, clustering, storage, serving — is downstream of that core capture.

---

## Scope (Explicit)

**In scope:**
- Interface-forward HTTP/HTTPS endpoints: XHR and fetch calls made by the browser during page load and interaction
- REST-style endpoints (parameterized paths, query strings, JSON bodies)
- GraphQL endpoints (operation name, query shape, variables)
- Form submissions (multipart, urlencoded)
- Authenticated flows (capture session-bound calls after login, store separately)

**Out of scope (initially):**
- WebSocket messages
- Server-sent events
- Endpoints never triggered by browser-visible flows (internal services, admin APIs not reachable from the frontend)
- Endpoint *execution* on behalf of agents — discovery first, execution later
- Any claim to "complete" endpoint coverage — we surface *observed* endpoints only

---

## Architecture

```
POST /crawl  →  [Browser + CDP Interceptor]  →  [Traffic Capture]
                        ↓
              [Normalizer / Clusterer]
                        ↓
              [Endpoint Registry (bbolt)]
                        ↓
GET /endpoints/{domain}  →  structured endpoint map
```

Two phases, strictly separated:

### Phase A — Observation

Headless Chrome loads the target URL. A CDP network interceptor records every XHR/fetch request and response verbatim as `ObservedRequest` records. The browser optionally drives a configured interaction script (click nav items, scroll, submit search) to trigger more API calls.

Nothing is inferred here. Everything saved is directly witnessed.

### Phase B — Normalization

After capture, a normalizer groups `ObservedRequest` records by structural similarity:
- Path parameter detection: `/users/123` and `/users/456` → `/users/{id}`
- Query param clustering: requests with the same base path and overlapping query keys
- GraphQL: group by `operationName` field in request body
- Method + path deduplication: retain unique examples, not duplicates

Output is a set of `EndpointPattern` records with representative request/response examples attached.

---

## Data Model

```go
// One raw intercepted network call. Never modified after save.
type ObservedRequest struct {
    ID          string
    Domain      string
    CrawlRunID  string
    URL         string
    Method      string
    RequestHeaders  map[string]string
    RequestBody     []byte        // raw, may be JSON/form/multipart
    StatusCode      int
    ResponseHeaders map[string]string
    ResponseBody    []byte        // raw
    ContentType     string
    Timestamp       time.Time
    TriggerPage     string        // which page URL was loaded when this fired
    Authenticated   bool          // was there a session cookie/token present?
}

// Normalized endpoint pattern derived from one or more ObservedRequests
type EndpointPattern struct {
    ID          string
    Domain      string
    Method      string
    PathPattern string            // e.g. /api/users/{id}/posts
    QueryParams []ParamSpec
    BodySchema  *JSONSchema       // inferred from observed bodies, nullable
    ResponseSchema *JSONSchema    // inferred from observed responses, nullable
    GraphQLOperation string       // non-empty for GraphQL
    AuthRequired    bool
    Confidence      float64       // 0–1; higher = more observations, stable schema
    ObservationIDs  []string      // back-references to raw records
    FirstSeen   time.Time
    LastSeen    time.Time
}

type ParamSpec struct {
    Name     string
    Example  string
    Required bool
}

// A single crawl job
type CrawlRun struct {
    ID          string
    Domain      string
    SeedURL     string
    Status      string    // queued | running | done | failed
    StartedAt   time.Time
    FinishedAt  *time.Time
    Pages       []string  // URLs visited
    RequestsN   int       // total observed requests
    Error       string
}
```

Key design property: `ObservedRequest` is append-only ground truth. `EndpointPattern` is derived and can be recomputed from raw records at any time.

---

## Components

### 1. CDP Interceptor (`internal/intercept`)

Attaches to a Rod-managed Chrome instance via Chrome DevTools Protocol. Hooks:
- `Network.requestWillBeSent` — captures outgoing requests
- `Network.responseReceived` + `Network.loadingFinished` — captures responses and bodies

Filters out:
- Same-origin navigation (page loads, not API calls)
- Static asset requests (`.js`, `.css`, `.png`, font files, etc.)
- First-party page HTML (`text/html` responses to navigation)

What remains after filtering is almost entirely API calls.

```go
type Interceptor struct {
    page    *rod.Page
    storage *Store        // writes ObservedRequest records
    filter  RequestFilter
}

func (i *Interceptor) Attach() error
func (i *Interceptor) Detach()
```

### 2. Crawler (`internal/crawler`)

Drives the browser through pages to maximize API call coverage.

**Default behavior (no interaction script):**
1. Load seed URL, wait for network idle
2. Follow same-origin `<a>` links up to configured depth/page budget
3. For each page: wait for network idle, collect intercepted requests

**With interaction script (optional, configured per-domain):**
- Click specified elements (nav, tabs, dropdowns)
- Submit search forms with test queries
- Scroll to trigger infinite scroll / lazy loading

The crawler does not try to infer site structure. It just navigates and lets the interceptor record.

```go
type Crawler struct {
    browser     *rod.Browser
    interceptor *Interceptor
    config      CrawlConfig
}

type CrawlConfig struct {
    SeedURL     string
    MaxPages    int           // default: 20
    MaxDepth    int           // default: 3
    SameOrigin  bool          // default: true
    IdleTimeout time.Duration // wait-for-network-idle timeout per page
    RespectRobotsTxt bool
    Interactions []Interaction // optional click/scroll/submit steps
}

func (c *Crawler) Run(ctx context.Context) (*CrawlRun, error)
```

### 3. Normalizer (`internal/normalize`)

Post-crawl pass over `ObservedRequest` records for a given `CrawlRun`. Produces `EndpointPattern` records.

Steps:
1. **Path parameterization:** Replace numeric segments and UUID-shaped segments with `{id}` or typed placeholders. Use frequency analysis — a path segment that varies across otherwise-identical paths is a parameter.
2. **Query key clustering:** Group requests sharing the same base path and at least N overlapping query param keys.
3. **GraphQL detection:** If `Content-Type: application/json` and body contains `{"query":` or `{"operationName":`, treat as GraphQL. Group by `operationName`.
4. **Schema inference:** For JSON bodies/responses, walk the union of all observed examples to produce a loose JSON Schema (all keys seen, types observed, no required constraints unless always present).
5. **Confidence scoring:** `observations / (observations + 1)` — more observed examples = higher confidence.

```go
type Normalizer struct{}

func (n *Normalizer) Normalize(runs []ObservedRequest) ([]EndpointPattern, error)
```

### 4. Registry (`internal/registry`)

bbolt-backed store. Two buckets:

- `observed_requests` — keyed by `{domain}/{run_id}/{request_id}`, stores raw `ObservedRequest`
- `endpoint_patterns` — keyed by `{domain}/{method}/{path_pattern_hash}`, stores `EndpointPattern`
- `crawl_runs` — keyed by `{domain}/{run_id}`, stores `CrawlRun`

TTL is tracked in `EndpointPattern.LastSeen`. Staleness is surfaced in the API but not auto-deleted.

### 5. API Server (`internal/api`)

Modeled after Cloudflare's Browser Rendering API: simple verbs on domain resources.

```
POST   /v1/crawl                         → Start a crawl job
GET    /v1/crawl/{run_id}                → Get job status + stats
GET    /v1/domains/{domain}/endpoints    → List all EndpointPatterns for a domain
GET    /v1/domains/{domain}/raw          → List raw ObservedRequests for a domain
GET    /v1/domains/{domain}/endpoints/{id} → Single endpoint with all examples
DELETE /v1/domains/{domain}              → Delete all data for a domain
GET    /v1/domains                       → List all known domains
```

**POST /v1/crawl** request:
```json
{
  "url": "https://example.com",
  "max_pages": 20,
  "interactions": [
    { "type": "click", "selector": "nav a" },
    { "type": "search", "selector": "input[type=search]", "query": "test" }
  ]
}
```

Response:
```json
{ "run_id": "01J...", "status": "queued" }
```

Crawls run as background jobs. The API returns `run_id` immediately. Poll `GET /v1/crawl/{run_id}` for status.

**GET /v1/domains/{domain}/endpoints** response:
```json
{
  "domain": "example.com",
  "endpoints": [
    {
      "id": "...",
      "method": "GET",
      "path_pattern": "/api/v2/products/{id}",
      "query_params": [{"name": "include", "example": "variants"}],
      "response_schema": { "type": "object", "properties": { "id": {"type": "integer"}, "name": {"type": "string"} } },
      "auth_required": false,
      "confidence": 0.92,
      "observation_count": 12,
      "last_seen": "2026-03-29T10:00:00Z"
    }
  ]
}
```

---

## Directory Structure

```
universalcrawl/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── intercept/
│   │   ├── interceptor.go       # CDP network event hooks
│   │   └── filter.go            # Static asset / nav request filtering
│   ├── crawler/
│   │   ├── crawler.go           # Page navigation driver
│   │   └── interaction.go       # Click/scroll/search automation
│   ├── normalize/
│   │   ├── normalize.go         # Orchestrates all normalization passes
│   │   ├── path.go              # Path parameterization
│   │   ├── cluster.go           # Query param clustering
│   │   ├── graphql.go           # GraphQL-specific handling
│   │   └── schema.go            # JSON schema inference from examples
│   ├── registry/
│   │   ├── registry.go          # bbolt store
│   │   └── models.go            # ObservedRequest, EndpointPattern, CrawlRun
│   └── api/
│       ├── server.go            # chi router
│       ├── handlers.go
│       └── middleware.go
├── go.mod
└── plan.md
```

---

## Build Phases

### Phase 1 — Core Capture Pipeline
- [ ] `go.mod` init + deps (rod, chi, bbolt)
- [ ] `ObservedRequest`, `EndpointPattern`, `CrawlRun` types
- [ ] CDP interceptor: attach, filter, save raw requests
- [ ] Basic crawler: load URL, wait for idle, collect
- [ ] Registry: store/retrieve observed requests
- [ ] API: `POST /v1/crawl`, `GET /v1/crawl/{id}`, `GET /v1/domains/{domain}/raw`

### Phase 2 — Normalization
- [ ] Path parameterization pass
- [ ] Query param clustering
- [ ] GraphQL detection and grouping
- [ ] JSON schema inference
- [ ] Confidence scoring
- [ ] `GET /v1/domains/{domain}/endpoints`

### Phase 3 — Coverage Breadth
- [ ] Multi-page crawl (follow same-origin links)
- [ ] Interaction scripts (click nav, submit search)
- [ ] robots.txt respect + crawl politeness (rate limiting, delay)
- [ ] Anti-bot detection awareness (user-agent, realistic timing)

### Phase 4 — Auth Flows
- [ ] Credential store (AES-256-GCM encrypted bbolt bucket)
- [ ] Pre-crawl login sequence (form login, token injection)
- [ ] Session persistence across crawl runs
- [ ] Mark authenticated vs. unauthenticated endpoints separately

### Phase 5 — Execution (Gated)
- [ ] Replay engine: given an `EndpointPattern` + params, fire the real request
- [ ] Safety classification: `read` / `write` / `auth` / `high_risk`
- [ ] Default policy: allow `read`, block `write`/`high_risk` unless explicitly unlocked
- [ ] `POST /v1/domains/{domain}/execute` (read-only by default)

---

## Key Technical Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Language | Go | Single binary, goroutine concurrency for parallel crawls |
| Browser automation | Rod (`go-rod/rod`) | Native Go, CDP, network interception without proxy |
| HTTP router | chi | Lightweight, middleware-composable |
| Capability store | bbolt | Embedded, zero-config, pure Go |
| Job model | Background goroutine + status in registry | Crawls are async; HTTP request should return immediately |
| Raw storage | Append-only `ObservedRequest` | Ground truth; normalization is always re-derivable |
| Schema inference | Union of all observed examples | Conservative — only claims what was actually seen |

---

## What This Is Not

- Not a full API documentation generator (we only see what the browser triggers, not the full backend surface)
- Not an executor by default (Phase 5 only, write-ops blocked)
- Not a replacement for official API docs when they exist
- Not capable of seeing server-to-server calls, internal microservices, or endpoints never triggered by the crawled flows

The output is: *"here are the API calls this website's frontend made while we watched it."* That framing is always accurate.
