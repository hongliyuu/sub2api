---
status: pending
priority: p1
issue_id: "008"
tags: [code-review, database, performance, reliability, rate-limiting]
dependencies: []
---

# Make per-model cooldown updates monotonic in the database

## Problem Statement
Per-bucket cooldowns are persisted in `accounts.extra.model_rate_limits` on upstream 429. The current implementation attempts to avoid shortening cooldowns by comparing against `account.modelRateLimitResetAt(key)` before writing.

That check is not atomic and uses an in-memory snapshot of `Account.Extra`, which can be stale across concurrent requests or instances. Under concurrency, a shorter cooldown can overwrite a longer one, creating stampedes and repeated 429 storms.

## Findings
- `backend/internal/service/ratelimit_service.go` checks existing cooldown via `account.modelRateLimitResetAt(key)` before calling `SetModelRateLimit`.
- `backend/internal/repository/account_repo.go` `SetModelRateLimit` unconditionally writes the JSON payload at the key.
- Under concurrent 429s:
  - cooldown can be shortened (correctness bug)
  - repeated writes/outbox events amplify load (performance risk)

## Proposed Solutions

### Option A: DB-level conditional update for JSONB key (Medium)
Update only when the new `rate_limit_reset_at` extends the existing value.

Pros:
- Preserves JSONB storage choice.
- Prevents cooldown shortening and reduces no-op writes.

Cons:
- JSONB timestamp extraction/comparison in SQL is more complex.

### Option B: Move cooldowns to a normalized table with UPSERT (Medium/Large)
Create `account_model_rate_limits(account_id, key, rate_limit_reset_at, rate_limited_at)` and use `INSERT .. ON CONFLICT .. DO UPDATE SET rate_limit_reset_at = GREATEST(...)`.

Pros:
- Clean monotonic semantics.
- Less JSONB rewrite pressure; better indexing and pruning.

Cons:
- Requires migration + more code changes.

### Option C: Hybrid gating (Medium)
Use Redis TTL gating to reduce DB writes and persist less frequently.

Pros:
- Reduces hot-row writes during 429 storms.

Cons:
- Adds another store and coherency complexity.

## Recommended Action

## Technical Details
- Write primitive: `backend/internal/repository/account_repo.go` `SetModelRateLimit`
- Read/check: `backend/internal/service/model_rate_limit.go`
- Callers: `backend/internal/service/ratelimit_service.go`, `backend/internal/service/gemini_messages_compat_service.go`, `backend/internal/service/antigravity_gateway_service.go`

## Acceptance Criteria
- [ ] Cooldown can never be shortened by concurrent writes.
- [ ] No-op writes (new reset <= existing reset) do not enqueue scheduler outbox events.
- [ ] Stress test: concurrent 429s do not cause repeated upstream stampedes after expiry.

## Work Log
### 2026-01-30 - Created from code review

**By:** OpenCode

**Actions:**
- Captured correctness/performance risk: non-atomic cooldown extension logic.

## Resources
- Branch: `fix/per-model-rate-limit-isolation`
