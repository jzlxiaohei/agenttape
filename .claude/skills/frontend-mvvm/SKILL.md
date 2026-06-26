---
name: frontend-mvvm
description: agenttape 前端 MVVM 架构与代码规范（Vite+React+TS / Zustand / TanStack Query / React Router）。写任何前端代码前先读。
---

# 前端 MVVM 规范（agenttape / 模块三）

上一代 viewer 的病:业务状态散落在 UI 的 `useState`、组件里直接 fetch、render 里堆解析,
`RequestInspector.tsx` 900 行 / `traceView.ts` 1900 行。本规范强制分层,违反即返工。

## 技术栈(已定型,勿换)

Vite + React 18 + TypeScript ｜ 路由 React Router ｜ **服务端状态 TanStack Query** ｜
**客户端状态 Zustand** ｜ 样式 Tailwind + shadcn/ui ｜ i18n react-i18next ｜
JSON/diff 查看 CodeMirror 6 ｜ 图表先用纯 CSS 条形。

## 目录结构(每个 feature 按这个分层)

```
frontend/src/
  api/         纯数据访问：fetch 封装 + 类型化 DTO（无 React、无 hooks）
  query/       TanStack Query 的 hooks（useSessions、useEvent…），唯一调用 api/ 的地方
  store/       Zustand store：跨组件业务/UI 状态 + actions（选中、过滤、tab、展开）
  viewmodel/   纯函数 selector / 派生 hook：把 query+store 组合成"视图就绪"数据
  ui/          组件：只接 props、渲染、触发 action（容器组件可调 query/store hook）
  components/   shadcn 基础组件（button/tabs/select…）
  i18n/        语言资源 + 配置
  routes/      路由页面（薄，组合 ui + viewmodel）
  lib/         纯工具函数
```

依赖方向单向:`routes/ui → viewmodel → (query + store) → api`。**UI 不直接 import api**。

## 硬规则

1. **业务状态禁止放 UI 组件**。选中的 session、tab、过滤条件、展开的 section、搜索词——
   全进 Zustand store(`store/`)。
2. **`useState` 只允许纯视图局部状态**:输入框草稿、hover、纯展示折叠(且不被别处读)。
   拿不准就进 store。
3. **数据获取只在 query 层**(TanStack Query)。组件里**不准出现 `fetch`/axios**;
   缓存/重试/失效/loading 统一由 Query 管。
4. **派生逻辑进 viewmodel**(纯函数 selector)。组件 render 里**不准**写
   `.filter().map().reduce()` 之类业务编排、不准解析 JSON、不准分组排序。
5. **一个组件一个职责**,超 ~150 行拆。容器组件(连 query/store)与展示组件(纯 props)分开。
6. **所有用户可见文案走 i18n key**,不硬编码中文/英文字符串(详见 frontend-design)。
7. **类型化**:api 层定义 DTO 类型,贯穿到 UI;不用 `any`(拿不到类型用 `unknown` + 收窄)。

## Zustand store 写法

- 一个 feature 一个 slice;state + actions 同文件。
- selector 取最小切片避免重渲染:`useStore(s => s.selectedId)`,不要整存。
- 复杂派生不写在组件,放 `viewmodel/`。

## 反例 → 正解

```tsx
// ✗ 业务状态 + fetch + 解析全在 UI
function SessionDetail({id}:{id:string}) {
  const [events,setEvents]=useState([])
  const [filter,setFilter]=useState('')
  useEffect(()=>{ fetch(`/api/sessions/${id}/events`).then(r=>r.json()).then(setEvents) },[id])
  const groups = events.filter(e=>e.kind===filter).reduce(...)   // 业务逻辑混进 render
  return ...
}

// ✓ 分层
function SessionDetail({id}:{id:string}) {
  const groups = useSessionEventGroups(id)   // viewmodel：内部用 useEvents(query)+useStore(filter) 派生
  return <EventGroupList groups={groups} />   // 纯展示
}
```

## 自查清单(提交前)

- [ ] 组件里没有 `fetch`/axios（都在 query 层）
- [ ] 组件里的 `useState` 都是纯视图局部状态
- [ ] render 里没有数据分组/过滤/解析（都在 viewmodel）
- [ ] 业务状态在 store 里有单一来源
- [ ] 用户可见文案都走 i18n key
- [ ] 没有 `any`
