---
status: pending
priority: p3
issue_id: "013"
tags: [code-review, quality, refactor, rate-limiting]
dependencies: []
---

# Rename model rate limit "scope" to "key" for clarity

## Problem Statement
The code now stores entries under `accounts.extra.model_rate_limits` keyed by strings like `"gemini:gemini-3-pro"`. The repository method parameter is still named `scope`, which is confusing alongside Antigravity quota scopes and legacy Sonnet scope.

## Findings
- `backend/internal/repository/account_repo.go`: `SetModelRateLimit(..., scope string, ...)` now expects a storage key.
- Several comments/constants still use “scope” terminology even though the semantics are “bucket key”.

## Proposed Solutions

### Option A: Rename params + constants (Small)
Rename `scope` -> `key` in interfaces, repos, mocks, and update constant names for clarity.

## Recommended Action

## Acceptance Criteria
- [ ] No remaining confusion between Antigravity quota scopes and model cooldown keys.

## Work Log
### 2026-01-30 - Created from code review

**By:** OpenCode

**Actions:**
- Captured naming cleanup for maintainability.
