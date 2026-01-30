---
status: pending
priority: p1
issue_id: "009"
tags: [code-review, correctness, rate-limiting]
dependencies: []
---

# Use effective model (post-mapping) for schedulability checks

## Problem Statement
Cooldowns are written using the model actually sent upstream (`effectiveModel` / `mappedModel`). However, schedulability gating uses `Account.IsSchedulableForModel(requestedModel)` and can end up checking cooldown keys derived from the requested model instead of the mapped model.

This can cause the scheduler to repeatedly select accounts that are cooled down for the effective model, increasing error rates and retry load.

## Findings
- Writes use mapped model:
  - `backend/internal/service/openai_gateway_service.go` uses `mappedModel`.
  - `backend/internal/service/gateway_service.go` uses `reqModel` after mapping.
  - `backend/internal/service/gemini_messages_compat_service.go` passes `mappedModel`.
- Scheduling gate uses requested model:
  - `backend/internal/service/antigravity_quota_scope.go` `Account.IsSchedulableForModel(requestedModel)` calls `a.isModelRateLimited(requestedModel)`.
- Mapping is account-dependent (`account.GetMappedModel(requestedModel)`), so the correct key for gating may require the same mapping logic.

## Proposed Solutions

### Option A: Apply mapping inside `IsSchedulableForModel` (Small/Medium)
If account type/platform mapping applies, compute effective model from requested and use that for cooldown keying.

Pros:
- Single choke point used broadly.

Cons:
- Must be careful not to change behavior for accounts where mapping should not apply.

### Option B: Introduce a shared helper for `effectiveModel` resolution (Medium)
Create one function used by both (1) request forwarding and (2) schedulability checks.

Pros:
- Prevents drift between write-path and read-path.

Cons:
- Requires refactoring multiple call sites.

## Recommended Action

## Technical Details
- Scheduling gate: `backend/internal/service/antigravity_quota_scope.go`
- Model cooldown read: `backend/internal/service/model_rate_limit.go`
- Mapping source: `backend/internal/service/account.go` (`GetMappedModel`)

## Acceptance Criteria
- [ ] If a model is mapped for upstream, the schedulability check uses the same mapped model for cooldown gating.
- [ ] Repeated 429s on a mapped model do not result in repeated re-selection of the same account.

## Work Log
### 2026-01-30 - Created from code review

**By:** OpenCode

**Actions:**
- Captured correctness bug risk: mismatched model identity between cooldown write and gating.

## Resources
- Branch: `fix/per-model-rate-limit-isolation`
