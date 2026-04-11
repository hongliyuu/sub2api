# 分销邀请系统设计文档

**日期**: 2026-04-11  
**状态**: 待实现  
**范围**: 2层分销系统，支持注册奖励 + 消费返佣，全后台可配置

---

## 1. 背景与目标

### 现状

项目目前有两套机制，但均不满足需求：

- **邀请码（InvitationCode）**：一次性 `redeem_code`，用于"邀请码注册模式"，本质是准入控制，不绑定邀请关系，不发放奖励。
- **注册优惠码（PromoCode）**：管理员批量生成的运营工具，不绑定邀请人，无法追踪邀请关系。

### 目标

构建一套独立的 2 层分销系统，允许用户生成专属邀请码，通过邀请朋友注册获得：
- **注册奖励**：朋友注册成功后，邀请人和被邀请人各自获得可配置的固定余额奖励
- **消费返佣**：朋友每次消费扣费后，邀请人获得可配置比例的余额返佣

所有奖励参数均可在管理后台独立配置和开关。

---

## 2. 分销层级模型

```
L1（邀请人）
 └── L2（被邀请人，直接下级）
      └── L3（L2 邀请的人，间接下级）
```

**奖励规则**：
- L2 注册 → L1 获注册奖励（inviter bonus），L2 获注册奖励（invitee bonus）
- L2 消费 → L1 获消费返佣（commission_rate_l1 比例）
- L3 注册 → L2 获注册奖励，L3 获注册奖励（L1 不参与）
- L3 消费 → L2 获消费返佣（commission_rate_l2 比例）（L1 不参与 L3 消费）

**严格 2 层**：L3 的下级注册/消费不再向上追溯。

---

## 3. 数据模型

### 3.1 新增表：`referral_codes`

用户专属邀请码，每个用户一条记录，注册时自动生成。

```sql
CREATE TABLE referral_codes (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL UNIQUE,
    code        VARCHAR(32) NOT NULL UNIQUE,  -- 全局唯一，如 "INV-A3K9X"
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_referral_codes_user_id ON referral_codes (user_id);
CREATE INDEX idx_referral_codes_code ON referral_codes (code);
```

### 3.2 新增表：`referral_records`

记录邀请关系及奖励明细，每对邀请关系一条记录。

```sql
CREATE TABLE referral_records (
    id          BIGSERIAL PRIMARY KEY,

    -- 邀请关系
    inviter_id  BIGINT NOT NULL,               -- 邀请人 user_id
    invitee_id  BIGINT NOT NULL UNIQUE,        -- 被邀请人 user_id（每人只能被邀请一次）
    level       SMALLINT NOT NULL,             -- 1=直接邀请(L1→L2), 2=间接邀请(L2→L3视角下L1→L3)
    status      VARCHAR(20) NOT NULL DEFAULT 'active',  -- active / disabled

    -- 注册奖励（注册时快照配置值，防止后续修改配置影响历史记录）
    signup_bonus_inviter  DECIMAL(20,8) NOT NULL DEFAULT 0,  -- 邀请人获得的注册奖励
    signup_bonus_invitee  DECIMAL(20,8) NOT NULL DEFAULT 0,  -- 被邀请人获得的注册奖励
    signup_rewarded_at    TIMESTAMPTZ,                        -- null=未发放

    -- 消费返佣
    commission_rate       DECIMAL(8,4) NOT NULL DEFAULT 0,   -- 返佣比例快照（如 0.05 = 5%）
    total_commission      DECIMAL(20,8) NOT NULL DEFAULT 0,  -- 累计返佣总额

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_referral_records_invitee ON referral_records (invitee_id);
CREATE INDEX idx_referral_records_inviter ON referral_records (inviter_id);
CREATE INDEX idx_referral_records_status ON referral_records (status);
```

> **设计说明**：
> - `invitee_id` 唯一约束确保每个用户只能有一个上级，邀请关系不可更改。
> - `signup_bonus_*` 和 `commission_rate` 在注册时快照，防止后台修改配置影响已有关系。
> - 2层关系通过 `level` 字段区分：level=1 表示直接邀请，用于消费返佣追踪。

