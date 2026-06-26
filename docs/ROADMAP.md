# AgentTape — 设计原则与方向

这份文件记录 AgentTape 的**设计原则、明确不做的范围和后续方向**。已实现的能力见
[`../README.md`](../README.md)，工程硬约束见 [`../CONVENTIONS.md`](../CONVENTIONS.md)，
Replay 细节见 [`REPLAY_LIB.md`](REPLAY_LIB.md)，安全边界见 [`SECURITY.md`](SECURITY.md)。

## 关键设计原则

- **原始证据优先。** 归一化便于比较，但不会替代原始请求与响应。
- **关联依赖结构与时序。** 工具生命周期按 ID 和因果顺序关联，不用文本关键词猜测。
- **不确定性显式分级。** Compaction 等跨事件结论区分确认、强疑似和弱疑似。
- **凭证不落盘。** 捕获头、注入 Key 和重放认证只保存在进程内存；细节以
  [`SECURITY.md`](SECURITY.md) 为准。
- **Replay Case 是实验素材，不是自动化评测任务。** Session 提供上游与凭证，Case 只保存
  请求形状和路由元信息。

## 明确暂不做

- 不把 Replay Library 扩成完整的评测 / 自动化平台：暂不做断言、批量运行、评分矩阵和
  能力回归检测。这里统一用“能力回归”或“行为回归”描述可观察结果，避免含混、拟人化的
  价值判断。只有产品定位改变时才重新评估。
- 不用单条 Replay Case 冒充完整 harness 实验。权限、Compaction、子 Agent 调度等能力
  必须结合多次请求与 Hook 观察。
- 不以最快支持最多客户端为目标；当前优先把 Claude Code / Codex 的行为链路做深。

## 后续方向（想法，不是承诺）

1. 在后端物化 turn 与 tool span，而不只依赖前端分组。
2. 建立逐轮 Context Ledger，展示消息增删、摘要引入、工具目录变化与 Token 增量。
3. 完善工具生命周期：权限、失败、重试、结果是否进入下一请求、工具输出 Token 成本。
4. 为选中轮次生成确定性的 “Explain this turn”，每一行都链接回 Trace 证据。
5. 继续深化 Compaction / Subagent episode，展示父子关系、保留内容和上下文变化。
