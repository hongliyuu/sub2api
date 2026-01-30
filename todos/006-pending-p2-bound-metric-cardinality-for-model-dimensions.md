---
status: pending
priority: p2
issue_id: "006"
tags: [code-review, observability, performance, rate-limiting]
dependencies: []
---

# Bound metric cardinality for model dimensions

## Problem Statement
The plan proposes metrics by `(provider, model)`. If `model` is derived from user input (even after normalization), metrics and logs can become high-cardinality and expensive, and can also become a log-injection vector.

## Findings
- Model identifiers are at least partially untrusted.
- Under throttling, metrics/log volume spikes.
- The plan mentions tracking distributions, but does not require cardinality controls.

## Proposed Solutions
### Option A: Use model buckets (Small)
Track top-N known models explicitly and aggregate the long tail into `other`.

Pros:
- Predictable time series count.

Cons:
- Less granular for long-tail debugging.

### Option B: Hash + sample (Medium)
Hash model keys for debugging correlation and sample detailed events.

Pros:
- Limits cardinality while retaining some diagnosis ability.

Cons:
- Requires careful operational ergonomics.

## Recommended Action

## Acceptance Criteria
- [ ] No high-cardinality unbounded labels derived from user input.
- [ ] Logs sanitize model strings (no newlines/control chars).
- [ ] Rollout dashboards remain stable in series count.

## Work Log
### 2026-01-30 - Created from plan review

**By:** OpenCode

**Actions:**
- Captured performance/ops risk: model strings in metrics can explode cardinality.

## Resources
- Plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
