# Replay Library — 设计与现状

> 模块定位：把"一条捕获到的模型请求"变成**可复用、可编辑、可重发**的素材，
> 围绕 6 个行为实验，沉淀出 Codex / Claude Code 两种 wire 格式的代表性请求形状。
> 最后更新：2026-06-24。

---

## 1. 它是什么

Replay Library（前端称"素材库 / Replay library"）是一组保存下来的请求 case。
每个 case = **一条发给上游的请求体** + 路由元信息。选一个 case、选一个 live
session（提供凭证与上游路由）、（可选）改 body、点 Run，就会真实重发并把响应
按抓包同一套 normalize 流程解析出来。

凭证**永不落盘**：case 只存请求体和 endpoint，auth 来自所选 session 的**内存**
headers，所以跑 case 需要一个**本进程内有凭证**的 live session。

> session 的**非密路由**（token/upstream/provider/mode）会持久化，让已起好的 agent
> 扛过 tracelab 重启（订阅模式自动恢复，API-key 模式需在 UI 重输一次 key——真 key 从不
> 落盘）。完整的凭证/持久化安全模型见 [`docs/SECURITY.md`](docs/SECURITY.md)。provider
> 相关差异（cc/codex，及如何加新 provider）集中在 `internal/server/agent_providers.go`。

---

## 2. 行为实验与单请求边界（重要约束）

一个 case 只能复现**一条 HTTP 请求**，而下面的实验大多跨多次模型请求 +
harness 编排（权限、diff 生成、compaction、子 agent 调度）。这些**编排逻辑无法
靠重放单个请求复现**。我们的做法是：把每个实验**定格成 agent loop 里最有代表性
的那一条请求**，做成 case。

| 实验 | 做成的 case | 能复现 | 不能复现 |
|---|---|---|---|
| 1 纯文本 | 带完整工具目录但要求"只回 hello" | 首请求形状 | 事件路径 |
| 2 读文件 | `tool_use`+`tool_result` / `function_call`+`output` 回灌 | tool result 如何进下一次请求 | PreToolUse 时序 |
| 3 改文件 | codex：`custom_tool_call`(apply_patch)；cc：结构化 `Edit`(old/new_string) | 两种改文件 wire 形状 | 权限 / diff 来源（harness 行为） |
| 4 执行失败 | `tool_result is_error` / 非零退出 output | 失败如何编码、模型如何重试 | 输出截断（harness 行为） |
| 5 compaction | `seed:cc-compaction`：按真实 /compact wire 形状合成的**触发请求**（要求模型把整段对话总结成 `<analysis>`/`<summary>`） | 压缩触发请求的形状 + `context_management`/adaptive thinking | 压缩后的"续接"请求、harness 何时自动触发 |
| 6 子 agent | —（无真实捕获，已跳过） | — | — |

### 六个实验的观察重点

1. **纯文本**：初始 Prompt 的组成、模型请求次数、`UserPromptSubmit` 到 `Stop` 的路径。
2. **读文件**：模型如何产生 Tool Call、`PreToolUse` 与网络响应的顺序、Tool Result 如何
   进入下一次请求。
3. **改文件**：Read / Edit / Write 的选择、写入前的权限事件、Diff 由模型还是 harness
   生成、修改结果如何回灌。
4. **执行失败**：`PostToolUseFailure`、stderr 编码、失败后的重试，以及 harness 是否截断
   过长输出。
5. **Compaction**：`PreCompact`、压缩前最后一次请求、`PostCompact`、压缩后第一次请求
   之间的内容与 Token 变化。
6. **子 Agent**：父子 Agent 的 System Prompt、工具集、History、工作目录是否隔离，以及
   子 Agent 最终向父 Agent 返回了什么。

