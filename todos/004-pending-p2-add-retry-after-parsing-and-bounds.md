---
status: resolved
priority: p2
issue_id: "004"
tags: [code-review, reliability, rate-limiting]
dependencies: []
---

# Add Retry-After parsing and bounded cooldown defaults

## Problem Statement
The plan proposes header parsing precedence starting with `Retry-After`, but the repo currently only allowlists `Retry-After` for passthrough; it does not parse it to compute cooldown state. Without explicit parsing and bounds, model cooldown timing will remain inconsistent across providers.

## Findings
- `Retry-After` is allowlisted in `backend/internal/util/responseheaders/responseheaders.go`.
- Current upstream error handling parses provider-specific signals (OpenAI x-ratelimit / Codex headers, Anthropic reset headers, body parsing fallbacks), but does not incorporate `Retry-After` as an input.
- The plan does not yet specify clamp rules (min/max) as acceptance criteria.

## Proposed Solutions
### Option A: Implement Retry-After parsing centrally (Small)
Parse `Retry-After` (seconds or HTTP-date) and use it as the highest-precedence reset signal.

Pros:
- Standard, provider-agnostic.

Cons:
- Must handle invalid/negative values safely.

### Option B: Keep provider-specific parsing first (Small)
Only use `Retry-After` as a fallback when provider headers are absent.

Pros:
- Minimizes behavior change.

Cons:
- Less consistent; can miss useful standard signals.

## Recommended Action

## Technical Details
- Code reference: `backend/internal/util/responseheaders/responseheaders.go`
- Likely touchpoints: `backend/internal/service/ratelimit_service.go`, gateway error handlers

## Acceptance Criteria
- [ ] `Retry-After` seconds and HTTP-date formats are supported.
- [ ] Cooldown is clamped to safe bounds (min/max) with small jitter to avoid stampedes.
- [ ] Invalid `Retry-After` values do not crash or wedge cooldown state.

## Work Log
### 2026-01-30 - Created from plan review

**By:** OpenCode

**Actions:**
- Captured missing gap between desired precedence and current implementation.

### 2026-01-30 - Implemented Retry-After parsing and bounded cooldowns

**By:** OpenCode

**Actions:**
- Added `parseRetryAfter`, `applyResetJitter`, and clamped cooldown logic in `backend/internal/service/ratelimit_service.go` so `Retry-After` values are honored safely.
- Documented the parsing/jitter/bounds behavior in the per-model rate limit plan and added tests covering seconds/date parsing, clamp bounds, and jitter range.

## Resources
- Plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
