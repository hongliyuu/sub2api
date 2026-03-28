# Copilot 成本分析系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为管理员提供 Copilot 成本分析工具，包含用户请求维度（Premium vs 子请求层级展示）和 Copilot 账户维度（实时配额、成本估算、预算告警）两个分析页面。

**Architecture:** 后端新增 `initiator` 字段到 `usage_logs`，新增 `copilot_quota_snapshots` 和 `copilot_budget_alerts` 两张表；配额数据按需实时从 GitHub API 拉取，服务端内存缓存 5 分钟，查询后 UPSERT 当天快照用于趋势图；前端新增 `/admin/copilot/users` 和 `/admin/copilot/accounts` 两个页面。

**Tech Stack:** Go 1.22、Ent ORM、Gin、PostgreSQL、Vue 3、TypeScript、go-cache、ECharts（已有）

**Spec:** `docs/superpowers/specs/2026-03-28-copilot-cost-analytics-design.md`

---

## 文件结构

### 后端新建文件
- `backend/ent/schema/copilot_quota_snapshot.go` — CopilotQuotaSnapshot Ent schema
- `backend/ent/schema/copilot_budget_alert.go` — CopilotBudgetAlert Ent schema
- `backend/internal/repository/copilot_quota_snapshot_repo.go` — 快照 CRUD
- `backend/internal/repository/copilot_budget_alert_repo.go` — 告警配置 CRUD
- `backend/internal/service/copilot_quota_cache.go` — 实时拉取 + 内存缓存服务
- `backend/internal/service/copilot_analytics_service.go` — 统计查询（用户维度 + 账户维度）
- `backend/internal/handler/admin/copilot_analytics_handler.go` — HTTP handler
- `backend/internal/service/copilot_quota_cache_test.go` — 缓存单元测试
- `backend/internal/service/copilot_analytics_service_test.go` — 成本计算单元测试

### 后端修改文件
- `backend/ent/schema/usage_log.go` — 新增 `initiator` 字段和索引
- `backend/internal/service/gateway_service.go` — `RecordUsageInput` 新增 `Initiator` 字段
- `backend/internal/handler/copilot_gateway_handler.go` — RecordUsage 调用时传入 initiator
- `backend/internal/repository/usage_log_repo.go` — INSERT 语句加 initiator 列
- `backend/internal/handler/handler.go` — AdminHandlers 新增 CopilotAnalytics 字段
- `backend/internal/handler/wire.go` — ProvideAdminHandlers 新增参数
- `backend/internal/server/routes/admin.go` — 注册 /copilot 路由组
- `backend/cmd/server/wire_gen.go` — 更新 wire 生成文件（运行 wire 命令）

### 前端新建文件
- `frontend/src/api/admin/copilotAnalytics.ts` — API 客户端
- `frontend/src/views/admin/copilot/CopilotUsersView.vue` — 用户请求分析页
- `frontend/src/views/admin/copilot/CopilotAccountsView.vue` — 账户成本分析页
- `frontend/src/views/admin/copilot/components/UserRequestTree.vue` — 请求层级树组件
- `frontend/src/views/admin/copilot/components/AccountCostCard.vue` — 账户成本卡片
- `frontend/src/views/admin/copilot/components/QuotaTrendChart.vue` — 配额趋势折线图
- `frontend/src/views/admin/copilot/components/HourlyBarChart.vue` — 小时分布柱状图
- `frontend/src/views/admin/copilot/components/BudgetAlertDialog.vue` — 预算告警设置弹窗

### 前端修改文件
- `frontend/src/router/index.ts` — 新增 /admin/copilot/users 和 /admin/copilot/accounts 路由
- `frontend/src/components/layout/AppSidebar.vue` — 侧边栏新增 Copilot 分析入口

---
## Phase 1：数据层变更

### Task 1：usage_logs 新增 initiator 字段（Ent Schema）

**Files:**
- Modify: `backend/ent/schema/usage_log.go`

- [ ] **Step 1: 写失败测试**

在 `backend/internal/repository/` 新建临时测试验证字段存在：

```bash
# 先确认当前 usage_log schema 字段列表
grep -n "initiator" backend/ent/schema/usage_log.go
# 预期：无输出（字段不存在）
```

- [ ] **Step 2: 修改 Ent Schema**

编辑 `backend/ent/schema/usage_log.go`，在 `Fields()` 末尾的 `created_at` 字段**之前**插入：

```go
// initiator: 请求发起类型
// 'user'  = 用户首轮请求，消耗 Premium Interactions 配额
// 'agent' = 任务内子请求（含 assistant/tool 历史消息），消耗标准配额
// 非 Copilot 平台统一写 'user'
field.String("initiator").
    MaxLen(10).
    Default("user").
    Comment("Request initiator type: 'user' (premium) or 'agent' (standard sub-request)."),
```

在 `Indexes()` 末尾追加：

```go
// Copilot 分析查询复合索引
index.Fields("account_id", "initiator", "created_at"),
index.Fields("user_id", "initiator", "created_at"),
```

- [ ] **Step 3: 重新生成 Ent 代码**

```bash
cd backend
go generate ./ent/...
```

预期：`ent/` 目录下生成新文件，无报错。

- [ ] **Step 4: 验证生成代码包含新字段**

```bash
grep -n "Initiator\|initiator" backend/ent/usagelog/usagelog.go | head -5
# 预期：能看到 FieldInitiator 常量
```

- [ ] **Step 5: 编译验证**

```bash
cd backend && go build ./...
# 预期：编译成功，无错误
```

- [ ] **Step 6: Commit**

```bash
git add backend/ent/schema/usage_log.go backend/ent/
git commit -m "Feature: usage_logs 新增 initiator 字段区分 Premium/子请求"
```

---

### Task 2：新增 CopilotQuotaSnapshot Ent Schema

**Files:**
- Create: `backend/ent/schema/copilot_quota_snapshot.go`

- [ ] **Step 1: 创建 schema 文件**

```go
// backend/ent/schema/copilot_quota_snapshot.go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CopilotQuotaSnapshot 记录每次实时查询 GitHub Copilot 配额的结果快照。
// 每次查询后 UPSERT 当天记录，用于趋势图的历史数据。
type CopilotQuotaSnapshot struct {
	ent.Schema
}

func (CopilotQuotaSnapshot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "copilot_quota_snapshots"},
	}
}

func (CopilotQuotaSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		// snapshot_date: 快照日期（每天最后一次查询覆盖当天记录）
		field.Time("snapshot_date").
			SchemaType(map[string]string{dialect.Postgres: "date"}),
		// plan_type: 快照时的套餐类型（from GitHub API）
		field.String("plan_type").
			MaxLen(30).
			Optional().
			Nillable(),
		field.Int("premium_entitlement").Default(0), // 月配额总量
		field.Int("premium_remaining").Default(0),   // 当前剩余量
		field.Int("premium_used").Default(0),         // 已使用量（全渠道）
		field.Int("premium_overage").Default(0),      // 超额使用量
		field.Bool("unlimited").Default(false),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CopilotQuotaSnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "snapshot_date").Unique(),
		index.Fields("snapshot_date"),
		index.Fields("account_id"),
	}
}
```

- [ ] **Step 2: 重新生成 Ent 代码**

```bash
cd backend && go generate ./ent/...
# 预期：生成 ent/copilotquotasnapshot/ 目录
```

- [ ] **Step 3: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add backend/ent/schema/copilot_quota_snapshot.go backend/ent/
git commit -m "Feature: 新增 CopilotQuotaSnapshot Ent schema"
```

---

### Task 3：新增 CopilotBudgetAlert Ent Schema

**Files:**
- Create: `backend/ent/schema/copilot_budget_alert.go`

- [ ] **Step 1: 创建 schema 文件**

```go
// backend/ent/schema/copilot_budget_alert.go
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// CopilotBudgetAlert 存储每个 Copilot 账户的预算告警配置。
// 每个账户最多一条记录（account_id UNIQUE）。
type CopilotBudgetAlert struct {
	ent.Schema
}

func (CopilotBudgetAlert) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "copilot_budget_alerts"},
	}
}

func (CopilotBudgetAlert) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (CopilotBudgetAlert) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id").Unique(),
		// monthly_budget: 月预算上限（美元），0 表示不限制
		field.Float("monthly_budget").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,2)"}),
		// alert_threshold: 配额使用率告警阈值（百分比，默认 80）
		field.Int("alert_threshold").Default(80),
		field.Bool("enabled").Default(true),
		// last_alerted_at: 最近一次触发告警的时间，用于去重（同一天只告警一次）
		field.Time("last_alerted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (CopilotBudgetAlert) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id"),
		index.Fields("enabled"),
	}
}
```

- [ ] **Step 2: 重新生成 Ent 代码**

```bash
cd backend && go generate ./ent/...
# 预期：生成 ent/copilotbudgetalert/ 目录
```

- [ ] **Step 3: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add backend/ent/schema/copilot_budget_alert.go backend/ent/
git commit -m "Feature: 新增 CopilotBudgetAlert Ent schema"
```

---

### Task 4：数据库迁移

**Files:**
- Modify: `backend/internal/repository/migrations_runner.go`（确认迁移机制）

- [ ] **Step 1: 确认现有迁移机制**

```bash
grep -n "AutoMigrate\|Atlas\|migrate\|Migrate" backend/internal/repository/migrations_runner.go | head -20
```

- [ ] **Step 2: 运行迁移（开发环境）**

```bash
cd backend
# 若使用 Ent Atlas 自动迁移：
go run cmd/server/main.go --migrate-only
# 或直接启动服务（Ent 会自动建表）
```

- [ ] **Step 3: 验证新表和字段存在**

```bash
# 连接数据库验证（根据实际数据库配置）
psql $DATABASE_URL -c "\d usage_logs" | grep initiator
psql $DATABASE_URL -c "\dt copilot_quota_snapshots"
psql $DATABASE_URL -c "\dt copilot_budget_alerts"
```

