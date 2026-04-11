# 分销邀请系统设计文档

**日期**: 2026-04-11  
**状态**: 待实现  
**范围**: 2层分销系统，支持注册奖励 + 消费返佣，引入分销员角色，全后台可配置

---

## 1. 背景与目标

### 现状

项目目前有两套机制，但均不满足需求：

- **邀请码（InvitationCode）**：一次性 `redeem_code`，用于"邀请码注册模式"，本质是准入控制，不绑定邀请关系，不发放奖励。
- **注册优惠码（PromoCode）**：管理员批量生成的运营工具，不绑定邀请人，无法追踪邀请关系。

现有角色：`admin`、`user`，无分销员角色。

### 目标

构建一套独立的 2 层分销系统，核心能力：

1. **普通用户推广**：任何用户均可通过专属邀请码邀请他人注册，获得注册奖励（可配）
2. **分销员推广**：具备 `distributor` 角色的用户，拥有独立奖励规则（注册奖励 + 消费返佣各自独立配置），并可查看直接下级的脱敏使用统计
3. **全后台可配置**：普通用户和分销员的所有奖励参数独立配置，支持独立开关

---

## 2. 角色体系

### 2.1 新增角色：`distributor`（分销员）

| 角色 | 值 | 说明 |
|------|----|------|
| 管理员 | `admin` | 现有，不变 |
| 普通用户 | `user` | 现有，不变 |
| **分销员** | `distributor` | **新增**，介于 user 和 admin 之间 |

分销员**不是管理员**，无法访问管理后台。拥有独立的分销员控制台页面。

### 2.2 分销员身份获取方式

**方式 A：管理员直接指定**
- 管理员在用户管理页面将用户角色从 `user` 改为 `distributor`
- 立即生效

**方式 B：用户申请 → 管理员审核**
- 用户在个人中心提交分销员申请（可填写申请原因）
- 管理员在后台审批列表中审核（通过 / 拒绝）
- 通过后角色自动变更为 `distributor`，拒绝则保持 `user`

### 2.3 分销员降级

管理员可将 `distributor` 降回 `user`：
- 已建立的邀请关系保留，历史记录不变
- 降级后的返佣规则切换为普通用户规则（使用记录中快照的比例不变，新发生的消费按普通用户规则？）

> **决策**：降级后已有 `referral_records` 中的 `commission_rate` 快照不变（已锁定），不影响已建立关系的返佣。但新邀请的用户会按降级时的规则快照。

---

## 3. 分销层级模型

```
L1（邀请人，可以是 user 或 distributor）
 └── L2（被邀请人）
      └── L3（L2 邀请的人）
```

**奖励规则**：
- L2 注册 → L1 获注册奖励，L2 获注册奖励（按 L1 角色用对应规则）
- L2 消费 → L1 获消费返佣（按 L1 角色用对应比例）
- L3 注册 → L2 获注册奖励，L3 获注册奖励（按 L2 角色用对应规则）
- L3 消费 → L2 获消费返佣（L1 不参与 L3 消费）

**严格 2 层**：L3 的下级不再向上追溯。

---

## 4. 数据模型

### 4.1 新增表：`referral_codes`

用户专属邀请码，每个用户一条记录，注册时自动生成。

```sql
CREATE TABLE referral_codes (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL UNIQUE,
    code        VARCHAR(32) NOT NULL UNIQUE,  -- 全局唯一，如 "INV-A3K9XZ8M"
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_referral_codes_user_id ON referral_codes (user_id);
CREATE INDEX idx_referral_codes_code    ON referral_codes (code);
```

### 4.2 新增表：`referral_records`

记录邀请关系及奖励明细，每对邀请关系一条记录。

```sql
CREATE TABLE referral_records (
    id          BIGSERIAL PRIMARY KEY,

    -- 邀请关系
    inviter_id        BIGINT NOT NULL,
    invitee_id        BIGINT NOT NULL UNIQUE,   -- 每人只能有一个上级
    inviter_role      VARCHAR(20) NOT NULL,     -- 邀请时的角色快照（user / distributor）
    level             SMALLINT NOT NULL,         -- 1=直接, 2=间接
    status            VARCHAR(20) NOT NULL DEFAULT 'active',  -- active / disabled

    -- 注册奖励快照（注册时锁定，防止后台修改影响历史）
    signup_bonus_inviter  DECIMAL(20,8) NOT NULL DEFAULT 0,
    signup_bonus_invitee  DECIMAL(20,8) NOT NULL DEFAULT 0,
    signup_rewarded_at    TIMESTAMPTZ,  -- null=未发放

    -- 消费返佣快照
    commission_rate       DECIMAL(8,4) NOT NULL DEFAULT 0,   -- 比例（0.05 = 5%）
    total_commission      DECIMAL(20,8) NOT NULL DEFAULT 0,  -- 累计返佣总额

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_referral_records_invitee  ON referral_records (invitee_id);
CREATE INDEX        idx_referral_records_inviter  ON referral_records (inviter_id);
CREATE INDEX        idx_referral_records_status   ON referral_records (status);
```

