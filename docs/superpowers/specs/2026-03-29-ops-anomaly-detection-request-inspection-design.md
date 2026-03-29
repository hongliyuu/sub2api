# 运维异常检测与请求排查增强设计

**日期：** 2026-03-29
**状态：** 待实现
**功能域：** 管理后台 / 运维工具

---

## 一、背景与目标

### 问题

1. **使用记录异常不可见**：当前使用记录列表所有行样式一致，管理员无法快速识别 token 全为 0 的无效请求、耗时过长的慢请求或超时请求，只能逐行查看。

2. **请求排查信息不足**：当前请求排查页面缺少发起者身份信息（用户、API Key、分组、上游账户），无法追溯问题根源；对于异常请求也没有原始请求/响应 body 数据，排查极为困难。

### 目标

- 异常行在列表中通过颜色立即可见，支持多选筛选异常类型
- 管理员可自定义异常阈值（慢请求/超时两级阈值）
- 请求详情面板展示完整身份链路：用户 → API Key → 分组 → 上游账户
- 异常请求保存完整原始数据（用户请求 body、上游请求 body、上游响应 body），方便排查

---

## 二、功能范围

### 功能1：异常标记与筛选

1. **行级异常标记**（A+B 混合样式）：
   - 行背景轻微染色 + 左侧彩色边框（行级感知）
   - 异常字段本身变色 + 独立"异常"标签列（字段级精准）
   - 红色 = 严重（token 全为 0 或耗时 > 超时阈值）
   - 黄色 = 警告（耗时 > 慢请求阈值但 ≤ 超时阈值）
   - 多异常叠加：一行可同时显示多个标签（如"超时 + 零Token"）

2. **多选异常筛选**：
   - 在现有筛选栏新增异常类型多选区
   - 选项：零Token / 慢请求 / 超时 / 错误请求
   - 多选时取"或"逻辑（命中任意一种即显示）

3. **管理员可配置阈值**：
   - 慢请求警告阈值（秒，默认 20s）
   - 超时严重阈值（秒，默认 60s）
   - 零Token检测开关
   - 保存异常原始数据开关
   - 配置持久化到 `settings` 表（key-value）

### 功能2：请求排查增强

1. **列表新增身份列**（可按列排序/筛选）：
   - 用户名（user display name）
   - API Key（末4位脱敏显示）
   - 分组名（group name）
   - 上游账户名（account name）

2. **详情面板扩展**：
   - 新增"身份信息"区块：用户/Key/分组/平台/账户
   - 新增"原始数据"区块（手风琴展开）：
     - 用户原始请求 body（`request_body`）
     - 上游请求 body（`upstream_request_body`）
     - 上游响应 body（`upstream_response_body`）
   - 原始数据仅在异常请求时存在

3. **异常原始数据存储**（仅异常请求）：
   - 新建 `request_logs` 表
   - 在请求被判定为异常时（网关处理完成后）异步写入
   - 存储完整 body（不截断），记录触发的异常类型
   - 不影响正常请求链路性能

---

## 三、数据库设计

### 3.1 新表：`request_logs`

```sql
-- Migration: 080_add_request_logs.sql
CREATE TABLE request_logs (
    id                    BIGSERIAL PRIMARY KEY,
    request_id            VARCHAR(64)  NOT NULL,
    usage_log_id          BIGINT       REFERENCES usage_logs(id),

    -- 身份关联（冗余存储，便于独立查询）
    user_id               BIGINT,
    api_key_id            BIGINT,
    account_id            BIGINT,
    group_id              BIGINT,

    -- 异常原因（bitmask 或枚举）
    anomaly_types         TEXT[]       NOT NULL DEFAULT '{}',
    -- 可取值: 'zero_token', 'slow_request', 'timeout', 'error'

    -- 原始数据
    request_body          JSONB,        -- 用户原始请求 body
    upstream_request_body JSONB,        -- 发往上游的请求 body
    upstream_response_body JSONB,       -- 上游响应 body（超时时为错误描述）

    -- 时间
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_logs_request_id ON request_logs(request_id);
CREATE INDEX idx_request_logs_created_at ON request_logs(created_at DESC);
CREATE INDEX idx_request_logs_user_id    ON request_logs(user_id);
CREATE INDEX idx_request_logs_api_key_id ON request_logs(api_key_id);
```