### 3.3 后台配置（扩展现有 `settings` 表）

新增以下配置项（key-value 形式）：

| Key | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `referral_enabled` | bool | false | 分销功能总开关 |
| `referral_signup_bonus_enabled` | bool | false | 注册奖励开关 |
| `referral_signup_bonus_inviter` | float | 0 | 邀请人注册奖励金额（元） |
| `referral_signup_bonus_invitee` | float | 0 | 被邀请人注册奖励金额（元） |
| `referral_commission_enabled` | bool | false | 消费返佣开关 |
| `referral_commission_rate_l1` | float | 0 | L1 对 L2 消费的返佣比例（0.05 = 5%） |
| `referral_commission_rate_l2` | float | 0 | L2 对 L3 消费的返佣比例（0.05 = 5%） |

---

## 4. 核心流程

### 4.1 专属码生成

用户**注册成功时**自动生成专属邀请码，无需手动申请：
1. 生成规则：`INV-` 前缀 + 8位大写字母数字随机串（如 `INV-A3K9XZ8M`）
2. 冲突重试：生成后检查唯一性，冲突则重新生成（最多 5 次）
3. 写入 `referral_codes` 表

### 4.2 注册流程（新增邀请码参数）

```
前端注册请求（新增 referral_code 字段）
  │
  ├─ URL 参数 ?ref=xxx → 自动填入表单
  └─ 用户手动填写
  
后端 RegisterWithVerification 新增逻辑：
  1. 若 referral_code 不为空：
     a. 查 referral_codes 表，找到邀请人（L1）
     b. 检查邀请人状态为 active（封禁用户无法成为邀请人）
     c. 注册成功后：
        - 为新用户生成专属邀请码
        - 写入 referral_records（inviter=L1, invitee=新用户, level=1）
        - 若 L1 本身也有上级（查 referral_records 找 L1 的 inviter），
          再写一条（inviter=L1的上级, invitee=新用户, level=2）
        - 发放注册奖励（若 referral_signup_bonus_enabled=true）：
          L1 余额 += signup_bonus_inviter
          新用户余额 += signup_bonus_invitee
          更新 signup_rewarded_at
  2. 若 referral_code 为空：仅生成专属码，不写邀请关系
```

### 4.3 消费返佣流程

挂载在现有计费扣款（`BillingService.Charge`）**之后**：

```
用户 U 被扣费 amount 元
  │
  └─ 查 referral_records WHERE invitee_id=U AND level=1 AND status=active
       │
       ├─ 未找到：不处理
       └─ 找到记录（inviter=L1）：
            │
            ├─ 若 referral_commission_enabled=true：
            │    commission = amount × commission_rate（快照值）
            │    L1 余额 += commission
            │    referral_records.total_commission += commission
            │    记录返佣日志
            └─ 结束
```

> **注意**：返佣使用记录创建时快照的 `commission_rate`，不受后台修改配置影响。

### 4.4 封禁联动

用户被管理员封禁时：
1. 将其所有 `referral_records`（作为 invitee）的 `status` 置为 `disabled`
2. 该用户的上级停止获得其消费返佣
3. 已发放的奖励和返佣**不追回**

---

## 5. API 设计

### 5.1 用户端 API

#### `GET /api/v1/user/referral`
获取当前用户的邀请信息。

**响应**：
```json
{
  "code": "INV-A3K9XZ8M",
  "invite_url": "https://example.com/register?ref=INV-A3K9XZ8M",
  "stats": {
    "total_invitees": 5,
    "total_signup_bonus": 10.00,
    "total_commission": 3.50
  },
  "records": [
    {
      "invitee_email": "u***@gmail.com",
      "created_at": "2026-04-10T10:00:00Z",
      "status": "active",
      "signup_bonus_inviter": 2.00,
      "total_commission": 1.20
    }
  ]
}
```

### 5.2 管理端 API

#### `GET /api/v1/admin/referral/settings`
获取分销配置。

#### `PUT /api/v1/admin/referral/settings`
更新分销配置。