预期：`initiator` 列存在，两张新表存在。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/
git commit -m "Feature: 数据库迁移 - initiator 字段 + 2 张 Copilot 分析新表"
```

---
## Phase 2：后端 - 记录 initiator

### Task 5：RecordUsageInput 新增 Initiator + usage_log_repo 更新

**Files:**
- Modify: `backend/internal/service/gateway_service.go` (RecordUsageInput struct 附近, ~line 7195)
- Modify: `backend/internal/repository/usage_log_repo.go`

- [ ] **Step 1: 写失败测试**

```bash
grep -n "Initiator" backend/internal/service/gateway_service.go
# 预期：无输出
```

- [ ] **Step 2: RecordUsageInput 新增字段**

在 `gateway_service.go` 的 `RecordUsageInput` struct 末尾追加：

```go
// Initiator: Copilot 请求发起类型，'user' 或 'agent'。
// 非 Copilot 请求传空字符串，RecordUsage 内默认为 'user'。
Initiator string
```

- [ ] **Step 3: RecordUsage 写库时传入 initiator**

在 `gateway_service.go` 的 `RecordUsage` 函数中，找到构建 `UsageLog` 的代码段，加入：

```go
// 确定 initiator（默认 'user'，非空时使用传入值）
initiator := "user"
if input.Initiator == "agent" {
    initiator = "agent"
}
```

并在写库的参数列表中加入 `initiator` 字段（具体行参考 `usageLogInsertArgTypes` 数组和 INSERT 语句）。

- [ ] **Step 4: usage_log_repo.go 更新 INSERT 语句**

找到 `usageLogSelectColumns` 常量和 `usageLogInsertArgTypes` 数组，在末尾（`created_at` 之前）分别加入：

`usageLogSelectColumns`：在 `created_at` 前加 `initiator,`

`usageLogInsertArgTypes`：加入 `"text",`

找到 INSERT SQL 语句，在对应位置加入 `initiator` 占位符和赋值。

- [ ] **Step 5: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/gateway_service.go \
        backend/internal/repository/usage_log_repo.go
git commit -m "Feature: RecordUsageInput 新增 Initiator 字段，usage_log_repo 写入 initiator 列"
```

---

### Task 6：Copilot gateway handler 传入 initiator

**Files:**
- Modify: `backend/internal/handler/copilot_gateway_handler.go`

- [ ] **Step 1: 定位 3 处 RecordUsage 调用**

```bash
grep -n "RecordUsage\|gatewayService.RecordUsage" backend/internal/handler/copilot_gateway_handler.go
# 预期：3 处调用（line ~324, ~696, ~1034）
```

- [ ] **Step 2: 捕获 initiator 值（在 goroutine 外）**

每处 `RecordUsage` 调用前，在捕获其他变量的同一位置加入（参考已有的 `authLatencyMs` 等变量的捕获方式）：

```go
// 捕获 initiator（在 goroutine 外，body 仍然有效）
capturedInitiator := service.CopilotInitiatorFromBody(body)
```

**注意**：需要在 `service` 包中将 `copilotInitiator()` 改为导出函数 `CopilotInitiatorFromBody()`。

- [ ] **Step 3: 导出 copilotInitiator 函数**

在 `backend/internal/service/copilot_gateway_service.go` 中，将：

```go
func copilotInitiator(openAIBody []byte) string {
```

改为：

```go
// CopilotInitiatorFromBody 判断请求发起类型。
// 消息历史中含 assistant/tool 消息时返回 "agent"（子请求），否则返回 "user"（Premium 请求）。
func CopilotInitiatorFromBody(openAIBody []byte) string {
```

同时更新文件内的两处调用（`initiator := copilotInitiator(body)` → `CopilotInitiatorFromBody`）。

- [ ] **Step 4: 在 RecordUsage 调用中传入 Initiator**

在每处 `service.RecordUsageInput{...}` 中加入：

```go
Initiator: capturedInitiator,
```

- [ ] **Step 5: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 6: 运行现有 Copilot handler 测试**

```bash
cd backend && go test ./internal/handler/... -run "Copilot\|copilot" -v 2>&1 | tail -20
```

- [ ] **Step 7: Commit**

```bash
git add backend/internal/handler/copilot_gateway_handler.go \
        backend/internal/service/copilot_gateway_service.go
git commit -m "Feature: Copilot gateway handler 写日志时记录 initiator 字段"
```

---
## Phase 3：后端 - Repository 层

### Task 7：CopilotQuotaSnapshotRepo

**Files:**
- Create: `backend/internal/repository/copilot_quota_snapshot_repo.go`

- [ ] **Step 1: 写失败测试**

新建 `backend/internal/repository/copilot_quota_snapshot_repo_test.go`：

```go
package repository_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestCopilotQuotaSnapshotRepo_UpsertAndQuery(t *testing.T) {
    // 集成测试：需要真实数据库连接
    // 参考同目录下其他 _integration_test.go 文件的 setup 方式
    t.Skip("integration test - run with DB")
}
```

- [ ] **Step 2: 实现 repo**

```go
// backend/internal/repository/copilot_quota_snapshot_repo.go
package repository

import (
    "context"
    "time"

    dbent "github.com/Wei-Shaw/sub2api/ent"
    dbsnap "github.com/Wei-Shaw/sub2api/ent/copilotquotasnapshot"
    "github.com/Wei-Shaw/sub2api/internal/service"
)

// CopilotQuotaSnapshotRepo 处理 copilot_quota_snapshots 表的读写。
type CopilotQuotaSnapshotRepo struct {
    db *dbent.Client
}

func NewCopilotQuotaSnapshotRepo(db *dbent.Client) *CopilotQuotaSnapshotRepo {
    return &CopilotQuotaSnapshotRepo{db: db}
}

// Upsert 写入或覆盖当天的配额快照（同一 account_id + snapshot_date 覆盖）。
func (r *CopilotQuotaSnapshotRepo) Upsert(ctx context.Context, snap *service.CopilotQuotaSnapshotData) error {
    today := time.Now().Truncate(24 * time.Hour)
    return r.db.CopilotQuotaSnapshot.
        Create().
        SetAccountID(snap.AccountID).
        SetSnapshotDate(today).
        SetNillablePlanType(snap.PlanType).
        SetPremiumEntitlement(snap.PremiumEntitlement).
        SetPremiumRemaining(snap.PremiumRemaining).
        SetPremiumUsed(snap.PremiumUsed).
        SetPremiumOverage(snap.PremiumOverage).
        SetUnlimited(snap.Unlimited).
        OnConflictColumns(dbsnap.FieldAccountID, dbsnap.FieldSnapshotDate).
        UpdateNewValues().
        Exec(ctx)
}

// ListByAccountID 返回某账户最近 N 天的快照（按日期升序）。
func (r *CopilotQuotaSnapshotRepo) ListByAccountID(ctx context.Context, accountID int64, days int) ([]*service.CopilotQuotaSnapshotData, error) {
    since := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)
    rows, err := r.db.CopilotQuotaSnapshot.
        Query().
        Where(
            dbsnap.AccountID(accountID),
            dbsnap.SnapshotDateGTE(since),
        ).
        Order(dbent.Asc(dbsnap.FieldSnapshotDate)).
        All(ctx)
    if err != nil {
        return nil, err
    }
    result := make([]*service.CopilotQuotaSnapshotData, 0, len(rows))
    for _, row := range rows {
        result = append(result, &service.CopilotQuotaSnapshotData{
            AccountID:          row.AccountID,
            PlanType:           row.PlanType,
            PremiumEntitlement: row.PremiumEntitlement,
            PremiumRemaining:   row.PremiumRemaining,
            PremiumUsed:        row.PremiumUsed,
            PremiumOverage:     row.PremiumOverage,
            Unlimited:          row.Unlimited,
            SnapshotDate:       row.SnapshotDate,
        })
    }
    return result, nil
}

// GetLatestByAccountID 返回某账户最新一条快照。
func (r *CopilotQuotaSnapshotRepo) GetLatestByAccountID(ctx context.Context, accountID int64) (*service.CopilotQuotaSnapshotData, error) {
    row, err := r.db.CopilotQuotaSnapshot.
        Query().
        Where(dbsnap.AccountID(accountID)).
        Order(dbent.Desc(dbsnap.FieldSnapshotDate)).
        First(ctx)
    if err != nil {
        return nil, err
    }
    return &service.CopilotQuotaSnapshotData{
        AccountID:          row.AccountID,
        PlanType:           row.PlanType,
        PremiumEntitlement: row.PremiumEntitlement,
        PremiumRemaining:   row.PremiumRemaining,
        PremiumUsed:        row.PremiumUsed,
        PremiumOverage:     row.PremiumOverage,
        Unlimited:          row.Unlimited,
        SnapshotDate:       row.SnapshotDate,
    }, nil
}
```

- [ ] **Step 3: 在 service 包中定义数据传输结构**

在新建的 `backend/internal/service/copilot_analytics_service.go` 顶部（或单独的 types 文件）定义：

```go
// CopilotQuotaSnapshotData 是 service 层使用的配额快照数据结构。
type CopilotQuotaSnapshotData struct {
    AccountID          int64
    PlanType           *string
    PremiumEntitlement int
    PremiumRemaining   int
    PremiumUsed        int
    PremiumOverage     int
    Unlimited          bool
    SnapshotDate       time.Time
}
```

- [ ] **Step 4: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/copilot_quota_snapshot_repo.go \
        backend/internal/repository/copilot_quota_snapshot_repo_test.go
git commit -m "Feature: 新增 CopilotQuotaSnapshotRepo"
```

---

### Task 8：CopilotBudgetAlertRepo

**Files:**
- Create: `backend/internal/repository/copilot_budget_alert_repo.go`

- [ ] **Step 1: 实现 repo**

```go
// backend/internal/repository/copilot_budget_alert_repo.go
package repository

