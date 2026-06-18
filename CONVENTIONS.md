# tracelab — 后端工程约定

本项目的代码由 AI 与人协作生成。这份约定是**硬规矩**，每次写后端代码前先读它。
目标：避免上一代 `aethertrace` 的问题（`viewer.go` 2400 行、cc/codex 揉在一起、
解析靠 hard code）。

## 1. 分层与依赖方向

```
event  ←  source      (采集层：产出 event.SourceEvent)
event  ←  normalize   (规范化层：消费 SourceEvent，产出 NormalizedEnvelope)
event  ←  sink        (落地：写 SourceEvent + NormalizedEnvelope)
```

- **`internal/event` 是解耦边界，不依赖任何其他内部包**，更不含 provider 语义
  （不出现 "anthropic"/"openai"/"codex" 等字样）。
- `source` 不 import `normalize`，`normalize` 不 import `source`。两者只共享 `event`。
- 依赖只能从外层指向 `event`，禁止反向或横向耦合。

## 2. provider 独立实现，只共享原子 helper

- 每个 provider（anthropic / openai-chat / openai-responses）一个**独立子包**，
  实现共同的 `normalize.Normalizer` 接口，**实现之间不互相 import**。
- 公共逻辑只允许是**原子、无状态的 helper**（token 估算、SSE 行解析、JSON 取值），
  放 `normalize/shared`。一旦某个 helper 里出现 `if provider == ...` 分支，就是设计错误，
  拆回各自实现。
- 新增一个 provider = 新增一个子包 + 在 registry 注册，**不改动**已有 provider 代码。

## 3. 文件与函数规模（防止再出现巨石文件）

- 单个 `.go` 文件 **≤ 400 行**（测试文件 ≤ 600 行）。超了就按职责拆文件。
- 单个函数 **≤ 80 行**。复杂解析拆成命名清晰的小函数。
- 一个文件只放一个主要职责；类型定义、解析、注册分文件。

## 4. 解析必须结构化，禁止关键字判断

- content block 的类型（text / reasoning / tool_call / tool_result）一律来自 provider
  返回 JSON 的**类型字段**，不允许"搜关键字猜类型"。
- 结构无法确定的语义信号（如 compaction / subagent），输出时必须带
  `Confidence` 且 `Suspected=true`，由上层决定怎么展示——绝不假装确定。

## 5. 错误与数据保真

- 解析失败不 panic、不丢数据：保留 `RawArtifact` 原始字节，错误写进 `event` 的 error 字段。
- 二进制/非 UTF-8 用 base64，不做有损字符串转换。
- 敏感 header（Authorization、Cookie 等）落盘前脱敏；脱敏策略可配置并标注。

## 6. 测试

- 每个 normalizer 子包配 golden 测试，输入用 `testdata/` 的真实 trace。
- 解耦验证：构造 hook + http 两种来源，断言产出同构 `SourceEvent`，下游消费代码一致。
- `go test ./...` 必须全绿；提交前 `go vet ./...`。