### 3.2 新增系统设置键

向现有 `settings` 表写入以下 key（若不存在则初始化默认值）：

| key | 默认值 | 说明 |
|-----|--------|------|
| `ops.anomaly.slow_request_threshold_ms` | `20000` | 慢请求警告阈值（毫秒） |
| `ops.anomaly.timeout_threshold_ms` | `60000` | 超时严重阈值（毫秒） |
| `ops.anomaly.detect_zero_token` | `true` | 是否检测零token |
| `ops.anomaly.save_raw_data` | `true` | 是否保存异常请求原始数据 |

---

## 四、后端设计

### 4.1 异常判定逻辑

异常判定在请求完成后执行（不在请求路径上，不影响响应延迟）：

```
func detectAnomalies(req *RequestContext, settings *AnomalySettings) []AnomalyType {
    var types []AnomalyType

    if settings.DetectZeroToken &&
       req.InputTokens == 0 && req.OutputTokens == 0 {
        types = append(types, AnomalyZeroToken)
    }

    if req.DurationMs > settings.TimeoutThresholdMs {
        types = append(types, AnomalyTimeout)
    } else if req.DurationMs > settings.SlowRequestThresholdMs {
        types = append(types, AnomalySlowRequest)
    }

    if req.StatusCode >= 500 {
        types = append(types, AnomalyError)
    }

    return types
}
```

### 4.2 原始数据写入时机

在 `CopilotGatewayHandler`（及 Sora/OpenAI 网关）的请求处理完成后，异步写入：

```
// 伪代码：在响应写完之后异步执行
go func() {
    anomalies := detectAnomalies(ctx, settings)
    if len(anomalies) > 0 && settings.SaveRawData {
        requestLogService.Save(&RequestLog{
            RequestID:            ctx.RequestID,
            UsageLogID:           ctx.UsageLogID,
            UserID:               ctx.UserID,
            APIKeyID:             ctx.APIKeyID,
            AccountID:            ctx.AccountID,
            GroupID:              ctx.GroupID,
            AnomalyTypes:         anomalies,
            RequestBody:          ctx.InboundRequestBody,
            UpstreamRequestBody:  ctx.UpstreamRequestBody,
            UpstreamResponseBody: ctx.UpstreamResponseBody,
        })
    }
}()
```

**关键约束：**
- 异步写入，goroutine 不阻塞响应
- 只有在 `save_raw_data` 开关打开时才写入 body 数据
- 阈值从内存缓存读取（TTL 30s），避免每次请求查数据库

### 4.3 新增 / 修改的 API 端点

#### 4.3.1 `GET /api/v1/admin/ops/requests`（已有，扩展参数）

新增 query 参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `anomaly_types` | `[]string` | 多选异常类型过滤（zero_token/slow_request/timeout/error） |
| `min_duration_ms` | `int` | 已有 |
| `max_duration_ms` | `int` | 已有 |

响应的 `OpsRequestDetail` 新增字段：

```go
type OpsRequestDetail struct {
    // ... 已有字段 ...

    // 新增
    UserName    *string `json:"user_name"`    // 用户显示名
    APIKeyLabel *string `json:"api_key_label"` // Key 标签（末4位）
    GroupName   *string `json:"group_name"`    // 分组名
    AccountName *string `json:"account_name"` // 上游账户名
    AnomalyTypes []string `json:"anomaly_types"` // 异常类型列表
}
```

#### 4.3.2 `GET /api/v1/admin/ops/usage-inspect`（已有，扩展响应）

响应的 `OpsUsageInspectDetail` 新增字段：