实验 6 在现有捕获里仍**找不到任何真实 subagent 请求**，因此不构造 seed，改用动手
实验卡引导用户实跑。实验 5 的 `seed:cc-compaction` 保留真实 /compact 请求的协议形状，
但会话内容已替换为虚构的 Tideline CLI 设计讨论，不再包含 TraceLab 自身的分析记录。
注意它是 compaction 的**触发半**（"把对话总结掉"）。当前 episode 检测必须等压缩后的请求 B 到来，所以孤立 seed
不会被标成一次完整 Compaction；这是证据边界，不是漏标，见 §7。

### 工具类 seed 是「多步任务」，重放会吐出真实工具调用（2026-06-24）

最初的工具类 seed（cc-edit / cc-tool-read / cc-tool-failure / codex-apply-patch /
codex-tool-read）对话都**停在一个 `tool_result`/`function_call_output` 上**，且首条
指令是单步任务（"改 errorRate""总结 README"）。重放时模型接到"工具已成功、活也干完
了"的局面，唯一合理的下一步就是回一句"改好了"——**响应里没有任何工具调用**（那个
工具调用早烤进了请求体的 assistant 轮，不在响应里）。这让重放结果看起来很空。

现在改成**多步任务**：保留烤进去的 `tool_use`+`tool_result`（请求形状 / 结果回灌照样
能看），但把首条指令改成"逐个改完所有字段""读完 README 再读源码""列完目录再总结"
这类还没干完的活，并让 `tool_result` 留出线索（如 cc-edit 的代码片段里**故意留下仍是
驼峰的字段** `windowStart`/`p50Ms`/`maxLatency`）。于是模型在响应里会**自然再发一个
工具调用**（下一个 Edit / 再 Read 一个文件 / 再 exec 一条命令）。两全：请求里的回灌
形状 + 响应里的真实工具调用都能看到。

> 注意：seed body 是**编译期 `go:embed` 进二进制**的（`internal/store/cases.go`），改完
> JSON 必须**重新 `go build`** 才生效；运行时不读磁盘上的 JSON。重编译后重启即可——
> seedCases 按内容指纹自动刷新内置 case，无需手动 SQL（见 §3）。

---

## 3. 架构

依赖方向：`routes/ui → viewmodel → (query + store) → api`（前端 MVVM 规范）。

### 数据模型（`internal/store/cases.go`）

```
ReplayCase { ID, Name, Tags, Provider, Method, Target, Endpoint, Body, Source, CreatedAt }
```

- `Source`：`seed`（内置）｜ `captured`（从抓到的事件保存）｜ `snapshot`（编辑后另存的派生）｜ `manual`（手填）。
- `Endpoint`：session 相对路径（如 `/responses`、`/v1/messages`）。case 只拥有
  请求形状 + endpoint，**上游由 session 提供**（`Target` 仅作展示元信息）。
- 表：`replay_cases`（`internal/store/schema.sql`）。

### 后端流程

- **列表 / 增删**：`internal/server/cases_api.go`
  - `handleAddCase`：从捕获事件保存（读 `request_body` raw 文件），或手填（`handleCreateManualCase`，会猜 provider/endpoint）。
  - `handleSnapshotCase`：编辑后**另存为新 case**，原 case 不动。
  - `handleOverwriteCase`：编辑后**就地覆盖**同一 case（不可撤销，UI 二次确认）。
  - `handleDeleteCase`：**只删 added 类**；seed（内置）不可删（API 返回 403，UI 也不显示删除）。内置集由 §3 的内容 digest 刷新。
- **运行**：`handleRunCase` → `executeCaseThroughSession`（`internal/server/replay.go`）：
  走真实客户端同款 session 代理 `/s/<token>/<endpoint>`，session 注入 auth、
  决定上游，响应经 `reg.Normalize` 解析。非 2xx 时把原始 body 一并回传
  （`replayResp.ResponseBody`），避免错误显示成空白。
- **curl 导出**：`handleCaseCurl`（`internal/server/case_curl.go`）两种模式：
  `proxy`（打 tracelab 代理，不含密钥）/ `direct`（直连含 auth，默认打码）。

