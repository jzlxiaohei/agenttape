# store（存储层）

> `internal/store`

## 为了做什么

把规范化后的事件持久化下来供查询/回放，并管理复盘库与 live session 路由。SQLite 存
结构化数据，**原始字节存磁盘文件**（`raw/`，库里只存指针——可直接打开是个 feature）。
`store.Store` 同时实现 `sink.Sink`，所以采集→规范化的产物直接 `Write` 进来。

## 基本流程

**打开**（`store.go` `Open`）：建数据目录 → 应用 `schema.sql`（`CREATE TABLE IF NOT
EXISTS`，幂等）→ `migrate`（加列等增量迁移）→ seed 内置 case → seed hook 事件默认集。

**写入**（`write.go` `Write(record)`）：一条 `SourceEvent` 拆进**脊柱 + 明细**两层——
`events`（每类事件共有的时间线/关联）+ `http_exchanges` / `hook_events`（各自明细），
外加 `raw_files`（原始字节指针）、`tags`、`sections`（token 占比）、`events_fts`（全文检索）。

**读取**：`query.go`（session 列表、搜索、facets）、`cases.go`（复盘 case）、
`live_sessions.go`（重连路由）。

## 关键文件

- `schema.sql`：全部表结构（一处定义，`Open` 时应用）。
- `store.go` / `write.go` / `query.go`：打开、写入、查询。
- `cases.go` + `seeds/*.json`：复盘库 case 模型 + 内置 seed（`go:embed`，按内容指纹刷新，
  见 [`REPLAY_LIB.md`](../REPLAY_LIB.md) §3）。
- `live_sessions.go`：跨重启的**非密** session 路由表。

## 安全

- 落库的 header 是**脱敏版**（采集层 `redactHeaders` 处理），库里**不含可用凭证**。
- `live_sessions` 只存路由事实（id/token/上游/provider/mode），**没有一行是凭证**；
  token 只是路由句柄，不能向上游认证。
- 内置 seed 来自真实捕获但已逐项脱敏（见 [`REPLAY_LIB.md`](../REPLAY_LIB.md) §4 /
  [`SECURITY.md`](../SECURITY.md) §6）。

> 数据目录是普通文件，同用户进程可读、且可能被备份/同步带离本机——库里不含凭证正是为了
> 让这种扩散不等于凭证泄露。
