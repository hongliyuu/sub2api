# Request Diagnosis & Tracing System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the admin a standardized tool to instantly see whether a request problem was caused by the downstream client, sub2api itself, or the upstream provider — with a waterfall latency view, smart fault-owner tags, and full span traces for each request.

**Architecture:** Instrument key phases (auth, routing, token-fetch, translate, upstream attempt, failover, response) as span events stored as JSONB in existing `usage_logs` and `ops_error_logs` tables. At query time, compute `fault_owner` and diagnosis summary from the raw spans. Extend the existing Ops request list UI with fault_owner pill filters, add a waterfall detail panel, and render the span tree inline.

**Tech Stack:** Go (gin, database/sql, PostgreSQL JSONB), Vue 3 (Composition API, TypeScript), Tailwind CSS, existing ops infrastructure patterns.

---

## File Structure

**New files:**
- `backend/migrations/082_add_spans_to_logs.sql` — adds `spans JSONB` to both log tables
- `backend/internal/service/ops_span.go` — `OpsSpan` struct, gin context key, helpers
- `frontend/src/views/admin/ops/components/OpsWaterfallPanel.vue` — waterfall chart for latency phases
- `frontend/src/views/admin/ops/components/OpsSpanTree.vue` — collapsible JSONB span tree

**Modified files:**
- `backend/internal/service/copilot_gateway_service.go` — add `appendOpsSpan()` calls at each phase
- `backend/internal/service/ops_upstream_context.go` — export `AppendOpsSpan()` (already has pattern)
- `backend/internal/service/ops_port.go` — add `Spans []*OpsSpan` and `SpansJSON *string` to `OpsInsertErrorLogInput`
- `backend/internal/service/usage_log.go` — add `Spans *string` to `UsageLog`
- `backend/internal/repository/ops_repo.go` — add `spans` to INSERT SQL and args (col 39)
- `backend/internal/repository/usage_log_repo.go` — add `spans` to INSERT SQL and arg types
- `backend/internal/handler/ops_error_logger.go` — read `OpsSpansKey` from context and attach to entry
- `backend/internal/handler/admin/ops_handler.go` — add `GET /admin/ops/requests/:id/spans` endpoint
- `frontend/src/api/admin/ops.ts` — add `OpsSpan` type, `OpsRequestDetail.spans`, `getRequestSpans()` fn
- `frontend/src/views/admin/ops/OpsRequestInspectView.vue` — add fault_owner filter + waterfall column
- `frontend/src/views/admin/ops/components/OpsRequestDetailPanel.vue` — embed `OpsWaterfallPanel` + `OpsSpanTree`

---

## Task 1: Database Migration — Add spans JSONB Column

**Files:**
- Create: `backend/migrations/082_add_spans_to_logs.sql`

- [ ] **Step 1: Write migration file**

```sql
-- 082_add_spans_to_logs.sql
-- Adds spans JSONB column to both error_logs and usage_logs for
-- request diagnosis and tracing. Spans store raw per-phase timing events;
-- fault_owner and diagnosis are computed at query time from this data.

ALTER TABLE ops_error_logs
  ADD COLUMN IF NOT EXISTS spans JSONB;

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS spans JSONB;

-- Partial indexes: only index rows that actually have spans data.
-- Avoids index bloat from the majority of rows without span data.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ops_error_logs_spans_gin
  ON ops_error_logs USING gin (spans)
  WHERE spans IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_spans_gin
  ON usage_logs USING gin (spans)
  WHERE spans IS NOT NULL;

COMMENT ON COLUMN ops_error_logs.spans IS
  'Per-phase span events as JSON array. Each span: {name, start_unix_ms, duration_ms, status, attrs}.';

COMMENT ON COLUMN usage_logs.spans IS
  'Per-phase span events as JSON array. Each span: {name, start_unix_ms, duration_ms, status, attrs}.';
```

- [ ] **Step 2: Register migration in migrations.go**

Open `backend/migrations/migrations.go` and add the new file to the embedded FS migrations list, following the existing pattern for 081:

```go
//go:embed 082_add_spans_to_logs.sql
var migration082 string
```

Then add it to the migrations slice:
```go
{Version: 82, Name: "add_spans_to_logs", SQL: migration082},
```

- [ ] **Step 3: Run migration and verify columns exist**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go run ./cmd/migrate/main.go up
```

Expected output: `Applying migration 082_add_spans_to_logs... done`

Then verify:
```bash
psql $DATABASE_URL -c "\d ops_error_logs" | grep spans
psql $DATABASE_URL -c "\d usage_logs" | grep spans
```

Expected: both show `spans | jsonb`

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/082_add_spans_to_logs.sql backend/migrations/migrations.go
git commit -m "Feature: 新增 spans JSONB 列到 ops_error_logs 和 usage_logs"
```

