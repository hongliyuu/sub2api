# CLAUDE.md

给未来 Claude 会话（和维护者）参考：本仓库在 fork 工作流 + 本地开发过程中踩过的坑与绕开方法。与 `DEV_GUIDE.md` 互补 —— 那份偏环境配置，这份偏「实际启动 / 合并 / 功能扩展时遇到的陷阱」。

---

## 1. 仓库与分支约定

- `main` —— 本仓库（shuanbao0/sub2api）的工作分支：upstream 最新 + 我们自己的定制（如卡券发放功能）。track `origin/main`。
- `upstream-main` —— 纯 upstream 镜像分支。track `Wei-Shaw/sub2api` 的 `main`，不放我们的改动。
- `origin` = `https://github.com/shuanbao0/sub2api.git`
- `upstream` = `https://github.com/Wei-Shaw/sub2api.git`

同步上游：
```bash
git fetch upstream
git checkout upstream-main && git pull --ff-only
git checkout main && git merge upstream-main
```

---

## 2. 本地开发启动流程

### 2.1 依赖容器

`deploy/docker-compose.deps.yml` 只起 Postgres + Redis：

```bash
cd deploy
docker compose -f docker-compose.deps.yml up -d
docker compose -f docker-compose.deps.yml ps
```

默认值（硬编码，改端口/密码改文件，**不用** env var 替换）：

| 服务 | 宿主端口 | 账号 / DB |
|---|---|---|
| Postgres | 55432 | sub2api / sub2api / db=sub2api |
| Redis | 56379 | 无密码 |

端口刻意用非默认（55432/56379 而非 5432/6379）避开系统已有的数据库冲突。

### 2.2 Backend

```bash
cd backend
go build -o bin/server ./cmd/server
# 首次: 见 §3.1
./bin/server
```

**默认 server port 是 18080（非 8080）**，对应 `backend/config.yaml` 里的 `server.port`。

### 2.3 Frontend

```bash
cd frontend
pnpm install
pnpm dev
```

在 `frontend/.env.local` 里配（不提交）：
```
VITE_DEV_PROXY_TARGET=http://localhost:18080
VITE_DEV_PORT=13000
```

---

## 3. 踩过的坑

### 坑 1：CLI setup 向导不能 pipe stdin

`./bin/server -setup` 是个交互式向导。代码里的 `promptPassword`（`backend/internal/setup/cli.go`）每次都 `bufio.NewReader(os.Stdin)` —— bufio 会预读 4KB 到内部缓冲，函数返回时这块缓冲被 GC，**后续管道输入的答案会被吞掉，导致所有字段对不齐**。

**绕开：** 写一个临时 main，直接调 `setup.Install(cfg)` 传硬编码配置。跑完删除。

```go
// backend/cmd/devsetup/main.go （用完即删，不要提交）
package main

import (
    "log"
    "github.com/Wei-Shaw/sub2api/internal/setup"
)

func main() {
    cfg := &setup.SetupConfig{
        Database: setup.DatabaseConfig{Host: "127.0.0.1", Port: 55432, User: "sub2api", Password: "sub2api", DBName: "sub2api", SSLMode: "disable"},
        Redis:    setup.RedisConfig{Host: "127.0.0.1", Port: 56379, DB: 0},
        Admin:    setup.AdminConfig{Email: "admin@sub2api.local", Password: "SubAdmin@2026"},
        Server:   setup.ServerConfig{Host: "0.0.0.0", Port: 18080, Mode: "debug"},
        JWT:      setup.JWTConfig{ExpireHour: 24},
        Timezone: "Asia/Shanghai",
    }
    if err := setup.Install(cfg); err != nil {
        log.Fatal(err)
    }
}
```

`setup.Install` 会测 DB/Redis 连通、跑 migrations、建 admin、写 `config.yaml` 和 `.installed` 锁。这两个文件都在 `.gitignore` 里。

### 坑 2：Migration checksum 不是文件原始内容的 sha256

`backend/internal/repository/migrations_runner.go` 里：

```go
content := strings.TrimSpace(string(contentBytes))
sum := sha256.Sum256([]byte(content))
```