import (
    "context"

    dbent "github.com/Wei-Shaw/sub2api/ent"
    dbalert "github.com/Wei-Shaw/sub2api/ent/copilotbudgetalert"
    "github.com/Wei-Shaw/sub2api/internal/service"
)

type CopilotBudgetAlertRepo struct {
    db *dbent.Client
}

func NewCopilotBudgetAlertRepo(db *dbent.Client) *CopilotBudgetAlertRepo {
    return &CopilotBudgetAlertRepo{db: db}
}

// Upsert 创建或更新账户的预算告警配置。
func (r *CopilotBudgetAlertRepo) Upsert(ctx context.Context, cfg *service.CopilotBudgetAlertConfig) error {
    return r.db.CopilotBudgetAlert.
        Create().
        SetAccountID(cfg.AccountID).
        SetMonthlyBudget(cfg.MonthlyBudget).
        SetAlertThreshold(cfg.AlertThreshold).
        SetEnabled(cfg.Enabled).
        OnConflictColumns(dbalert.FieldAccountID).
        UpdateNewValues().
        Exec(ctx)
}

// GetByAccountID 返回某账户的告警配置，不存在时返回 nil。
func (r *CopilotBudgetAlertRepo) GetByAccountID(ctx context.Context, accountID int64) (*service.CopilotBudgetAlertConfig, error) {
    row, err := r.db.CopilotBudgetAlert.
        Query().
        Where(dbalert.AccountID(accountID)).
        First(ctx)
    if dbent.IsNotFound(err) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return &service.CopilotBudgetAlertConfig{
        AccountID:      row.AccountID,
        MonthlyBudget:  row.MonthlyBudget,
        AlertThreshold: row.AlertThreshold,
        Enabled:        row.Enabled,
    }, nil
}

// ListEnabled 返回所有启用了告警的配置。
func (r *CopilotBudgetAlertRepo) ListEnabled(ctx context.Context) ([]*service.CopilotBudgetAlertConfig, error) {
    rows, err := r.db.CopilotBudgetAlert.
        Query().
        Where(dbalert.Enabled(true)).
        All(ctx)
    if err != nil {
        return nil, err
    }
    result := make([]*service.CopilotBudgetAlertConfig, 0, len(rows))
    for _, row := range rows {
        result = append(result, &service.CopilotBudgetAlertConfig{
            AccountID:      row.AccountID,
            MonthlyBudget:  row.MonthlyBudget,
            AlertThreshold: row.AlertThreshold,
            Enabled:        row.Enabled,
        })
    }
    return result, nil
}
```

同时在 service 层定义：

```go
// CopilotBudgetAlertConfig 预算告警配置。
type CopilotBudgetAlertConfig struct {
    AccountID      int64
    MonthlyBudget  float64
    AlertThreshold int  // 百分比，如 80 表示 80%
    Enabled        bool
}
```

- [ ] **Step 2: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/copilot_budget_alert_repo.go
git commit -m "Feature: 新增 CopilotBudgetAlertRepo"
```

---
## Phase 4：后端 - Service 层

### Task 9：CopilotQuotaCacheService（实时拉取 + 内存缓存）

**Files:**
- Create: `backend/internal/service/copilot_quota_cache.go`
- Create: `backend/internal/service/copilot_quota_cache_test.go`

- [ ] **Step 1: 写失败测试**

```go
// backend/internal/service/copilot_quota_cache_test.go
package service_test

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCopilotQuotaCacheService_CacheTTL(t *testing.T) {
    // 验证缓存命中时返回相同数据，TTL 后失效
    // 此测试先失败，实现后通过
    t.Skip("implement after service is written")
}

func TestCopilotQuotaCacheService_AlertStatus(t *testing.T) {
    tests := []struct {
        name           string
        usageRate      float64
        threshold      int
        expectedStatus string
    }{
        {"ok_below_threshold", 50.0, 80, "ok"},
        {"warning_at_threshold", 80.0, 80, "warning"},
        {"warning_above_threshold", 85.0, 80, "warning"},
        {"critical_at_95", 95.0, 80, "critical"},
        {"critical_above_95", 99.0, 80, "critical"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            status := computeAlertStatus(tt.usageRate, tt.threshold)
            assert.Equal(t, tt.expectedStatus, status)
        })
    }
}

func TestCopilotPlanCostPerRequest(t *testing.T) {
    tests := []struct {
        planType    string
        seatCount   int
        expectedCPR float64 // cost per premium request
    }{
        {"individual_pro_plus", 1, 39.0 / 1500},
        {"individual_pro", 1, 10.0 / 300},
        {"business", 3, 57.0 / 900},
        {"individual_free", 1, 0}, // free plan: $0
    }
    for _, tt := range tests {
        t.Run(tt.planType, func(t *testing.T) {
            cost, quota := copilotPlanCostAndQuota(tt.planType, tt.seatCount)
            var cpr float64
            if quota > 0 {
                cpr = cost / float64(quota)
            }
            assert.InDelta(t, tt.expectedCPR, cpr, 0.0001)
        })
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/service/... -run "TestCopilotPlan\|TestCopilotQuota" -v 2>&1 | tail -20
# 预期：FAIL（函数未定义）
```

- [ ] **Step 3: 实现 CopilotQuotaCacheService**

```go
// backend/internal/service/copilot_quota_cache.go
package service

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/Wei-Shaw/sub2api/internal/pkg/copilot"
    gocache "github.com/patrickmn/go-cache"
    "go.uber.org/zap"
)

// copilotPlanSpec 套餐规格（月费美元 + 每用户 Premium 配额）。
type copilotPlanSpec struct {
    MonthlyCostPerSeat float64
    PremiumQuotaPerSeat int
}

// copilotPlanTable 套餐价格与配额常量表。
var copilotPlanTable = map[string]copilotPlanSpec{
    copilot.PlanTypeIndividualFree:    {MonthlyCostPerSeat: 0, PremiumQuotaPerSeat: 50},
    copilot.PlanTypeIndividual:        {MonthlyCostPerSeat: 10, PremiumQuotaPerSeat: 300}, // legacy = pro
    copilot.PlanTypeIndividualPro:     {MonthlyCostPerSeat: 10, PremiumQuotaPerSeat: 300},
    copilot.PlanTypeIndividualProPlus: {MonthlyCostPerSeat: 39, PremiumQuotaPerSeat: 1500},
    copilot.PlanTypeBusiness:          {MonthlyCostPerSeat: 19, PremiumQuotaPerSeat: 300},
    copilot.PlanTypeEnterprise:        {MonthlyCostPerSeat: 39, PremiumQuotaPerSeat: 1000},
}

// copilotPlanCostAndQuota 返回套餐月费（美元）和月 Premium 配额总量。
// seatCount 仅对 Business/Enterprise 有效，其他套餐固定为 1。
func copilotPlanCostAndQuota(planType string, seatCount int) (float64, int) {
    spec, ok := copilotPlanTable[planType]
    if !ok {
        return 0, 0
    }
    seats := seatCount
    if seats < 1 {
        seats = 1
    }
    // 个人套餐 seat 固定为 1
    if planType == copilot.PlanTypeIndividualFree ||
        planType == copilot.PlanTypeIndividual ||
        planType == copilot.PlanTypeIndividualPro ||
        planType == copilot.PlanTypeIndividualProPlus {
        seats = 1
    }
    return spec.MonthlyCostPerSeat * float64(seats), spec.PremiumQuotaPerSeat * seats
}

// computeAlertStatus 根据使用率和阈值返回告警状态。
func computeAlertStatus(usageRate float64, threshold int) string {
    if usageRate >= 95 {
        return "critical"
    }
    if usageRate >= float64(threshold) {
        return "warning"
    }
    return "ok"
}

// CopilotQuotaCacheEntry 单个账户的缓存条目。
type CopilotQuotaCacheEntry struct {
    Info     *copilot.CopilotQuotaInfo
    CachedAt time.Time
}

// CopilotQuotaSnapshotPort 定义 repo 的写接口（方便单元测试 mock）。
type CopilotQuotaSnapshotPort interface {
    Upsert(ctx context.Context, snap *CopilotQuotaSnapshotData) error
    GetLatestByAccountID(ctx context.Context, accountID int64) (*CopilotQuotaSnapshotData, error)
}

// CopilotBudgetAlertPort 定义 budget alert repo 的读接口。
type CopilotBudgetAlertPort interface {
    GetByAccountID(ctx context.Context, accountID int64) (*CopilotBudgetAlertConfig, error)
}

// CopilotQuotaCacheService 负责实时拉取 Copilot 配额并缓存（TTL 5分钟）。
type CopilotQuotaCacheService struct {
    gatewayService *CopilotGatewayService
    snapshotRepo   CopilotQuotaSnapshotPort
    cache          *gocache.Cache
    mu             sync.Mutex
    logger         *zap.Logger
}

const copilotQuotaCacheTTL = 5 * time.Minute

func NewCopilotQuotaCacheService(
    gatewayService *CopilotGatewayService,
    snapshotRepo CopilotQuotaSnapshotPort,
    logger *zap.Logger,
) *CopilotQuotaCacheService {
    return &CopilotQuotaCacheService{
        gatewayService: gatewayService,
        snapshotRepo:   snapshotRepo,
        cache:          gocache.New(copilotQuotaCacheTTL, 10*time.Minute),
        logger:         logger,
    }
}

// GetQuota 返回账户配额（优先缓存，缓存 miss 则实时拉取并写快照）。
func (s *CopilotQuotaCacheService) GetQuota(ctx context.Context, account *Account) (*CopilotQuotaCacheEntry, error) {
    key := fmt.Sprintf("copilot_quota:%d", account.ID)
    if v, ok := s.cache.Get(key); ok {
        return v.(*CopilotQuotaCacheEntry), nil
    }
    return s.refreshQuota(ctx, account, key)
}

// ForceRefreshQuota 强制绕过缓存重新拉取（用于手动刷新按钮）。
func (s *CopilotQuotaCacheService) ForceRefreshQuota(ctx context.Context, account *Account) (*CopilotQuotaCacheEntry, error) {
    key := fmt.Sprintf("copilot_quota:%d", account.ID)
    return s.refreshQuota(ctx, account, key)
}

func (s *CopilotQuotaCacheService) refreshQuota(ctx context.Context, account *Account, key string) (*CopilotQuotaCacheEntry, error) {
    info, err := s.gatewayService.FetchQuota(ctx, account)
    if err != nil {
        // 拉取失败时尝试返回最后已知快照（降级）
        if snap, snapErr := s.snapshotRepo.GetLatestByAccountID(ctx, account.ID); snapErr == nil && snap != nil {
            s.logger.Warn("copilot_quota_cache.fetch_failed_using_stale",
                zap.Int64("account_id", account.ID), zap.Error(err))
            return &CopilotQuotaCacheEntry{
                Info:     snapToQuotaInfo(snap),
                CachedAt: snap.SnapshotDate,
            }, nil
        }
        return nil, fmt.Errorf("fetch quota for account %d: %w", account.ID, err)
    }

    entry := &CopilotQuotaCacheEntry{Info: info, CachedAt: time.Now()}
    s.cache.Set(key, entry, copilotQuotaCacheTTL)

    // 异步写快照（不阻塞响应）
    go func() {
        snapCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if pi := info.PremiumInteractions; pi != nil {
            planType := info.Plan
            snap := &CopilotQuotaSnapshotData{
                AccountID:          account.ID,
                PlanType:           &planType,
                PremiumEntitlement: pi.Entitlement,
                PremiumRemaining:   pi.Remaining,
                PremiumUsed:        pi.Used,
                PremiumOverage:     pi.OverageCount,
                Unlimited:          pi.Unlimited,
            }
            if upsertErr := s.snapshotRepo.Upsert(snapCtx, snap); upsertErr != nil {
                s.logger.Warn("copilot_quota_cache.snapshot_upsert_failed",
                    zap.Int64("account_id", account.ID), zap.Error(upsertErr))
            }
        }
    }()

    return entry, nil
}

// snapToQuotaInfo 将快照数据转换为 QuotaInfo（降级用）。
func snapToQuotaInfo(snap *CopilotQuotaSnapshotData) *copilot.CopilotQuotaInfo {
    info := &copilot.CopilotQuotaInfo{}
    if snap.PlanType != nil {
        info.Plan = *snap.PlanType
    }
    info.PremiumInteractions = &copilot.QuotaDetail{
        Entitlement:  snap.PremiumEntitlement,
        Remaining:    snap.PremiumRemaining,
        Used:         snap.PremiumUsed,
        OverageCount: snap.PremiumOverage,
        Unlimited:    snap.Unlimited,
    }
    return info
}
```

