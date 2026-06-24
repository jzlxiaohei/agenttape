# frontend（Viewer）

> `frontend/`（Vite + React + TypeScript）

## 为了做什么

浏览器里的研究界面：看捕获到的会话时间线，把一次执行画成 Flow（hook + HTTP 交错），对相邻
请求做 Context Diff，跑 Replay Library，以及从 Launch 页起会话。后端只提供 `/api/*`，所有
呈现/派生在前端。

## 基本流程（MVVM，依赖单向）

```
routes / ui  ──►  viewmodel  ──►  (query + store)  ──►  api
   页面/组件      纯函数 selector     服务端/客户端状态     fetch + DTO
```

- `api/`：纯数据访问，定义 DTO，唯一 `fetch` 的地方（无 React）。
- `query/`：TanStack Query hooks（服务端状态：缓存/重试/失效/loading）。
- `store/`：Zustand（客户端状态：选中、过滤、tab、展开…）。
- `viewmodel/`：纯函数 selector，把 query+store 组合成"视图就绪"数据。
- `ui/` `routes/`：只接 props、渲染、触发 action；render 里不写业务编排/解析。

## 关键目录

- `routes/`：路由页面（薄，组合 ui + viewmodel）。
- `ui/`：组件（如 `CasesPanel.tsx` 复盘库、Flow / Diff 视图）。
- `i18n/{zh,en}.json`：**所有用户可见文案走 key**，中英双语。
- `index.css`：design token（颜色/间距/圆角），组件里不写裸值。

## 写代码前先读

前端有两份**硬规范**（违反即返工），动任何前端代码前先读：

- **frontend-mvvm**：分层与状态归属、数据获取只在 query 层、派生只在 viewmodel。
- **frontend-design**：浅色聊天 App 风、design token、语义色、i18n 双语。

（二者是仓库根 `.claude/skills/frontend-mvvm` 与 `.claude/skills/frontend-design` 下的 skill。）
