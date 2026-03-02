# sub2api 测试计划

## 一、单元测试 (Go) — 已有 262 个文件

### 核心模块覆盖
- [x] Handler — 请求处理（OpenAI/Gemini/Sora gateway）
- [x] Admin — 管理后台 API
- [x] Failover — 故障转移逻辑
- [x] Idempotency — 幂等处理
- [x] Usage — 用量记录和统计
- [x] Backoff — 退避策略
- [x] FastPath — 快速路径优化

### CI 已有
- [x] make test-unit（单元测试）
- [x] make test-integration（集成测试）
- [x] golangci-lint（代码风格）
- [x] security-scan（安全扫描）

## 二、E2E / 集成测试

### 待补充
- [ ] 端到端 API 调用链：client → sub2api → upstream provider
- [ ] 订阅 key 轮换场景
- [ ] 限流触发场景
- [ ] Admin 后台操作流程

## 三、冒烟测试（部署后）

1. Cloud Run 健康检查 → 200
2. /v1/models → 返回模型列表
3. Admin 登录 → 成功
4. 创建 API key → 可用于请求

## 四、全流程回归（CI 自动跑）

```bash
cd backend && make test-unit
cd backend && make test-integration
golangci-lint run
```

注意：集成测试需要 PostgreSQL + Redis。CI 里用 docker service container。