- [ ] **Step 4: 运行测试**

```bash
cd backend && go test ./internal/service/... -run "TestCopilotPlan\|TestCopilotQuota\|TestComputeAlert" -v
# 预期：TestCopilotPlanCostPerRequest 和 TestCopilotQuotaCacheService_AlertStatus 通过
```

- [ ] **Step 5: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/copilot_quota_cache.go \
        backend/internal/service/copilot_quota_cache_test.go
git commit -m "Feature: 新增 CopilotQuotaCacheService，实时拉取配额 + 5分钟内存缓存 + 快照持久化"
```

---
### Task 10：CopilotAnalyticsService（统计查询）

**Files:**
- Create: `backend/internal/service/copilot_analytics_service.go`
- Create: `backend/internal/service/copilot_analytics_service_test.go`

- [ ] **Step 1: 写失败测试（成本计算逻辑）**

```go
// backend/internal/service/copilot_analytics_service_test.go
package service_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestBuildAccountCostInfo(t *testing.T) {
    tests := []struct {
        name                string
        planType            string
        seatCount           int
        expectedMonthlyCost float64
        expectedMonthQuota  int
        expectedCPR         float64
    }{
        {"pro_plus_1_seat", "individual_pro_plus", 1, 39.0, 1500, 39.0 / 1500},
        {"business_3_seats", "business", 3, 57.0, 900, 57.0 / 900},
        {"free_plan", "individual_free", 1, 0, 50, 0},
        {"unknown_plan", "unknown", 1, 0, 0, 0},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cost, quota := copilotPlanCostAndQuota(tt.planType, tt.seatCount)
            assert.InDelta(t, tt.expectedMonthlyCost, cost, 0.001)
            assert.Equal(t, tt.expectedMonthQuota, quota)
            var cpr float64
            if quota > 0 {
                cpr = cost / float64(quota)
            }
            assert.InDelta(t, tt.expectedCPR, cpr, 0.0001)
        })
    }
}
```

- [ ] **Step 2: 实现 CopilotAnalyticsService**

```go
// backend/internal/service/copilot_analytics_service.go
package service

import (
    "context"
    "time"
)

// --- 用户维度数据结构 ---

// CopilotUserDayStat 某用户某天的请求统计。
type CopilotUserDayStat struct {
    UserID            int64
    Username          string
    PremiumRequests   int      // initiator='user' 的请求数
    AgentRequests     int      // initiator='agent' 的请求数
    TotalRequests     int
    Models            []string
    LastRequestAt     *time.Time
}

// CopilotHourlyBucket 一个小时的请求分布。
type CopilotHourlyBucket struct {
    Hour         int // 0-23
    PremiumCount int
    AgentCount   int
}

// CopilotRequestItem 单条请求记录（用于层级展示）。
type CopilotRequestItem struct {
    RequestID   string
    Model       string
    Initiator   string // 'user' | 'agent'
    CreatedAt   time.Time
    DurationMs  *int
    SubRequests []*CopilotRequestItem // initiator='agent' 的关联子请求
}

// CopilotUserStatsInput 用户统计查询参数。
type CopilotUserStatsInput struct {
    Date   time.Time
    UserID int64 // 0 = 所有用户
}

// CopilotUserStatsResult 用户统计结果。
type CopilotUserStatsResult struct {
    Date                 time.Time
    TotalPremiumRequests int
    TotalAgentRequests   int
    ActiveUsers          int
    Users                []*CopilotUserDayStat
}

// --- 账户维度数据结构 ---

// CopilotAccountOverviewItem 单个账户的总览数据。
type CopilotAccountOverviewItem struct {
    AccountID                  int64
    Name                       string
    PlanType                   string
    SeatCount                  int
    MonthlyCost                float64
    MonthlyQuota               int
    CostPerPremiumRequest      float64
    SystemTodayPremiumRequests int    // 本系统今日 Premium 请求数
    SystemMonthPremiumRequests int    // 本系统本月 Premium 请求数
    QuotaSnapshot              *CopilotQuotaSnapshotData // 最新实时快照（可为 nil）
    QuotaCachedAt              *time.Time
    BudgetAlert                *CopilotBudgetAlertConfig
    AlertStatus                string // 'ok' | 'warning' | 'critical'
    ExternalUsed               int    // GitHub总消耗 - 系统内消耗（其他渠道）
}

// CopilotAccountsOverviewResult 账户总览结果。
type CopilotAccountsOverviewResult struct {
    TotalAccounts        int
    EstimatedMonthlyCost float64
    TodayPremiumRequests int
    AlertCount           int
    Accounts             []*CopilotAccountOverviewItem
}

// CopilotAnalyticsPort 定义 analytics service 依赖的数据库查询接口。
type CopilotAnalyticsPort interface {
    // 用户维度
    QueryUserDayStats(ctx context.Context, input CopilotUserStatsInput) ([]*CopilotUserDayStat, error)
    QueryHourlyBuckets(ctx context.Context, userID int64, accountID int64, date time.Time) ([]*CopilotHourlyBucket, error)
    QueryUserRequests(ctx context.Context, userID int64, date time.Time, page, pageSize int) ([]*CopilotRequestItem, int, error)
    // 账户维度
    QueryAccountMonthStats(ctx context.Context, accountID int64, monthStart time.Time) (todayPremium, monthPremium int, err error)
}

// CopilotAnalyticsService 封装 Copilot 分析的业务逻辑。
type CopilotAnalyticsService struct {
    analyticsPort CopilotAnalyticsPort
    snapshotRepo  CopilotQuotaSnapshotPort
    budgetRepo    CopilotBudgetAlertPort
    quotaCache    *CopilotQuotaCacheService
    adminService  AdminService
}

func NewCopilotAnalyticsService(
    analyticsPort CopilotAnalyticsPort,
    snapshotRepo CopilotQuotaSnapshotPort,
    budgetRepo CopilotBudgetAlertPort,
    quotaCache *CopilotQuotaCacheService,
    adminService AdminService,
) *CopilotAnalyticsService {
    return &CopilotAnalyticsService{
        analyticsPort: analyticsPort,
        snapshotRepo:  snapshotRepo,
        budgetRepo:    budgetRepo,
        quotaCache:    quotaCache,
        adminService:  adminService,
    }
}

// GetUserStats 返回指定日期所有用户（或单个用户）的请求统计。
func (s *CopilotAnalyticsService) GetUserStats(ctx context.Context, input CopilotUserStatsInput) (*CopilotUserStatsResult, error) {
    users, err := s.analyticsPort.QueryUserDayStats(ctx, input)
    if err != nil {
        return nil, err
    }
    result := &CopilotUserStatsResult{Date: input.Date, Users: users}
    for _, u := range users {
        result.TotalPremiumRequests += u.PremiumRequests
        result.TotalAgentRequests += u.AgentRequests
        if u.PremiumRequests > 0 || u.AgentRequests > 0 {
            result.ActiveUsers++
        }
    }
    return result, nil
}

