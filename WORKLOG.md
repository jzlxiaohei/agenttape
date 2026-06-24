# TraceLab — 当前状态与决策记录

这份文件记录**已经落地的能力、仍需验证的事项和明确不做的范围**。它不是把所有想法都
堆进来的任务清单。对外介绍见 [`README.md`](README.md)，工程硬约束见
[`CONVENTIONS.md`](CONVENTIONS.md)，Replay 细节见 [`REPLAY_LIB.md`](REPLAY_LIB.md)，
安全边界见 [`docs/SECURITY.md`](docs/SECURITY.md)。最后更新：2026-06-24。

## 已完成

### Capture 与归一化

- Claude Code 与 Codex 的 HTTP 反向代理捕获，以及两种 runtime 的 harness hooks。
- HTTP / Hook 统一落为 `SourceEvent`，Provider normalizer 独立解析 Anthropic Messages、
  OpenAI Responses 和 OpenAI Chat。
- Hook 使用接收时间戳；`tool_use_id` / `call_id` 与工具名进入关联链路。
- SQLite、原始字节、跨会话搜索、标签与请求组成 Token 估算。

### Viewer

- Hook-first Flow：Hook 是执行流主干，HTTP 请求通过 chip 打开侧栏证据。
- Context Diff：比较相邻请求的 System、Tools、Messages 与 Tool Results。
- URL 路由可分享、可刷新，Go 服务支持 SPA deep-link fallback。
- Compaction 按跨事件 episode 分级：Hook 证据为 `confirmed`，历史收缩加内容血缘为
  `strong_suspected`，仅历史收缩为 `weak_suspected`。
- Replay：可编辑并重发一条捕获请求，使用同一 normalizer 展示结果；运行结果不写回 Trace。

### Replay Library

- Case 持久化、从捕获保存、手工创建、就地覆盖、另存 Snapshot、删除自建 Case。
- 内置 11 个 Claude Code / Codex 请求 seed，并提供 compaction 与子 Agent 动手实验卡。
- Case 可绑定 live session 真实运行，或导出 Proxy / Direct 两种 cURL；Direct 默认隐藏凭证。
- Seed 使用内容 digest 自动刷新；修改 `internal/store/seeds/*.json` 后重新构建、重启即可，
  不再需要手动清理数据库标记。

### Launch 与 session 重连

- CLI / Viewer 启动 Claude Code、Codex CLI；支持订阅登录与 API Key 模式。
- API Key 只在 TraceLab 内存中，Agent 收到占位符，由代理转发时注入真实凭证。
- `live_sessions` 只持久化非密路由。重启后订阅模式自动恢复；API Key 模式需在 UI
  重新输入一次 Key。
- Codex Desktop 支持配置逐字节备份、临时代理注入和恢复；受 `-allow-launch` 控制。
- Viewer 的服务端启动当前默认开启，可用 `-allow-launch=false` 关闭；无论是否开启，
  页面都会提供可复制的手动命令。

## 关键决策

- **原始证据优先。** 归一化便于比较，但不会替代原始请求与响应。
- **关联依赖结构与时序。** 工具生命周期按 ID 和因果顺序关联，不用文本关键词猜测。
- **不确定性显式分级。** Compaction 等跨事件结论区分确认、强疑似和弱疑似。
- **凭证不落盘。** 捕获头、注入 Key 和重放认证只保存在进程内存；细节以
  [`docs/SECURITY.md`](docs/SECURITY.md) 为准。
- **Replay Case 是实验素材，不是自动化评测任务。** Session 提供上游与凭证，Case 只保存
  请求形状和路由元信息。

## 仍需验证或处理

- 在真实 macOS + Codex Desktop 中手动验证配置安装、一次性 Hook 信任与逐字节恢复。
- 在有凭证的 live session 下逐个运行内置 seed，校准多步工具任务是否稳定产生下一次
  工具调用。该项需要真实上游请求，可能计费。
- 发布前安全整改、依赖与发布物复扫以 [`security-audit/CHECKLIST.md`](security-audit/CHECKLIST.md)
  为准；2026-06-23 的报告是历史快照，不作为当前状态表。
- 低优先级：`Session` 类型目前仍在 `httpcap`，以后可移到跨 adapter 的 `internal/source`。

## 明确暂不做

- 不把 Replay Library 扩成完整的评测 / 自动化平台：暂不做断言、批量运行、评分矩阵和
  **能力回归检测**。这里统一用“能力回归”或“行为回归”描述可观察结果，避免含混、拟人化
  的价值判断。只有产品定位改变时才重新评估。
- 不用单条 Replay Case 冒充完整 harness 实验。权限、Compaction、子 Agent 调度等能力
  必须结合多次请求与 Hook 观察。
- 不以最快支持最多客户端为目标；当前优先把 Claude Code / Codex 的行为链路做深。

## 后续产品方向（想法，不是承诺）

这些想法从原独立的 Viewer 方向文档合并而来，集中放在这里，避免和当前任务混淆：

1. 在后端物化 turn 与 tool span，而不只依赖前端分组。
2. 建立逐轮 Context Ledger，展示消息增删、摘要引入、工具目录变化与 Token 增量。
3. 完善工具生命周期：权限、失败、重试、结果是否进入下一请求、工具输出 Token 成本。
4. 为选中轮次生成确定性的 “Explain this turn”，每一行都链接回 Trace 证据。
5. 继续深化 Compaction / Subagent episode，展示父子关系、保留内容和上下文变化。

## 运行与验证

```bash
go build -o ./tracelab ./cmd/tracelab
(cd frontend && npm run build)
go test ./...

./tracelab serve \
  -data ./tracelab-data \
  -listen 127.0.0.1:8787 \
  -viewer ./frontend/dist
```

打开 <http://127.0.0.1:8787/viewer/>。Replay / Case Run 会访问真实上游并可能计费。

## 常见误区

- 端口上残留的旧进程可能使用另一份数据目录，看起来像“数据丢了”；先确认正在访问的进程
  与 `-data` 参数。
- Seed 通过 `go:embed` 编进二进制；只改 JSON、不重新构建不会生效。
- API Key 模式重启后只恢复路由，不恢复 Key；这是凭证不落盘的设计结果。
