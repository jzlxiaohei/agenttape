# launch（启动 agent）

> `internal/server/launch_api.go` · `internal/server/codex_desktop_api.go` ·
> `internal/server/agent_providers.go` · `internal/launcher`

## 为了做什么

把一个 coding agent（Claude Code / Codex）**通过 agenttape 代理**拉起来，这样它和模型之间
的流量就会被自动捕获；可选地把 hook 也注入进去，让 harness 事件一并进时间线。

## 基本流程

1. **注册 session**：拿到一个 `token` 和对应上游（`Sessions.Register`）。
2. **构造启动方式**，把 agent 指向 `…/s/<token>/`：
   - cc：`ANTHROPIC_BASE_URL` 环境变量；codex：`-c model_providers.*` 覆盖。
   - 完整捕获（含 hook）走 `agenttape launch` CLI（`internal/launcher`）；轻量只抓 HTTP 的
     走"自己运行"的复制粘贴命令。
3. **运行**：服务端在新终端窗口起进程（macOS），或用户自己粘命令跑。
4. 两种凭证模式：**订阅**（用 agent 已登录的账号）/ **key**（用户给 key）。

**codex 桌面版**（`codex_desktop_api.go` + `launcher/codex_desktop.go`）是特例：桌面版没有
按会话路由的办法，只能**改写全局 `~/.codex/config.toml`**——所以会先备份原文件、结束后恢复。

## 关键文件

- `launch_api.go`：launch / manual-command / re-inject key 等 HTTP 处理。
- `agent_providers.go`：**cc/codex 全部差异的单一来源**（client、provider id、key 环境变量、
  订阅与 key 各自上游、key→auth 头）。加新 provider 就加一条。
- `launcher/launcher.go`：构造 `agenttape launch` 实际跑的命令（非侵入）。
- `launcher/codex_desktop.go`：全局 config.toml 的合并/标记。

## 安全（本仓库最敏感的一块）

- **服务端会起本地进程**：默认开启但受 `-allow-launch` 控制、仅 macOS、工作目录校验、且
  有 `sameOriginOK` 同源校验防浏览器偷触发。关掉后只给复制粘贴命令、不执行任何东西。
- **API-key 模式：真 key 只进内存**。启动给 agent 的是占位符
  `agenttape-proxy-placeholder`，代理转发时才换成真 key——key 不进 agent 进程、不进终端
  历史、不落盘。重启后 key 消失，需在 UI 重输（routing 会自动恢复，见 capture.md）。
- **codex 桌面版改写全局配置**：这是桌面版唯一的路由方式；务必保留"先备份 + 结束恢复"，
  且只动路由、**不碰 `~/.codex/auth.json`**，仅订阅模式。
- 加新 provider 必须保持上述不变量（占位符、key 不落盘、injectAuth 只造头）。

完整清单见 [`SECURITY.md`](../SECURITY.md) §3/§4/§7。