### 种子机制（内容指纹 / digest）

seed body 在**编译期**经 `//go:embed seeds/*.json` 烤进二进制（`cases.go`），运行时
**不读磁盘 JSON**。每次启动 `store.Open()` 调 `seedCases()`，它对整个 seed 集（每个
seed 的字段 + body）算一个 SHA-256 指纹，存在 `schema_meta.cases_seed_digest`：

- **指纹没变** → 直接返回，什么都不做。所以发布版二进制里内置集稳定，用户对内置 case
  的就地覆盖**跨重启保留**（旧 `TestSeedCasesOnce` 仍成立；内置 case 不能通过 API 删除）。
- **指纹变了**（改了 `seeds/*.json` 重新 `go build`，或新版本增删 seed）→ 用
  `INSERT OR REPLACE` 把**所有** seed 行按新 body 重装一遍，再写回新指纹。

> 因此**改 seed 的正确姿势**：改 JSON → `go build` → 重启。**不再需要手动 SQL**。

权衡（诚实写明）：指纹一变就**重装整套内置集**，所以用户就地覆盖过的内置 case
会在那次重编译后**回归（刷成最新）**。自建 case 永远不受影响——`INSERT OR
REPLACE` 只命中带命名空间的 `seed:*` id。旧的 `cases_seeded` 布尔标志已废弃，刷新时
会顺手 `DELETE` 掉。

### 前端

- `ui/CasesPanel.tsx`：**卡片画廊**——顶部 provider 过滤（`ProviderFilter`，状态在
  `store/ui.ts` 的 `casesProvider`），下方按 内置 / 后加 分组的卡片网格
  （`CaseCard`：provider icon + 标题 + 实验说明 + 核心字段 method/endpoint/model/
  tools/stream + source 徽标 + 删除）。点卡片在 Dialog 里打开 CaseRunner（选
  session、编辑 body、Run、覆盖、快照、curl、结果 Sheet）。
- `viewmodel/cases.ts`：`caseSections` 分组、`filterCasesByProvider` / `caseProviders`
  过滤、`caseProviderClient`（provider→品牌 icon）、`caseCardMeta`（从 body 抽
  model/tools/stream）、`caseDescription`（内置实验说明 i18n）、`caseEndpoint` /
  `caseRunURL` 派生、`caseDisplayName(c, t)` 把内置标题映射到 i18n（见 §5）。
- `query/cases.ts`：TanStack Query hooks（`useCases` / `useRunCase` / `useSnapshotCase`
  / `useOverwriteCase` / `useDeleteCase` / `useCreateCase` / `useCaseCurl`）。

---

## 4. 内置 seed 集（11 个）

文件在 `internal/store/seeds/*.json`，用 `//go:embed` 进二进制。

| seed id | provider | endpoint | 实验 | 文件 |
|---|---|---|---|---|
| `seed:codex-pure-text` | openai-responses | /responses | 1 | codex-pure-text.json |
| `seed:codex-tool-read` | openai-responses | /responses | 2 | codex-tool-read.json |
| `seed:codex-apply-patch` | openai-responses | /responses | 3 | codex-apply-patch.json |
| `seed:codex-tool-failure` | openai-responses | /responses | 4 | codex-tool-failure.json |
| `seed:cc-pure-text` | anthropic | /v1/messages | 1 | cc-pure-text.json |
| `seed:cc-tool-read` | anthropic | /v1/messages | 2 | cc-tool-read.json |
| `seed:cc-edit` | anthropic | /v1/messages | 3（结构化 Edit） | cc-edit.json |
| `seed:cc-tool-failure` | anthropic | /v1/messages | 4 | cc-tool-failure.json |
| `seed:cc-title` | anthropic | /v1/messages | 附赠：会话标题生成 | cc-title.json |
| `seed:cc-full-claude` | anthropic | /v1/messages | 完整请求形状（cache_control + adaptive thinking） | cc-full-messages.json |
| `seed:cc-compaction` | anthropic | /v1/messages | 5（compaction 触发） | cc-compaction.json |

