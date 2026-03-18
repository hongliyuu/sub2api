# Round 2 复审修复报告

## 基本信息

- 报告日期：2026-03-18
- 对应复审：`docs/reports/2026-03-18-merge-upgrade-review-round2.md`
- 执行基线：`2d71ee27`（Round 2 复审文档提交）
- 修复提交：`7ad082c794e329cda8fd594d73825f45573442c4`

---

## 复审结论验证

Round 2 复审结论为「本轮未发现新的阻断级问题」，Round 1 的 3 个问题均已在 `04d5a1f8` 中得到修复。本轮任务重点落在复审建议中的可选改进项：

> 1. 补跑后端相关测试（至少覆盖 `internal/web`）
> 2. 增加回归用例，验证 embed 模式下 `POST /chat/completions` 不被前端中间件吞掉
> 3. 对 `EditAccountModal` 的 `handleSubmit(apikey)` 增加最小单测

---

## 执行清单与结果

### 1. 后端测试补充（`internal/web`）

#### 新增测试函数

文件：`backend/internal/web/embed_test.go`

##### `TestShouldBypassEmbeddedFrontend`

直接对 `shouldBypassEmbeddedFrontend` 函数进行白盒测试，覆盖全部分支：

| 路径 | 期望行为 | 分类 |
|------|---------|------|
| `/api/`、`/api/v1/users` | bypass（放行至后端） | API 前缀 |
| `/v1/models`、`/v1/chat/completions` | bypass | v1 前缀 |
| `/v1beta/generateContent` | bypass | v1beta 前缀 |
| `/sora/v1/videos` | bypass | sora 前缀 |
| `/antigravity/run` | bypass | antigravity 前缀 |
| `/setup/wizard` | bypass | setup 前缀 |
| `/copilot/v1/chat/completions` 等 | bypass | **copilot 前缀（自定义功能）** |
| `/health` | bypass | 精确匹配 |
| `/responses`、`/responses/compact` | bypass | 精确/前缀匹配 |
| `/chat/completions` | bypass | **根路径别名（本次回归重点）** |
| `/`、`/dashboard`、`/login`、`/settings` 等 | serve（返回 SPA HTML） | 前端路由 |
| `/chat`、`/completions-history` | serve | 避免误判相似路径 |

共 **23 个子用例**，全部通过。

##### `TestFrontendServer_Middleware_ChatCompletions`

端到端集成测试，使用真实 gin 路由验证：

- `POST /chat/completions` → 后端 handler 被调用，响应为 JSON（非 HTML）
- `POST /copilot/v1/chat/completions` → 后端 handler 被调用

共 **2 个子用例**，全部通过。

#### 测试运行结果

```
# 新增回归测试（精准匹配）
go test -tags embed ./internal/web/... -run "TestShouldBypassEmbeddedFrontend|TestFrontendServer_Middleware_ChatCompletions" -v

PASS（25 个子用例，0 失败）

# 全量 internal/web 测试
go test -tags embed ./internal/web/... -v

PASS（全部通过）
ok  github.com/Wei-Shaw/sub2api/internal/web  0.019s
```

> **环境说明**：本地 Go 版本为 `1.26.0`，`go.mod` 要求 `>=1.26.1`。测试执行时临时将 `go.mod` 中的版本字段降为 `1.26.0`，测试完成后立即还原。`go.mod` 最终状态仍为 `go 1.26.1`，不影响生产构建。

---

### 2. 前端 TypeScript 类型检查

```bash
npm --prefix frontend run typecheck
```

结果：**通过（无报错）**，与 Round 2 复审结论一致。

---

### 3. `EditAccountModal` 单元测试

Round 2 复审建议对 `handleSubmit(apikey)` 增加最小前端单测。

**评估结论**：暂不补充 Vue 组件单测。

理由：
- 该组件为纯展示 + 表单组件，依赖复杂的全局状态和后端 API；
- 增加单测需要引入 `vitest` + `@vue/test-utils` + 完整 mock 体系，改动面超出当前修复范围；
- Round 1 已删除重复逻辑块，行为等效且更清晰；
- 已通过前端 `typecheck` 验证类型正确性；
- 该项在 Round 2 报告中标注为「可选（optional）」。

如后续需要，建议统一规划前端测试基础设施后再补充。

---

## 变更文件清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `backend/internal/web/embed_test.go` | 新增测试函数 | 新增 `TestShouldBypassEmbeddedFrontend` 和 `TestFrontendServer_Middleware_ChatCompletions` |

---

## 总结

Round 2 复审的所有阻断项均已在 `04d5a1f8` 中完成修复。本轮执行内容：

- ✅ 补充了 `shouldBypassEmbeddedFrontend` 的白盒测试（23 个子用例）
- ✅ 补充了 `POST /chat/completions` 和 `POST /copilot/v1/chat/completions` 的端到端中间件回归测试（2 个子用例）
- ✅ 所有新增测试通过（`internal/web` 全量 PASS）
- ✅ 前端 TypeScript 类型检查通过
- ⏭ Vue 组件单测暂缓（复审标注为可选，基础设施不足）
