# Copilot 成本分析系统设计文档

**日期**: 2026-03-28
**状态**: 待实现
**作者**: 管理员

---

## 1. 背景与目标

sub2api 主要使用 GitHub Copilot 平台账户作为上游，Copilot 按请求次数（Premium Interactions）计费。当前系统缺乏对 Copilot 资源消耗的精细追踪，无法回答以下问题：

- 哪些用户每天消耗了多少次 Premium 请求？
- 用户的一次"任务"产生了多少次子请求？
- 某个 Copilot 账户的配额还剩多少？按当前速度能撑到月底吗？
- 本月所有 Copilot 账户的总运营成本是多少？

**目标**：为管理员提供完整的 Copilot 成本分析工具，包含两个维度：
1. **用户请求维度** — 每个用户消耗了多少次 Premium/子请求，层级展示
2. **Copilot 账户维度** — 每个账户的配额消耗趋势、套餐费用、预算告警

---

## 2. 核心概念

### 2.1 Premium 请求 vs 子请求

Copilot API 通过 `X-Initiator` 请求头区分两类请求：

| 类型 | X-Initiator 值 | 消耗配额 | 说明 |
|------|--------------|--------|------|
| **Premium 请求** | `user` | Premium Interactions 配额 | 用户发起的首轮对话，消息历史中无 assistant/tool 消息 |
| **子请求** | `agent` | 标准配额（不计入 Premium） | 任务内后续轮次，消息历史中有 assistant 或 tool 消息 |

现有 `copilotInitiator()` 函数已实现此判断逻辑，但结果未持久化到数据库。

### 2.2 套餐价格与配额

| 套餐 | 月付价格 | Premium 配额/月 | 备注 |
|------|--------|--------------|------|
| Free | $0 | 50次 | 个人 |
| Pro | $10/用户 | 300次 | 个人 |
| Pro+ | $39/用户 | 1500次 | 个人高级 |
| Business | $19/用户/月 | 300次/用户 | 按座席 |
| Enterprise | $39/用户/月 | 1000次/用户 | 按座席 |

**单次 Premium 请求成本** = 套餐月费 / 套餐月配额
Business/Enterprise 多座席：月费 = 单价 × 座席数，月配额 = 配额/用户 × 座席数

---

## 3. 数据层设计

### 3.1 `usage_logs` 新增字段

在现有 `usage_logs` 表新增一个字段，记录每次请求的发起类型：

```sql
ALTER TABLE usage_logs ADD COLUMN initiator TEXT NOT NULL DEFAULT 'user';
-- 取值: 'user' | 'agent'
-- 非 Copilot 平台统一写 'user'（不影响其他平台逻辑）
```

**写入时机**：Copilot gateway 写日志前调用 `copilotInitiator()` 函数，结果写入此字段。

**Ent Schema 变更**：在 `ent/schema/usage_log.go` 的 `Fields()` 中新增：

```go
field.String("initiator").
    MaxLen(10).
    Default("user").
    Comment("Request initiator type: 'user' (premium) or 'agent' (standard sub-request)."),
```

新增索引（优化 Copilot 分析查询）：

```go
index.Fields("account_id", "initiator", "created_at"),
index.Fields("user_id", "initiator", "created_at"),
```

### 3.2 账户 `extra` 字段扩展

在账户管理页面新增 Copilot 套餐配置，存入账户的 `extra` JSONB 字段：

```json
{
  "copilot_plan_type": "individual_pro_plus",
  "copilot_seat_count": 1
}
```

- `copilot_plan_type`：套餐类型，与现有 `PlanType` 常量对齐
- `copilot_seat_count`：座席数（仅 Business/Enterprise 有效，默认 1）

无需修改数据库 Schema，直接利用现有 `extra` JSONB 字段。

### 3.3 新增表 `copilot_quota_snapshots`

每天定时记录每个 Copilot 账户的配额快照，用于趋势分析：

```go
// CopilotQuotaSnapshot 定义 Copilot 账户配额快照 schema。
// 每天由定时任务写入一条，记录当天的配额使用情况，用于趋势分析和成本预测。
type CopilotQuotaSnapshot struct {
    ent.Schema
}

func (CopilotQuotaSnapshot) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("account_id"),
        field.Time("snapshot_date").
            SchemaType(map[string]string{dialect.Postgres: "date"}),
        field.String("plan_type").
            MaxLen(30).
            Optional().Nillable(),
        field.Int("premium_entitlement").Default(0),  // 月配额总量
        field.Int("premium_remaining").Default(0),    // 剩余量
        field.Int("premium_used").Default(0),         // 已用量
        field.Int("premium_overage").Default(0),      // 超额使用量
        field.Bool("unlimited").Default(false),
        field.Time("created_at").
            Default(time.Now).Immutable().
            SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
    }
}

func (CopilotQuotaSnapshot) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("account_id", "snapshot_date").Unique(),
        index.Fields("snapshot_date"),
    }
}
```