> `seed:cc-compaction` 曾保留一整段分析 TraceLab 自身的真实对话。现已改为**合成会话**：
> 使用虚构的 Tideline 日志 CLI 作为任务，只保留 /compact 触发请求所需的消息序列、
> `context_management`、adaptive thinking、工具定义和总结指令。素材不再包含本项目文件、
> 本机路径、历史决策或带价值判断的评测措辞。

合并/取舍：
- 实验 2 与 3 在"单请求形状"上同构（都是工具调用 → 结果回灌）。cc 侧最初只留一个
  read 回灌；后应要求补了 `seed:cc-edit`（结构化 `Edit`）——它与 read 同构，留着是
  为了**显式展示"cc 怎么改文件"**，与 codex 的 `apply_patch` 对照。codex 侧 `apply_patch`
  是 **custom freeform grammar 工具**（形状确实不同），单列。
- **cc 没有 `apply_patch`**：它用结构化工具 `Edit`(old_string/new_string)/`Write`/
  `MultiEdit` 改文件，就是普通 `tool_use`+JSON input，wire 形状与任意工具调用一致。
- codex 无子 agent，实验 6 对 codex 不适用。

### seed 来源与脱敏（混合策略）

发现：一条 362B 的真实 codex 请求（无 `tools`、`instructions` 仅一句）也被
chatgpt backend 200 接受 → 说明 codex 的 `tools` 和长 `instructions` **可选**。
据此采用混合构造，兼顾"被 API 接受"与"体积 / 隐私"：

- **纯文本**：用已验证的最小真实形状（model/instructions/`input[input_text]`/
  `store:false`/`stream:true`/`include:[reasoning.encrypted_content]`）。
- **工具 / 失败 / apply_patch**：短 instructions/system + **从真实捕获提取的单个
  工具 schema**（codex `exec_command`/`apply_patch`；cc `Read`/`Bash`/`Edit`，
  均为厂商公开定义、非用户 PII）+ **手写的无害剧本**（读 README、跑 go test 等）。
- **cc-title**：直接取真实标题生成请求，替换会话内容 + 删 `metadata`。

每个 seed 约 1–9KB。**彻底脱敏**：丢弃 codex 的 `client_metadata`/`prompt_cache_key`
（含 installation_id、session_id、workspace 路径、git hash）和 cc 的
`metadata.user_id`（device_id/account_uuid）；路径统一 `/workspace`；真实邮箱 →
`example@example.com`。逐 token 复查无 `/Users/`、真实邮箱、姓名、`tracelab` 残留。
（`noreply@anthropic.com` 是 Bash 工具 schema 自带的公共地址，保留。）

> ⚠️ codex `chatgpt.com/backend-api/codex/responses` 是订阅路径；旧 seed 曾指向
> `api.openai.com/v1/responses`，对订阅账号必 401，且 body 过简会 400——已废弃。

---

## 5. i18n

内置标题不硬编码语言：DB 里 `Name` 存英文 fallback，前端 `caseDisplayName`
按 `cases.seed.<id>`（id 去掉 `seed:` 前缀、`-`→`_`）取 i18n，随语言切换。
key 在 `frontend/src/i18n/{en,zh}.json` 的 `cases.seed.*`。用户自建 case 用原名。

---

## 6. UI 行为约定

- **覆盖 / 另存快照**按钮**始终可见**，未编辑 body 时禁用（`cases.edit_to_enable`
  悬停提示）——避免"编辑前不渲染导致找不到"。
- Run 前二次确认（真实计费请求）。
- direct 模式 curl 默认打码，需显式 reveal。

---

## 7. 当前状态与剩余验证

### 已落地

- 实验 5（compaction）：补了 `seed:cc-compaction`（真实协议形状 + 合成会话，见 §4）
  **+** 一张动手实验卡（见下）——压缩的"前/后"完整故事跨多请求 + hooks，单 seed 看不全。
