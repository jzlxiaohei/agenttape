# 模块指南 / Module guides

agenttape 的每个模块一页：**为了做什么** + **基本流程** + **关键文件**，涉及安全的地方
单独标一段。想动哪块代码，先读对应这页。深层约定见 [`CONVENTIONS.md`](../../CONVENTIONS.md)，
凭证/安全不变量见 [`SECURITY.md`](../SECURITY.md)，复盘库见 [`REPLAY_LIB.md`](../REPLAY_LIB.md)。

## 总览：一条数据的旅程

```
 coding agent ↔ LLM
      │  (反向代理 /s/<token>/…)            harness hooks (POST /_hook)
      ▼                                          │
  [capture] httpcap ──┐                          ▼
                      ├──►  event.SourceEvent  ◄── hook
  原始 HTTP 事实        │     (raw, provider 无关)
                      ▼
                 [normalize] ──►  NormalizedEnvelope  (typed blocks / usage / sections / 信号)
                      │
                      ▼
                  [store]  ──►  SQLite + raw/ 原始字节
                      ▲
                      │  /api/*
                 [server]  ──►  [frontend] Viewer (Flow / Diff / Replay / Launch)
```

依赖单向指向 `internal/event`（解耦边界，不含任何 provider 语义）。`source` 与
`normalize` 互不 import，只共享 `event`。

## 模块清单

| 模块 | 一句话 | 文档 | 安全相关 |
|---|---|---|---|
| capture（采集） | 反向代理抓 HTTP + 接收 hook 事件 | [capture.md](capture.md) | ⚠️ 经手真实凭证（内存） |
| normalize（规范化） | 把原始事件解析成 provider 无关的结构化信封 | [normalize.md](normalize.md) | — |
| store（存储） | SQLite + 原始字节落盘；复盘 case；live session 路由 | [store.md](store.md) | ⚠️ 落盘前脱敏 |
| server（服务） | 把代理/hook/Viewer API 组装成一个 HTTP 服务 | [server.md](server.md) | ⚠️ 同源校验 |
| launch（启动） | 把 agent 通过代理拉起来、可选注入 hook | [launch.md](launch.md) | ⚠️⚠️ 起进程 / 注入 key / 改全局配置 |
| frontend（Viewer） | 浏览器里看 trace、做 Flow/Diff/Replay/Launch | [frontend.md](frontend.md) | — |

> 标 ⚠️ 的模块每页末尾有「安全」小节；最敏感的是 **launch**。