### 3.4 新增表 `copilot_budget_alerts`

存储每个账户的预算告警配置：

```go
type CopilotBudgetAlert struct {
    ent.Schema
}

func (CopilotBudgetAlert) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("account_id").Unique(),
        field.Float("monthly_budget").
            SchemaType(map[string]string{dialect.Postgres: "decimal(10,2)"}),
        field.Int("alert_threshold").Default(80),  // 配额使用率告警阈值（%）
        field.Bool("enabled").Default(true),
        field.Time("last_alerted_at").Optional().Nillable().
            SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
    }
}

func (CopilotBudgetAlert) Mixin() []ent.Mixin {
    return []ent.Mixin{mixins.TimeMixin{}}
}

func (CopilotBudgetAlert) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("account_id"),
        index.Fields("enabled"),
    }
}
```

---

## 4. 后端 API 设计

### 4.1 路由注册

在 `/api/v1/admin/` 下新增 `copilot` 路由组：

```
/api/v1/admin/copilot/users/stats              GET
/api/v1/admin/copilot/users/:id/timeline       GET
/api/v1/admin/copilot/users/:id/requests       GET
/api/v1/admin/copilot/accounts/overview        GET
/api/v1/admin/copilot/accounts/:id/quota-trend GET
/api/v1/admin/copilot/accounts/:id/hourly-stats GET
/api/v1/admin/copilot/accounts/:id/quota-refresh POST
/api/v1/admin/copilot/accounts/:id/budget      PUT
```

### 4.2 用户维度 API

**GET `/api/v1/admin/copilot/users/stats`**

Query params: `date` (YYYY-MM-DD, 默认今天), `user_id` (可选)

Response:
```json
{
  "date": "2026-03-28",
  "summary": {
    "total_premium_requests": 342,
    "total_agent_requests": 1205,
    "active_users": 28
  },
  "users": [
    {
      "user_id": 1,
      "username": "alice",
      "premium_requests": 15,
      "agent_requests": 48,
      "total_requests": 63,
      "models": ["gpt-4o", "claude-3.5-sonnet"],
      "last_request_at": "2026-03-28T14:23:05Z"
    }
  ]
}
```

**GET `/api/v1/admin/copilot/users/:id/timeline`**

Query params: `date`

Response: 24个小时桶
```json
{
  "user_id": 1,
  "date": "2026-03-28",
  "hourly": [
    { "hour": 0, "premium_count": 0, "agent_count": 0 },
    { "hour": 9, "premium_count": 3, "agent_count": 12 },
    ...
  ]
}
```

**GET `/api/v1/admin/copilot/users/:id/requests`**

Query params: `date`, `page`, `page_size`

层级关联规则：同一用户，`initiator='agent'` 的请求归属到时间上最近的前一条 `initiator='user'` 请求（时间窗口：前后30秒内）。

Response:
```json
{
  "total": 63,
  "items": [
    {
      "request_id": "req_abc123",
      "model": "gpt-4o",
      "initiator": "user",
      "created_at": "2026-03-28T14:23:01Z",
      "duration_ms": 1200,
      "sub_requests": [
        {
          "request_id": "req_def456",
          "model": "gpt-4o",
          "initiator": "agent",
          "created_at": "2026-03-28T14:23:02Z",
          "duration_ms": 800
        }
      ]
    }
  ]
}
```

### 4.3 账户维度 API

**GET `/api/v1/admin/copilot/accounts/overview`**

Response:
```json
{
  "summary": {
    "total_accounts": 5,
    "estimated_monthly_cost": 195.00,
    "today_premium_requests": 342,
    "alert_count": 1
  },
  "accounts": [
    {
      "account_id": 10,
      "name": "copilot-pro-plus-01",
      "plan_type": "individual_pro_plus",
      "seat_count": 1,
      "monthly_cost": 39.00,
      "cost_per_premium_request": 0.026,
      "system_today_premium_requests": 45,
      "system_month_premium_requests": 312,
      "quota_snapshot": {
        "entitlement": 1500,
        "remaining": 300,
        "github_total_used": 1200,
        "overage": 0,
        "unlimited": false,
        "external_used": 888,
        "cached_at": "2026-03-28T14:20:00Z"
      },
      "budget_alert": {
        "monthly_budget": 39.00,
        "alert_threshold": 80,
        "enabled": true
      },
      "alert_status": "warning"
    }
  ]
}
```

**GET `/api/v1/admin/copilot/accounts/:id/quota-trend`**

Query params: `days` (默认30)

Response:
```json
{
  "account_id": 10,
  "trend": [
    {
      "date": "2026-03-01",
      "premium_used": 150,
      "premium_remaining": 1350,
      "premium_entitlement": 1500
    }
  ]
}
```