---

## Task 2: Span Data Model — OpsSpan Struct and Context Helpers

**Files:**
- Create: `backend/internal/service/ops_span.go`

- [ ] **Step 1: Write failing test**

Create `backend/internal/service/ops_span_test.go`:

```go
package service_test

import (
	"testing"
	"time"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestAppendOpsSpan_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	now := time.Now()
	service.AppendOpsSpan(c, service.OpsSpan{
		Name:        "auth.verify",
		StartUnixMs: now.UnixMilli(),
		DurationMs:  12,
		Status:      "ok",
	})

	spans := service.GetOpsSpans(c)
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "auth.verify" {
		t.Errorf("expected name auth.verify, got %s", spans[0].Name)
	}
}

func TestMarshalOpsSpans_EmptyIsNil(t *testing.T) {
	result := service.MarshalOpsSpans(nil)
	if result != nil {
		t.Errorf("expected nil for empty spans, got %v", result)
	}
}

func TestMarshalOpsSpans_Valid(t *testing.T) {
	spans := []*service.OpsSpan{
		{Name: "routing.select_account", StartUnixMs: 1000, DurationMs: 5, Status: "ok"},
	}
	result := service.MarshalOpsSpans(spans)
	if result == nil {
		t.Fatal("expected non-nil JSON string")
	}

	var parsed []*service.OpsSpan
	if err := json.Unmarshal([]byte(*result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Name != "routing.select_account" {
		t.Errorf("parsed span mismatch: %+v", parsed)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go test ./internal/service/ -run TestAppendOpsSpan -v
```

Expected: FAIL with `undefined: service.OpsSpan`

- [ ] **Step 3: Implement ops_span.go**

```go
package service

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// OpsSpansKey is the gin context key for the accumulated span slice.
	OpsSpansKey = "ops_spans"
)

// OpsSpan represents a single timed phase within a gateway request.
// Stored as a JSONB array in usage_logs.spans / ops_error_logs.spans.
//
// Phase name conventions:
//   auth.verify          — API key lookup + permission check
//   routing.select       — account selection (first attempt)
//   token.fetch          — Copilot access-token fetch
//   translate.req        — client→upstream body translation
//   upstream.post        — single upstream HTTP attempt (per failover attempt)
//   failover.select      — account re-selection after upstream error
//   translate.resp       — upstream→client body translation
//   body.truncation      — large request body truncation event
type OpsSpan struct {
	// Name identifies the phase (e.g. "auth.verify", "upstream.post").
	Name string `json:"name"`

	// StartUnixMs is the wall-clock start of this span (Unix milliseconds).
	StartUnixMs int64 `json:"start_unix_ms"`

	// DurationMs is the elapsed time for this span in milliseconds.
	DurationMs int64 `json:"duration_ms"`

	// Status is "ok", "error", or "skipped".
	Status string `json:"status,omitempty"`

	// Attrs holds span-specific key-value attributes (e.g. account_id, model, attempt).
	// Keep values small — strings and numbers only, no nested objects.
	Attrs map[string]any `json:"attrs,omitempty"`
}

// NewOpsSpan creates a span with StartUnixMs set to now.
// Call span.End() when the phase finishes.
func NewOpsSpan(name string) *OpsSpan {
	return &OpsSpan{
		Name:        name,
		StartUnixMs: time.Now().UnixMilli(),
	}
}

// End closes a span, setting DurationMs from StartUnixMs to now.
func (s *OpsSpan) End(status string) {
	if s == nil {
		return
	}
	s.DurationMs = time.Now().UnixMilli() - s.StartUnixMs
	if status != "" {
		s.Status = status
	}
}

// AppendOpsSpan appends a span to the gin context span slice.
// Safe to call with a nil context (no-op).
func AppendOpsSpan(c *gin.Context, span OpsSpan) {
	if c == nil {
		return
	}
	if span.StartUnixMs <= 0 {
		span.StartUnixMs = time.Now().UnixMilli()
	}
	var existing []*OpsSpan
	if v, ok := c.Get(OpsSpansKey); ok {
		if arr, ok := v.([]*OpsSpan); ok {
			existing = arr
		}
	}
	spanCopy := span
	existing = append(existing, &spanCopy)
	c.Set(OpsSpansKey, existing)
}

// GetOpsSpans reads the accumulated spans from the gin context.
// Returns nil if none were recorded.
func GetOpsSpans(c *gin.Context) []*OpsSpan {
	if c == nil {
		return nil
	}
	v, ok := c.Get(OpsSpansKey)
	if !ok {
		return nil
	}
	arr, _ := v.([]*OpsSpan)
	return arr
}

// MarshalOpsSpans serialises span slice to a JSON string for DB storage.
// Returns nil when spans is empty.
func MarshalOpsSpans(spans []*OpsSpan) *string {
	if len(spans) == 0 {
		return nil
	}
	raw, err := json.Marshal(spans)
	if err != nil || len(raw) == 0 {
		return nil
	}
	s := string(raw)
	return &s
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go test ./internal/service/ -run "TestAppendOpsSpan|TestMarshalOpsSpans" -v
```

