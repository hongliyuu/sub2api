---
status: pending
priority: p2
issue_id: "005"
tags: [code-review, api, agent-native, rate-limiting]
dependencies: []
---

# Make model-scoped rate limit errors machine-actionable for agents

## Problem Statement
Per-model cooldowns are most useful when clients (including agents) can reliably detect: (1) that a request is rate-limited, (2) the scope (model vs provider/global), (3) which model key is blocked, and (4) when to retry.

The plan currently treats additive fields like `error.code`, `error.model`, and `error.retry_after_seconds` as optional.

## Findings
- Agents can’t depend on provider-specific headers because headers are filtered and providers differ.
- Without a stable scope/model/retry hint, agents may spam retries or switch models ineffectively.
- The plan already suggests optional additions but does not commit to them as a contract.

## Proposed Solutions
### Option A: Add stable JSON fields on 429 (Small/Medium)
Guarantee additive fields like:
- `error.code` (e.g., `model_rate_limited`)
- `error.scope` (`model|provider|account|unknown`)
- `error.model` (required when scope=model)
- `error.retry_after_seconds` (required on all 429)

Pros:
- Provider-agnostic, agent-friendly.

Cons:
- Expands public API surface (additive).

### Option B: Add stable `x-sub2api-*` headers (Small/Medium)
Provide stable headers for scope/model/reset; keep body unchanged.

Pros:
- Minimal body changes.

Cons:
- Some clients ignore headers; still filtered lists must include them.

## Recommended Action

## Technical Details
- Plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
- Headers filter: `backend/internal/util/responseheaders/responseheaders.go`

## Acceptance Criteria
- [ ] Agents can programmatically decide whether switching models is likely to help.
- [ ] A 429 includes retry guidance even when upstream headers are missing.
- [ ] No sensitive internal routing/account data is leaked.

## Work Log
### 2026-01-30 - Created from plan review

**By:** OpenCode

**Actions:**
- Converted agent-native review feedback into a tracked todo.

## Resources
- Plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