**先 TrimSpace 再 hash**。所以 `shasum file.sql` 算出来的值**不等于**数据库里存的 checksum。要比对时用 Go 或 Python：

```python
content = open(f, 'rb').read().decode('utf-8').strip()
hashlib.sha256(content.encode('utf-8')).hexdigest()
```

### 坑 3：合并 upstream 时 migration checksum 爆炸

上游偶尔会修改 **已发布** 的 migration 文件内容（例：`109_auth_identity_compat_backfill.sql` 被拆分，部分 DDL 移到 `108a` / `121`），这违反迁移不可变原则，但他们会在 `migrations_runner.go` 的 `migrationChecksumCompatibilityRules` 白名单里配双 checksum 做兼容。

我们的 DB 如果曾用更旧版本的 upstream 建库，**DB 里存的 checksum 可能两个白名单值都不在**，启动会硬拒。

**处理方案：**

1. **对齐 checksum（推荐，dev/prod 都适用）：** 只要 upstream 那些修改都是"把 DDL 挪到后续幂等 migration"的形式，就只需要对齐 `schema_migrations` 表的 checksum 字段：

   ```sql
   UPDATE schema_migrations SET checksum = '<file_checksum>' WHERE filename = '<migration_name>';
   ```

   file_checksum 的正确算法见 §坑 2（TrimSpace + sha256）。或直接看镜像启动日志里的 `file=<hash>` 值。

2. **清库重来（仅限 dev）：**
   ```bash
   docker exec <postgres_container> psql -U sub2api -d sub2api \
     -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
   ```
   重启 backend 会按镜像 / 代码里的 migrations 从头建。

**判断哪种安全的准则：** 看 upstream 对该 migration 的 diff。如果只是把 DDL 挪到新的幂等 migration（`CREATE ... IF NOT EXISTS` / 带条件的 `DO $$ ... END $$`），对齐 checksum 是安全的。如果真改了 schema 逻辑，得清库或手工补 DDL。

### 坑 4：`users_signup_source_check` CHECK 约束需要随新来源扩展

`108_auth_identity_foundation_core.sql` 里的约束只允许 `('email', 'linuxdo', 'wechat', 'oidc')`。新增 signup_source 值（如 cards_issue 用户）必须写新 migration 扩展：

```sql
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_signup_source_check') THEN
        ALTER TABLE users DROP CONSTRAINT users_signup_source_check;
    END IF;
    ALTER TABLE users ADD CONSTRAINT users_signup_source_check
        CHECK (signup_source IN ('email', 'linuxdo', 'wechat', 'oidc', 'cards_issue'));
END $$;
```

我们的 `119_cards_issue_signup_source.sql` 就是这个。**这个问题单测抓不到**（stub adminService 绕过了真实 INSERT），只有真实 DB 调用才会暴露 500。

### 坑 5：docker-compose 端口绑定的 IP 前缀

```yaml
ports:
  - "127.0.0.1:55432:5432"   # 只绑 loopback，跨机器不可见
  - "0.0.0.0:55432:5432"     # 绑所有接口（明确）
  - "55432:5432"             # 默认等价于 0.0.0.0（更简洁）
```

如果依赖容器跑在 A 机器，backend 跑在 B 机器，绑了 `127.0.0.1:` 的 A 机器端口从 B 机器连不到（`Connection refused`）。

### 坑 6：远程 Docker 部署时 config.yaml 不能写 127.0.0.1

backend 容器里的 `127.0.0.1` 是容器内部，不是宿主机。如果 DB/Redis 在别的宿主机（或者哪怕在同一宿主的另一个容器里），config.yaml 得写那台机器的 LAN IP，不能 `127.0.0.1`。

### 坑 7：`setup.NeedsSetup()` 依据两个文件

- `<DATA_DIR>/config.yaml` 存在 → setup 已完成
- `<DATA_DIR>/.installed` 存在 → setup 已完成（防重装）

DATA_DIR 查找顺序：
1. `DATA_DIR` 环境变量
2. `/app/data`（Docker）
3. `.`（当前目录）—— 本地 `./bin/server` 时 config.yaml 落在 `backend/`
4. `./config`
5. `/etc/sub2api`