Expected: PASS (3 tests)

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/ops_span.go backend/internal/service/ops_span_test.go
git commit -m "Feature: 新增 OpsSpan 数据模型和 gin context 辅助函数"
```

---

## Task 3: Propagate Spans Through the Data Pipeline

**Files:**
- Modify: `backend/internal/service/ops_port.go` — add `Spans`, `SpansJSON` fields
- Modify: `backend/internal/service/usage_log.go` — add `Spans` field
- Modify: `backend/internal/handler/ops_error_logger.go` — read spans from context
- Modify: `backend/internal/repository/ops_repo.go` — add spans to INSERT
- Modify: `backend/internal/repository/usage_log_repo.go` — add spans to INSERT

- [ ] **Step 1: Add Spans fields to OpsInsertErrorLogInput in ops_port.go**

Read the file first, then edit. Locate the `UpstreamErrorsJSON *string` field and add after it:

```go
	// Spans captures per-phase timing events for request diagnosis.
	// Populated from gin context key OpsSpansKey during handler middleware.
	Spans []*OpsSpan
	// SpansJSON is the marshalled JSON stored into ops_error_logs.spans.
	// Set by OpsService.RecordError before persisting.
	SpansJSON *string
```

- [ ] **Step 2: Add Spans field to UsageLog struct in usage_log.go**

Find the `UsageLog` struct definition. Add after `ResponseLatencyMs`:

```go
	// Spans holds per-phase timing events as a serialised JSON string.
	Spans *string
```

- [ ] **Step 3: Read spans from gin context in ops_error_logger.go**

In `applyOpsLatencyFieldsFromContext`, add span reading after the existing latency fields:

```go
func applyOpsLatencyFieldsFromContext(c *gin.Context, entry *service.OpsInsertErrorLogInput) {
	if c == nil || entry == nil {
		return
	}
	entry.AuthLatencyMs = getContextLatencyMs(c, service.OpsAuthLatencyMsKey)
	entry.RoutingLatencyMs = getContextLatencyMs(c, service.OpsRoutingLatencyMsKey)
	entry.UpstreamLatencyMs = getContextLatencyMs(c, service.OpsUpstreamLatencyMsKey)
	entry.ResponseLatencyMs = getContextLatencyMs(c, service.OpsResponseLatencyMsKey)
	entry.TimeToFirstTokenMs = getContextLatencyMs(c, service.OpsTimeToFirstTokenMsKey)
	// Attach span trace for request diagnosis
	entry.Spans = service.GetOpsSpans(c)
}
```

- [ ] **Step 4: Marshal spans in OpsService.RecordError (ops_service.go)**

Find the method that populates `UpstreamErrorsJSON` from `UpstreamErrors` and add the same pattern for spans. In `backend/internal/service/ops_service.go`, find where `UpstreamErrorsJSON` is set and add:

```go
	// Marshal spans for storage
	if len(input.Spans) > 0 {
		input.SpansJSON = MarshalOpsSpans(input.Spans)
	}
```

- [ ] **Step 5: Add spans to ops_repo.go INSERT (column 39)**

In `backend/internal/repository/ops_repo.go`, update `insertOpsErrorLogSQL`:

Change the column list to add `spans` after `created_at`:
```sql
INSERT INTO ops_error_logs (
  request_id, client_request_id, user_id, api_key_id, account_id, group_id,
  client_ip, platform, model, request_path, stream, user_agent,
  error_phase, error_type, severity, status_code, is_business_limited, is_count_tokens,
  error_message, error_body, error_source, error_owner,
  upstream_status_code, upstream_error_message, upstream_error_detail, upstream_errors,
  auth_latency_ms, routing_latency_ms, upstream_latency_ms, response_latency_ms, time_to_first_token_ms,
  request_body, request_body_truncated, request_body_bytes, request_headers,
  is_retryable, retry_count, created_at, spans
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39
)
```

Update `opsInsertErrorLogArgs` to add the 39th argument at the end:

```go
func opsInsertErrorLogArgs(input *service.OpsInsertErrorLogInput) []any {
	return []any{
		// ... existing 38 args ...
		opsNullString(ptrStringVal(input.SpansJSON)), // $39 spans
	}
}
```

Where `ptrStringVal` dereferences `*string` safely:
```go
func ptrStringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
```

(Check if this helper already exists; if so use the existing one.)

- [ ] **Step 6: Add spans to usage_log_repo.go INSERT**

In `usage_log_repo.go`, find the `INSERT INTO usage_logs` statements. There are multiple (single insert, batch insert, bulk insert). Each needs `spans` added to:
1. The column list
2. The `$N` placeholder
3. The arg type array (`"jsonb"` type)
4. The args construction (pass `log.Spans`)

For each INSERT statement, add `spans` as the last column. For example the main single INSERT:

Column list — append `, spans` before the closing paren.
Placeholder — increment to next `$N`.
Arg types array `usageLogInsertArgTypes` — append `"jsonb"`.
Args — append `sql.NullString{String: ptrStr(log.Spans), Valid: log.Spans != nil}`.

- [ ] **Step 7: Build and verify**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./...
go vet ./...
```

Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add backend/internal/service/ops_port.go \
        backend/internal/service/usage_log.go \
        backend/internal/handler/ops_error_logger.go \
        backend/internal/repository/ops_repo.go \
        backend/internal/repository/usage_log_repo.go
