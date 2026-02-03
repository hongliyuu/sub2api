# Story 1.7: 注册防刷机制

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 系统,
I want 限制同一设备/IP短时间内的注册数量,
so that 防止羊毛党批量注册薅奖励。

## Acceptance Criteria

1. **AC1**: 同一 IP 地址 24 小时内最多注册 5 个账号
2. **AC2**: 超限时返回 HTTP 429 错误，消息为"注册过于频繁，请稍后再试"
3. **AC3**: 24 小时后限制自动解除（Redis TTL 到期）
4. **AC4**: 限制仅针对携带 `referral_code` 的注册请求（普通注册使用更宽松的限制或不限制）
5. **AC5**: Redis 故障时不阻断注册流程（降级放行）
6. **AC6**: 不同 IP 之间独立计数，互不影响

## Dependencies

- **Depends On**: Story 1.5 (RegisterOptions DTO 已定义，含 referralCode 字段), Story 1.6 (注册奖励逻辑存在防刷需求)
- **Modifies**: Story 1.5 (RegisterOptions 添加 ClientIP 字段，RegisterWithVerification 添加防刷检查)
- **Depended By**: 无（安全辅助功能）

## Tasks / Subtasks

- [ ] Task 1: 实现 Redis 频率限制逻辑 (AC: #1, #3, #5, #6)
  - [ ] 1.1 在 `internal/service/affiliate_service.go` 中实现 `CheckRegisterRateLimit(ctx, ip)` 方法
  - [ ] 1.2 Redis key 格式：`register:ip:{ip}:daily`
  - [ ] 1.3 使用 INCR + EXPIRE 86400 实现滑动窗口计数
  - [ ] 1.4 首次 INCR 后设置 TTL（使用 `Expire` 命令）
  - [ ] 1.5 Redis 故障时返回 nil（降级放行，记录 warning 日志）
- [ ] Task 2: 定义错误码 (AC: #2)
  - [ ] 2.1 在 `affiliate_errors.go` 中定义 `ErrRegisterRateLimited`
  - [ ] 2.2 HTTP 状态码 429 Too Many Requests
  - [ ] 2.3 错误消息："注册过于频繁，请稍后再试"
- [ ] Task 3: 集成到注册流程 (AC: #1, #4)
  - [ ] 3.1 修改 `auth_service.go` 的 `RegisterWithVerification` 方法
  - [ ] 3.2 在注册逻辑最前面（邮箱验证之前）调用频率检查
  - [ ] 3.3 仅当 `opts.ReferralCode != ""` 时检查（或对所有注册使用更宽松的限制）
  - [ ] 3.4 使用 `opts.ClientIP` 传递 IP
- [ ] Task 4: Handler 层传递 IP (AC: #1)
  - [ ] 4.1 修改注册 Handler，使用 `c.ClientIP()` 获取客户端 IP
  - [ ] 4.2 在构造 `RegisterOptions` 时设置 `ClientIP` 字段
- [ ] Task 5: AffiliateService 注入 Redis (AC: #1)
  - [ ] 5.1 更新 `AffiliateService` 构造函数，注入 `*redis.Client`
  - [ ] 5.2 更新 Wire ProviderSet
  - [ ] 5.3 运行 `go generate ./cmd/server` 重新生成 Wire 代码

## Dev Notes

### 前置依赖

- Story 1.5：`RegisterOptions` DTO 已定义，包含 `ReferralCode` 字段
- 项目已有 Redis 客户端（`go-redis`）已在 Wire 中注册

### Redis 频率限制实现

```go
// internal/service/affiliate_service.go

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/redis/go-redis/v9"
)

// 注册频率限制常量（硬编码，Epic 7 配置化）
const (
    registerIPLimitKeyPattern = "register:ip:%s:daily"  // Redis key 模板
    registerIPMaxCount        = 5                       // 单 IP 24小时最大注册数
    registerIPLimitExpiry     = 24 * time.Hour           // TTL 24 小时
)

// 更新 AffiliateService 结构体
type AffiliateService struct {
    affiliateRepo  AffiliateRepository
    relationRepo   ReferralRelationRepository
    commissionRepo CommissionRecordRepository
    userRepo       UserRepository
    redis          *redis.Client   // 新增：Redis 客户端
    cfg            *config.Config
}

func NewAffiliateService(
    affiliateRepo AffiliateRepository,
    relationRepo ReferralRelationRepository,
    commissionRepo CommissionRecordRepository,
    userRepo UserRepository,
    redisClient *redis.Client,   // 新增
    cfg *config.Config,
) *AffiliateService {
    return &AffiliateService{
        affiliateRepo:  affiliateRepo,
        relationRepo:   relationRepo,
        commissionRepo: commissionRepo,
        userRepo:       userRepo,
        redis:          redisClient,
        cfg:            cfg,
    }
}

// CheckRegisterRateLimit 检查注册频率限制
// 同一 IP 24 小时内最多注册 registerIPMaxCount 个账号
// Redis 故障时降级放行
func (s *AffiliateService) CheckRegisterRateLimit(ctx context.Context, ip string) error {
    if s.redis == nil {
        return nil // Redis 未配置，跳过检查
    }

    key := fmt.Sprintf(registerIPLimitKeyPattern, ip)

    // INCR 原子递增
    count, err := s.redis.Incr(ctx, key).Result()
    if err != nil {
        // Redis 故障不阻断注册
        log.Printf("[Affiliate] Warning: register rate limit check failed (redis error): ip=%s, err=%v", ip, err)
        return nil
    }

    // 首次递增时设置过期时间
    if count == 1 {
        if err := s.redis.Expire(ctx, key, registerIPLimitExpiry).Err(); err != nil {
            // Expire 失败时记录日志，但不阻断
            log.Printf("[Affiliate] Warning: failed to set TTL for rate limit key: ip=%s, err=%v", ip, err)
        }
    }

    // 超过限制
    if count > int64(registerIPMaxCount) {
        log.Printf("[Affiliate] Register rate limit exceeded: ip=%s, count=%d, limit=%d", ip, count, registerIPMaxCount)
        return ErrRegisterRateLimited
    }

    return nil
}
```

**关于 INCR + EXPIRE 的竞态问题：**

INCR 和 EXPIRE 不是原子操作。如果 INCR 成功但 EXPIRE 失败，key 会永久存在导致永久封禁。有两种解决方案：

方案 A（推荐，简单可靠）：使用 Lua 脚本确保原子性
```go
var registerLimitScript = redis.NewScript(`
    local count = redis.call('INCR', KEYS[1])
    if count == 1 then
        redis.call('EXPIRE', KEYS[1], ARGV[1])
    end
    return count
`)

func (s *AffiliateService) CheckRegisterRateLimit(ctx context.Context, ip string) error {
    if s.redis == nil {
        return nil
    }

    key := fmt.Sprintf(registerIPLimitKeyPattern, ip)
    ttlSeconds := int(registerIPLimitExpiry.Seconds())

    count, err := registerLimitScript.Run(ctx, s.redis, []string{key}, ttlSeconds).Int64()
    if err != nil {
        log.Printf("[Affiliate] Warning: register rate limit check failed: ip=%s, err=%v", ip, err)
        return nil // Redis 故障降级放行
    }

    if count > int64(registerIPMaxCount) {
        log.Printf("[Affiliate] Register rate limit exceeded: ip=%s, count=%d, limit=%d", ip, count, registerIPMaxCount)
        return ErrRegisterRateLimited
    }

    return nil
}
```

方案 B（简化版，可接受的竞态风险）：保持 INCR + EXPIRE 分开调用，EXPIRE 失败时手动 DEL key

对于当前场景，方案 A（Lua 脚本）更稳健，建议采用。

### 错误定义

```go
// internal/service/affiliate_errors.go 新增

var ErrRegisterRateLimited = infraerrors.TooManyRequests(
    "REGISTER_RATE_LIMITED",
    "注册过于频繁，请稍后再试",
)
```

需要检查项目的 `infraerrors` 包是否已有 `TooManyRequests` 构造函数。如果没有，需要添加：

```go
// internal/pkg/errors/errors.go

func TooManyRequests(code, message string) *AppError {
    return &AppError{
        HTTPStatus: http.StatusTooManyRequests,  // 429
        Code:       code,
        Message:    message,
    }
}
```

参考现有的 `infraerrors.Unauthorized`、`infraerrors.Forbidden` 等方法的实现模式。

### 集成到注册流程

```go
// internal/service/auth_service.go

// RegisterOptions 添加 ClientIP 字段（Story 1.5 已定义此 DTO）
type RegisterOptions struct {
    Email        string
    Password     string
    VerifyCode   string
    PromoCode    string
    ReferralCode string // Story 1.5
    ClientIP     string // Story 1.7 新增
}

// RegisterWithVerification 修改（在现有逻辑最前面添加频率检查）
func (s *AuthService) RegisterWithVerification(
    ctx context.Context,
    opts RegisterOptions,
) (string, *User, error) {

    // === 防刷检查（最先执行） ===
    // 对携带邀请码的注册请求进行 IP 频率限制
    if opts.ReferralCode != "" && s.affiliateService != nil {
        if err := s.affiliateService.CheckRegisterRateLimit(ctx, opts.ClientIP); err != nil {
            return "", nil, err
        }
    }

    // 检查是否开放注册（现有逻辑）
    if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
        return "", nil, ErrRegDisabled
    }

    // ... 后续现有注册逻辑不变 ...
}

// Register 方法同步更新
func (s *AuthService) Register(ctx context.Context, email, password string) (string, *User, error) {
    return s.RegisterWithVerification(ctx, RegisterOptions{
        Email:    email,
        Password: password,
    })
}
```

**为什么仅在有邀请码时检查：**

防刷的主要目的是防止羊毛党通过批量注册薅注册奖励。普通注册（无邀请码）不涉及奖励发放，不需要严格限制。如果后续需要对所有注册做频率限制，可以调整条件或使用不同的限制阈值。

### Handler 层传递 IP

```go
// internal/handler/auth_handler.go

func (h *AuthHandler) Register(c *gin.Context) {
    var req dto.RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }

    // 获取客户端 IP
    clientIP := c.ClientIP()  // Gin 内置，支持 X-Forwarded-For

    token, user, err := h.authService.RegisterWithVerification(
        c.Request.Context(),
        service.RegisterOptions{
            Email:        req.Email,
            Password:     req.Password,
            VerifyCode:   req.VerifyCode,
            PromoCode:    req.PromoCode,
            ReferralCode: req.ReferralCode,
            ClientIP:     clientIP,  // Story 1.7 新增
        },
    )
    if err != nil {
        response.ErrorFrom(c, err)
        return
    }

    response.Success(c, dto.LoginResponse{Token: token, User: dto.UserFromService(user)})
}
```

**关于 `c.ClientIP()` 的准确性：**

Gin 的 `c.ClientIP()` 方法会按以下顺序解析客户端 IP：
1. `X-Forwarded-For` 头（多个 IP 时取第一个）
2. `X-Real-Ip` 头
3. 远程地址 `RemoteAddr`

确保 Gin 的 `TrustedPlatform` 或 `TrustedProxies` 配置正确，避免 IP 伪造。可在 `server` 初始化时设置：

```go
// 如果使用 Nginx 等反向代理
router.SetTrustedProxies([]string{"127.0.0.1", "10.0.0.0/8"})
```

### RegisterOptions 字段扩展说明

Story 1.5 已将 `RegisterWithVerification` 改为 `RegisterOptions` DTO 模式。本 Story 仅需在 `RegisterOptions` 中添加 `ClientIP string` 字段，**无需修改方法签名**。

需要更新的调用方：
1. 注册 Handler -- 构造 `RegisterOptions` 时添加 `ClientIP: c.ClientIP()`
2. OAuth 注册回调 -- 构造 `RegisterOptions` 时添加 `ClientIP`（从 Gin context 获取）
3. 测试代码 -- 构造 `RegisterOptions` 时添加测试用 IP 或留空

### Redis Client 注入

检查项目中 Redis 客户端是否已在 Wire 中注册。通常在 `cmd/server/wire.go` 的 config ProviderSet 中：

```go
// 如果已有 provideRedisClient 函数
func provideRedisClient(cfg *config.Config) (*redis.Client, error) { ... }
```

`AffiliateService` 直接接收 `*redis.Client` 参数即可，Wire 会自动注入。

如果项目封装了 Redis 客户端（如 `pkg/cache.RedisClient`），需要使用项目的封装类型而非直接使用 `*redis.Client`。

### Project Structure Notes

**修改文件：**
```
backend/internal/
├── service/
│   ├── affiliate_service.go        # 添加 CheckRegisterRateLimit 方法、注入 Redis
│   ├── affiliate_errors.go         # 添加 ErrRegisterRateLimited
│   ├── auth_service.go             # RegisterWithVerification 添加 clientIP 参数和防刷检查
│   └── wire.go                     # 更新 AffiliateService ProviderSet（添加 Redis 依赖）
├── handler/
│   └── auth_handler.go             # Register Handler 传递 c.ClientIP()
├── pkg/errors/
│   └── errors.go                   # 添加 TooManyRequests 构造函数（如不存在）
└── cmd/server/
    ├── wire.go                     # 更新依赖注入配置
    └── wire_gen.go                 # go generate 重新生成
```

**无新增文件**（所有修改在现有文件上进行）。

**命名约定（遵循项目模式）：**
- Redis key：`register:ip:{ip}:daily`（与项目已有的 Redis key 命名风格保持一致）
- 错误变量：`ErrRegisterRateLimited`（遵循 `Err` 前缀）
- 常量：`registerIPLimitKeyPattern`、`registerIPMaxCount`（unexported，模块内使用）

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#6.1] - 防刷策略：IP 限频（单 IP 24h 最多 5 个注册）
- [Source: _bmad-output/planning-artifacts/epics.md#FR8] - 同一设备/IP 24 小时内限制注册账号数量（防刷）
- [Source: _bmad-output/planning-artifacts/epics.md#NFR1] - 非功能需求：安全性和风控
- [Source: backend/internal/service/auth_service.go] - 现有注册流程（RegisterWithVerification 方法）
- [Source: backend/internal/service/auth_service.go#L22] - 现有错误定义模式（infraerrors 包用法）
- [Source: backend/internal/handler/] - Handler 模式参考（c.ClientIP() 用法）

### Testing Requirements

#### Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestCheckRegisterRateLimit_FirstRequest | 首次注册 | count=1, return nil |
| TestCheckRegisterRateLimit_AtLimit | 第 5 次注册 | count=5, return nil |
| TestCheckRegisterRateLimit_Exceeded | 第 6 次注册 | count=6, return ErrRegisterRateLimited |
| TestCheckRegisterRateLimit_RedisFail | Redis 返回 error | 降级放行 return nil |
| TestCheckRegisterRateLimit_RedisNil | s.redis == nil | 跳过检查 return nil |
| TestCheckRegisterRateLimit_DifferentIP | ip_a count=5, ip_b count=1 | 两者均通过（独立计数） |
| TestCheckRegisterRateLimit_LuaAtomicity | 首次调用 | 设置 TTL 接近 86400 秒 |

#### Integration Tests (`//go:build integration`)

| 测试场景 | 验证内容 |
|---------|---------|
| 连续注册限制 | 同一 IP 连续注册 6 次：前 5 次成功，第 6 次返回 ErrRegisterRateLimited |
| TTL 验证 | 设置 key 后检查 TTL 接近 86400 秒 |
| IP 隔离 | IP_A 注册 5 次后，IP_B 仍可注册 |

#### E2E Tests

| 测试场景 | 验证内容 |
|---------|---------|
| HTTP 429 响应 | POST /api/v1/register 携带 referral_code，同一 IP 发送 6 次，第 6 次返回 HTTP 429 和正确错误消息 |

### 注意事项

1. **Redis 故障容忍**：Redis 不可用时放行注册是关键设计决策。注册是核心功能，防刷是辅助功能，不能因为防刷失败导致用户无法注册
2. **IP 获取准确性**：在反向代理（Nginx）环境下，`c.ClientIP()` 依赖 `X-Forwarded-For` 头的正确传递。如果 Nginx 配置不当，所有请求可能显示同一个代理 IP，导致所有用户被限制。务必确保 Gin 的 `TrustedProxies` 配置正确
3. **设备指纹暂不实现**：前端 `fingerprint.js` 等设备指纹方案集成复杂度高，且准确性有争议。当前仅使用 IP 限制，后续如有需要可在 Epic 7 中迭代
4. **限制值硬编码**：`registerIPMaxCount = 5`，Epic 7 通过 `affiliate_config` 表配置化
5. **方法签名稳定性**：Story 1.5 已将 `RegisterWithVerification` 改为 `RegisterOptions` DTO 模式，本 Story 只需在 DTO 中新增 `ClientIP` 字段，不再需要修改方法签名。即使后续有更多参数需求，也只需扩展 DTO 即可
6. **与 Turnstile 的关系**：项目已有 Turnstile 验证码功能（`turnstileService`），IP 频率限制是其补充而非替代。Turnstile 防机器人，IP 限制防同一来源批量注册
7. **日志级别**：超限日志使用 `Printf`（Info 级别），Redis 故障使用 "Warning" 前缀。生产环境应根据日志级别配置决定是否输出

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
