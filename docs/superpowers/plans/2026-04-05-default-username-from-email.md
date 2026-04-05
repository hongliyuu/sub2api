# Default Username From Email Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用户通过邮箱+密码注册后，若未设置用户名，则自动将邮箱地址作为默认用户名。

**Architecture:** 仅修改 `RegisterWithVerification` 函数中创建 `User` 结构体的代码，在 `Username` 字段赋值为 `email`。改动极小，不影响 OAuth 注册路径，不影响已有用户。

**Tech Stack:** Go, testify/require (unit tests)

---

### Task 1: 写失败测试，验证注册后 Username 等于 email

**Files:**
- Modify: `backend/internal/service/auth_service_register_test.go`

- [ ] **Step 1: 在 `TestAuthService_Register_Success` 末尾追加对 Username 的断言**

在文件的 `TestAuthService_Register_Success` 函数（约第 302-320 行）中，在最后一行 `require.True(t, user.CheckPassword("password"))` 之后添加：

```go
require.Equal(t, "user@test.com", user.Username)
```

完整修改后的函数末尾如下：

```go
require.True(t, user.CheckPassword("password"))
require.Equal(t, "user@test.com", user.Username)
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
cd backend && go test -tags unit ./internal/service/... -run TestAuthService_Register_Success -v
```

预期输出（失败）：
```
--- FAIL: TestAuthService_Register_Success
    auth_service_register_test.go:XXX: 
        Error Trace: auth_service_register_test.go:XXX
        Error:       Not equal: 
                     expected: "user@test.com"
                     actual  : ""
FAIL
```

- [ ] **Step 3: 修改实现，在创建 User 时设置 Username = email**

修改 `backend/internal/service/auth_service.go` 第 191-198 行的 `User` 结构体字面量：

```go
// 创建用户
user := &User{
    Email:        email,
    Username:     email,
    PasswordHash: hashedPassword,
    Role:         RoleUser,
    Balance:      defaultBalance,
    Concurrency:  defaultConcurrency,
    Status:       StatusActive,
}
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
cd backend && go test -tags unit ./internal/service/... -run TestAuthService_Register_Success -v
```

预期输出：
```
--- PASS: TestAuthService_Register_Success
PASS
```

- [ ] **Step 5: 运行全部 unit 测试，确认无回归**

```bash
cd backend && go test -tags unit ./internal/service/... -v
```

预期：所有测试通过，无 FAIL。

- [ ] **Step 6: 提交**

```bash
cd /Users/ziji/personal/github/sub2api
git add backend/internal/service/auth_service.go backend/internal/service/auth_service_register_test.go
git commit -m "Feature: 用户注册时默认使用邮箱作为用户名"
```