git commit -m "Feature: 将 spans 字段贯穿数据管道（port → handler → repo）"
```

---

## Task 4: Instrument copilot_gateway_service.go with Span Recording

**Files:**
- Modify: `backend/internal/service/copilot_gateway_service.go`

The goal is to call `AppendOpsSpan` at 7 key phases in `ForwardChatCompletions`, `ForwardResponses`, `ForwardMessages`, and `forwardMessagesViaResponses`.

- [ ] **Step 1: Add span instrumentation to ForwardChatCompletions**

Read `ForwardChatCompletions` (lines ~112–206). Add spans at:

**auth.verify** — token fetch:
```go
// Before: token, err := s.tokenProvider.GetAccessToken(ctx, account)
tokenStart := time.Now()
token, err := s.tokenProvider.GetAccessToken(ctx, account)
AppendOpsSpan(c, OpsSpan{
    Name:        "token.fetch",
    StartUnixMs: tokenStart.UnixMilli(),
    DurationMs:  time.Since(tokenStart).Milliseconds(),
    Status:      func() string {
        if err != nil { return "error" }
        return "ok"
    }(),
    Attrs: map[string]any{"account_id": account.ID},
})
if err != nil {
    return nil, fmt.Errorf("copilot auth: %w", err)
}
```

**translate.req** — body rewrite (no network call, so just mark start time before `rewriteCopilotUpstreamModel`):
```go
translateStart := time.Now()
body = mergeConsecutiveSameRoleMessagesInOpenAIBody(body)
body, logModel := rewriteCopilotUpstreamModel(body, account)
body = clampCopilotUpstreamMaxTokens(body, account)
AppendOpsSpan(c, OpsSpan{
    Name:        "translate.req",
    StartUnixMs: translateStart.UnixMilli(),
    DurationMs:  time.Since(translateStart).Milliseconds(),
    Status:      "ok",
    Attrs: map[string]any{"model": extractModelFromBody(body)},
})
```

**upstream.post** — HTTP round-trip:
```go
// Before: resp, err := s.httpClient.Do(req)
upstreamStart := time.Now()
resp, err := s.httpClient.Do(req)
upstreamDuration := time.Since(upstreamStart).Milliseconds()
SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, upstreamDuration)
upstreamStatus := "ok"
if err != nil { upstreamStatus = "error" }
AppendOpsSpan(c, OpsSpan{
    Name:        "upstream.post",
    StartUnixMs: upstreamStart.UnixMilli(),
    DurationMs:  upstreamDuration,
    Status:      upstreamStatus,
    Attrs: map[string]any{
        "account_id": account.ID,
        "endpoint":   "chat/completions",
    },
})
```

- [ ] **Step 2: Add span instrumentation to the three remaining forward functions**

Apply the same pattern to `ForwardResponses`, `ForwardMessages`, and `forwardMessagesViaResponses`. Each has the same phases: `token.fetch`, `translate.req`, `upstream.post`. Add appropriate `endpoint` attr to `upstream.post` (`"responses"`, `"messages"`).

For `ForwardMessages` and `forwardMessagesViaResponses` also add **failover.select** span in the failover loop (when a new account is selected after upstream error):

```go
// In failover loop, after SelectAccountForModelWithExclusions succeeds:
AppendOpsSpan(c, OpsSpan{
    Name:        "failover.select",
    StartUnixMs: time.Now().UnixMilli(),
    DurationMs:  0,
    Status:      "ok",
    Attrs: map[string]any{
        "attempt":    switchCount,
        "account_id": account.ID,
    },
})
```

- [ ] **Step 3: Add routing.select span in copilot_gateway_handler.go**

In `copilot_gateway_handler.go`, after `SelectAccountForModel` succeeds (first account selection, before entering the forward call), add:

```go
// After successful account selection, before forwarding:
service.AppendOpsSpan(c, service.OpsSpan{
    Name:        "routing.select",
    StartUnixMs: routingStart.UnixMilli(), // set routingStart before SelectAccountForModel
    DurationMs:  time.Since(routingStart).Milliseconds(),
    Status:      "ok",
    Attrs: map[string]any{"account_id": account.ID, "platform": "copilot"},
})
```

Note: `routingStart` should be recorded just before the `SelectAccountForModel` call. Check `copilot_gateway_handler.go` for the exact call site.

- [ ] **Step 4: Build and verify**

```bash
cd /Users/ziji/personal/github/sub2api/backend
go build ./...
go vet ./...
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/copilot_gateway_service.go \
        backend/internal/handler/copilot_gateway_handler.go