> **`inviter_role` 快照说明**：记录邀请建立时邀请人的角色，用于明确该关系适用哪套规则。角色快照一旦写入不再更新（降级不影响历史返佣比例）。

### 4.3 新增表：`distributor_applications`

分销员申请记录。

```sql
CREATE TABLE distributor_applications (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending / approved / rejected
    reason       TEXT,                    -- 申请原因（用户填写）
    admin_notes  TEXT,                    -- 审批备注（管理员填写）
    reviewed_by  BIGINT,                  -- 审批管理员 user_id
    reviewed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_distributor_applications_user_id ON distributor_applications (user_id);
CREATE INDEX idx_distributor_applications_status  ON distributor_applications (status);
```

### 4.4 后台配置（扩展现有 `settings` 表）

**普通用户（user）规则**：

| Key | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `referral_enabled` | bool | false | 分销功能总开关（影响所有角色） |
| `referral_user_signup_bonus_enabled` | bool | false | 普通用户注册奖励开关 |
| `referral_user_signup_bonus_inviter` | float | 0 | 普通用户：邀请人获得的注册奖励（元） |
| `referral_user_signup_bonus_invitee` | float | 0 | 普通用户：被邀请人获得的注册奖励（元） |
| `referral_user_commission_enabled` | bool | false | 普通用户消费返佣开关 |
| `referral_user_commission_rate_l1` | float | 0 | 普通用户 L1 对 L2 消费返佣比例 |
| `referral_user_commission_rate_l2` | float | 0 | 普通用户 L2 对 L3 消费返佣比例 |

**分销员（distributor）规则**：

| Key | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `referral_dist_signup_bonus_enabled` | bool | false | 分销员注册奖励开关 |
| `referral_dist_signup_bonus_inviter` | float | 0 | 分销员：邀请人获得的注册奖励（元） |
| `referral_dist_signup_bonus_invitee` | float | 0 | 分销员：被邀请人获得的注册奖励（元） |
| `referral_dist_commission_enabled` | bool | false | 分销员消费返佣开关 |
| `referral_dist_commission_rate_l1` | float | 0 | 分销员 L1 对 L2 消费返佣比例 |
| `referral_dist_commission_rate_l2` | float | 0 | 分销员 L2 对 L3 消费返佣比例 |

---

## 5. 核心流程

### 5.1 专属码生成

用户**注册成功时**自动生成：
1. 生成规则：`INV-` + 8位大写字母数字随机串（如 `INV-A3K9XZ8M`）
2. 冲突重试：最多 5 次
3. 所有角色（user / distributor）均自动生成

### 5.2 注册流程（新增 `referral_code` 参数）

```
前端注册请求（新增 referral_code 字段）
  │
  ├─ URL ?ref=xxx → 自动填入表单（可手动修改）
  └─ 用户手动填写

后端处理：
  1. referral_code 为空 → 仅生成专属码，不建立邀请关系
  2. referral_code 不为空：
     a. 查 referral_codes 找邀请人（L1）
     b. 校验 L1 状态为 active（封禁用户不能成为邀请人）
     c. 校验 L1 != 新用户（防自邀）
     d. 注册成功后（事务内）：
        - 生成新用户专属码
        - 读取 L1 角色 → 按角色读取对应奖励配置
        - 写 referral_records（inviter=L1, invitee=新用户, level=1,
                               inviter_role=L1.role, 快照当前配置）
        - 若 L1 本身有上级 L0（查 referral_records WHERE invitee_id=L1 AND level=1）：
          写 referral_records（inviter=L0, invitee=新用户, level=2,
                               inviter_role=L0.role, 快照当前配置）
        - 若注册奖励开关开启：
          L1 余额 += signup_bonus_inviter
          新用户余额 += signup_bonus_invitee
          更新 signup_rewarded_at
```

### 5.3 消费返佣流程

挂载在 `BillingService.Charge` **之后**（异步执行，不阻塞主流程）：

