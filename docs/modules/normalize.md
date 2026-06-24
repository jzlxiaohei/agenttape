# normalize（规范化层）

> `internal/normalize`（+ `anthropic` / `openai` / `shared` / `providers` 子包）

## 为了做什么

把采集层产出的原始 `SourceEvent`（一坨厂商各异的 JSON）解析成**统一的、provider 无关
的** `NormalizedEnvelope`：有类型的 content block（text / reasoning / tool_call /
tool_result / error）、token 用量、各 section 占比、以及 compaction/subagent 这类**带置信
度的信号**。让上层（存储、Viewer）不必关心 cc 和 codex 的 wire 差异。

## 基本流程

```
SourceEvent ──► Registry.Pick(ev) ──► Normalizer.Normalize(ev) ──► NormalizedEnvelope
```

- `Registry` 持有一组 `Normalizer`；`Pick` 按事件特征选中匹配的那个（`normalize.go`）。
- 每个 provider 是**独立子包**，各自实现 `Normalizer` 接口，**实现之间不互相 import**。
- 公共逻辑只能是 `shared` 里**原子无状态的 helper**（token 估算、SSE 行解析、JSON 取值）。
  一旦某个 helper 里出现 `if provider == …`，就是设计错误，拆回各自实现。

## 关键文件

- `normalize.go`：`Registry` / `Normalizer` 接口 / `Pick` / `Normalize`。
- `providers/providers.go`：注册内置 provider（anthropic、openai-responses、openai-chat）。
- `anthropic/` `openai/`：各 provider 的解析实现。
- `shared/`：跨 provider 的原子 helper。

## 约定（务必遵守）

- **结构化解析，禁止关键字猜类型**：block 类型一律来自 JSON 的类型字段。
- 结构无法确定的语义信号必须带 `Confidence` + `Suspected=true`，由上层决定怎么展示，
  绝不假装确定（见 [`CONVENTIONS.md`](../../CONVENTIONS.md) §4）。
- **新增一个 provider = 新增一个子包 + 在 registry 注册**，不改动已有 provider 代码。
- 每个子包配 `testdata/` 真实 trace 的 golden 测试。

> 注：这里的 provider 是**解析 wire 格式**的维度（anthropic / openai-responses…）。启动/
> 注入凭证那条线的 provider 抽象在 `internal/server/agent_providers.go`，是另一回事。