git commit -m "Feature: 在 copilot gateway 的关键阶段添加 span 埋点"
```

---

## Task 5: Expose Spans in the Request Detail API

**Files:**
- Modify: `backend/internal/service/ops_port.go` — add `SpansJSON` to `OpsRequestDetail`
- Modify: `backend/internal/repository/ops_repo.go` — include `spans` in `ListRequestDetails` SELECT
- Modify: `backend/internal/handler/admin/ops_handler.go` — return spans in request detail response

- [ ] **Step 1: Add Spans to OpsRequestDetail struct in ops_port.go**

Find the `OpsRequestDetail` struct. Add after `ResponseLatencyMs`:

```go
	// SpansJSON is the serialised JSON spans array for this request (if recorded).
	SpansJSON *string `json:"spans,omitempty"`
```

- [ ] **Step 2: Include spans in ListRequestDetails query**

In `ops_repo.go`, find `ListRequestDetails`. The UNION query selects from `usage_logs` and `ops_error_logs`. Add `spans` to both SELECT arms:

For `usage_logs` arm:
```sql
-- Add to SELECT: COALESCE(spans::text, '') as spans_json
```

For `ops_error_logs` arm:
```sql
-- Add to SELECT: COALESCE(spans::text, '') as spans_json
```

Then scan `spans_json` into `OpsRequestDetail.SpansJSON`.

- [ ] **Step 3: Build and verify**

```bash
go build ./...
go vet ./...
```

Expected: no errors

- [ ] **Step 4: Manual API test**

```bash
# Start dev server
cd /Users/ziji/personal/github/sub2api && ./scripts/dev.sh

# Make a test request, get request_id from logs
# Then call the API:
curl -s "http://localhost:3000/api/v1/admin/ops/requests?time_range=1h&kind=all&page=1&page_size=5" \
  -H "Authorization: Bearer <admin_token>" | jq '.data[0].spans'
```

Expected: `null` for old requests, JSON array for new instrumented requests.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/ops_port.go \
        backend/internal/repository/ops_repo.go \
        backend/internal/handler/admin/ops_handler.go
git commit -m "Feature: 在请求列表 API 中暴露 spans 字段"
```

---

## Task 6: Frontend — OpsSpan Types and API Helpers

**Files:**
- Modify: `frontend/src/api/admin/ops.ts`

- [ ] **Step 1: Add OpsSpan types and update OpsRequestDetail**

Open `frontend/src/api/admin/ops.ts`. Add after the `OpsUpstreamErrorEvent` type:

```typescript
/** A single timed phase within a gateway request. */
export interface OpsSpan {
  name: string
  start_unix_ms: number
  duration_ms: number
  status?: 'ok' | 'error' | 'skipped'
  attrs?: Record<string, string | number | boolean>
}

/** Computed fault owner for a request — determined from spans + status_code at query time. */
export type FaultOwner = 'client' | 'upstream' | 'platform' | 'ok'
```

Then update `OpsRequestDetail` interface — add after `response_latency_ms`:

```typescript
  /** Raw span trace for this request. Populated from spans JSONB column. */
  spans?: OpsSpan[] | null

  /**
   * Computed fault owner.
   * 'client'   — 4xx caused by client (bad auth, bad request)
   * 'upstream' — upstream provider returned 5xx or timeout
   * 'platform' — sub2api internal error (routing, internal)
   * 'ok'       — successful request (status < 400)
   */
  fault_owner?: FaultOwner | null
```

- [ ] **Step 2: Add computeFaultOwner utility function**

Add this helper at the bottom of the file, before the exported API functions:

```typescript
/**
 * Derive fault owner from request detail fields.
 * This matches the backend classifyOpsErrorOwner logic.
 */
export function computeFaultOwner(row: OpsRequestDetail): FaultOwner {
  if (!row.status_code || row.status_code < 400) return 'ok'
  if (row.status_code === 499) return 'client' // client disconnected
  if (row.phase === 'upstream') return 'upstream'
  if (row.phase === 'auth' || row.phase === 'request') return 'client'
  if (row.phase === 'routing' || row.phase === 'internal') return 'platform'
  // Fallback: use error_owner from backend if available
  return 'platform'
}
```