```
用户 U 被扣费 amount 元
  │
  └─ 查 referral_records WHERE invitee_id=U AND level=1 AND status=active
       │
       ├─ 未找到 → 结束
       └─ 找到（inviter=L1, commission_rate=快照值）：
            │
            └─ 按 inviter_role 判断对应开关是否开启：
                 ├─ 开关关闭 → 结束
                 └─ 开关开启：
                      commission = amount × commission_rate
                      若 commission > 0：
                        L1 余额 += commission
                        total_commission += commission
                        记录返佣日志
```

> 返佣异步执行，失败时记录错误日志，不影响用户正常使用。

### 5.4 分销员申请流程

```
用户提交申请 → 写 distributor_applications（status=pending）
  │
  └─ 管理员在后台审批：
       ├─ 通过 → status=approved, reviewed_by, reviewed_at
       │         users.role 改为 distributor
       └─ 拒绝 → status=rejected, admin_notes 说明原因
```

限制：同一用户只能有一条 `pending` 状态的申请（已有 pending 申请时不允许重复提交）。

### 5.5 封禁联动

用户被封禁时：
1. 其所有 `referral_records`（作为 invitee）`status` 置为 `disabled`
2. 上级停止获得该用户的消费返佣
3. 已发放的奖励**不追回**

---

## 6. 分销员控制台

分销员（`distributor` 角色）拥有独立的控制台页面（`/distributor`），**与管理后台完全隔离**。

### 6.1 可见数据

仅可查看**直接邀请的下级（L2）**的数据，L3 的数据不可见。

**统计概览**（脱敏）：

| 字段 | 说明 |
|------|------|
| 邮箱 | 脱敏显示（`u***@gmail.com`） |
| 注册时间 | 完整显示 |
| 状态 | active / disabled |
| 消费总额 | 累计扣费金额（元） |
| 调用次数 | 累计 API 调用次数 |
| 最近活跃时间 | 最后一次调用时间 |
| 为我贡献返佣 | 来自该下级的累计返佣金额 |

**无法看到**：prompt 内容、具体调用记录、API Key、余额、充值记录。

### 6.2 控制台页面结构

```
/distributor
├── 概览（总邀请人数、累计注册奖励、累计返佣）
├── 我的邀请码（专属码 + 一键复制链接）
├── 下级列表（L2 统计，分页，可按邮箱搜索）
└── 申请记录（查看自己的申请状态）
```

---

## 7. API 设计

### 7.1 用户端 API（所有登录用户）

#### `GET /api/v1/user/referral`
获取当前用户邀请信息（普通用户版，不含下级统计）。

```json
{
  "code": "INV-A3K9XZ8M",
  "invite_url": "https://example.com/register?ref=INV-A3K9XZ8M",
  "stats": {
    "total_invitees": 3,
    "total_signup_bonus": 6.00,
    "total_commission": 0.00
  }
}
```

#### `POST /api/v1/user/referral/apply-distributor`
提交分销员申请。

```json
{ "reason": "我有大量潜在用户资源..." }
```

### 7.2 分销员 API（`distributor` 角色）

#### `GET /api/v1/distributor/overview`
分销员控制台概览统计。

```json
{
  "code": "INV-A3K9XZ8M",
  "invite_url": "https://example.com/register?ref=INV-A3K9XZ8M",
  "total_invitees": 12,
  "total_signup_bonus": 24.00,
  "total_commission": 15.80
}
```

#### `GET /api/v1/distributor/invitees`
直接下级（L2）列表，分页，支持邮箱模糊搜索。

```json
{
  "total": 12,
  "items": [
    {
      "masked_email": "u***@gmail.com",
      "registered_at": "2026-04-10T10:00:00Z",
      "status": "active",
      "total_spend": 8.50,
      "api_call_count": 342,
      "last_active_at": "2026-04-11T08:30:00Z",
      "commission_earned": 0.43
    }
  ]
}
```

### 7.3 管理端 API

#### `GET/PUT /api/v1/admin/referral/settings`
读取/更新分销配置（含普通用户和分销员两套规则）。

#### `GET /api/v1/admin/referral/records`
邀请记录列表，支持按邀请人、被邀请人、层级、状态筛选。

#### `GET /api/v1/admin/referral/stats`
全局统计：总邀请人数、总注册奖励、总返佣。

#### `GET /api/v1/admin/distributor/applications`
分销员申请列表（pending / approved / rejected）。