```go
type OpsUsageInspectDetail struct {
    // ... 已有字段（包含 account_name, group_name）...

    // 新增
    UserName             *string          `json:"user_name"`
    APIKeyLabel          *string          `json:"api_key_label"`
    AnomalyTypes         []string         `json:"anomaly_types"`
    RequestBody          *json.RawMessage `json:"request_body,omitempty"`
    UpstreamRequestBody  *json.RawMessage `json:"upstream_request_body,omitempty"`
    UpstreamResponseBody *json.RawMessage `json:"upstream_response_body,omitempty"`
}
```

#### 4.3.3 `GET /api/v1/admin/ops/settings/anomaly`（新增）

读取异常检测配置：

```go
type AnomalySettings struct {
    SlowRequestThresholdMs int64 `json:"slow_request_threshold_ms"`
    TimeoutThresholdMs     int64 `json:"timeout_threshold_ms"`
    DetectZeroToken        bool  `json:"detect_zero_token"`
    SaveRawData            bool  `json:"save_raw_data"`
}
```

#### 4.3.4 `PUT /api/v1/admin/ops/settings/anomaly`（新增）

更新配置，写入 `settings` 表，清除内存缓存。

### 4.4 数据库查询扩展

**`ops_repo_request_details.go`** 的查询需要：
1. 新增与 `users` 和 `api_keys` 的 LEFT JOIN（已有 accounts, groups 的 JOIN）
2. 新增 `anomaly_types` 过滤条件：通过 duration_ms 范围和 token 为 0 的 WHERE 条件在查询层面实现（不依赖 request_logs 表），避免额外 JOIN

**`ops_repo_usage_inspect.go`** 的查询需要：
1. 在已有 30 列基础上新增 users.display_name 和 api_keys 末4位 LEFT JOIN
2. 查询完成后，异步查询 `request_logs` 表获取原始数据（以 request_id 为 key）

### 4.5 设置服务（SettingsService）

新增或扩展现有设置服务：

```go
type AnomalySettingsCache struct {
    mu       sync.RWMutex
    settings *AnomalySettings
    expireAt time.Time
}

// GetAnomalySettings 带 30s TTL 内存缓存
func (s *SettingsService) GetAnomalySettings(ctx context.Context) (*AnomalySettings, error)

// UpdateAnomalySettings 更新并清除缓存
func (s *SettingsService) UpdateAnomalySettings(ctx context.Context, settings *AnomalySettings) error
```

---

## 五、前端设计

### 5.1 OpsRequestInspectView.vue 改动

**筛选栏新增**（在现有 kind/platform/search 之后）：

```vue
<!-- 异常类型多选筛选 -->
<div class="anomaly-filter">
  <AnomalyFilterChips
    v-model="filters.anomalyTypes"
    :options="anomalyTypeOptions"
  />
</div>
```

`anomalyTypeOptions` 定义：
```ts
const anomalyTypeOptions = [
  { value: 'zero_token',    label: '零Token',  severity: 'critical' },
  { value: 'slow_request',  label: '慢请求',   severity: 'warning'  },
  { value: 'timeout',       label: '超时',     severity: 'critical' },
  { value: 'error',         label: '错误请求', severity: 'critical' },
]
```

**表格列新增**（在现有列之后，可折叠）：

| 新增列 | 字段 | 样式 |
|--------|------|------|
| 用户 | `user_name` | badge 蓝色 |
| Key | `api_key_label` | monospace 灰色 |
| 分组 | `group_name` | badge 灰色 |
| 上游账户 | `account_name` | 紫色文字 |
| 异常 | `anomaly_types` | badge 列表（红/黄） |

**行样式计算**（computed）：

```ts
function getRowClass(row: OpsRequestDetail): string {
  const types = row.anomaly_types ?? []
  if (types.includes('zero_token') || types.includes('timeout') || types.includes('error')) {
    return 'row-critical'
  }
  if (types.includes('slow_request')) {
    return 'row-warning'
  }
  return 'row-normal'
}
```