- 实验 6（子 agent）：**不做 seed**。子 agent 的精髓是 harness 编排（fork→跑→只回摘要），
  跨多请求 + hooks，单条可重放请求承载不了。改为**动手实验卡**：自包含 prompt（`pwd` +
  让子 agent 自报，任意目录可跑，不依赖源码——OSS 二进制用户也能跑），引导用户开 hooks
  实跑，在 flow 里看 SubagentStart/Stop。
- 实验卡 = replay lib 里**「实验」分组**下的特殊只读卡（不可 Run）。**前端静态**
  （`frontend/src/lib/experiments.ts` + `ui/ExperimentCard.tsx`，文案在 i18n `experiments.*`），
  不进 `replay_cases`、不需预抓真实数据——用户跑出来的 session 就是他自己的数据。
  目前两张：子 agent、compaction。
- **compaction 检测重做为分级跨事件 episode**（采纳同事 review）：
  `anthropic/signals.go` **不再产任何 compaction 关键词标签**（关键词会对"讨论
  compaction 的会话"假阳性）。改在 `internal/server/compaction.go` 按 session 跨
  事件判定，配对相邻 completion (A,B) 分级：

  - `confirmed`：A、B 之间有 PreCompact/PostCompact hook（harness ground truth）。
  - `strong_suspected`：历史大幅收缩 **且** A 的响应（摘要）被 B 的请求承接
    （内容血缘——证明数据从 A 输出流到 B 输入，区别于截断/重置/改写）。
  - `weak_suspected`：仅历史收缩。

  判定**必须等 B 到来**，故是跨事件 episode、不在 A 入库时打永久标签；
  `GET /api/sessions/{id}/compaction-episodes` 在读时实时算（旧 session 也适用）。
  前端 CompactionPanel 显示分级徽标（已确认/强疑似/弱疑似）。只有触发请求 A、没有
  后续 B 的孤立 case 不出面板——单看 A 本就无法确认。
- **改了 seed 旧库会自动刷新**（2026-06-24，原"旧库不会自动出现新 seed"已解决）：
  改为 `cases_seed_digest` 内容指纹（§3）。改 `seeds/*.json` → `go build` → 重启即自动
  重装，无需手动 SQL。代价：内置集一变就整套重装，用户对内置的就地覆盖会回归（§3）。

### 仍需 live 验证

- [ ] seed 是否被上游接受，需在 live session 下逐个 Run 验证（本进程无凭证无法自验）；
  形状均来自被 200 接受过的真实字段，置信度高，万一 400 错误体会显示在结果里。
- [ ] **工具类 seed 的多步措辞需 live Run 校准**：路线 A 靠 prompt 措辞 +「留个没改完的
  线索」诱导模型在响应里再发工具调用（见 §2）。措辞够不够硬要实跑才知道；若某 case
  仍只回文本，把"还没干完"挑得更明显（如 cc-edit 的 tool_result 里多留几个驼峰字段）。

---

## 8. 关键文件索引

```
internal/store/cases.go            数据模型、seedCases、EndpointForTarget
internal/store/seeds/*.json        11 个内置请求体
internal/store/schema.sql          replay_cases 表
internal/server/cases_api.go       list/add/snapshot/overwrite/delete/run handler
internal/server/replay.go          executeReplay / executeCaseThroughSession / buildReplayEvent
internal/server/case_curl.go       curl 导出（proxy / direct）
frontend/src/ui/CasesPanel.tsx     列表 + CaseRunner + CurlSheet
frontend/src/viewmodel/cases.ts    分组 / endpoint / caseDisplayName
frontend/src/query/cases.ts        TanStack Query hooks
frontend/src/api/cases.ts          DTO + fetch 封装
frontend/src/i18n/{en,zh}.json     cases.* 文案（含 seed.*）
```