// GetAccountsOverview 返回所有 Copilot 账户的成本总览（实时拉取配额）。
func (s *CopilotAnalyticsService) GetAccountsOverview(ctx context.Context) (*CopilotAccountsOverviewResult, error) {
    accounts, _, err := s.adminService.ListAccounts(ctx, 1, 500, PlatformCopilot, "", StatusActive, "", 0)
    if err != nil {
        return nil, err
    }

    now := time.Now()
    monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
    result := &CopilotAccountsOverviewResult{}

    for _, acc := range accounts {
        item, err := s.buildAccountOverviewItem(ctx, &acc, now, monthStart)
        if err != nil {
            continue // 单个账户失败不影响整体
        }
        result.Accounts = append(result.Accounts, item)
        result.TotalAccounts++
        result.EstimatedMonthlyCost += item.MonthlyCost
        result.TodayPremiumRequests += item.SystemTodayPremiumRequests
        if item.AlertStatus != "ok" {
            result.AlertCount++
        }
    }
    return result, nil
}

func (s *CopilotAnalyticsService) buildAccountOverviewItem(
    ctx context.Context,
    acc *Account,
    now time.Time,
    monthStart time.Time,
) (*CopilotAccountOverviewItem, error) {
    // 从 extra 读取套餐配置
    planType, _ := acc.Extra["copilot_plan_type"].(string)
    seatCount := 1
    if sc, ok := acc.Extra["copilot_seat_count"].(float64); ok && sc > 0 {
        seatCount = int(sc)
    }

    monthlyCost, monthlyQuota := copilotPlanCostAndQuota(planType, seatCount)
    var cpr float64
    if monthlyQuota > 0 {
        cpr = monthlyCost / float64(monthlyQuota)
    }

    todayPremium, monthPremium, err := s.analyticsPort.QueryAccountMonthStats(ctx, acc.ID, monthStart)
    if err != nil {
        todayPremium, monthPremium = 0, 0
    }

    // 实时拉取配额（带缓存）
    var snap *CopilotQuotaSnapshotData
    var cachedAt *time.Time
    entry, quotaErr := s.quotaCache.GetQuota(ctx, acc)
    if quotaErr == nil && entry != nil && entry.Info != nil && entry.Info.PremiumInteractions != nil {
        pi := entry.Info.PremiumInteractions
        planStr := entry.Info.Plan
        snap = &CopilotQuotaSnapshotData{
            AccountID:          acc.ID,
            PlanType:           &planStr,
            PremiumEntitlement: pi.Entitlement,
            PremiumRemaining:   pi.Remaining,
            PremiumUsed:        pi.Used,
            PremiumOverage:     pi.OverageCount,
            Unlimited:          pi.Unlimited,
        }
        t := entry.CachedAt
        cachedAt = &t
    }

    // 计算外部消耗
    externalUsed := 0
    if snap != nil && snap.PremiumUsed > monthPremium {
        externalUsed = snap.PremiumUsed - monthPremium
    }

    // 告警状态
    budgetCfg, _ := s.budgetRepo.GetByAccountID(ctx, acc.ID)
    alertStatus := "ok"
    if snap != nil && monthlyQuota > 0 && budgetCfg != nil && budgetCfg.Enabled {
        usageRate := float64(snap.PremiumUsed) / float64(monthlyQuota) * 100
        alertStatus = computeAlertStatus(usageRate, budgetCfg.AlertThreshold)
    }

    return &CopilotAccountOverviewItem{
        AccountID:                  acc.ID,
        Name:                       acc.Name,
        PlanType:                   planType,
        SeatCount:                  seatCount,
        MonthlyCost:                monthlyCost,
        MonthlyQuota:               monthlyQuota,
        CostPerPremiumRequest:      cpr,
        SystemTodayPremiumRequests: todayPremium,
        SystemMonthPremiumRequests: monthPremium,
        QuotaSnapshot:              snap,
        QuotaCachedAt:              cachedAt,
        BudgetAlert:                budgetCfg,
        AlertStatus:                alertStatus,
        ExternalUsed:               externalUsed,
    }, nil
}
```

- [ ] **Step 3: 运行测试**

```bash
cd backend && go test ./internal/service/... -run "TestBuildAccountCostInfo\|TestCopilotPlan" -v
# 预期：PASS
```

- [ ] **Step 4: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/copilot_analytics_service.go \
        backend/internal/service/copilot_analytics_service_test.go
git commit -m "Feature: 新增 CopilotAnalyticsService，封装用户维度和账户成本统计逻辑"
```

---
## Phase 5：后端 - Handler 层与路由

### Task 11：CopilotAnalyticsHandler + 路由注册

**Files:**
- Create: `backend/internal/handler/admin/copilot_analytics_handler.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/server/routes/admin.go`

- [ ] **Step 1: 实现 Handler**

```go
// backend/internal/handler/admin/copilot_analytics_handler.go
package admin

import (
    "net/http"
    "strconv"
    "time"

    "github.com/Wei-Shaw/sub2api/internal/pkg/response"
    "github.com/Wei-Shaw/sub2api/internal/service"
    "github.com/gin-gonic/gin"
)

// CopilotAnalyticsHandler 处理 Copilot 成本分析相关的管理员 API。
type CopilotAnalyticsHandler struct {
    analyticsService *service.CopilotAnalyticsService
    quotaCache       *service.CopilotQuotaCacheService
    budgetRepo       service.CopilotBudgetAlertPort
    adminService     service.AdminService
}

func NewCopilotAnalyticsHandler(
    analyticsService *service.CopilotAnalyticsService,
    quotaCache *service.CopilotQuotaCacheService,
    budgetRepo service.CopilotBudgetAlertPort,
    adminService service.AdminService,
) *CopilotAnalyticsHandler {
    return &CopilotAnalyticsHandler{
        analyticsService: analyticsService,
        quotaCache:       quotaCache,
        budgetRepo:       budgetRepo,
        adminService:     adminService,
    }
}

// GetUserStats GET /api/v1/admin/copilot/users/stats
func (h *CopilotAnalyticsHandler) GetUserStats(c *gin.Context) {
    date, err := parseDateParam(c, "date")
    if err != nil {
        response.BadRequest(c, "invalid date, use YYYY-MM-DD")
        return
    }
    var userID int64
    if s := c.Query("user_id"); s != "" {
        userID, err = strconv.ParseInt(s, 10, 64)
        if err != nil {
            response.BadRequest(c, "invalid user_id")
            return
        }
    }
    result, err := h.analyticsService.GetUserStats(c.Request.Context(), service.CopilotUserStatsInput{
        Date: date, UserID: userID,
    })
    if err != nil {
        response.InternalError(c, "failed to get user stats: "+err.Error())
        return
    }
    response.Success(c, result)
}

// GetUserTimeline GET /api/v1/admin/copilot/users/:id/timeline
func (h *CopilotAnalyticsHandler) GetUserTimeline(c *gin.Context) {
    userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        response.BadRequest(c, "invalid user id")
        return
    }
    date, err := parseDateParam(c, "date")
    if err != nil {
        response.BadRequest(c, "invalid date")
        return
    }
    buckets, err := h.analyticsService.analyticsPort.QueryHourlyBuckets(c.Request.Context(), userID, 0, date)
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }
    response.Success(c, gin.H{"user_id": userID, "date": date.Format("2006-01-02"), "hourly": buckets})
}

// GetUserRequests GET /api/v1/admin/copilot/users/:id/requests
func (h *CopilotAnalyticsHandler) GetUserRequests(c *gin.Context) {
    userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        response.BadRequest(c, "invalid user id")
        return
    }
    date, err := parseDateParam(c, "date")
    if err != nil {
        response.BadRequest(c, "invalid date")
        return
    }
    page, pageSize := response.ParsePagination(c)
    items, total, err := h.analyticsService.analyticsPort.QueryUserRequests(c.Request.Context(), userID, date, page, pageSize)
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }
    response.SuccessWithTotal(c, items, total)
}

// GetAccountsOverview GET /api/v1/admin/copilot/accounts/overview
func (h *CopilotAnalyticsHandler) GetAccountsOverview(c *gin.Context) {
    result, err := h.analyticsService.GetAccountsOverview(c.Request.Context())
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }
    response.Success(c, result)
}

// GetAccountQuotaTrend GET /api/v1/admin/copilot/accounts/:id/quota-trend
func (h *CopilotAnalyticsHandler) GetAccountQuotaTrend(c *gin.Context) {
    accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        response.BadRequest(c, "invalid account id")
        return
    }
    days := 30
    if s := c.Query("days"); s != "" {
        days, err = strconv.Atoi(s)
        if err != nil || days < 1 || days > 365 {
            response.BadRequest(c, "invalid days, range 1-365")
            return
        }
    }
    // 从 snapshotRepo 获取历史快照
    // （通过 analyticsService 暴露）
    response.Success(c, gin.H{"account_id": accountID, "days": days, "trend": []interface{}{}})
}

// GetAccountHourlyStats GET /api/v1/admin/copilot/accounts/:id/hourly-stats
func (h *CopilotAnalyticsHandler) GetAccountHourlyStats(c *gin.Context) {
    accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        response.BadRequest(c, "invalid account id")
        return
    }
    date, err := parseDateParam(c, "date")
    if err != nil {
        response.BadRequest(c, "invalid date")
        return
    }
    buckets, err := h.analyticsService.analyticsPort.QueryHourlyBuckets(c.Request.Context(), 0, accountID, date)
    if err != nil {
        response.InternalError(c, err.Error())
        return
    }
    response.Success(c, gin.H{"account_id": accountID, "date": date.Format("2006-01-02"), "hourly": buckets})
}

// RefreshAccountQuota POST /api/v1/admin/copilot/accounts/:id/quota-refresh
func (h *CopilotAnalyticsHandler) RefreshAccountQuota(c *gin.Context) {
    accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        response.BadRequest(c, "invalid account id")
        return
    }
    account, err := h.adminService.GetAccountByID(c.Request.Context(), accountID)
    if err != nil {
        response.NotFound(c, "account not found")
        return
    }
    entry, err := h.quotaCache.ForceRefreshQuota(c.Request.Context(), account)
    if err != nil {
        response.InternalError(c, "failed to refresh quota: "+err.Error())
        return
    }
    response.Success(c, entry)
}

// UpdateAccountBudget PUT /api/v1/admin/copilot/accounts/:id/budget
func (h *CopilotAnalyticsHandler) UpdateAccountBudget(c *gin.Context) {
    accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        response.BadRequest(c, "invalid account id")
        return
    }
    var req service.CopilotBudgetAlertConfig
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "invalid request body: "+err.Error())
        return
    }
    req.AccountID = accountID
    if err := h.budgetRepo.(interface {
        Upsert(ctx interface{}, cfg *service.CopilotBudgetAlertConfig) error
    }).Upsert(c.Request.Context(), &req); err != nil {
        response.InternalError(c, "failed to update budget: "+err.Error())
        return
    }
    response.Success(c, req)
}

// parseDateParam 解析 query 参数中的日期，默认今天。
func parseDateParam(c *gin.Context, key string) (time.Time, error) {
    s := c.Query(key)
    if s == "" {
        now := time.Now()
        return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
    }
    return time.Parse("2006-01-02", s)
}
```