**GET `/api/v1/admin/copilot/accounts/:id/hourly-stats`**

Query params: `date`

Response: 与用户 timeline 格式一致，24个小时桶。

**POST `/api/v1/admin/copilot/accounts/:id/quota-refresh`**

强制绕过缓存，实时调用 `FetchQuota()` 拉取单个账户最新配额，更新内存缓存并 UPSERT 当天快照。
返回最新的配额数据（含 `cached_at`）。

**PUT `/api/v1/admin/copilot/accounts/:id/budget`**

Body:
```json
{
  "monthly_budget": 39.00,
  "alert_threshold": 80,
  "enabled": true
}
```

### 4.4 配额数据策略：按需实时 + 短时缓存

**无定时任务**。配额数据在用户访问账户成本分析页时实时拉取，原因：
- 管理员查看时需要看到最新状态，定时任务最多延迟数小时
- 账户可能在 sub2api 之外被使用（Claude Code、Copilot 插件等），定时快照无法反映实时消耗

**缓存策略**（服务端内存缓存，可用 `sync.Map` 或 `go-cache`）：
- TTL：**5 分钟**
- Key：`copilot_quota:{account_id}`
- 进入账户成本页 → 触发 `FetchAllCopilotQuotas()` 并发拉取所有账户
- 命中缓存 → 直接返回，并在响应中附带 `cached_at` 时间戳
- 手动点击"刷新"按钮 → 强制绕过缓存，重新拉取单个账户

**快照持久化**（`copilot_quota_snapshots` 表的新用途）：
- 每次实时查询成功后，UPSERT 当天的快照记录（同一天多次查询覆盖）
- 用途：为趋势折线图提供历史数据（每天最后一次查询的结果作为当天快照）
- 系统重启后前端展示"最后已知状态"，避免页面空白

**告警检查时机**：
- 每次实时查询成功后（无论是页面加载还是手动刷新）同步检查告警条件
- 配额使用率 ≥ `alert_threshold` → `alert_status = 'warning'`
- 配额使用率 ≥ 95% → `alert_status = 'critical'`
- 写入系统日志（ops error log）

---

## 5. 前端页面设计

### 5.1 路由配置

```
/admin/copilot/users     -- Copilot 用户请求分析
/admin/copilot/accounts  -- Copilot 账户成本分析
```

### 5.2 用户请求分析页 `/admin/copilot/users`

**布局**：
```
┌─────────────────────────────────────────────────────┐
│  Copilot 用户请求分析          [日期选择器]           │
├─────────────────────────────────────────────────────┤
│  汇总卡片（3个）                                      │
│  [今日Premium请求] [今日子请求] [活跃用户数]          │
├─────────────────────────────────────────────────────┤
│  全局小时分布柱状图（所有用户合计）                    │
│  X轴: 0-23小时  ■Premium请求  ■子请求               │
├─────────────────────────────────────────────────────┤
│  用户列表表格              [搜索用户名]               │
│  用户名 | Premium | 子请求 | 模型 | 最后请求时间       │
│  > 展开行：请求层级列表                               │
│    ├─ [USER] gpt-4o  14:23:01  1200ms               │
│    │   ├─ [AGENT] gpt-4o  14:23:02  800ms           │
│    │   └─ [AGENT] gpt-4o  14:23:05  950ms           │
│    └─ [USER] claude-3.5  15:10:33  2300ms           │
│        └─ [AGENT] claude-3.5  15:10:35  1100ms      │
└─────────────────────────────────────────────────────┘
```

- 展开行内的每条请求可点击，跳转到 `/admin/ops/request-inspect?request_id=xxx`
- `[USER]` 标签用绿色，`[AGENT]` 标签用蓝色

### 5.3 账户成本分析页 `/admin/copilot/accounts`