Docker 镜像需要挂 `/app/data` 做持久化，否则每次容器重启都会触发 setup 向导（开 HTTP wizard 在 8080）。

---

## 4. 合并冲突高发文件

与上游合并时下列文件几乎总有冲突（都是 struct 字段 / slice 追加型，容易解）：

- `backend/cmd/server/wire_gen.go` —— Wire 自动生成，改完手动 wire.go 后 `go generate` 重跑
- `backend/internal/handler/handler.go` —— AdminHandlers / Handlers struct
- `backend/internal/handler/wire.go` —— ProvideAdminHandlers 参数列表 + ProviderSet
- `backend/internal/server/routes/admin.go` —— 路由注册
- `frontend/src/api/admin/index.ts` —— adminAPI barrel export
- `frontend/src/components/layout/AppSidebar.vue` —— 侧边栏菜单
- `frontend/src/i18n/locales/{en,zh}.ts` —— 翻译 key（上游改得最频繁，一轮合并动二三十次）
- `frontend/src/router/index.ts`

**减少冲突面的做法：** 新功能尽量放独立文件（如 `cards_issue*.go`、`routes/custom.go`、`CardsIssueView.vue`），仅在必要的装配点（wire、router 注册、i18n）动已有文件。

---

## 5. 卡券发放功能（Cards Issue）

### 接口概览

- 对外：`POST /api/custom/cards/issue`，Bearer Key 认证（与管理员 JWT 隔离），挂在 `routes/custom.go` 避开版本化路由
- 管理后台：`/api/v1/admin/cards-issue/{config,key/regenerate,key}`

### 业务语义

- 按 `buyer_id` 查找或创建用户，按 `order_amount × order_quantity` 充值
- 新用户：确定性登录邮箱 `buyer_<sha256(buyer_id)[:10]>@cards-sync.invalid` + 16 字节随机密码（crypto/rand），**只在首次返回**
- 已存在用户（通过 `buyer_id` binding 查找命中）：响应里 `login_email` / `password` 留空，避免泄露
- 基于 `order_id` 幂等（默认 TTL 2h），重放响应附 `X-Idempotency-Replayed: true` 并把 password 脱敏为 `***`

### 输入上限（防爆）

定义在 `backend/internal/service/domain_constants.go`：
- `CardsIssueMaxOrderAmount = 1_000_000`（单价上限）
- `CardsIssueMaxOrderQuantity = 10_000`（数量上限）
- `CardsIssueMaxRechargeAmount = 1_000_000`（总额上限）
- 同时拒绝 NaN / Inf

### 关键文件（隔离分布）

- `backend/internal/service/cards_issue.go` —— `IssueOrder` 主流程
- `backend/internal/service/cards_issue_settings.go` —— Bearer Key / 模板配置
- `backend/internal/repository/cards_issue_binding_repo.go` —— `buyer_id` 绑定存储（复用 UserAttribute 机制）
- `backend/internal/handler/cards_issue_handler.go` —— 对外端点
- `backend/internal/handler/admin/cards_issue_handler.go` —— 管理端
- `backend/internal/server/routes/custom.go` —— 独立路由文件
- `backend/migrations/119_cards_issue_signup_source.sql` —— CHECK 约束扩展
- `frontend/src/views/admin/CardsIssueView.vue` + `src/api/admin/cardsIssue.ts`

---

## 6. 端口约定速查

| 项 | 端口 |
|---|---|
| Backend dev | 18080 |
| Frontend dev（pnpm dev） | 13000 |
| Postgres（deps compose） | 55432 |
| Redis（deps compose） | 56379 |
| Backend release 默认 | 8080 |

故意全走非默认以避开系统级 5432/6379/8080 的冲突。

---

## 7. 不要提交的本地文件

都已在 `.gitignore`，但新人容易手滑：

- `backend/config.yaml` —— 包含 DB/Redis/JWT 密码
- `backend/.installed` —— 安装锁
- `backend/bin/` —— 编译产物
- `backend/cmd/devsetup/` —— 临时绕开 setup 向导的 helper（用完即删）
- `frontend/.env.local` —— 本地 vite env
- `frontend/node_modules/`