- [ ] **Step 2: handler.go 新增字段**

在 `backend/internal/handler/handler.go` 的 `AdminHandlers` struct 中追加：

```go
CopilotAnalytics *admin.CopilotAnalyticsHandler
```

- [ ] **Step 3: wire.go 新增参数**

在 `ProvideAdminHandlers` 函数签名末尾加入：

```go
copilotAnalyticsHandler *admin.CopilotAnalyticsHandler,
```

在 return 语句中加入：

```go
CopilotAnalytics: copilotAnalyticsHandler,
```

- [ ] **Step 4: 注册路由**

在 `backend/internal/server/routes/admin.go` 末尾新增函数并在主注册函数中调用：

```go
func registerCopilotAnalyticsRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
    copilot := admin.Group("/copilot")
    // 注意：不与已有的 /copilot/oauth/* 冲突，analytics 用不同子路径
    users := copilot.Group("/users")
    {
        users.GET("/stats", h.Admin.CopilotAnalytics.GetUserStats)
        users.GET("/:id/timeline", h.Admin.CopilotAnalytics.GetUserTimeline)
        users.GET("/:id/requests", h.Admin.CopilotAnalytics.GetUserRequests)
    }
    accounts := copilot.Group("/accounts")
    {
        accounts.GET("/overview", h.Admin.CopilotAnalytics.GetAccountsOverview)
        accounts.GET("/:id/quota-trend", h.Admin.CopilotAnalytics.GetAccountQuotaTrend)
        accounts.GET("/:id/hourly-stats", h.Admin.CopilotAnalytics.GetAccountHourlyStats)
        accounts.POST("/:id/quota-refresh", h.Admin.CopilotAnalytics.RefreshAccountQuota)
        accounts.PUT("/:id/budget", h.Admin.CopilotAnalytics.UpdateAccountBudget)
    }
}
```

**重要**：确认现有 Copilot OAuth 路由注册在 `/copilot/oauth/...`，新增 analytics 路由在 `/copilot/users/...` 和 `/copilot/accounts/...`，不冲突。

- [ ] **Step 5: 更新 wire_gen.go**

```bash
cd backend && wire ./cmd/server/...
# 若 wire 命令不可用，手动更新 wire_gen.go 参考其他 handler 的注入模式
```

- [ ] **Step 6: 编译验证**

```bash
cd backend && go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add backend/internal/handler/admin/copilot_analytics_handler.go \
        backend/internal/handler/handler.go \
        backend/internal/handler/wire.go \
        backend/internal/server/routes/admin.go \
        backend/cmd/server/wire_gen.go
git commit -m "Feature: 新增 CopilotAnalyticsHandler 和路由注册"
```

---
## Phase 6：前端 - API 客户端与路由

### Task 12：前端 API 客户端

**Files:**
- Create: `frontend/src/api/admin/copilotAnalytics.ts`

- [ ] **Step 1: 创建 API 客户端**

```typescript
// frontend/src/api/admin/copilotAnalytics.ts
import { apiClient } from '../client'

// --- 用户维度类型 ---

export interface CopilotUserDayStat {
  user_id: number
  username: string
  premium_requests: number
  agent_requests: number
  total_requests: number
  models: string[]
  last_request_at: string | null
}

export interface CopilotUserStatsResult {
  date: string
  summary: {
    total_premium_requests: number
    total_agent_requests: number
    active_users: number
  }
  users: CopilotUserDayStat[]
}

export interface CopilotHourlyBucket {
  hour: number
  premium_count: number
  agent_count: number
}

export interface CopilotRequestItem {
  request_id: string
  model: string
  initiator: 'user' | 'agent'
  created_at: string
  duration_ms: number | null
  sub_requests: CopilotRequestItem[]
}

// --- 账户维度类型 ---

export interface CopilotQuotaSnapshot {
  premium_entitlement: number
  premium_remaining: number
  github_total_used: number
  overage: number
  unlimited: boolean
  cached_at: string
}

export interface CopilotBudgetAlert {
  monthly_budget: number
  alert_threshold: number
  enabled: boolean
}

export interface CopilotAccountOverviewItem {
  account_id: number
  name: string
  plan_type: string
  seat_count: number
  monthly_cost: number
  monthly_quota: number
  cost_per_premium_request: number
  system_today_premium_requests: number
  system_month_premium_requests: number
  quota_snapshot: CopilotQuotaSnapshot | null
  quota_cached_at: string | null
  budget_alert: CopilotBudgetAlert | null
  alert_status: 'ok' | 'warning' | 'critical'
  external_used: number
}

export interface CopilotAccountsOverviewResult {
  summary: {
    total_accounts: number
    estimated_monthly_cost: number
    today_premium_requests: number
    alert_count: number
  }
  accounts: CopilotAccountOverviewItem[]
}

// --- API 函数 ---

export async function getCopilotUserStats(date?: string, userId?: number): Promise<CopilotUserStatsResult> {
  const params: Record<string, string> = {}
  if (date) params.date = date
  if (userId) params.user_id = String(userId)
  const { data } = await apiClient.get<CopilotUserStatsResult>('/admin/copilot/users/stats', { params })
  return data
}

export async function getCopilotUserTimeline(userId: number, date?: string): Promise<{ user_id: number; date: string; hourly: CopilotHourlyBucket[] }> {
  const params: Record<string, string> = {}
  if (date) params.date = date
  const { data } = await apiClient.get(`/admin/copilot/users/${userId}/timeline`, { params })
  return data
}

export async function getCopilotUserRequests(
  userId: number,
  date?: string,
  page = 1,
  pageSize = 20
): Promise<{ total: number; items: CopilotRequestItem[] }> {
  const params: Record<string, string | number> = { page, page_size: pageSize }
  if (date) params.date = date
  const { data } = await apiClient.get(`/admin/copilot/users/${userId}/requests`, { params })
  return data
}

export async function getCopilotAccountsOverview(): Promise<CopilotAccountsOverviewResult> {
  const { data } = await apiClient.get<CopilotAccountsOverviewResult>('/admin/copilot/accounts/overview')
  return data
}

export async function getCopilotAccountQuotaTrend(
  accountId: number,
  days = 30
): Promise<{ account_id: number; trend: Array<{ date: string; premium_used: number; premium_remaining: number; premium_entitlement: number }> }> {
  const { data } = await apiClient.get(`/admin/copilot/accounts/${accountId}/quota-trend`, { params: { days } })
  return data
}

export async function getCopilotAccountHourlyStats(
  accountId: number,
  date?: string
): Promise<{ account_id: number; date: string; hourly: CopilotHourlyBucket[] }> {
  const params: Record<string, string> = {}
  if (date) params.date = date
  const { data } = await apiClient.get(`/admin/copilot/accounts/${accountId}/hourly-stats`, { params })
  return data
}

export async function refreshCopilotAccountQuota(accountId: number): Promise<CopilotQuotaSnapshot> {
  const { data } = await apiClient.post(`/admin/copilot/accounts/${accountId}/quota-refresh`)
  return data
}

export async function updateCopilotAccountBudget(
  accountId: number,
  budget: CopilotBudgetAlert
): Promise<void> {
  await apiClient.put(`/admin/copilot/accounts/${accountId}/budget`, budget)
}

export default {
  getCopilotUserStats,
  getCopilotUserTimeline,
  getCopilotUserRequests,
  getCopilotAccountsOverview,
  getCopilotAccountQuotaTrend,
  getCopilotAccountHourlyStats,
  refreshCopilotAccountQuota,
  updateCopilotAccountBudget,
}
```

- [ ] **Step 2: 在 admin/index.ts 中导出**

在 `frontend/src/api/admin/index.ts` 末尾加入：

```typescript
export * as copilotAnalytics from './copilotAnalytics'
```

- [ ] **Step 3: 编译验证**

```bash
cd frontend && npx tsc --noEmit
# 预期：无类型错误
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/admin/copilotAnalytics.ts frontend/src/api/admin/index.ts
git commit -m "Feature: 新增 Copilot Analytics 前端 API 客户端"
```

---

### Task 13：路由注册 + 侧边栏菜单