**布局**：
```
┌─────────────────────────────────────────────────────┐
│  Copilot 账户成本分析                                 │
├─────────────────────────────────────────────────────┤
│  汇总卡片（4个）                                      │
│  [账户总数] [本月估算总费用] [今日Premium消耗] [告警数]│
├─────────────────────────────────────────────────────┤
│  账户卡片列表                                         │
│  ┌──────────────────────────────────────────────┐   │
│  │ copilot-pro-plus-01  [Pro+ · $39/月]  [⚠告警]  │   │
│  │ GitHub配额: ████████░░  1200/1500 (80%)        │   │
│  │ ├ 系统内: 今日45次 / 本月312次                  │   │
│  │ └ 外部消耗: 888次  单次成本: $0.026             │   │
│  │ 更新: 14:20  [查看趋势 ▾]  [设置预算]  [刷新]   │   │
│  └──────────────────────────────────────────────┘   │
│                                                       │
│  展开"查看趋势"后：                                   │
│  ┌──────────────────────────────────────────────┐   │
│  │ 近30天配额消耗趋势（折线图）                    │   │
│  │ 今日小时分布（柱状图）                          │   │
│  │ 预算告警设置: 月预算 $[___]  阈值 [__]%        │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

- 告警状态颜色：`ok` 绿色，`warning` 黄色，`critical` 红色
- 配额进度条颜色：< 60% 绿色，60-80% 黄色，> 80% 红色
- 超额部分页面标注"超额部分另计"

### 5.4 导航菜单

在管理员侧边栏 `运维监控` 入口下方新增 `Copilot 分析` 分组：
- Copilot 用户请求（图标：👥）
- Copilot 账户成本（图标：💰）

### 5.5 账户管理页扩展

在现有 `/admin/accounts` 的账户编辑表单中，Copilot 平台账户新增两个字段：
- `套餐类型` 下拉：Free / Pro / Pro+ / Business / Enterprise
- `座席数` 数字输入（仅 Business/Enterprise 显示）

---

## 6. 成本计算逻辑

```
// 套餐配额表（常量）
planQuota = {
  individual_free:    { monthly_cost: 0,   premium_quota: 50   },
  individual_pro:     { monthly_cost: 10,  premium_quota: 300  },
  individual_pro_plus:{ monthly_cost: 39,  premium_quota: 1500 },
  business:           { monthly_cost: 19,  premium_quota: 300  },  // per seat
  enterprise:         { monthly_cost: 39,  premium_quota: 1000 }, // per seat
}

// 计算月费
monthly_cost = planQuota[plan_type].monthly_cost × seat_count

// 计算月配额
monthly_quota = planQuota[plan_type].premium_quota × seat_count

// 单次 Premium 请求成本
cost_per_request = monthly_cost / monthly_quota
// 例：Pro+ = $39 / 1500 = $0.026/次
// 例：Business 3座席 = $57 / 900 = $0.063/次

// 配额使用率（用于告警判断）
usage_rate = premium_used / monthly_quota × 100
```

---

## 7. 实现步骤

### Phase 1：数据层（后端）
1. Ent Schema：`usage_logs` 新增 `initiator` 字段
2. Ent Schema：新增 `CopilotQuotaSnapshot` 实体
3. Ent Schema：新增 `CopilotBudgetAlert` 实体
4. 执行数据库迁移
5. Copilot gateway handler：写日志时记录 `initiator` 值

### Phase 2：后端 API
6. Repository 层：`CopilotQuotaSnapshotRepo`、`CopilotBudgetAlertRepo`
7. Service 层：`CopilotAnalyticsService`（封装统计查询逻辑）
8. Service 层：`CopilotQuotaCacheService`（内存缓存，TTL 5分钟，复用 `FetchAllCopilotQuotas()`）
9. Handler 层：`CopilotAnalyticsHandler`（用户维度 + 账户维度 API）
10. 路由注册：`/api/v1/admin/copilot/...`

### Phase 3：前端
11. API 客户端：新增 Copilot 分析相关接口调用
12. 用户请求分析页：`CopilotUsersView.vue`
13. 账户成本分析页：`CopilotAccountsView.vue`
14. 路由注册 + 侧边栏菜单
15. 账户编辑表单扩展（套餐类型 + 座席数）

### Phase 4：测试与完善
16. 后端单元测试（成本计算、层级关联逻辑）
17. 前端 E2E 测试（关键交互流程）
18. 版本发布

---

## 8. 不在本期范围内

- 超额费用计算（GitHub 未公开超额定价）
- 用户端可见的 Premium 请求消耗（用户只看 token，不看 Premium）
- 告警邮件通知（第一期只写系统日志）
- 跨月历史成本汇总报表

---

## 9. 风险与注意事项

1. **GitHub API 限流**：`FetchQuota()` 调用有频率限制，内存缓存 TTL 5分钟可大幅降低调用频率；`FetchAllCopilotQuotas()` 已实现并发处理，直接复用
2. **缓存冷启动**：服务重启后缓存为空，首次访问会触发全量拉取，耗时较长；前端展示 loading 状态并从 `copilot_quota_snapshots` 读取最后已知状态作为占位数据
3. **历史数据**：`initiator` 字段新增后历史记录默认为 `'user'`，历史统计数据存在偏差，页面应标注"数据从 v0.1.126 起准确"
4. **层级关联**：时间窗口（30秒）为启发式规则，极端情况下可能不准确，前端标注"近似关联"
5. **Free 套餐成本**：月费为 $0，单次成本为 $0，页面显示"免费套餐"而非除以零
6. **外部消耗计算**：`external_used = github_total_used - system_month_premium_requests`，若系统内统计有遗漏（如请求失败未记录）可能出现负值，前端显示为 0
