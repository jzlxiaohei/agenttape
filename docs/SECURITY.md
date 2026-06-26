# agenttape — 安全设计 / Security design

> 面向用户与贡献者的安全说明。agenttape 会经手**真实的模型 API 凭证**（订阅登录态或
> API key），所以"凭证去了哪、什么落了盘、谁能读到"必须写清楚。本文档是这些不变量的
> 单一事实来源；改动凭证/持久化相关代码时请同步更新。最后更新：2026-06-24。

This document describes how agenttape handles credentials. It is written for users
deciding whether to trust the tool, and for contributors who must preserve these
invariants. The normative invariants are in **§2**.

---

## 1. 它是什么 / 威胁模型

agenttape 是一个**纯本地**工具：它在 `127.0.0.1` 上起一个反向代理，把你本机的 coding
agent（Claude Code / Codex）↔ 模型 API 之间的流量抓下来，供你审视、重放、做实验。

- 没有服务端、不向第三方上报、不做遥测外发。
- 唯一的网络出口是**你的 agent 本来就要打的那个上游**（api.anthropic.com /
  chatgpt.com / api.openai.com）。
- 数据库（SQLite）和抓到的原始字节都在你指定的 `-data` 目录里。

"纯本地"消掉的是**网络侧**威胁（无服务端拖库、无传输截获）。它**挡不住**同机其它进程
读文件、备份/同步把文件带离本机、以及"把数据文件当调试产物分享出去"。后两类正是凭证
落盘的主要风险面——所以 agenttape 的设计原则是**凭证永不落盘**（§2）。

---

## 2. 核心不变量（改代码勿破）

1. **凭证永不写盘。** 真实 auth（订阅 token、API key、捕获到的请求头）只存在于
   **进程内存**，随进程退出而消失。SQLite 库、`raw/` 原始字节、任何导出文件里都不含
   可用凭证。
2. **API-key 模式下，key 不离开 agenttape 进程。** 用户把 key 交给 agenttape 后，agent
   被启动时只拿到一个**占位符**（`agenttape-proxy-placeholder`）。代理在转发时把占位符
   换成内存里的真 key。key 不进 agent 进程、不进终端、不进 shell history、不落盘。
3. **持久化的 session 路由是非密的。** 为了让 agent 扛过 agenttape 重启，`live_sessions`
   表只存路由事实（id、token、upstream、provider、mode），**不含任何凭证**。token 只是
   一个路由句柄（`/s/<token>/…`），不是 secret——它本身不能向上游认证。
4. **重放复用内存凭证，不另存。** 重放一条捕获请求时，auth 取自所选 session 的内存
   头；没有内存凭证就直接报错，而不是从盘上找。

任何新功能/新 provider 都必须保持这 4 条。

---

## 3. 两种凭证模式

agenttape 启动 agent 走代理时，凭证有两种来源：

### 3.1 订阅模式（subscription）

agent 用**它自己已登录的账号**（如 `claude login` 的 OAuth 态）。每次请求 agent 自带
真实 `Authorization`，代理只是**透传**，不持有也不替换任何凭证。

- 真凭证**始终在 agent 手里**，agenttape 从不持有。
- 代理会把每次请求的头（含 auth）记进**内存** `headers`，仅供重放复用——内存、不落盘。

### 3.2 API-key 模式（key）

用户在 agenttape UI/CLI 里提供一个真实 API key。

- 真 key 进 agenttape **内存** `inject`；启动 agent 时给它的环境变量是占位符。
- 代理转发时：删掉 agent 带来的占位符头，换成内存里的真 key（`proxy.go`）。
- **代价**：真 key 只在内存。agenttape 重启后它消失（见 §4）。这是不变量 §2.2 的必然
  结果——要让它扛过重启，只能落盘（破坏 §2.1）或把真 key 直接交给 agent（破坏 §2.2），
  两者都不做。

> 这个**故意的非对称**很重要：订阅模式重启后能自动恢复（真凭证在 agent 手里），
> API-key 模式重启后必须**重新输入一次 key**（真凭证只在 agenttape 内存里，已随重启消失）。

---

## 4. 跨重启的 session 重连（re-attach）

抓包/重放要看到改动就得重启 agenttape，但重启会清空内存里的 session 注册表——已经起好的
agent 会拿到 `502 unknown session token`。为减少这个摩擦，agenttape 持久化**非密**的
session 路由（`live_sessions` 表），启动时回填代理注册表（`Sessions.BindPersister`）。

| 模式 | 重启后自动恢复？ | 为什么 |
|---|---|---|
| 订阅 | ✅ 全自动 | 路由回填 + agent 自带真凭证透传 → 立即可用；它的头被重新记入内存,重放也恢复 |
| API-key | ⚠️ 半自动 | 路由回填(token 能路由了),但真 key 没落盘 → `NeedsKey`；用户在 UI 重输一次 key,即注入内存,agent 无感恢复 |

