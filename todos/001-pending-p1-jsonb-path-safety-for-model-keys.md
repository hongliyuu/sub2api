---
status: done
priority: p1
issue_id: "001"
tags: [code-review, security, database, rate-limiting]
dependencies: []
---

# JSONB path safety for per-model cooldown keys

## Problem Statement
The plan proposes storing per-model cooldown state in `accounts.extra.model_rate_limits` keyed by `(provider_id, exact_model_id)`. The existing repository write primitive for `model_rate_limits` builds a `jsonb_set` path using string concatenation, and is currently safe only because keys are controlled (e.g., a fixed `claude_sonnet` scope).

If we start using arbitrary model IDs as JSON keys, special characters (commas/braces/quotes) can break the Postgres array path literal, cause SQL errors, or create injection-adjacent edge cases.

## Findings
- `backend/internal/repository/account_repo.go` currently constructs JSONB path strings like `"{model_rate_limits," + scope + "}"`.
- The current `scope` is effectively controlled (substring-derived), but the new design makes the key user-influenced via `model`.
- The plan mentions canonicalization/validation, but it does not explicitly require “safe as jsonb_set path element” constraints.

## Proposed Solutions
### Option A: Restrict and validate key characters (Small)
Constrain `(provider_id, exact_model_id)` to a safe character set before any DB write.

Pros:
- Minimal code change surface.

Cons:
- Still relies on fragile string-path assembly.
- Hard to guarantee across all future changes.

### Option B: Change repository write primitive to pass structured path (Medium)
Stop using ad-hoc array-literal strings; use a safer approach to address JSON paths (or represent keys in a way that does not require embedding arbitrary strings inside a path literal).

Pros:
- Robust by construction.

Cons:
- More implementation work; must ensure ent/sql builder supports it.

### Option C: Encode model key (Small/Medium)
Encode keys (e.g., URL-safe base64) before storage, and decode at read-time.

Pros:
- Avoids unsafe characters entirely.

Cons:
- Reduces human readability in DB.

## Recommended Action

- Describe the `service.ModelRateLimitKey`/`service.IsSafeModelRateLimitKey` guard in the plan so reviewers know why the path is safe.
- Enforce the guard in `SetModelRateLimit`, refuse unsafe keys, and emit scheduler outbox payloads that include the affected `model_rate_limit_keys` so downstream caches see the change.
- Cover the helper with unit tests that exercise both good and bad characters.
## Technical Details
- Affected plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
- Affected code path: `backend/internal/repository/account_repo.go` (model_rate_limits JSONB writes)

## Acceptance Criteria
- [ ] Model/provider keys used in JSONB paths cannot break SQL or JSON path semantics.
- [ ] Attempts to write cooldown for invalid keys are rejected safely (no partial writes).
- [ ] Tests cover keys containing commas, braces, quotes, whitespace, and control characters.

## Work Log
### 2026-01-30 - Created from plan review

**By:** OpenCode

**Actions:**
- Captured risk that JSONB path construction becomes unsafe with arbitrary model keys.

**Learnings:**
- Existing `SetModelRateLimit` behavior is safe today only because scope is controlled.

### 2026-01-31 - Guarded per-model keys and documented flow

**By:** OpenCode

**Actions:**
- Added `service.ModelRateLimitKey`/`service.IsSafeModelRateLimitKey` and wired `SetModelRateLimit` to refuse unsafe keys while emitting scheduler outbox payloads that carry `model_rate_limit_keys`.
- Added unit tests that confirm the helper accepts canonical keys, rejects characters that would break JSON paths, and documented the safety mechanism in the main plan.

**Learnings:**
- The scheduler continues to rebuild from account changes when extra fields change, so carrying the key names in the outbox payload makes downstream observability easier.

## Resources
- Plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
- Repo reference: `backend/internal/repository/account_repo.go`
