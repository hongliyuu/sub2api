---
status: done
priority: p2
issue_id: "012"
tags: [code-review, api, agent-native, rate-limiting]
dependencies: []
---

# Expose machine-readable model-bucket rate limit signals to clients

## Problem Statement
Internal scheduling can now avoid a rate-limited model bucket and still use other buckets. However, API clients (including agents) still receive generic 429 responses with no stable indication of scope (bucket vs global) or which bucket was cooled down.

## Findings
- Error responses for 429 are generic in gateway error writers.
- `RateLimitService` computes reset times internally (including `Retry-After` parsing), but error responses do not consistently include a retry hint.

## Proposed Solutions

### Option A: Add stable headers on 429 (Small/Medium)
Always set:
- `Retry-After` (computed if missing)
- `X-Sub2API-RateLimit-Scope: model_bucket|account|unknown`
- `X-Sub2API-RateLimit-Key: <provider>:<bucket>` when available

### Option B: Add additive JSON fields (Medium)
Include `error.code`, `error.scope`, `error.quota_bucket`, `error.retry_after_seconds`.

## Recommended Action

## Acceptance Criteria
- [ ] An agent can programmatically decide whether switching models/buckets is likely to work.
- [ ] 429 responses always include retry guidance, even when upstream headers are missing.
- [ ] No sensitive internal routing/account ids are leaked.

## Work Log
### 2026-01-30 - Created from code review

**By:** OpenCode

**Actions:**
- Captured agent-native gap: internal behavior improved but external signals remain generic.

### 2026-01-31 - Emit machine-readable throttling signals

**By:** OpenCode

**Actions:**
- Added `RateLimitSignal` plumbing so 429 handling now returns the cooled bucket context.
- Surface `Retry-After`, `X-Sub2API-RateLimit-Scope`, and `X-Sub2API-RateLimit-Key` in Gateway, OpenAI, and Gemini responses.
- Documented the new header contract in `docs/reference/rate-limit-signals.md`.