关键点：**`live_sessions` 里没有一行是凭证**。token 能路由不等于能认证——上游仍要真
auth，订阅模式由 agent 提供、API-key 模式由用户重输提供。即使这张表/整个库泄露，攻击者
拿到的也只是"打哪个上游、用哪个本地路由路径"，无法冒充你向上游发请求。

重输 key 的接口 `POST /api/active-sessions/{id}/key` 把 key 直接放进内存 `inject`，
**不写盘**，路径与首次启动完全一致。

---

## 5. 网络与本地访问面

- **默认只听 `127.0.0.1`**（`serve.go` 默认 `127.0.0.1:8787`）。不监听公网。
- **会触发本地进程执行/凭证注入的写接口有同源校验**（`sameOriginOK`）：带 `Origin`
  的跨站请求被拒，防 CSRF / DNS-rebinding 从浏览器偷触发 launch / 注入 key。
- **curl 导出**：`direct` 模式（直连含 auth）默认**打码**，需显式 reveal；`proxy` 模式
  打 agenttape 代理、**不含密钥**。
- 抓到的**请求/响应头在入库前会脱敏**（`redactHeaders`）——库里的头是打码版，真值只在
  内存供重放。

仍需注意（本地工具的固有面，非 agenttape 特有）：数据目录是普通文件，**同用户的任何进程
都能读**；若被 Time Machine / iCloud / Dropbox / git 纳入，会被带离本机。库里**不含凭证**
正是为了让这种扩散不等于凭证泄露。

---

## 6. 内置 seed 的脱敏

复盘库的内置 case（`internal/store/seeds/*.json`）保留**真实捕获的协议形状**，但请求内容
使用合成任务或最小化样例，不保留真实项目会话。遥测、installation_id、session_id、
workspace 路径和 git hash 会被删除；示例邮箱与路径统一使用明显虚构的值。详见
`REPLAY_LIB.md` §4。

> 为什么单列：agenttape 的数据文件**默认会被分享/开源**（你正在做的就是这件事）。在一个
> "会被分享"的产物里塞凭证是经典泄露模式——所以 seed 脱敏 + 凭证不落盘是同一条原则的
> 两面。

---

## 7. 新增 provider 的安全清单

目前只实现 cc（Anthropic）和 codex（OpenAI）。provider 相关的差异集中在一处：
`internal/server/agent_providers.go` 的 `agentProviders` 表。新增一个 provider（如
Gemini）= 往表里加一条 `agentProvider`，填：

- `Kind` / `Client` / `Provider`：UI 选择器、capture client id、normalize/wire id。
- `KeyEnv`：该 CLI 读 key 的环境变量名（API-key 模式给它占位符用）。
- `SubURL` / `KeyURL`：订阅态与 API key 各自的上游（两者常不同，弄错会 401）。
- `injectAuth(key)`：把真 key 变成上游要的 auth 头（这是该 provider key→header 形状的
  **唯一定义处**，重输 key 的接口也复用它）。

加完后，launch / 注入 / 重连三条路径都自动支持，**无需**在别处加 `if provider == …`。

**任何新 provider 必须保持 §2 的 4 条不变量**，尤其：

1. API-key 模式给 agent 的必须是**占位符**，真 key 只进内存 `inject`。
2. 不要把 key 写进 `live_sessions` 或任何持久化结构。
3. `injectAuth` 只构造 auth 头，不做任何落盘/外发。

> 注意：除了这张表，"复制粘贴启动命令"的模板（`buildManualEnvCommand` / `manualCommand`）
> 是**结构性 provider 相关**的（cc 用 base-url 环境变量，codex 用 `-c` TOML 覆盖），新
> provider 还需为它补一段命令模板。这部分无法只靠表驱动，已知。

---

## 8. 非目标 / 已知局限

- **不防御已被攻陷的本机/同用户进程。** 凭证在内存、数据文件可被同用户读——目标是"不把
  凭证写成持久明文产物 / 不外发"，不是抵御本机 root 或 keylogger。
- **API-key 模式不扛重启**（设计取舍，§3.2/§4）。想要免重输，可改走 OS keychain（把真
  key 放系统钥匙串、库里只存引用）——尚未实现，欢迎 PR，但要保持 §2.1"明文不落盘"。
- **`live_sessions` 是非密路由，但仍是元数据**：它暴露你打过哪些上游、起过哪些会话。
  介意的话，关闭 session（UI 的关闭按钮）会同时删掉这条持久路由。

---

## 报告安全问题 / Reporting

发现可能的凭证泄露或绕过上述不变量的问题，请优先私下联系维护者，不要直接开公开 issue 附
复现细节。
