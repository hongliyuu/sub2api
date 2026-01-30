---
status: pending
priority: p2
issue_id: "003"
tags: [code-review, quality, rate-limiting]
dependencies: []
---

# Canonicalization consistency for model keys

## Problem Statement
The plan introduces `exact_model_id` normalization rules that differ from existing repo behavior (which lowercases model identifiers in multiple places). If canonicalization is inconsistent across reads/writes, per-model cooldown gating will be flaky and hard to debug.

## Findings
- The plan says “do not case-fold unless model ids are guaranteed case-insensitive.”
- Existing code already lowercases in scope resolution:
  - `backend/internal/service/antigravity_quota_scope.go`
  - `backend/internal/service/model_rate_limit.go`
- Divergent normalization can lead to two separate cooldown keys for the same logical model.

## Proposed Solutions
### Option A: Standardize on current behavior (Small)
Always lowercase + trim + strip `models/` consistently across all per-model cooldown reads/writes.

Pros:
- Aligns with existing patterns.

Cons:
- Requires confidence model IDs are effectively case-insensitive in practice.

### Option B: Stop lowercasing everywhere (Medium)
Remove lowercasing from existing scope resolvers and treat model IDs as case-sensitive.

Pros:
- Strict correctness if providers are case-sensitive.

Cons:
- Likely larger refactor and higher regression risk.

### Option C: Normalize only the internal bucket key (Small)
Keep request model case for outward-facing fields, but always canonicalize the bucket key deterministically.

Pros:
- Minimizes client-visible changes.

Cons:
- Needs careful documentation.

## Recommended Action

## Technical Details
- Plan definitions: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
- Existing code: `backend/internal/service/antigravity_quota_scope.go`, `backend/internal/service/model_rate_limit.go`

## Acceptance Criteria
- [ ] Single canonicalization function is defined for bucket keys.
- [ ] All per-model cooldown reads and writes use that function.
- [ ] Tests prove that casing/whitespace differences do not create distinct buckets.

## Work Log
### 2026-01-30 - Created from plan review

**By:** OpenCode

**Actions:**
- Noted mismatch between plan guidance and current normalization patterns.

## Resources
- Plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
