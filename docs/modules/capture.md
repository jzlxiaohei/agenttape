# capture（采集层）

> `internal/source/httpcap` + `internal/source/hook`

## 为了做什么

把"coding agent ↔ LLM"之间真实发生的事抓下来，统一成 `event.SourceEvent`（原始、
provider 无关）。两个来源：

- **httpcap**：一个反向代理，agent 把请求打到 `…/s/<token>/<上游路径>`，代理转发到真正
  上游，同时把请求/响应原样录下来。
- **hook**：接收 agent harness 推来的生命周期/工具事件（PreToolUse、SessionStart、
  PreCompact…），让时间线上不止有 HTTP，还有"harness 在干什么"。

## 基本流程

**HTTP 抓取**（`httpcap/proxy.go`）：
1. 请求进 `/s/<token>/…` → 按 token 在 `Sessions` 注册表查到这条会话的上游。
2. 透传到上游（强制 `Accept-Encoding: identity`，拿未压缩的可读字节）。
3. 流式回写给 agent 的同时，留一份完整字节用于落库。
4. `emit` 一个 `KindHTTPExchange` 的 `SourceEvent`（含 req/resp 原始 artifact）。

**Hook 抓取**（`hook/hook.go`）：agent 配置成对 `POST /_hook?runtime=&event=&session=`
发事件；cc 和 codex 用同一种载荷形状，统一 `emit` 成 `KindHookEvent`。

## 关键文件

- `httpcap/proxy.go`：反向代理 + 事件构造。
- `httpcap/session.go`：`Sessions` 注册表（token↔上游、内存凭证、跨重启重连）。
- `httpcap/headers.go`：落库前的 header 脱敏。
- `hook/{hook,claude,codex}.go`：hook 端点 + 两种 runtime 的接入。

## 安全

这层**经手真实凭证**，原则是**只在内存、永不落盘**：

- 订阅模式：agent 自带 auth，代理纯透传，自己不持有凭证。
- API-key 模式：真 key 只在内存 `inject`，转发时替换掉 agent 携带的占位符——key 不进
  agent、不进磁盘。
- 落库前 header 经 `redactHeaders` 脱敏；真实头只在内存供重放复用。
- session 的**非密路由**（token/上游/provider/mode）会持久化以扛重启，但**不含任何凭证**。

完整不变量见 [`SECURITY.md`](../SECURITY.md)。