- [ ] **Step 3: Build frontend and verify no type errors**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npm run type-check
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/admin/ops.ts
git commit -m "Feature: 前端 OpsSpan 类型定义和 FaultOwner 计算辅助函数"
```

---

## Task 7: Frontend — Waterfall Panel Component

**Files:**
- Create: `frontend/src/views/admin/ops/components/OpsWaterfallPanel.vue`

- [ ] **Step 1: Write the component**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import type { OpsSpan, OpsRequestDetail } from '@/api/admin/ops'

const props = defineProps<{
  row: OpsRequestDetail
}>()

interface WaterfallBar {
  label: string
  durationMs: number
  offsetMs: number
  color: string
  status: string
}

const PHASE_COLORS: Record<string, string> = {
  'token.fetch':      'bg-yellow-400',
  'translate.req':    'bg-blue-300',
  'upstream.post':    'bg-indigo-500',
  'failover.select':  'bg-orange-400',
  'routing.select':   'bg-emerald-400',
  'auth.verify':      'bg-teal-400',
  'translate.resp':   'bg-sky-300',
  'body.truncation':  'bg-red-400',
}

const PHASE_LABELS: Record<string, string> = {
  'token.fetch':      'Token 获取',
  'translate.req':    '请求转译',
  'upstream.post':    '上游请求',
  'failover.select':  'Failover 切换',
  'routing.select':   '路由选择',
  'auth.verify':      'Auth 验证',
  'translate.resp':   '响应转译',
  'body.truncation':  'Body 截断',
}

const bars = computed((): WaterfallBar[] => {
  const spans = props.row.spans
  if (!spans || spans.length === 0) return []

  // Find the earliest start to compute offsets
  const t0 = Math.min(...spans.map(s => s.start_unix_ms))
  const totalMs = props.row.duration_ms ?? 1

  return spans.map((span: OpsSpan) => ({
    label: PHASE_LABELS[span.name] ?? span.name,
    durationMs: span.duration_ms,
    offsetMs: span.start_unix_ms - t0,
    color: span.status === 'error'
      ? 'bg-red-500'
      : (PHASE_COLORS[span.name] ?? 'bg-gray-400'),
    status: span.status ?? 'ok',
  }))
})

const totalMs = computed(() => props.row.duration_ms ?? 1)
</script>

<template>
  <div class="space-y-1 py-2">
    <div
      v-for="(bar, i) in bars"
      :key="i"
      class="flex items-center gap-2 text-xs"
    >
      <!-- Label -->
      <div class="w-28 flex-shrink-0 truncate text-right text-[11px] text-gray-500 dark:text-gray-400">
        {{ bar.label }}
      </div>
      <!-- Track -->
      <div class="relative flex-1 h-4 rounded bg-gray-100 dark:bg-dark-800 overflow-hidden">
        <div
          class="absolute top-0 h-4 rounded transition-all"
          :class="bar.color"
          :style="{
            left: `${(bar.offsetMs / totalMs) * 100}%`,
            width: `${Math.max((bar.durationMs / totalMs) * 100, 0.5)}%`,
          }"
        />
      </div>
      <!-- Duration badge -->
      <div class="w-16 flex-shrink-0 text-right font-mono text-[11px] text-gray-600 dark:text-gray-300">
        {{ bar.durationMs }}ms
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="bars.length === 0"
      class="py-4 text-center text-xs text-gray-400"
    >
      暂无 Span 数据（此请求发生在埋点上线之前）
    </div>
  </div>
</template>
```

