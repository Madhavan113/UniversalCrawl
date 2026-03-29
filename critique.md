# Critique of `plan.md`

## Overall Assessment

`plan.md` presents a strong high-level direction for a universal web capability server, especially in its focus on giving agents a stable interface instead of exposing them directly to HTML, DOM trees, or fragile selectors. The architecture is also sensibly divided into crawler, extractor, registry, executor, and API layers.

That said, the current plan overstates what can be reliably discovered and executed on arbitrary websites. The largest issues are around scope realism, data modeling, safety, and operational design. As written, the system risks promising a level of autonomy and endpoint extraction that browser-driven crawling cannot consistently deliver.

## Primary Critiques

### 1. The claim of handling "any website" is too broad

The plan frames the product as a server that can autonomously crawl any website and extract its endpoints. In practice, a crawler can only discover what it can directly observe:

- public pages it reaches
- forms present in rendered HTML
- navigation links
- network requests actually triggered during the crawl
- exposed API documentation such as OpenAPI or Swagger, when available

It cannot reliably infer the full backend surface area of a site. Many endpoints remain invisible unless the crawler reaches very specific authenticated states, feature-flagged flows, admin paths, or rare user journeys. Some APIs may also never be called from the browser at all.

This means the current phrasing should be narrowed. A more accurate statement would be that the system discovers observed capabilities and network endpoints reachable through crawlable site flows.

### 2. The action model assumes a cleaner world than most sites provide

The current `Action` model is attractive, but it is too opinionated for arbitrary web applications. It assumes that raw signals can be normalized into a reusable operation with:

- a stable semantic name
- a clean endpoint
- typed parameters
- a known response schema

This works for a subset of sites, but many real systems use:

- GraphQL
- signed URLs
- CSRF tokens
- opaque JSON bodies
- dynamic headers
- websocket messages
- multi-step stateful workflows
- request shapes that depend on prior DOM interaction

Because of that, the plan should separate raw observed evidence from higher-level inferred actions. Without that separation, the system will blur what it truly saw versus what it guessed.

### 3. Discovery and execution are mixed too early

The plan moves quickly from observing signals to executing actions generically. That is a risky jump. There is a major difference between:

- seeing that a request happened
- understanding why it happened
- knowing how to replay it safely
- knowing whether replaying it is read-only or mutating

An executor built on top of imperfect inference will eventually do the wrong thing on arbitrary sites. That is especially dangerous when a guessed action corresponds to deletion, checkout, purchase, profile edits, message sending, or administrative side effects.

The system should treat execution as a later-stage capability that only applies to replayable, well-understood flows with explicit safety metadata.

### 4. Safety policy is under-specified

The current design allows agents to discover and execute actions on live websites, but it does not define a strong risk model. That is one of the biggest missing pieces in the plan.

At minimum, each capability should be classified into something like:

- read
- auth
- write
- high_risk

The default policy for a general-purpose infrastructure server should probably be:

- allow read-only capabilities
- require confirmation for authentication
- block writes unless explicitly enabled
- always gate high-risk actions behind human approval

Without those controls, the executor becomes too dangerous for autonomous use.

### 5. The crawl API likely needs asynchronous job handling

The proposed API suggests that capability discovery can happen inline when a cache miss occurs. That will be brittle in practice. Real crawls may involve:

- browser startup time
- SPA hydration
- network interception
- retries
- redirects
- throttling
- login flows
- pagination or multi-page exploration

Those are not good fits for a synchronous request path. A more robust design would treat crawling as a job with status tracking, logs, timestamps, and partial progress.

## Data Model Concerns

The current core data model is too compressed for debugging and trustworthiness. A single `SiteCapability` object does not capture enough detail about how knowledge was gathered.

The system should preserve provenance for every discovered item, including:

- whether it came from OpenAPI, forms, anchors, XHR/fetch interception, or LLM inference
- when it was last observed
- what page or state produced it
- whether it required authentication
- whether it was directly observed or inferred
- what confidence score was assigned
- what example request and response were seen

Without provenance, false positives will be hard to diagnose and confidence in the capability map will degrade quickly.

## Suggested Architectural Reframe

The current architecture would be stronger if it explicitly separated three layers:

### 1. Discovery

This layer should only collect evidence:

- URLs
- forms
- scripts
- network requests
- page states
- API docs
- sitemap and robots information where relevant

Its job is observation, not interpretation.

### 2. Inference

This layer can cluster and normalize the observed evidence into candidate capabilities. It may:

- group similar requests
- infer likely parameters
- assign names
- detect likely CRUD patterns
- identify auth requirements
- estimate risk
- assign confidence scores

This is the right place for optional LLM assistance.

### 3. Replay / Execution

This layer should only operate on capabilities that are considered replayable and sufficiently understood. It should preserve:

- required headers
- cookies
- CSRF behavior
- prerequisite navigation state
- timing constraints
- safety restrictions

That separation makes the system much easier to reason about and much less likely to overclaim.

## Recommended Additional Types

To support the above, the registry should likely store more than a single capability document. Useful records would include:

- `CrawlRun`
- `PageArtifact`
- `ObservedRequest`
- `ObservedForm`
- `CandidateCapability`
- `ExecutionRecipe`

This would let the product retain raw evidence, inferred meaning, and replay configuration as separate concerns.

## MVP Recommendation

The current roadmap would be much more achievable if version one were narrowed significantly.

Recommended MVP:

1. Public websites only
2. No login flows initially
3. Discovery-first, not execution-first
4. Read-only capabilities only
5. Explicitly market the output as observed endpoint and capability discovery, not complete endpoint extraction

A strong MVP output would include:

- discovered pages
- discovered forms
- observed XHR/fetch requests
- grouped endpoint patterns
- candidate actions with confidence and provenance

That still delivers real value while keeping the first implementation realistic.

## Missing Operational Concerns

Several infrastructure concerns should be added to the plan before implementation:

- crawl frontier strategy such as max depth, same-origin rules, and page budget
- rate limiting and politeness controls
- robots.txt and legal/compliance posture
- browser pool sizing and crash recovery
- anti-bot detection handling
- concurrency limits per domain
- crawl observability such as logs, traces, screenshots, or HAR capture
- GraphQL-specific handling
- websocket and streaming request handling
- benchmark sites and evaluation criteria for recall and precision

These details will materially affect whether the system works reliably in production.

## Most Important Recommendation

The single most important change is to stop treating "observed request," "inferred capability," and "safe executable action" as the same thing.

Those should be modeled separately:

- observed evidence is factual
- inferred capability is probabilistic
- executable action is policy-gated

Once those layers are separated, the rest of the architecture becomes easier to make realistic and robust.

## Conclusion

`plan.md` is a compelling foundation for a capability discovery server, but it currently overstates what a browser-based crawler can autonomously achieve on arbitrary sites. The design becomes much stronger if it narrows the claim, stores provenance, adds risk classification, and treats execution as a gated extension rather than a default outcome of crawling.

With those changes, the project could become a credible and useful infrastructure layer for agents interacting with the web.
