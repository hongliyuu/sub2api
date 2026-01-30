---
status: pending
priority: p2
issue_id: "010"
tags: [code-review, correctness, rate-limiting]
dependencies: []
---

# Pass effective model through retry-exhausted side effects

## Problem Statement
One code path still calls retry-exhausted side effects without an effective model, which may cause fallback to global cooldown behavior when a model-scoped cooldown is intended.

## Findings
- `backend/internal/service/gateway_service.go` has a call that passes `""` as effective model in `handleRetryExhaustedError`.
- When `effectiveModel` is empty, `RateLimitService` cannot compute a bucket key, and the system may fall back to global `SetRateLimited`.

## Proposed Solutions

### Option A: Thread `reqModel` through retry-exhausted handlers (Small)
Pass the mapped/effective request model into `handleRetryExhaustedError` and into `handleRetryExhaustedSideEffects`.

Pros:
- Minimal change.

Cons:
- Must ensure the model is available at that call site.

### Option B: Use request context storage (Small/Medium)
Store effective model in gin context keys and retrieve in retry-exhausted handlers.

Pros:
- Avoids changing function signatures.

Cons:
- More implicit coupling.

## Recommended Action

## Acceptance Criteria
- [ ] Retry-exhausted 429s can still write model-scoped cooldowns when model is known.

## Work Log
### 2026-01-30 - Created from code review

**By:** OpenCode

**Actions:**
- Captured remaining path that drops effective model.
