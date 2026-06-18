---
name: frontend-design
description: tracelab 前端视觉设计与 i18n 规范（Tailwind + shadcn/ui，暗色 DevTools 风，中英双语）。写任何样式/文案前先读。
---

# 前端设计 & i18n 规范（tracelab / 模块三）

定位:**浅色、柔和、聊天 App 风**的研究工具,审美参照 Cumora / Slack / Linear。
布局=最左图标栏 + 会话列表 + 中间对话流(+ 右侧研究面板)。圆角、头像、留白舒适,
但保留研究工具需要的密度与信息(token 占比、tags、原始/diff)。上一代 viewer "样式难看",
本规范是基线。

## 1. design token,禁止魔法值

- 颜色、间距、圆角、字号、阴影只能用 token(Tailwind theme / CSS 变量),
  组件里**不写** `#3b7af0`、`13px`、`margin:7px` 这类裸值。
- shadcn 的 CSS 变量(`--background`/`--foreground`/`--muted`/`--border`…)作为基础调色板;
  缺语义色就**先加 token 再用**。

## 2. 浅色为主 + 语义色

- 默认浅色:白/极浅灰背景 + 柔和边框 + 一个 indigo 强调色(参照 Cumora 的 Convene 按钮)。
- **内容类型语义色固定**(贯穿全站,对应规范化的 block 类型):
  `text` 普通 ｜ `reasoning` 思考 ｜ `tool_call` ｜ `tool_result` ｜ `error`。
- **`suspected`(疑似)必须视觉上明确"不确定"**:低饱和警示色 + 斜体或虚线边,
  且 hover 出 evidence 说明("为什么疑似")。诚实优先(next.md 3.3)。
- 头像/角色:对话流里每条消息带头像 + 角色标签(USER/ASSISTANT/TOOL/REASONING),
  助记参照聊天 App。
- 大段 JSON/代码用等宽字体 + CodeMirror 语法高亮。

## 3. 间距 / 排版(4px 基准)

- 间距用 4 的倍数(4/8/12/16/24/32)。字号梯度收敛到 4-5 档。
- 信息密度优先:列表/表格紧凑,但分组留白清晰。

## 4. 布局(聊天 App 式)

- **最左细图标栏**(会话/搜索/统计/设置)→ **会话列表**(头像+名称+预览+计数,
  像会话列表)→ **中间对话流**(消息气泡)→ 需要时**右侧研究面板**(token 占比/tags/原始/diff)。
- cc / codex 多会话并行,列表/标识区分清晰(next.md 3.1)。
- section(system/tools/messages/reasoning/tool)可折叠,标题区清晰可点。
- token 占比:统一的 **CSS 条形/环形**视觉语言,不每处自创。

## 5. 组件库

- 统一用 shadcn/ui 基础组件(button/input/select/tabs/badge/card/tooltip/dialog…),
  **先看库里有没有再造**;新组件遵循同一 token 体系。

## 6. 交互细节

- 关键信息一键复制:JSON 路径、请求体、token 数。
- loading / empty / error 三态都要有明确样式,不留白屏。
- 探测/控制请求(GET/HEAD 探活)与真实补全在视觉上区分,不混列(M1 dump 已区分,沿用)。

## 7. i18n(中英双语)——硬约定

- **所有用户可见文案走 `t('key')`**,源码里不留裸中文/英文字符串(含按钮、标签、
  提示、空状态、错误)。
- 资源文件:`i18n/en.json`、`i18n/zh.json`,key 按 feature 命名空间组织
  (如 `sessions.empty`、`event.section.tools`)。
- **包括语义/标签**:`suspected`→"疑似 / suspected"、section 名、tag 名都要有双语。
- 提供语言切换;默认跟随浏览器,可手动切并记住(存 store/localStorage)。
- 数字/时间用 `Intl` 本地化格式。

## 自查清单

- [ ] 没有裸色值/裸像素(都走 token)
- [ ] 用了既有 shadcn 组件而非新造
- [ ] 暗色下对比度可读
- [ ] suspected / 语义色按规范区分,且 suspected 有 evidence tooltip
- [ ] 没有硬编码文案,全部 `t('key')` 且 en/zh 都有