- [ ] **Step 2: Build frontend and check for errors**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npm run type-check
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/ops/components/OpsWaterfallPanel.vue
git commit -m "Feature: 新增 OpsWaterfallPanel 瀑布图组件"
```

---

## Task 8: Frontend — Span Tree Component

**Files:**
- Create: `frontend/src/views/admin/ops/components/OpsSpanTree.vue`

- [ ] **Step 1: Write the component**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import type { OpsSpan } from '@/api/admin/ops'

const props = defineProps<{
  spans: OpsSpan[]
}>()

const expanded = ref(true)

function statusClass(status?: string): string {
  switch (status) {
    case 'error':   return 'text-red-500'
    case 'skipped': return 'text-gray-400'
    default:        return 'text-emerald-500'
  }
}

function statusIcon(status?: string): string {
  switch (status) {
    case 'error':   return '✕'
    case 'skipped': return '—'
    default:        return '✓'
  }
}
</script>

<template>
  <div class="font-mono text-xs">
    <!-- Header toggle -->
    <button
      class="mb-1 flex items-center gap-1 text-[11px] font-bold uppercase tracking-wider text-gray-400 hover:text-gray-600"
      @click="expanded = !expanded"
    >
      <span>{{ expanded ? '▾' : '▸' }}</span>
      <span>Span 追踪 ({{ spans.length }})</span>
    </button>

    <div v-if="expanded" class="space-y-0.5 pl-3">
      <div
        v-for="(span, i) in spans"
        :key="i"
        class="flex items-start gap-2 py-0.5"
      >
        <!-- Status icon -->
        <span :class="statusClass(span.status)" class="w-3 flex-shrink-0 text-center">
          {{ statusIcon(span.status) }}
        </span>
        <!-- Span name -->
        <span class="w-32 flex-shrink-0 text-gray-700 dark:text-gray-300">
          {{ span.name }}
        </span>
        <!-- Duration -->
        <span class="w-16 flex-shrink-0 text-right text-gray-500">
          {{ span.duration_ms }}ms
        </span>
        <!-- Attrs -->
        <span
          v-if="span.attrs && Object.keys(span.attrs).length > 0"
          class="flex flex-wrap gap-x-2 text-[10px] text-gray-400"
        >
          <template v-for="(val, key) in span.attrs" :key="key">
            <span>{{ key }}=<span class="text-gray-600 dark:text-gray-300">{{ val }}</span></span>
          </template>
        </span>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Build and verify**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npm run type-check
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/ops/components/OpsSpanTree.vue
git commit -m "Feature: 新增可折叠 OpsSpanTree span 树形组件"
```

---

## Task 9: Frontend — Fault Owner Tag Component and Request List Integration

**Files:**
- Create: `frontend/src/views/admin/ops/components/FaultOwnerTag.vue`
- Modify: `frontend/src/views/admin/ops/OpsRequestInspectView.vue`

- [ ] **Step 1: Create FaultOwnerTag component**

```vue
<script setup lang="ts">
import type { FaultOwner } from '@/api/admin/ops'

const props = defineProps<{
  owner: FaultOwner | null | undefined
}>()

const CONFIG: Record<string, { label: string; classes: string }> = {
  client:   { label: '客户端',   classes: 'bg-amber-100   text-amber-700   dark:bg-amber-900/30  dark:text-amber-400'  },
  upstream: { label: '上游',     classes: 'bg-red-100     text-red-700     dark:bg-red-900/30    dark:text-red-400'    },
  platform: { label: 'sub2api',  classes: 'bg-purple-100  text-purple-700  dark:bg-purple-900/30 dark:text-purple-400' },
  ok:       { label: '正常',     classes: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'},
}

const config = () => CONFIG[props.owner ?? 'ok'] ?? CONFIG['ok']
</script>

<template>
  <span
    class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold"
    :class="config().classes"
  >
    {{ config().label }}
  </span>
</template>
```

- [ ] **Step 2: Add FaultOwnerTag to request list table in OpsRequestInspectView.vue**

Read `OpsRequestInspectView.vue` first, then find the table column definitions. Import the component and `computeFaultOwner`:

```typescript
import FaultOwnerTag from './components/FaultOwnerTag.vue'
import { computeFaultOwner } from '@/api/admin/ops'
```

Add a "责任方" column in the table (after the status_code column):

```html
<!-- In the table header -->
<th class="...">责任方</th>

<!-- In each table row -->
<td class="...">
  <FaultOwnerTag :owner="computeFaultOwner(row)" />
</td>
```

Also add a `fault_owner` filter bar above the table using four chip buttons (全部 / 客户端 / 上游 / sub2api), controlling a local `faultOwnerFilter` ref that filters `rows` client-side:

```typescript
const faultOwnerFilter = ref<FaultOwner | 'all'>('all')

const filteredRows = computed(() => {
  if (faultOwnerFilter.value === 'all') return rows.value
  return rows.value.filter(row => computeFaultOwner(row) === faultOwnerFilter.value)
})
```

- [ ] **Step 3: Build and verify**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npm run type-check
npm run build
```

Expected: no errors, build succeeds

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/admin/ops/components/FaultOwnerTag.vue \
        frontend/src/views/admin/ops/OpsRequestInspectView.vue
git commit -m "Feature: 请求列表新增责任方标签和筛选条"
```

---

## Task 10: Frontend — Waterfall + Span Tree in Detail Panel

**Files:**
- Modify: `frontend/src/views/admin/ops/components/OpsRequestDetailPanel.vue`

- [ ] **Step 1: Import and embed the new components**

Read `OpsRequestDetailPanel.vue` first. Then add imports in `<script setup>`:

