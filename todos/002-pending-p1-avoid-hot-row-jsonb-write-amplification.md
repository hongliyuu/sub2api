---
status: pending
priority: p1
issue_id: "002"
tags: [code-review, performance, database, rate-limiting]
dependencies: []
---

# Avoid hot-row JSONB write amplification on 429 storms

## Problem Statement
Per-model cooldowns are written on the error path (429), which spikes exactly when the system is under stress. If the chosen storage is `accounts.extra` JSONB, we risk turning upstream rate limiting into hot-row updates and lock contention, causing cascading latency and potentially replication/WAL pressure.

## Findings
- Option A stores cooldown state in `accounts.extra.model_rate_limits` (JSONB).
- Under bursty 429 conditions, many workers can attempt to update the same account row.
- JSONB updates can be TOAST-heavy and can cause row-level lock contention.
- Plan calls out monotonic semantics but doesn’t yet specify write suppression/debouncing (beyond monotonicity).

## Proposed Solutions
### Option A: Write suppression + monotonic max-only (Small)
Only persist cooldown when `new_reset_at` strictly extends the stored value, optionally with a small epsilon threshold.

Pros:
- High impact, low effort.

Cons:
- Still leaves hot-row contention under certain traffic patterns.

### Option B: Separate table for cooldowns (Medium/Large)
Move to a normalized table keyed by `(account_id, provider_id, exact_model_id)` with an index on `rate_limit_reset_at`.

Pros:
- Reduces JSON rewrite amplification; better queryability.

Cons:
- Requires migration + ent schema + operational rollout.

### Option C: Add a short-lived external gate (Medium)
Use a TTL store (e.g., Redis) for fast gating and reduce DB writes; persist less frequently.

Pros:
- Reduces DB hot-row writes and can smooth stampedes.

Cons:
- Adds another storage system and coherency considerations.

## Recommended Action

- Suppress JSONB writes in `accounts.extra.model_rate_limits` unless the incoming reset timestamp strictly extends the value currently stored for that key (with a short epsilon for clock skew). The repository layer already implements this via a conditional `jsonb_set` update that checks the existing `rate_limit_reset_at` before mutating the row, so the majority of concurrent 429s become noop reads instead of competing writes.
- Keep the existing JSONB path for now, but monitor `cooldown_write_total` vs. DB lock wait metrics; revisit the dedicated cooldown table (Option B) if contention remains under sustained bursts.
- Optionally, consider a short-lived cache/gate (Option C) for even faster suppression later, but ship the conditional update first because it reuses existing schema and keeps release scope small.

## Technical Details
 - Plan storage option: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
 - Hot row candidate: `accounts` table row per account with `extra.model_rate_limits`

## Acceptance Criteria
- [ ] Cooldown writes are suppressed when they do not extend `reset_at`.
- [ ] System remains stable under sustained upstream 429 conditions (no runaway DB lock waits).
- [ ] Metrics exist for `cooldown_write_total` and DB lock wait/latency signals.

## Work Log
### 2026-01-30 - Created from plan review

**By:** OpenCode

**Actions:**
- Captured risk that JSONB-in-accounts becomes a bottleneck under throttling.

**Learnings:**
- Error-path persistence needs explicit write amplification controls.

## Resources
- Plan: `docs/plans/2026-01-30-fix-per-model-rate-limit-isolation-plan.md`
