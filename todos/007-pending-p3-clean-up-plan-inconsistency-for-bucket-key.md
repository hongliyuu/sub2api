---
status: resolved
priority: p3
issue_id: "007"
tags: [code-review, documentation, rate-limiting]
dependencies: []
---

# Clean up plan inconsistency: bucket key mentions without provider

## Problem Statement
The plan mostly uses `(account_id, provider_id, exact_model_id)` as the bucket key, but a few remaining sentences still reference `(account_id, exact_model_id)`.

This is documentation-only, but it can cause confusion during implementation.

## Findings
- `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md` includes one lingering mention in the Proposed Solution section.

## Proposed Solutions
### Option A: Edit plan to be consistent (Small)
Replace remaining references with the final key shape and optionally show an example key format.

Pros:
- Removes ambiguity.

Cons:
- None.

## Recommended Action

## Acceptance Criteria
- [x] Plan uses a single bucket key definition consistently.

## Work Log
### 2026-01-30 - Created from plan review

**By:** OpenCode

**Actions:**
- Captured minor documentation inconsistency.

### 2026-01-31 - Clarified bucket terminology

**By:** OpenCode

**Actions:**
- Clarified that the bucket key tuple `(account_id, provider_id, quota_bucket_id)` is distinct from the scope label, and updated the agent signal section to describe both clearly in `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`.

## Resources
- Plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
