---
status: pending
priority: p2
issue_id: "011"
tags: [code-review, reliability, performance, rate-limiting]
dependencies: []
---

# Revisit reset-time clamping and reset-source strategy

## Problem Statement
Reset time parsing now aggregates multiple sources and clamps the final reset to a maximum of 24 hours. Some providers/quota windows can legitimately exceed 24 hours (multi-day windows). Hard clamping can cause the system to retry too early, creating repeated 429 storms.

## Findings
- `backend/internal/service/ratelimit_service.go`:
  - selects `resetAt` as the latest future time among candidates.
  - clamps to `[now+1s, now+24h]`.
- This is safe against pathological headers but may be incorrect for long reset windows.

## Proposed Solutions

### Option A: Source-aware max clamp (Small/Medium)
Apply stricter clamps to untrusted sources (e.g., `Retry-After`, body parsing), but allow longer windows for trusted provider headers.

### Option B: Increase global max clamp (Small)
Raise max to something like 7–10 days while keeping the min clamp.

### Option C: Remove max clamp and keep only min clamp (Small)
Accept long resets, relying on key safety and provider parsing to avoid wedging.

## Recommended Action

## Acceptance Criteria
- [ ] Legitimate multi-day resets do not get reduced to 24h.
- [ ] Pathological values still cannot wedge scheduling for weeks/months.

## Work Log
### 2026-01-30 - Created from code review

**By:** OpenCode

**Actions:**
- Captured correctness/perf risk: too-aggressive 24h clamp.