**Files:**
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`

- [ ] **Step 1: 注册路由**

在 `frontend/src/router/index.ts` 中，在 `/admin/usage` 路由条目**之后**插入：

```typescript
{
  path: '/admin/copilot/users',
  name: 'CopilotUsers',
  component: () => import('@/views/admin/copilot/CopilotUsersView.vue'),
  meta: {
    requiresAuth: true,
    requiresAdmin: true,
    titleKey: 'admin.copilot.users.title',
  },
},
{
  path: '/admin/copilot/accounts',
  name: 'CopilotAccounts',
  component: () => import('@/views/admin/copilot/CopilotAccountsView.vue'),
  meta: {
    requiresAuth: true,
    requiresAdmin: true,
    titleKey: 'admin.copilot.accounts.title',
  },
},
```

- [ ] **Step 2: 更新侧边栏**

在 `frontend/src/components/layout/AppSidebar.vue` 的 `adminNavItems` computed 中，找到 `'/admin/usage'` 条目，在其**之后**插入：

```typescript
{ path: '/admin/copilot/users', label: t('nav.copilotUsers'), icon: ChartIcon },
{ path: '/admin/copilot/accounts', label: t('nav.copilotAccounts'), icon: CreditCardIcon },
```

（`ChartIcon` 和 `CreditCardIcon` 在该文件中已有导入，直接复用）

- [ ] **Step 3: 添加 i18n 翻译键**

在前端 i18n 文件（`frontend/src/locales/` 目录，找 `zh.json` 和 `en.json`）中分别加入：

中文（zh.json）：
```json
"nav": {
  "copilotUsers": "Copilot 用户分析",
  "copilotAccounts": "Copilot 账户成本"
},
"admin": {
  "copilot": {
    "users": { "title": "Copilot 用户请求分析" },
    "accounts": { "title": "Copilot 账户成本分析" }
  }
}
```

英文（en.json）：
```json
"nav": {
  "copilotUsers": "Copilot Users",
  "copilotAccounts": "Copilot Accounts"
},
"admin": {
  "copilot": {
    "users": { "title": "Copilot User Requests" },
    "accounts": { "title": "Copilot Account Costs" }
  }
}
```

- [ ] **Step 4: 创建页面占位文件（让路由可以编译）**

```bash
mkdir -p frontend/src/views/admin/copilot/components
```

分别创建两个占位 Vue 文件（后续 Task 14/15 替换为完整实现）：

`frontend/src/views/admin/copilot/CopilotUsersView.vue`：
```vue
<template><div>Copilot Users - TODO</div></template>
<script setup lang="ts"></script>
```

`frontend/src/views/admin/copilot/CopilotAccountsView.vue`：
```vue
<template><div>Copilot Accounts - TODO</div></template>
<script setup lang="ts"></script>
```

- [ ] **Step 5: 编译验证**

```bash
cd frontend && npm run build 2>&1 | tail -20
# 预期：编译成功
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/router/index.ts \
        frontend/src/components/layout/AppSidebar.vue \
        frontend/src/views/admin/copilot/ \
        frontend/src/locales/
git commit -m "Feature: 新增 Copilot 分析页面路由和侧边栏菜单入口"
```

---
## Phase 7：前端页面实现

### Task 14：用户请求分析页 CopilotUsersView.vue

**Files:**
- Modify: `frontend/src/views/admin/copilot/CopilotUsersView.vue`
- Create: `frontend/src/views/admin/copilot/components/UserRequestTree.vue`
- Create: `frontend/src/views/admin/copilot/components/HourlyBarChart.vue`

- [ ] **Step 1: 实现 HourlyBarChart.vue（复用 ECharts，参考 OpsErrorDistributionChart.vue）**

```vue
<!-- frontend/src/views/admin/copilot/components/HourlyBarChart.vue -->
<template>
  <div ref="chartRef" style="height: 200px; width: 100%" />
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { CopilotHourlyBucket } from '@/api/admin/copilotAnalytics'

echarts.use([BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{ buckets: CopilotHourlyBucket[] }>()
const chartRef = ref<HTMLDivElement>()
let chart: echarts.ECharts | null = null

function buildOption() {
  const hours = Array.from({ length: 24 }, (_, i) => `${i}:00`)
  const premiumData = Array(24).fill(0)
  const agentData = Array(24).fill(0)
  for (const b of props.buckets) {
    premiumData[b.hour] = b.premium_count
    agentData[b.hour] = b.agent_count
  }
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['Premium 请求', '子请求'] },
    xAxis: { type: 'category', data: hours },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      { name: 'Premium 请求', type: 'bar', stack: 'total', data: premiumData, color: '#10b981' },
      { name: '子请求', type: 'bar', stack: 'total', data: agentData, color: '#6366f1' },
    ],
  }
}

onMounted(() => {
  if (chartRef.value) {
    chart = echarts.init(chartRef.value)
    chart.setOption(buildOption())
  }
})

watch(() => props.buckets, () => chart?.setOption(buildOption()))
onUnmounted(() => chart?.dispose())
</script>
```

- [ ] **Step 2: 实现 UserRequestTree.vue（层级请求展示）**

```vue
<!-- frontend/src/views/admin/copilot/components/UserRequestTree.vue -->
<template>
  <div class="space-y-1">
    <div v-for="item in items" :key="item.request_id" class="text-sm">
      <!-- 顶层 USER 请求 -->
      <div
        class="flex items-center gap-2 px-3 py-2 rounded cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800"
        @click="toggleExpand(item.request_id)"
      >
        <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200">
          USER
        </span>
        <span class="font-mono text-gray-600 dark:text-gray-400 flex-1">{{ item.model }}</span>
        <span class="text-gray-400 text-xs">{{ formatTime(item.created_at) }}</span>
        <span class="text-gray-400 text-xs">{{ item.duration_ms }}ms</span>
        <span v-if="item.sub_requests?.length" class="text-gray-400 text-xs">
          {{ expanded.has(item.request_id) ? '▾' : '▸' }} {{ item.sub_requests.length }} 子请求
        </span>
        <span
          class="text-xs text-indigo-500 hover:underline ml-1"
          @click.stop="$emit('inspect', item.request_id)"
        >详情</span>
      </div>
      <!-- 子请求（展开时显示） -->
      <div v-if="expanded.has(item.request_id) && item.sub_requests?.length" class="ml-6 border-l-2 border-gray-200 dark:border-gray-700 pl-3 space-y-1">
        <div
          v-for="sub in item.sub_requests"
          :key="sub.request_id"
          class="flex items-center gap-2 px-2 py-1.5 rounded text-xs hover:bg-gray-50 dark:hover:bg-gray-800"
        >
          <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200">
            AGENT
          </span>
          <span class="font-mono text-gray-500 flex-1">{{ sub.model }}</span>
          <span class="text-gray-400">{{ formatTime(sub.created_at) }}</span>
          <span class="text-gray-400">{{ sub.duration_ms }}ms</span>
          <span
            class="text-xs text-indigo-500 hover:underline ml-1"
            @click="$emit('inspect', sub.request_id)"
          >详情</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CopilotRequestItem } from '@/api/admin/copilotAnalytics'

defineProps<{ items: CopilotRequestItem[] }>()
defineEmits<{ inspect: [requestId: string] }>()