**阈值配置入口**：在筛选栏右侧添加"⚙ 配置阈值"按钮，点击弹出 `AnomalySettingsModal`。

### 5.2 OpsRequestDetailPanel.vue 改动

在现有详情面板基础上新增两个区块：

**区块1：身份信息**（插入在 request_id 和 metadata 之间）

```vue
<DetailSection title="身份信息" v-if="usageDetail">
  <DetailRow label="用户" :value="usageDetail.user_name" type="user" />
  <DetailRow label="API Key" :value="usageDetail.api_key_label" type="monospace" />
  <DetailRow label="分组" :value="usageDetail.group_name" type="badge" />
  <DetailRow label="上游平台" :value="usageDetail.platform" />
  <DetailRow label="上游账户" :value="usageDetail.account_name" type="accent" />
</DetailSection>
```

**区块2：原始数据**（在详情底部，仅当 anomaly_types 非空时显示）

```vue
<DetailSection
  title="原始数据"
  v-if="usageDetail?.anomaly_types?.length > 0"
  class="section-danger"
>
  <RawDataAccordion
    label="用户原始请求"
    :data="usageDetail.request_body"
  />
  <RawDataAccordion
    label="上游请求"
    :data="usageDetail.upstream_request_body"
  />
  <RawDataAccordion
    label="上游响应"
    :data="usageDetail.upstream_response_body"
    :error="isTimeout(usageDetail)"
  />
</DetailSection>
```

### 5.3 新增组件

| 组件 | 职责 |
|------|------|
| `AnomalyFilterChips.vue` | 多选异常类型筛选 chips |
| `AnomalyBadge.vue` | 单个异常类型标签（zero_token/slow/timeout/error） |
| `AnomalySettingsModal.vue` | 阈值配置弹窗 |
| `RawDataAccordion.vue` | 手风琴式 JSON 数据展示 |

### 5.4 新增 API 客户端函数（ops.ts）

```ts
// 获取异常配置
export function getAnomalySettings(): Promise<AnomalySettings>

// 更新异常配置
export function updateAnomalySettings(settings: AnomalySettings): Promise<void>
```

新增类型：

```ts
export type AnomalyType = 'zero_token' | 'slow_request' | 'timeout' | 'error'

export interface AnomalySettings {
  slow_request_threshold_ms: number
  timeout_threshold_ms:      number
  detect_zero_token:         boolean
  save_raw_data:             boolean
}

// OpsRequestDetail 新增字段
export interface OpsRequestDetail {
  // ...已有...
  user_name?:     string
  api_key_label?: string
  group_name?:    string
  account_name?:  string
  anomaly_types?: AnomalyType[]
}

// OpsUsageInspectDetail 新增字段
export interface OpsUsageInspectDetail {
  // ...已有...
  user_name?:              string
  api_key_label?:          string
  anomaly_types?:          AnomalyType[]
  request_body?:           unknown
  upstream_request_body?:  unknown
  upstream_response_body?: unknown
}
```

---

## 六、错误处理与边界情况

### 6.1 原始数据存储失败

- 写入 `request_logs` 失败不影响主请求链路（goroutine 独立）
- 写入失败记录日志，不重试（append-only 语义，异常已发生，数据仅供参考）
- 若 `save_raw_data` 为 false，body 字段返回 null，前端不显示该区块

### 6.2 body 数据过大

- JSONB 列无固定大小限制，PostgreSQL 支持大 JSONB
- 若 body 超过 1MB，截断并附加 `{"_truncated": true, "_original_size": N}` 字段
- 截断逻辑在写入前执行

### 6.3 阈值配置读取失败

- 若数据库读取失败，使用内存中最后一次有效缓存
- 若缓存也不存在，使用硬编码默认值（20000ms / 60000ms）
- 降级逻辑对用户透明

### 6.4 异常类型前端计算 vs 后端存储

