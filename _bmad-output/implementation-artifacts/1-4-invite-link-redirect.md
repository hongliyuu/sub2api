# Story 1.4: 邀请链接跳转处理

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 潜在用户,
I want 点击邀请链接后被引导到注册页面,
so that 完成注册并绑定邀请关系。

## Acceptance Criteria

1. **AC1**: 用户点击 `{domain}/r/{code}` 后，前端路由匹配该 URL 并提取邀请码
2. **AC2**: 邀请码同时存储到 localStorage（key: `referral_code`）和 Cookie（name: `referral_code`，7 天过期，path=/）
3. **AC3**: 存储完成后自动跳转到注册页面 `/register`
4. **AC4**: 邀请码无效时（后端校验返回 404）仍跳转注册页面，但不存储该无效码
5. **AC5**: 可选功能：调用后端 `GET /api/v1/affiliate/validate-code/:code` 接口校验邀请码有效性
6. **AC6**: 已登录用户点击邀请链接时，跳转到首页而非注册页（邀请码不存储）
7. **AC7**: 注册页面能够读取 Cookie/localStorage 中的 referral_code 并在注册请求中传递给后端

## Dependencies

- **Depends On**: Story 1.1 (邀请码存在性校验), Story 1.2 (邀请链接格式 `{site_url}/r/{code}`)
- **Depended By**: Story 1.5 (注册时读取 referral_code 并传递给后端)

## Tasks / Subtasks

