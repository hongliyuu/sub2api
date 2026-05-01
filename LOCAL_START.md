# 本地启动指南

本文档记录 sub2api 在 Windows 本地环境中的启动方式、端口约定和常见问题处理。

## 一、推荐方式：一体化启动

一体化启动会先构建前端，然后用 Go 的 `embed` 构建标签把前端页面嵌入后端服务。启动后只需要访问一个地址：

```text
http://localhost:3000
```

### 1. 构建前端

```powershell
cd d:\WorkSpace\ai\sub2api\frontend
pnpm run build
```

构建成功后，前端产物会输出到：

```text
backend/internal/web/dist
```

### 2. 启动后端一体化服务

```powershell
cd d:\WorkSpace\ai\sub2api\backend
$env:DATA_DIR='d:\WorkSpace\ai\sub2api\backend'
go run -tags embed ./cmd/server
```

启动成功后，日志中应出现类似内容：

```text
Server started on 0.0.0.0:3000
```

### 3. 浏览器访问

```text
http://localhost:3000
```

如果登录接口返回 `401`，说明请求已经正常到达后端，只是账号密码认证失败或当前未登录，不是代理问题。

## 二、前后端分离启动

前后端分离适合频繁修改 Vue/TypeScript 代码的开发场景，因为 Vite 支持热更新。

### 端口约定

默认建议：

```text
前端：http://localhost:3000
后端：http://localhost:8080
```

前端 Vite 配置中默认代理目标是：

```text
http://localhost:8080
```

对应配置在：

```text
frontend/vite.config.ts
```

关键配置：

```ts
const backendUrl = env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080'
const devPort = Number(env.VITE_DEV_PORT || 3000)
```

### 1. 配置后端端口

如果使用分离启动，建议将 `backend/config.yaml` 配置为：

```yaml
server:
    host: 0.0.0.0
    port: 8080
    mode: release
```

### 2. 启动后端

```powershell
cd d:\WorkSpace\ai\sub2api\backend
go run ./cmd/server
```

### 3. 启动前端

另开一个终端：

```powershell
cd d:\WorkSpace\ai\sub2api\frontend
pnpm run dev
```

### 4. 浏览器访问

```text
http://localhost:3000
```

### 不改后端端口的临时启动方式

如果后端继续监听 `3000`，前端就不能也监听 `3000`。可以临时改前端端口和代理目标：

```powershell
cd d:\WorkSpace\ai\sub2api\frontend
$env:VITE_DEV_PORT='5173'
$env:VITE_DEV_PROXY_TARGET='http://localhost:3000'
pnpm run dev
```

然后访问：

```text
http://localhost:5173
```

## 三、首次安装向导

首次启动时，项目可能进入安装向导流程，相关接口包括：

```text
/setup/status
/setup/test-db
/setup/test-redis
/setup/install
```

安装完成后，如果日志出现：

```text
Service restart via exit only works on Linux with systemd
```

这不表示 Windows 不能启动项目。它的意思是：程序尝试通过 Linux systemd 自动重启服务，但 Windows 本地没有 systemd，所以需要手动停止旧进程，再重新启动后端。

一体化模式下重新启动：

```powershell
cd d:\WorkSpace\ai\sub2api\backend
go run -tags embed ./cmd/server
```

前后端分离模式下重新启动后端：

```powershell
cd d:\WorkSpace\ai\sub2api\backend
go run ./cmd/server
```

安装完成后，如果 `/setup/status` 返回：

```json
{
  "needs_setup": false
}
```

就不要再访问 `/setup/install`，直接访问首页即可。

## 四、常见问题

### 1. 首页返回 404

现象：

```text
404 page not found
```

常见原因是只启动了普通后端：

```powershell
go run ./cmd/server
```

普通后端不会内置前端页面。

解决方式：

```powershell
cd d:\WorkSpace\ai\sub2api\frontend
pnpm run build
```

```powershell
cd d:\WorkSpace\ai\sub2api\backend
go run -tags embed ./cmd/server
```

### 2. Vite 代理 ECONNREFUSED

现象：

```text
[vite] http proxy error: /api/v1/auth/login
AggregateError [ECONNREFUSED]
```

原因通常是 Vite 代理目标端口没有后端服务。

默认情况下：

```text
前端端口：3000
后端代理目标：8080
```

如果后端实际也跑在 `3000`，就会出现端口冲突或代理失败。

解决方式二选一：

1. 后端改成 `8080`，前端继续 `3000`。
2. 使用一体化启动，不启动 Vite dev server。

### 3. 安装完成后页面超时

如果提交 `/setup/install` 后页面超时，但 `/setup/status` 已经返回 `needs_setup: false`，通常说明安装已经完成，只是 Windows 无法通过 systemd 自动重启。

处理方式：

1. 停掉旧的后端进程。
2. 重新启动后端。
3. 直接访问首页。

### 4. 日志目录无权限

Windows 本地运行时可能出现：

```text
write /app/data/logs/sub2api.log: Access is denied.
```

这通常不会阻止服务启动，程序会降级输出到标准输出。

如果需要消除该警告，可以在本地配置中调整日志文件路径或关闭文件日志输出。

### 5. 端口被占用

检查端口占用：

```powershell
Get-NetTCPConnection -State Listen | Where-Object { $_.LocalPort -in 3000,8080,5173 } | Select-Object LocalAddress,LocalPort,OwningProcess
```

如果端口被占用，可以关闭对应进程，或者调整 `backend/config.yaml` / `VITE_DEV_PORT`。

## 五、常用命令速查

### 一体化启动

```powershell
cd d:\WorkSpace\ai\sub2api\frontend
pnpm run build
```

```powershell
cd d:\WorkSpace\ai\sub2api\backend
go run -tags embed ./cmd/server
```

访问：

```text
http://localhost:3000
```

### 前后端分离启动

后端：

```powershell
cd d:\WorkSpace\ai\sub2api\backend
go run ./cmd/server
```

前端：

```powershell
cd d:\WorkSpace\ai\sub2api\frontend
pnpm run dev
```

访问前端终端输出的地址。

### 构建一体化可执行文件

如果不想每次 `go run`，可以构建可执行文件：

```powershell
cd d:\WorkSpace\ai\sub2api\frontend
pnpm run build
```

```powershell
cd d:\WorkSpace\ai\sub2api\backend
go build -tags embed -o sub2api.exe ./cmd/server
```

运行：

```powershell
.\sub2api.exe
```