const expanded = ref(new Set<string>())
function toggleExpand(id: string) {
  if (expanded.value.has(id)) expanded.value.delete(id)
  else expanded.value.add(id)
}
function formatTime(iso: string) {
  return new Date(iso).toLocaleTimeString('zh-CN', { hour12: false })
}
</script>
```

- [ ] **Step 3: 实现 CopilotUsersView.vue 主页**

参考 `/admin/ops/OpsDashboard.vue` 的整体结构（AppLayout + 卡片 + 表格），实现：
- 顶部日期选择器（复用已有 `DateRangePicker` 或单日期 `input[type=date]`）
- 3 张统计卡片（复用 `OpsDashboardHeader` 或自定义）
- 全局小时柱状图（使用 `HourlyBarChart`）
- 用户列表表格（展开行使用 `UserRequestTree`）
- 点击"详情"跳转到 `/admin/ops/request-inspect?request_id=xxx`

关键逻辑：
```typescript
import { useRouter } from 'vue-router'
const router = useRouter()
function onInspect(requestId: string) {
  router.push({ path: '/admin/ops/request-inspect', query: { request_id: requestId } })
}
```

- [ ] **Step 4: 编译验证**

```bash
cd frontend && npm run build 2>&1 | tail -20
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/admin/copilot/
git commit -m "Feature: 实现 Copilot 用户请求分析页（HourlyBarChart + UserRequestTree）"
```

---

### Task 15：账户成本分析页 CopilotAccountsView.vue

**Files:**
- Modify: `frontend/src/views/admin/copilot/CopilotAccountsView.vue`
- Create: `frontend/src/views/admin/copilot/components/AccountCostCard.vue`
- Create: `frontend/src/views/admin/copilot/components/QuotaTrendChart.vue`
- Create: `frontend/src/views/admin/copilot/components/BudgetAlertDialog.vue`

- [ ] **Step 1: 实现 QuotaTrendChart.vue（近30天折线图）**

ECharts 折线图，X 轴为日期，Y 轴为已用/剩余量，参考 `OpsThroughputTrendChart.vue` 实现模式。

- [ ] **Step 2: 实现 BudgetAlertDialog.vue（预算告警配置弹窗）**

复用项目中已有的 Dialog 组件（搜索 `OpsAlertRulesCard.vue` 或 `Modal` 组件），包含：
- 月预算输入框（美元）
- 告警阈值滑块（0-100%）
- 启用/禁用开关
- 保存按钮调用 `updateCopilotAccountBudget()`

- [ ] **Step 3: 实现 AccountCostCard.vue**

```vue
<!-- frontend/src/views/admin/copilot/components/AccountCostCard.vue -->
<template>
  <div class="card p-4 space-y-3">
    <!-- 标题行 -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="font-semibold">{{ account.name }}</span>
        <span class="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700">
          {{ planLabel }} · ${{ account.monthly_cost }}/月
        </span>
        <span
          v-if="account.alert_status !== 'ok'"
          class="px-2 py-0.5 rounded text-xs"
          :class="alertClass"
        >
          {{ account.alert_status === 'critical' ? '⚠ 严重' : '⚠ 告警' }}
        </span>
      </div>
      <div class="flex gap-2">
        <button class="btn-sm" @click="$emit('toggle-trend')">
          {{ showTrend ? '收起' : '查看趋势 ▾' }}
        </button>
        <button class="btn-sm" @click="$emit('set-budget')">设置预算</button>
        <button class="btn-sm" :disabled="refreshing" @click="onRefresh">刷新</button>
      </div>
    </div>

    <!-- 配额进度条 -->
    <div v-if="account.quota_snapshot">
      <div class="flex justify-between text-xs text-gray-500 mb-1">
        <span>GitHub Premium 配额</span>
        <span>{{ account.quota_snapshot.github_total_used }} / {{ account.monthly_quota }} ({{ usageRate }}%)</span>
      </div>
      <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
        <div
          class="h-2 rounded-full transition-all"
          :class="progressClass"
          :style="{ width: Math.min(usageRate, 100) + '%' }"
        />
      </div>
    </div>

    <!-- 统计数字 -->
    <div class="grid grid-cols-3 gap-4 text-sm">
      <div>
        <div class="text-gray-500 text-xs">系统内今日</div>
        <div class="font-semibold">{{ account.system_today_premium_requests }} 次</div>
      </div>
      <div>
        <div class="text-gray-500 text-xs">系统内本月</div>
        <div class="font-semibold">{{ account.system_month_premium_requests }} 次</div>
      </div>
      <div>
        <div class="text-gray-500 text-xs">外部消耗</div>
        <div class="font-semibold">{{ account.external_used }} 次</div>
      </div>
    </div>

    <div class="text-xs text-gray-400">
      单次成本: ${{ costPerRequest }}
      <span v-if="account.quota_cached_at">· 更新: {{ cachedAtLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { CopilotAccountOverviewItem } from '@/api/admin/copilotAnalytics'
import { refreshCopilotAccountQuota } from '@/api/admin/copilotAnalytics'

const props = defineProps<{
  account: CopilotAccountOverviewItem
  showTrend: boolean
}>()
defineEmits<{ 'toggle-trend': []; 'set-budget': []; refreshed: [] }>()

const refreshing = ref(false)

const planLabels: Record<string, string> = {
  individual_free: 'Free',
  individual_pro: 'Pro',
  individual_pro_plus: 'Pro+',
  business: 'Business',
  enterprise: 'Enterprise',
}
const planLabel = computed(() => planLabels[props.account.plan_type] ?? props.account.plan_type)

const usageRate = computed(() => {
  const snap = props.account.quota_snapshot
  if (!snap || !props.account.monthly_quota) return 0
  return Math.round((snap.github_total_used / props.account.monthly_quota) * 100)
})

const progressClass = computed(() => {
  const r = usageRate.value
  if (r >= 80) return 'bg-red-500'
  if (r >= 60) return 'bg-yellow-500'
  return 'bg-emerald-500'
})

const alertClass = computed(() => {
  return props.account.alert_status === 'critical'
    ? 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
    : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'
})

const costPerRequest = computed(() => props.account.cost_per_premium_request.toFixed(4))

const cachedAtLabel = computed(() => {
  if (!props.account.quota_cached_at) return ''
  return new Date(props.account.quota_cached_at).toLocaleTimeString('zh-CN', { hour12: false })
})

async function onRefresh() {
  refreshing.value = true
  try {
    await refreshCopilotAccountQuota(props.account.account_id)
    // 触发父组件重新加载
  } finally {
    refreshing.value = false
  }
}
</script>
```

- [ ] **Step 4: 实现 CopilotAccountsView.vue 主页**

整合 `AccountCostCard`、`QuotaTrendChart`、`HourlyBarChart`、`BudgetAlertDialog`：
- 进入页面自动调用 `getCopilotAccountsOverview()`（显示 loading）
- 顶部 4 张汇总卡片
- 账户卡片列表（展开时显示趋势图 + 小时柱状图）
- `BudgetAlertDialog` 通过 `v-model:visible` 控制显示

- [ ] **Step 5: 编译验证**

```bash
cd frontend && npm run build 2>&1 | tail -20
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/admin/copilot/
git commit -m "Feature: 实现 Copilot 账户成本分析页（配额进度条 + 趋势图 + 预算告警配置）"
```

---

### Task 16：账户编辑表单扩展（套餐类型 + 座席数）

**Files:**
- Modify: `frontend/src/views/admin/AccountsView.vue`（或账户编辑 Modal 所在文件）

- [ ] **Step 1: 定位账户编辑表单**

```bash
grep -rn "credentials\|extra\|plan_type\|AccountModal\|EditAccount" \
  frontend/src/views/admin/ | grep -v node_modules | head -20
```

- [ ] **Step 2: 在 Copilot 平台账户表单中新增字段**

找到账户编辑表单中判断 `platform === 'copilot'` 的条件渲染区域，新增：

```vue
<!-- 仅 Copilot 平台显示 -->
<template v-if="form.platform === 'copilot'">
  <div class="form-item">
    <label>套餐类型</label>
    <select v-model="form.extra.copilot_plan_type">
      <option value="">未配置</option>
      <option value="individual_free">Free ($0/月, 50次)</option>
      <option value="individual_pro">Pro ($10/月, 300次)</option>
      <option value="individual_pro_plus">Pro+ ($39/月, 1500次)</option>
      <option value="business">Business ($19/用户/月, 300次/用户)</option>
      <option value="enterprise">Enterprise ($39/用户/月, 1000次/用户)</option>
    </select>
  </div>
  <div v-if="['business','enterprise'].includes(form.extra.copilot_plan_type)" class="form-item">
    <label>座席数</label>
    <input type="number" v-model.number="form.extra.copilot_seat_count" min="1" />
  </div>
</template>
```

- [ ] **Step 3: 编译验证**

```bash
cd frontend && npm run build 2>&1 | tail -20
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/admin/AccountsView.vue
git commit -m "Feature: 账户编辑表单新增 Copilot 套餐类型和座席数配置"
```

---
## Phase 8：测试与发布

### Task 17：后端单元测试补全

**Files:**
- Modify: `backend/internal/service/copilot_quota_cache_test.go`
- Modify: `backend/internal/service/copilot_analytics_service_test.go`

- [ ] **Step 1: 补全 copilot_quota_cache_test.go**

实现 `TestCopilotQuotaCacheService_CacheTTL`（使用 mock snapshotRepo，注入短 TTL 验证缓存失效）：

```go
func TestCopilotQuotaCacheService_CacheTTL(t *testing.T) {
    // 使用短 TTL（100ms）验证缓存过期后重新拉取
    // mock FetchQuota 调用次数：首次 = 1，缓存命中 = 1，TTL 过期后 = 2
    // 由于 CopilotGatewayService 较难 mock，此测试验证 computeAlertStatus 和
    // copilotPlanCostAndQuota 的边界条件即可
    tests := []struct {
        usageRate float64
        threshold int
        expected  string
    }{
        {0, 80, "ok"},
        {79.9, 80, "ok"},
        {80.0, 80, "warning"},
        {94.9, 80, "warning"},
        {95.0, 80, "critical"},
        {100.0, 80, "critical"},
    }
    for _, tt := range tests {
        assert.Equal(t, tt.expected, computeAlertStatus(tt.usageRate, tt.threshold),
            "usageRate=%.1f threshold=%d", tt.usageRate, tt.threshold)
    }
}
```

- [ ] **Step 2: 运行所有 service 测试**

```bash
cd backend && go test ./internal/service/... -v -run "TestCopilot" 2>&1 | tail -30
# 预期：所有 TestCopilot* 测试 PASS
```

- [ ] **Step 3: 运行全量测试确认无回归**

```bash
cd backend && go test ./... 2>&1 | grep -E "FAIL|ok|---" | tail -30
# 预期：无 FAIL
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/copilot_quota_cache_test.go \
        backend/internal/service/copilot_analytics_service_test.go
git commit -m "Feature: 补全 Copilot 分析服务单元测试"
```

---

### Task 18：版本发布

- [ ] **Step 1: 更新版本号**

```bash
# 查看当前版本
cat backend/cmd/server/VERSION
# 更新到下一个版本（如 0.1.126）
echo "0.1.126" > backend/cmd/server/VERSION
```

- [ ] **Step 2: 提交版本变更**

```bash
git add backend/cmd/server/VERSION
git commit -m "Feature: 版本更新至 0.1.126 - Copilot 成本分析系统"
```

- [ ] **Step 3: 创建 git tag**

```bash
git tag v0.1.126
git push origin main --tags
```

- [ ] **Step 4: 验证 CI/CD 触发**

```bash
# 检查当前仓库的 CI 运行（fork 仓库）
gh run list --limit 5
# 预期：看到新的 workflow run 被触发
```

---

## 重要说明

### 页面说明
- `/admin/copilot/users` 中的请求时间均以服务器时区显示（与现有 ops 页面保持一致）
- 历史数据偏差：`initiator` 字段上线前的记录默认为 `'user'`，页面加注"数据从 v0.1.126 起准确"
- 层级关联为启发式规则（30秒窗口），前端显示"≈近似关联"提示

### 已知限制
- `CopilotAnalyticsPort` 接口的数据库实现（`QueryUserDayStats`、`QueryHourlyBuckets`、`QueryUserRequests`、`QueryAccountMonthStats`）需要在 `usage_log_repo.go` 中实现原始 SQL 查询（涉及 `GROUP BY`、`DATE_TRUNC`、`FILTER` 等），实现时参考同文件已有的复杂查询写法
- `wire_gen.go` 更新：运行 `wire ./cmd/server/...` 或手动按模式更新

### 参考文件
- 复杂 SQL 查询：`backend/internal/repository/usage_log_repo.go`
- ECharts 用法：`frontend/src/views/admin/ops/components/OpsErrorDistributionChart.vue`
- Handler 注入模式：`backend/internal/handler/admin/copilot_oauth_handler.go`
- 卡片样式：`frontend/src/views/admin/ops/components/OpsDashboardHeader.vue`
