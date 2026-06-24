# server（服务层）

> `internal/server`

## 为了做什么

组装根：把采集代理、hook 端点、规范化、存储、Viewer API 拼成一个本地 HTTP 服务。它本身
不含业务解析逻辑——只做路由编排和把各层接起来。

## 基本流程

**装配**（`server.go` `New(sink, reg)`）挂载基础端点：

- `/s/…` 反向代理（capture）、`/_hook` hook 接收、`/_register` 注册 session、`/healthz`。
- `emit()`：每个事件先 `reg.Normalize` 再 `sink.Write`（规范化失败也不丢，错误记进 record）。

**API**（`api.go` `EnableAPI(store)`，仅 SQLite 模式）挂 `/api/*`：session 列表、搜索、
facets、事件详情、复盘 case（list/run/snapshot/overwrite/curl）、active-sessions、launch、
codex-desktop、compaction episodes。`EnableViewer` 把前端 dist 挂到 `/viewer`。

**几块值得知道的逻辑**：
- **复盘 / cases**（`cases_api.go` + `replay.go`）：选一个 case + 一个 live session，真实重发
  到上游，响应走同一套 normalize 解析。auth 来自 session 内存，不另存。
- **compaction 检测**（`compaction.go`）：**跨事件分级**判定（confirmed / strong / weak），
  按证据强度而非关键字；读时实时算，`GET /api/sessions/{id}/compaction-episodes`。
- **launch**：见单独的 [launch.md](launch.md)（安全敏感）。

## 关键文件

- `server.go`：装配 + `emit` + `/_register` / `/_sessions`。
- `api.go`：`/api/*` 路由 + active-sessions（含 re-attach 的 needs_key）。
- `cases_api.go` / `replay.go`：复盘库运行。
- `compaction.go`：分级 compaction episode。
- `agent_providers.go`：cc/codex 启动/注入差异的单一来源（见 launch.md）。

## 安全

- 启动监听默认 `127.0.0.1`，不对外。
- **会触发本地进程执行 / 凭证注入的写接口都有同源校验**（`sameOriginOK`），挡浏览器侧
  CSRF / DNS-rebinding 偷触发。
- 复盘 case 是**真实计费请求**，前端有二次确认；curl 导出默认打码。
- 详见 [launch.md](launch.md) 与 [`SECURITY.md`](../SECURITY.md)。
