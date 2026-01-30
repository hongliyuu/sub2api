---
status: pending
priority: p3
issue_id: "014"
tags: [code-review, simplicity, rate-limiting]
dependencies: []
---

# Simplify 429 reset-time selection logic

## Problem Statement
`RateLimitService.handle429` collects candidate reset times into a slice and then selects the best. This is correct but slightly more complex than needed for a hot error path.

## Findings
- `backend/internal/service/ratelimit_service.go` builds `candidates []time.Time` and then calls `latestFutureTime`.

## Proposed Solutions

### Option A: Compute best candidate incrementally (Small)
Track a single best reset time as you parse sources, avoiding slice allocations.

Pros:
- Slightly lower GC pressure under 429 storms.

Cons:
- Minor change; not a functional fix.

## Recommended Action

## Acceptance Criteria
- [ ] Behavior remains identical (same reset chosen for the same inputs).

## Work Log
### 2026-01-30 - Created from code review

**By:** OpenCode

**Actions:**
- Captured simplification opportunity for a hot error-path function.