- [ ] Task 1: 前端路由配置 (AC: #1, #3, #6)
  - [ ] 1.1 在 `src/router/index.ts` 中添加 `/r/:code` 路由定义
  - [ ] 1.2 路由指向 `ReferralRedirect.vue` 组件
  - [ ] 1.3 路由 meta 设置 `requiresAuth: false`
- [ ] Task 2: 创建 ReferralRedirect.vue 组件 (AC: #1, #2, #3, #4, #6)
  - [ ] 2.1 创建 `src/views/auth/ReferralRedirect.vue`
  - [ ] 2.2 从路由参数中提取 `code`
  - [ ] 2.3 检查用户是否已登录，已登录则跳转首页
  - [ ] 2.4 存储邀请码到 localStorage 和 Cookie
  - [ ] 2.5 跳转到注册页面
- [ ] Task 3: Cookie/localStorage 存储工具 (AC: #2)
  - [ ] 3.1 创建 `src/utils/referral.ts` 工具模块
  - [ ] 3.2 实现 `saveReferralCode(code)` 方法
  - [ ] 3.3 实现 `getReferralCode()` 方法（优先读 Cookie，fallback 到 localStorage）
  - [ ] 3.4 实现 `clearReferralCode()` 方法
- [ ] Task 4: 可选 - 后端邀请码校验接口 (AC: #4, #5)
  - [ ] 4.1 后端 `AffiliateHandler` 添加 `ValidateCode` 方法
  - [ ] 4.2 路由注册 `GET /api/v1/affiliate/validate-code/:code`（公开接口，无需认证）
  - [ ] 4.3 前端在跳转前调用校验接口
- [ ] Task 5: 注册页面集成 (AC: #7)
  - [ ] 5.1 修改 `src/views/auth/RegisterView.vue`，读取 referral_code
  - [ ] 5.2 注册 API 请求中携带 `referral_code` 字段
  - [ ] 5.3 注册成功后调用 `clearReferralCode()` 清除存储

## Dev Notes

### 前置依赖

Story 1.1 和 1.2 的代码必须先完成：
- `user_affiliate` 表和 `referral_code` 字段已存在
- `AffiliateHandler` 和路由基础设施已就绪
- `affiliate_repo.go` 的 `ExistsByCode()` 方法可用

### 前端路由配置

在 `frontend/src/router/index.ts` 中添加路由，位置在 Public Routes 区域内：

```typescript
// ==================== Referral Routes ====================
{
  path: '/r/:code',
  name: 'ReferralRedirect',
  component: () => import('@/views/auth/ReferralRedirect.vue'),
  meta: {
    requiresAuth: false,
    title: 'Referral'
  }
},
```

注意：该路由必须放在 404 catch-all 路由 `/:pathMatch(.*)*` 之前，否则会被 404 捕获。参考现有路由定义格式（如 `/login`、`/register` 等公开路由）。

### ReferralRedirect.vue 组件

```vue
<!-- src/views/auth/ReferralRedirect.vue -->
<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { saveReferralCode } from '@/utils/referral'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

onMounted(async () => {
  const code = route.params.code as string

  // 已登录用户直接跳转首页
  if (authStore.isAuthenticated) {
    router.replace('/dashboard')
    return
  }

  if (code) {
    // 可选：校验邀请码有效性
    try {
      // const { data } = await affiliateApi.validateCode(code)
      // if (data.valid) {
      saveReferralCode(code)
      // }
    } catch {
      // 校验失败时不存储，静默处理
    }
  }

  // 跳转到注册页
  router.replace('/register')
})
</script>

<template>
  <div class="flex items-center justify-center min-h-screen">
    <div class="text-center">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
      <p class="mt-4 text-gray-500">正在跳转...</p>
    </div>
  </div>
</template>
```

### Cookie/localStorage 存储工具

```typescript
// src/utils/referral.ts

const REFERRAL_CODE_KEY = 'referral_code'
const COOKIE_EXPIRY_DAYS = 7

/**
 * 设置 Cookie
 */
function setCookie(name: string, value: string, days: number): void {
  const expires = new Date()
  expires.setTime(expires.getTime() + days * 24 * 60 * 60 * 1000)
  document.cookie = `${name}=${encodeURIComponent(value)};expires=${expires.toUTCString()};path=/;SameSite=Lax`
}

/**
 * 读取 Cookie
 */
function getCookie(name: string): string | null {
  const nameEQ = `${name}=`
  const cookies = document.cookie.split(';')
  for (const cookie of cookies) {
    const c = cookie.trim()
    if (c.startsWith(nameEQ)) {
      return decodeURIComponent(c.substring(nameEQ.length))
    }
  }
  return null
}

/**
 * 删除 Cookie
 */
function deleteCookie(name: string): void {
  document.cookie = `${name}=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/`
}

/**
 * 保存邀请码到 Cookie 和 localStorage
 */
export function saveReferralCode(code: string): void {
  if (!code) return
  setCookie(REFERRAL_CODE_KEY, code, COOKIE_EXPIRY_DAYS)
  localStorage.setItem(REFERRAL_CODE_KEY, code)
}

/**
 * 获取邀请码（优先 Cookie，fallback localStorage）
 */
export function getReferralCode(): string | null {
  return getCookie(REFERRAL_CODE_KEY) || localStorage.getItem(REFERRAL_CODE_KEY)
}

/**
 * 清除邀请码
 */
export function clearReferralCode(): void {
  deleteCookie(REFERRAL_CODE_KEY)
  localStorage.removeItem(REFERRAL_CODE_KEY)
}
```

### 可选：后端邀请码校验接口

```
GET /api/v1/affiliate/validate-code/:code
```

**Response 200（有效）:**
```json
{
  "code": 0,
  "data": {
    "valid": true,
    "code": "ABC123"
  }
}
```

**Response 200（无效）:**
```json
{
  "code": 0,
  "data": {
    "valid": false
  }
}
```

后端 Handler 实现：

```go
// internal/handler/affiliate_handler.go

func (h *AffiliateHandler) ValidateCode(c *gin.Context) {
    code := c.Param("code")
    if code == "" {
        response.Success(c, gin.H{"valid": false})
        return
    }

    exists, err := h.affiliateService.ValidateReferralCode(c.Request.Context(), code)
    if err != nil {
        // 校验失败时返回无效，不报错
        response.Success(c, gin.H{"valid": false})
        return
    }

    response.Success(c, gin.H{"valid": exists, "code": code})
}
```

Service 层：

```go
// internal/service/affiliate_service.go

func (s *AffiliateService) ValidateReferralCode(ctx context.Context, code string) (bool, error) {
    return s.affiliateRepo.ExistsByCode(ctx, code)
}
```

路由注册（公开接口，不需要 JWT 认证）：

```go
// internal/server/routes/affiliate.go
// 在 RegisterAffiliateRoutes 中添加公开路由

public := v1.Group("/affiliate")
{
    public.GET("/validate-code/:code", h.Affiliate.ValidateCode)
}
```

### 注册页面集成

修改 `frontend/src/views/auth/RegisterView.vue`，在注册表单提交时读取邀请码：

```typescript
import { getReferralCode, clearReferralCode } from '@/utils/referral'

// 在注册请求中携带 referral_code
async function handleRegister() {
  const referralCode = getReferralCode()

  const payload = {
    email: form.email,
    password: form.password,
    verify_code: form.verifyCode,
    promo_code: form.promoCode,
    referral_code: referralCode || undefined, // 新增字段
  }

  const response = await authApi.register(payload)

  // 注册成功后清除邀请码
  if (response.code === 0) {
    clearReferralCode()
  }
}
```

后端 `RegisterWithVerification` 方法签名需要在 Story 1.5 中扩展以接受 `referralCode` 参数。本 Story 仅负责前端传递，后端处理在 1.5 中完成。

### Project Structure Notes

**新增文件：**
```
frontend/
├── src/
│   ├── views/auth/
│   │   └── ReferralRedirect.vue       # 邀请链接跳转组件
│   └── utils/
│       └── referral.ts                # 邀请码存储工具
```

**修改文件：**
```
frontend/
├── src/
│   ├── router/index.ts                # 添加 /r/:code 路由
│   └── views/auth/RegisterView.vue    # 读取 referral_code 并传给注册 API

backend/
├── internal/
│   ├── handler/affiliate_handler.go   # 添加 ValidateCode 方法（可选）
│   └── server/routes/affiliate.go     # 添加公开校验路由（可选）
```

**命名约定（遵循项目模式）：**
- Vue 组件：PascalCase（`ReferralRedirect.vue`）
- 工具模块：camelCase（`referral.ts`）
- 路由名称：PascalCase（`ReferralRedirect`）
- Cookie/localStorage key：snake_case（`referral_code`）

### References

- [Source: _bmad-output/affiliate/TDD-affiliate-system.md#3.1] - API 设计
- [Source: _bmad-output/planning-artifacts/epics.md#FR6] - 邀请链接注册绑定需求
- [Source: frontend/src/router/index.ts] - 现有 Vue Router 配置模式
- [Source: frontend/src/views/auth/RegisterView.vue] - 注册页面实现
- [Source: frontend/src/stores/auth.ts] - 认证状态管理
- [Source: _bmad-output/implementation-artifacts/1-2-invite-link-qrcode.md] - Story 1.2（邀请链接格式 `{site_url}/r/{code}`）

### Testing Requirements

#### 前端 Unit Tests (Vitest)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestSaveReferralCode | 保存邀请码 | Cookie 和 localStorage 均写入 |
| TestGetReferralCode_Cookie | Cookie 存在 | 优先读取 Cookie |
| TestGetReferralCode_LocalStorage | Cookie 不存在 | 降级读取 localStorage |
| TestClearReferralCode | 清除邀请码 | Cookie 和 localStorage 均清除 |
| TestRedirect_LoggedIn | 已登录用户 | 跳转 dashboard，不存储邀请码 |
| TestRedirect_NotLoggedIn | 未登录用户 | 存储邀请码，跳转 register |
| TestRedirect_InvalidCode | 无效邀请码 | 不存储，仍跳转 register |

#### 后端 Unit Tests (`//go:build unit`)

| 测试函数 | 场景 | 期望结果 |
|---------|------|---------|
| TestValidateCode_Valid | 有效邀请码 | 返回 valid=true |
| TestValidateCode_Invalid | 不存在的码 | 返回 valid=false |
| TestValidateCode_Empty | 空字符串 | 返回 valid=false |

#### Integration Tests

| 测试场景 | 验证内容 |
|---------|---------|
| 完整跳转流程 | 访问 /r/ABC123 → 存储邀请码 → 跳转注册 → 注册时携带 referral_code |
| 无效码场景 | 访问 /r/INVALID → 不存储 → 仍跳转注册 |

### 注意事项

1. **Cookie 安全性**：使用 `SameSite=Lax` 防止 CSRF，不使用 `Secure` 标志以支持开发环境 HTTP 访问（生产环境可在 Nginx 层强制 HTTPS）
2. **双重存储**：同时使用 Cookie 和 localStorage 确保兼容性，Cookie 作为主要来源，localStorage 作为 fallback
3. **竞态处理**：用户快速多次点击不同邀请链接时，以最后一次为准（覆盖写入）
4. **路由顺序**：`/r/:code` 路由必须在 `/:pathMatch(.*)*` 之前定义
5. **前后端分离**：本 Story 主要是前端工作，后端校验接口为可选实现，Story 1.5 负责后端注册绑定逻辑

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