- **列表查询**：后端在查询层根据 duration_ms 和 token 字段动态计算异常类型（不依赖 request_logs），保证全量数据都能筛选
- **详情面板**：通过 request_logs 表获取原始数据（可能不存在，如开关关闭期间的请求）
- 两者解耦：`anomaly_types` 字段由后端列表查询动态产出，`request_body` 系列字段从 request_logs 读取

### 6.5 并发写入

- `request_logs` 使用 BIGSERIAL 主键，并发写入安全
- 同一 request_id 可能有多条（理论上不应出现，但不约束 UNIQUE，避免写入失败）

---

## 七、实现阶段

### Phase 1：数据库与后端基础（2天）
1. 迁移文件 `080_add_request_logs.sql`
2. ent schema for `RequestLog`
3. `SettingsService` 异常配置读写（带缓存）
4. `RequestLogService` 异步写入逻辑
5. 网关 handler 注入检测逻辑（Copilot + Sora + OpenAI 三个网关）
6. 异常配置 API 端点（GET/PUT /settings/anomaly）

### Phase 2：列表 API 扩展（1天）
1. `ops_repo_request_details.go` 新增 users/api_keys JOIN 和身份字段
2. `ops_handler.go` 新增 `anomaly_types` 过滤参数
3. 列表 API 响应中动态计算并附加 `anomaly_types`

### Phase 3：详情 API 扩展（1天）
1. `ops_repo_usage_inspect.go` 新增 user_name / api_key_label JOIN
2. 从 `request_logs` 读取原始数据，合并入 `OpsUsageInspectDetail`

### Phase 4：前端实现（2天）
1. 新增 `AnomalyFilterChips`、`AnomalyBadge`、`AnomalySettingsModal`、`RawDataAccordion` 组件
2. `OpsRequestInspectView.vue` 改造（筛选栏、表格列、行样式）
3. `OpsRequestDetailPanel.vue` 改造（身份区块、原始数据区块）
4. `ops.ts` 新增类型和 API 函数

---

## 八、不在本次范围内

- 原始数据的保留策略（按时间清理）— 可后续补充定时任务
- 异常数量统计图表（可后续加入 OpsDashboard）
- 邮件/Webhook 告警通知
- 用户侧可见的异常统计

---

## 九、文件变更清单

### 新增文件

| 文件 | 说明 |
|------|------|
| `backend/migrations/080_add_request_logs.sql` | request_logs 表 |
| `backend/ent/schema/request_log.go` | ent schema |
| `backend/internal/repository/request_log_repo.go` | 数据访问层 |
| `backend/internal/service/request_log_service.go` | 写入逻辑 |
| `backend/internal/service/anomaly_settings_service.go` | 配置缓存服务 |
| `frontend/src/views/admin/ops/components/AnomalyFilterChips.vue` | 筛选 chips |
| `frontend/src/views/admin/ops/components/AnomalyBadge.vue` | 标签组件 |
| `frontend/src/views/admin/ops/components/AnomalySettingsModal.vue` | 配置弹窗 |
| `frontend/src/views/admin/ops/components/RawDataAccordion.vue` | 原始数据展示 |

### 修改文件

| 文件 | 改动 |
|------|------|
| `backend/internal/handler/admin/ops_handler.go` | 新增 anomaly settings 端点，扩展 list 参数 |
| `backend/internal/handler/copilot_gateway_handler.go` | 注入异常检测逻辑 |
| `backend/internal/handler/sora_gateway_handler.go` | 注入异常检测逻辑 |
| `backend/internal/repository/ops_repo_request_details.go` | 新增 JOIN，anomaly 过滤 |
| `backend/internal/repository/ops_repo_usage_inspect.go` | 新增身份字段 JOIN，原始数据合并 |
| `backend/cmd/server/wire_gen.go` | 注入新服务依赖 |
| `frontend/src/views/admin/ops/OpsRequestInspectView.vue` | 筛选栏、表格、行样式 |
| `frontend/src/views/admin/ops/components/OpsRequestDetailPanel.vue` | 身份区块、原始数据区块 |
| `frontend/src/api/admin/ops.ts` | 新增类型和函数 |