**请求体**：
```json
{
  "referral_enabled": true,
  "referral_signup_bonus_enabled": true,
  "referral_signup_bonus_inviter": 2.00,
  "referral_signup_bonus_invitee": 1.00,
  "referral_commission_enabled": true,
  "referral_commission_rate_l1": 0.05,
  "referral_commission_rate_l2": 0.02
}
```

#### `GET /api/v1/admin/referral/records`
邀请记录列表，支持分页和筛选。

**查询参数**：`page`, `page_size`, `inviter_id`, `invitee_id`, `status`, `level`

#### `GET /api/v1/admin/referral/stats`
全局分销统计：总邀请人数、总注册奖励发放、总返佣发放。

---

## 6. 前端变更

### 6.1 注册页（`RegisterView.vue`）
- 新增邀请码输入框（可选）
- 页面加载时读取 URL `?ref=` 参数自动填入
- 与现有邀请码（`invitation_code`）字段并列，样式保持一致

### 6.2 用户个人中心（新增邀请模块）
在 `views/user/` 下新增邀请页面或在现有页面新增 tab：
- **我的邀请码**：显示专属码 + 一键复制链接按钮
- **邀请统计**：已邀请人数 / 累计注册奖励 / 累计返佣
- **邀请记录**：分页列表，含被邀请人（邮箱脱敏）、状态、注册时间、奖励金额

### 6.3 管理后台
- **分销配置页**：所有配置项的表单，含总开关、各子功能开关、金额/比例输入
- **邀请记录管理页**：可按邀请人/被邀请人搜索，查看完整邀请链

---

## 7. Ent Schema 变更

新增两个 Schema：
- `backend/ent/schema/referral_code.go`
- `backend/ent/schema/referral_record.go`

`User` Schema 新增 edges：
```go
edge.To("referral_code", ReferralCode.Type),      // 用户的专属码
edge.To("referral_invitees", ReferralRecord.Type), // 作为邀请人的记录
edge.From("referral_inviter", ReferralRecord.Type), // 作为被邀请人的记录
```

---

## 8. 文件变更清单

### 后端新增
- `backend/ent/schema/referral_code.go`
- `backend/ent/schema/referral_record.go`
- `backend/migrations/087_referral_system.sql`
- `backend/internal/service/referral_service.go`
- `backend/internal/repository/referral_repo.go`
- `backend/internal/handler/user/referral_handler.go`
- `backend/internal/handler/admin/referral_handler.go`

### 后端修改
- `backend/ent/schema/user.go` — 新增 edges
- `backend/internal/service/auth_service.go` — 注册时生成专属码、写邀请关系、发注册奖励
- `backend/internal/service/billing_service.go` — 扣款后触发消费返佣
- `backend/internal/service/setting_service.go` — 新增分销配置项
- `backend/internal/setup/handler.go` — 注册新路由

### 前端新增
- `frontend/src/views/user/ReferralView.vue`
- `frontend/src/views/admin/ReferralSettingsView.vue`
- `frontend/src/views/admin/ReferralRecordsView.vue`
- `frontend/src/api/user/referral.ts`
- `frontend/src/api/admin/referral.ts`

### 前端修改
- `frontend/src/views/auth/RegisterView.vue` — 新增邀请码输入框
- `frontend/src/router/index.ts` — 新增路由
- `frontend/src/i18n/locales/zh.ts` — 新增中文翻译
- `frontend/src/i18n/locales/en.ts` — 新增英文翻译

---

## 9. 边界条件与防护

| 场景 | 处理方式 |
|------|---------|
| 用户填写自己的邀请码 | 后端校验 inviter != invitee，拒绝 |
| 邀请码不存在 | 静默忽略（不报错，正常注册，无邀请关系） |
| 邀请人已被封禁 | 静默忽略，不建立邀请关系 |
| 注册奖励发放失败 | 不影响注册成功，记录错误日志，可后续补发 |
| 返佣计算结果为 0 | 跳过，不写日志 |
| 并发注册相同邀请码 | `invitee_id` 唯一约束保证幂等 |

---

## 10. 不在本期范围内

- 提现/结算功能（返佣仅入余额，不支持提现）
- 3层及以上分销
- 邀请人手动禁用某个下级的返佣
- 返佣记录的独立对账报表
- 防刷机制（IP 限制、手机号验证等）