#### `PUT /api/v1/admin/distributor/applications/:id`
审批申请（approve / reject）。

#### `PUT /api/v1/admin/users/:id/role`
直接修改用户角色（现有接口扩展，支持 `distributor`）。

---

## 8. 前端变更

### 8.1 注册页（`RegisterView.vue`）
- 新增邀请码输入框（可选，与 `invitation_code` 字段并列）
- 页面加载时读取 URL `?ref=` 参数自动填入

### 8.2 用户个人中心
- 新增"邀请"tab：显示专属码、一键复制链接、邀请统计
- 新增"申请分销员"入口（当前角色为 `user` 时显示，`distributor` 时隐藏）
- 申请进度查看

### 8.3 分销员控制台（新页面，独立路由 `/distributor`）
- 路由守卫：仅 `distributor` 角色可访问
- 概览卡片（总邀请、总返佣）
- 专属码 + 复制链接
- 下级列表（脱敏统计，分页）

### 8.4 管理后台
- **分销配置页**：两套规则（普通用户/分销员）各自的开关和金额/比例配置
- **邀请记录页**：完整邀请链，可搜索
- **分销员申请管理页**：待审批列表，支持通过/拒绝并填写备注
- **用户管理页**：角色修改支持 `distributor`

---

## 9. 后端文件变更清单

### 新增
- `backend/ent/schema/referral_code.go`
- `backend/ent/schema/referral_record.go`
- `backend/ent/schema/distributor_application.go`
- `backend/migrations/087_referral_system.sql`
- `backend/internal/service/referral_service.go`
- `backend/internal/repository/referral_repo.go`
- `backend/internal/handler/user/referral_handler.go`
- `backend/internal/handler/distributor/overview_handler.go`
- `backend/internal/handler/admin/referral_handler.go`
- `backend/internal/handler/admin/distributor_application_handler.go`
- `backend/internal/server/middleware/distributor_only.go`

### 修改
- `backend/internal/domain/constants.go` — 新增 `RoleDistributor = "distributor"`
- `backend/internal/service/domain_constants.go` — 同步新角色常量
- `backend/ent/schema/user.go` — 新增 referral 相关 edges
- `backend/internal/service/auth_service.go` — 注册时生成专属码、写邀请关系、发注册奖励
- `backend/internal/service/billing_service.go` — 扣款后异步触发消费返佣
- `backend/internal/service/setting_service.go` — 新增分销配置项（两套规则）
- `backend/internal/service/admin_service.go` — 角色修改支持 distributor、封禁时联动
- `backend/internal/setup/handler.go` — 注册新路由

## 10. 前端文件变更清单

### 新增
- `frontend/src/views/user/ReferralView.vue`
- `frontend/src/views/distributor/DistributorLayout.vue`
- `frontend/src/views/distributor/OverviewView.vue`
- `frontend/src/views/distributor/InviteesView.vue`
- `frontend/src/views/admin/ReferralSettingsView.vue`
- `frontend/src/views/admin/ReferralRecordsView.vue`
- `frontend/src/views/admin/DistributorApplicationsView.vue`
- `frontend/src/api/user/referral.ts`
- `frontend/src/api/distributor/index.ts`
- `frontend/src/api/admin/referral.ts`

### 修改
- `frontend/src/views/auth/RegisterView.vue` — 新增邀请码输入框
- `frontend/src/router/index.ts` — 新增路由和分销员路由守卫
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

---

## 11. 边界条件与防护

| 场景 | 处理方式 |
|------|---------|
| 用户填写自己的邀请码 | 后端拒绝（inviter == invitee） |
| 邀请码不存在 | 静默忽略，正常注册，无邀请关系 |
| 邀请人已被封禁 | 静默忽略，不建立邀请关系 |
| 注册奖励发放失败 | 不影响注册成功，记录错误日志 |
| 返佣计算为 0 | 跳过，不写日志 |
| 并发注册同一邀请码 | `invitee_id` 唯一约束保证幂等 |
| 重复提交分销员申请 | 有 pending 申请时拒绝新提交 |
| 分销员被降级后的返佣 | 历史 `commission_rate` 快照不变，不受影响 |
| 分销员查看非直属下级 | API 严格过滤 level=1，后端强制执行 |

---

## 12. 不在本期范围内

- 提现/结算（返佣仅入余额）
- 3层及以上分销
- 分销员手动管理下级（如手动禁用某个下级的返佣）
- 返佣独立对账报表
- 防刷机制（IP 限制、手机号验证等）
- 分销员查看 L3 数据