```typescript
import OpsWaterfallPanel from './OpsWaterfallPanel.vue'
import OpsSpanTree from './OpsSpanTree.vue'
import type { OpsSpan } from '@/api/admin/ops'

const parsedSpans = computed((): OpsSpan[] => {
  const raw = props.row?.spans
  if (!raw || raw.length === 0) return []
  return raw
})

const hasSpans = computed(() => parsedSpans.value.length > 0)
```

- [ ] **Step 2: Add waterfall section to template**

In the success detail template (the `v-else` branch showing usage data), add after the latency section:

```html
<!-- Waterfall section -->
<div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
  <div class="mb-2 text-xs font-bold uppercase tracking-wider text-gray-400">
    请求瀑布图
  </div>
  <OpsWaterfallPanel :row="row" />
</div>

<!-- Span tree section -->
<div v-if="hasSpans" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
  <OpsSpanTree :spans="parsedSpans" />
</div>
```

For the **error detail** path (kind === 'error'), the `OpsErrorDetailPanel` is used inline. Since error spans are fetched with the error log data (SpansJSON is in OpsRequestDetail), the same waterfall can be shown. Add the waterfall at the bottom of the error detail template:

```html
<!-- After OpsErrorDetailPanel: show waterfall if spans available -->
<div v-if="row.spans && row.spans.length > 0" class="border-t border-gray-100 dark:border-dark-700 px-6 py-4">
  <div class="mb-2 text-xs font-bold uppercase tracking-wider text-gray-400">请求瀑布图</div>
  <OpsWaterfallPanel :row="row" />
  <div class="mt-3">
    <OpsSpanTree :spans="row.spans" />
  </div>
</div>
```

- [ ] **Step 3: Build and verify**

```bash
cd /Users/ziji/personal/github/sub2api/frontend
npm run type-check
npm run build
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/admin/ops/components/OpsRequestDetailPanel.vue
git commit -m "Feature: 在请求详情面板中嵌入瀑布图和 span 树"
```

---

## Task 11: End-to-End Verification

- [ ] **Step 1: Start the full dev stack**

```bash
cd /Users/ziji/personal/github/sub2api
./scripts/dev.sh
```

Wait for both frontend (Vite) and backend (Go) to be ready.

- [ ] **Step 2: Send a test request through the proxy**

```bash
# Use a real API key configured in sub2api
curl -s http://localhost:3000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: <test_api_key>" \
  -d '{"model":"claude-3-5-sonnet-20241022","max_tokens":10,"messages":[{"role":"user","content":"hi"}]}'
```

Expected: 200 response with content

- [ ] **Step 3: Verify spans saved to DB**

```bash
psql $DATABASE_URL -c "
  SELECT request_id, jsonb_array_length(spans) as span_count, spans
  FROM usage_logs
  WHERE spans IS NOT NULL
  ORDER BY created_at DESC
  LIMIT 1;"
```

Expected: `span_count >= 3` (at minimum: routing.select, token.fetch, upstream.post)

- [ ] **Step 4: Verify spans appear in API response**

```bash
curl -s "http://localhost:3000/api/v1/admin/ops/requests?time_range=5m&kind=all" \
  -H "Authorization: Bearer <admin_token>" \
  | jq '.data[] | select(.spans != null) | {request_id, span_count: (.spans | length)}'
```

Expected: entries with `span_count > 0`

- [ ] **Step 5: Open admin UI and verify waterfall renders**

Navigate to `http://localhost:5173/admin/ops/requests`. Verify:
1. Fault owner tags (正常 / 客户端 / 上游 / sub2api) appear in the request list
2. Fault owner filter chips work (click "上游" to show only upstream errors)
3. Clicking a row shows the waterfall with colored phase bars
4. The span tree is collapsible and shows `account_id`, `model` attrs

- [ ] **Step 6: Commit final verification note**

```bash
git tag v0.spans-tracing-e2e-verified
git commit --allow-empty -m "Optimize: 请求诊断追踪系统 E2E 验证通过"
```

---

## Spec Coverage Check

| Requirement | Task |
|-------------|------|
| Spans JSONB in DB | Task 1 |
| OpsSpan struct + context helpers | Task 2 |
| Span data flows to DB in both tables | Task 3 |
| auth.verify span | Tasks 3 + 4 |
| routing.select span | Task 4 |
| token.fetch span | Task 4 |
| translate.req span | Task 4 |
| upstream.post span (per attempt) | Task 4 |
| failover.select span | Task 4 |
| Spans exposed in request list API | Task 5 |
| Frontend OpsSpan types + computeFaultOwner | Task 6 |
| Waterfall chart component | Task 7 |
| Span tree component (collapsible) | Task 8 |
| Fault owner tag + filter bar | Task 9 |
| Waterfall + span tree in detail panel | Task 10 |
| E2E verification | Task 11 |
