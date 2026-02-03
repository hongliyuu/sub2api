# Story 1.2: 获取邀请链接和二维码

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 普通用户,
I want 获取我的专属邀请链接和二维码图片,
so that 分享给朋友进行推广。

## Acceptance Criteria

1. **AC1**: 已登录用户调用 `GET /api/v1/affiliate/info` 返回完整推广信息
2. **AC2**: 响应包含 referral_code、referral_link（格式 `{site_url}/r/{code}`）、qrcode_url
3. **AC3**: 二维码基于邀请链接动态生成，返回 base64 Data URL 或服务端生成的图片 URL
4. **AC4**: 响应包含用户分销状态信息：tier_level、tier_name、commission_rate、effective_count、total_earnings、withdrawable
5. **AC5**: 响应包含 can_withdraw 字段和 withdraw_threshold（提现门槛）
6. **AC6**: 未创建分销信息的用户返回 404 错误
7. **AC7**: 二维码图片支持下载（前端功能）

## Dependencies

- **Depends On**: Story 1.1 (user_affiliate 表和 AffiliateService 基础设施)
- **Depended By**: Story 1.3 (海报生成依赖 qrcode_data_url), Story 1.4 (邀请链接格式), Story 3.1 (推广中心首页)

## Tasks / Subtasks

- [ ] Task 1: 添加 site_url 配置 (AC: #2)
  - [ ] 1.1 在 `config.go` 的 Config struct 中添加 `SiteURL` 字段
  - [ ] 1.2 在 `config.yaml` 中添加 `site_url` 配置项
- [ ] Task 2: 实现 AffiliateService 的 GetAffiliateInfo 方法 (AC: #1, #2, #4, #5, #6)
  - [ ] 2.1 在 `affiliate_service.go` 中添加 `GetAffiliateInfo(ctx, userID)` 方法
  - [ ] 2.2 拼装 referral_link = `{site_url}/r/{referral_code}`
  - [ ] 2.3 根据 tier_level 返回 tier_name 和 commission_rate（硬编码阶梯规则）
  - [ ] 2.4 计算 can_withdraw（withdrawable >= threshold）
  - [ ] 2.5 计算 next_tier_threshold（下一档位所需邀请数）
- [ ] Task 3: 实现二维码生成 (AC: #3)
  - [ ] 3.1 添加 `github.com/skip2/go-qrcode` 依赖
  - [ ] 3.2 在 `affiliate_service.go` 中实现 `GenerateQRCode(link)` 方法，返回 base64 Data URL
  - [ ] 3.3 二维码尺寸 256x256，RecoveryLevel = Medium
- [ ] Task 4: 创建 AffiliateHandler (AC: #1, #6)
  - [ ] 4.1 创建 `internal/handler/affiliate_handler.go`
  - [ ] 4.2 实现 `GetAffiliateInfo(c *gin.Context)` 处理方法
  - [ ] 4.3 定义响应 DTO 结构体
- [ ] Task 5: 注册路由和 Wire 依赖 (AC: #1)
  - [ ] 5.1 创建 `internal/server/routes/affiliate.go`，注册 `GET /affiliate/info` 路由
  - [ ] 5.2 在 `handler/wire.go` 中注册 `NewAffiliateHandler`
  - [ ] 5.3 在 `service/wire.go` 中注册 `NewAffiliateService`（如 Story 1.1 未完成）
  - [ ] 5.4 在 `handler/handlers.go` 的 `Handlers` struct 中添加 `Affiliate` 字段
  - [ ] 5.5 运行 `go generate ./cmd/server` 重新生成 Wire 代码
- [ ] Task 6: Repository 层扩展 (AC: #4)
  - [ ] 6.1 确保 `affiliate_repo.go` 中有 `GetByUserID(ctx, userID)` 方法
  - [ ] 6.2 该方法已在 Story 1.1 中定义，此处复用

## Dev Notes

### 前置依赖

Story 1.1（用户分销信息初始化）的代码必须先完成：
- `user_affiliate` Ent Schema 和数据库表已创建
- `affiliate_repo.go` 的 `GetByUserID` 方法已实现
- `affiliate_service.go` 的基本结构已存在
- Wire ProviderSet 中已注册 AffiliateService

如果 Story 1.1 尚未实现，本 Story 需要先完成 1.1 的所有任务。

### API 设计

```
GET /api/v1/affiliate/info
Authorization: Bearer {jwt_token}
```

**Response 200:**
```json
{
  "code": 0,
  "data": {
    "referral_code": "ABC123",
    "referral_link": "https://code.ai80.vip/r/ABC123",
    "qrcode_data_url": "data:image/png;base64,iVBOR...",
    "tier_level": 1,
    "tier_name": "青铜",
    "commission_rate": 0.05,
    "effective_count": 0,
    "next_tier_threshold": 11,
    "total_earnings": 0.00,
    "withdrawable": 0.00,
    "withdraw_threshold": 100.00,
    "can_withdraw": false
  }
}
```

**Response 404（无分销信息）:**
```json
{
  "code": -1,
  "message": "affiliate info not found"
}
```

### 二维码生成方案

使用 `github.com/skip2/go-qrcode` 在服务端生成，返回 base64 Data URL：

```go
import "github.com/skip2/go-qrcode"

func GenerateQRCodeDataURL(link string) (string, error) {
    png, err := qrcode.Encode(link, qrcode.Medium, 256)
    if err != nil {
        return "", err
    }
    return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
```

选择 base64 Data URL 而非独立图片 URL 的原因：
- 不需要额外的文件存储或 CDN
- 简化架构，无需管理图片生命周期
- 256x256 PNG 的 base64 大小约 2-4KB，完全可接受
- 后续如果需要 CDN 缓存，可在 Epic 3 UI 开发时调整

### 阶梯规则硬编码（待 Epic 7 配置化）

```go
var defaultTierRules = []TierRule{
    {Level: 1, Name: "青铜", MinCount: 0, MaxCount: 10, Rate: 0.05},
    {Level: 2, Name: "白银", MinCount: 11, MaxCount: 30, Rate: 0.08},
    {Level: 3, Name: "黄金", MinCount: 31, MaxCount: 0, Rate: 0.12}, // MaxCount=0 表示无上限
}

type TierRule struct {
    Level    int
    Name     string
    MinCount int
    MaxCount int // 0=无上限
    Rate     float64
}
```

### site_url 配置

在 `backend/internal/config/config.go` 中添加：

```go
type Config struct {
    // 现有字段...
    SiteURL string `mapstructure:"site_url"` // 站点 URL，如 https://code.ai80.vip
}
```

在 `config.yaml` 中：

```yaml
site_url: "https://code.ai80.vip"
```

参考现有的 `wechat_pay.notify_url` 配置方式。

### Handler 实现模式

严格遵循现有 Handler 模式（参考 `user_handler.go`）：

```go
type AffiliateHandler struct {
    affiliateService *service.AffiliateService
}

func NewAffiliateHandler(affiliateService *service.AffiliateService) *AffiliateHandler {
    return &AffiliateHandler{affiliateService: affiliateService}
}

func (h *AffiliateHandler) GetAffiliateInfo(c *gin.Context) {
    subject, ok := middleware2.GetAuthSubjectFromContext(c)
    if !ok {
        response.Unauthorized(c, "User not authenticated")
        return
    }

    info, err := h.affiliateService.GetAffiliateInfo(c.Request.Context(), subject.UserID)
    if err != nil {
        response.ErrorFrom(c, err)
        return
    }

    response.Success(c, dto.AffiliateInfoFromService(info))
}
```

### 路由注册模式

参考 `routes/user.go` 创建 `routes/affiliate.go`：

```go
func RegisterAffiliateRoutes(
    v1 *gin.RouterGroup,
    h *handler.Handlers,
    jwtAuth middleware.JWTAuthMiddleware,
) {
    authenticated := v1.Group("")
    authenticated.Use(gin.HandlerFunc(jwtAuth))
    {
        affiliate := authenticated.Group("/affiliate")
        {
            affiliate.GET("/info", h.Affiliate.GetAffiliateInfo)
        }
    }
}
```

然后在 `server/router.go` 中调用 `RegisterAffiliateRoutes`。

### Wire 注册

**handler/wire.go** - 在 ProviderSet 中添加 `NewAffiliateHandler`，在 `ProvideHandlers` 参数中添加：

```go
func ProvideHandlers(
    // ...existing params...
    affiliateHandler *AffiliateHandler,
) *Handlers {
    return &Handlers{
        // ...existing fields...
        Affiliate: affiliateHandler,
    }
}
```

**handler/handlers.go** - 在 Handlers struct 添加字段：

```go
type Handlers struct {
    // ...existing fields...
    Affiliate *AffiliateHandler
}
```

**service/wire.go** - 确保 `NewAffiliateService` 已注册（Story 1.1 应已完成）。

### DTO 定义

```go
// internal/handler/dto/affiliate_dto.go

type AffiliateInfoResponse struct {
    ReferralCode       string  `json:"referral_code"`
    ReferralLink       string  `json:"referral_link"`
    QRCodeDataURL      string  `json:"qrcode_data_url"`
    TierLevel          int     `json:"tier_level"`
    TierName           string  `json:"tier_name"`
    CommissionRate     float64 `json:"commission_rate"`
    EffectiveCount     int     `json:"effective_count"`
    NextTierThreshold  int     `json:"next_tier_threshold"`
    TotalEarnings      float64 `json:"total_earnings"`
    Withdrawable       float64 `json:"withdrawable"`
    WithdrawThreshold  float64 `json:"withdraw_threshold"`
    CanWithdraw        bool    `json:"can_withdraw"`
}
```

### Project Structure Notes

**新增文件：**
```
backend/
├── internal/
│   ├── handler/
│   │   ├── affiliate_handler.go       # 新增 Handler
│   │   └── dto/
│   │       └── affiliate_dto.go       # 新增 DTO
│   └── server/routes/
│       └── affiliate.go               # 新增路由
```

**修改文件：**
```
backend/
├── internal/
│   ├── config/config.go               # 添加 SiteURL 字段
│   ├── handler/
│   │   ├── handlers.go                # 添加 Affiliate 字段
│   │   └── wire.go                    # 注册 AffiliateHandler
│   ├── service/
│   │   └── affiliate_service.go       # 添加 GetAffiliateInfo 方法
│   └── server/router.go               # 注册 affiliate 路由
├── config.yaml                         # 添加 site_url
└── go.mod / go.sum                     # 添加 qrcode 依赖
```

**命名约定（遵循项目模式）：**
- Handler 文件：`affiliate_handler.go`，类型 `AffiliateHandler`
- DTO 文件：`affiliate_dto.go`，类型 `AffiliateInfoResponse`
- Route 文件：`affiliate.go`，函数 `RegisterAffiliateRoutes`

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.1.1] - API 设计：GET /api/v1/affiliate/info 完整响应结构
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#2.2.1] - user_affiliate 表结构
- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#5.1] - 缓存策略：`aff:code:{code}` 24h TTL
- [Source: _bmad-output/planning-artifacts/epics.md#Story-1.2] - Story 需求定义：FR1, FR4
- [Source: _bmad-output/implementation-artifacts/1-1-user-affiliate-init.md] - Story 1.1 实现指南（前置依赖）
- [Source: backend/internal/handler/user_handler.go] - Handler 模式参考
- [Source: backend/internal/server/routes/user.go] - 路由注册模式参考
- [Source: backend/internal/handler/wire.go] - Wire Handler 注册模式
- [Source: backend/internal/config/config.go] - 配置管理模式

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestGetAffiliateInfo_Normal | 用户有分销信息 | 返回完整推广信息 |
| TestGetAffiliateInfo_NotFound | 用户无分销信息 | 返回 404 |
| TestGetAffiliateInfo_TierMatch | tier_level=1/2/3 | 正确匹配 tier_name 和 commission_rate |
| TestGetAffiliateInfo_CanWithdraw | withdrawable=150 | can_withdraw=true |
| TestGetAffiliateInfo_CannotWithdraw | withdrawable=50 | can_withdraw=false |
| TestGetAffiliateInfo_NextTier | effective_count=5, tier=1 | next_tier_threshold=11 |
| TestGenerateQRCode_Valid | 有效链接 | 返回 data:image/png;base64,... |
| TestGenerateQRCode_Empty | 空链接 | 返回 error |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 完整 API 流程 | 认证 → 获取推广信息 → 响应格式验证 |
| 未认证请求 | 返回 401 |
| 无分销信息用户 | 返回 404 |

### 注意事项

1. **AffiliateService 需要注入 Config**：用于获取 site_url，构造函数需要添加 `config *config.Config` 参数
2. **二维码生成不缓存**：当前阶段每次请求动态生成，后续可考虑 Redis 缓存
3. **提现门槛硬编码**：`withdraw_threshold = 100.00`，Epic 7 实现配置化
4. **前端二维码下载**：前端从 base64 Data URL 创建下载链接即可，无需后端额外接口
5. **与 Story 1.1 的关系**：如果 1.1 代码已存在，本 Story 在其基础上扩展；如果不存在，需要先实现 1.1 的全部代码

### 从 Story 1.1 继承的关键信息

- 邀请码生成算法：8 位，字符集 `23456789ABCDEFGHJKLMNPQRSTUVWXYZ`
- 错误定义：`ErrAffiliateNotFound`（在 `affiliate_errors.go` 中）
- Repository 接口：`GetByUserID(ctx, userID) (*UserAffiliate, error)`
- Service 构造函数：`NewAffiliateService(repo AffiliateRepository)`

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
