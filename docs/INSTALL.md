# 安装 / Install

agenttape 是**单个本地 Go 程序**:`agenttape serve` 起本地服务(代理 + hook + API + 内嵌
Viewer),`agenttape launch` 把 cc/codex 通过代理拉起来。它必须跑在**和你的 coding agent
同一台机器**上。下面几种装法按"省事程度"排,挑一种即可。

> 装完都一样:跑 `agenttape serve`,浏览器开 <http://127.0.0.1:8787/viewer/>。

---

## 方式一:预编译二进制(推荐,发布后)

从 GitHub Releases 下对应你系统 / 架构的二进制(已把 Viewer 内嵌进去,**单文件即用**)。

```bash
# 1. 下载对应平台的文件后，赋予执行权限
chmod +x agenttape

# 2. 跑起来
./agenttape serve
```

**macOS 注意(Gatekeeper):** 从网上下载的未签名二进制会被拦("无法验证开发者")。解隔离一次即可:

```bash
xattr -d com.apple.quarantine ./agenttape   # 或者:右键 → 打开 → 确认一次
```

把它放进 PATH(如 `/usr/local/bin` 或 `~/bin`)后就能直接 `agenttape serve` / `agenttape launch`。

> 状态:发布渠道仍在搭建中。在二进制 Release 出来之前,用**方式二(源码编译)**。

---

## 方式二:源码编译(当前可用)

**前置:** Go 1.26+、Node 18+、git。

```bash
git clone <repo-url> agenttape
cd agenttape

# 1. 先编前端 —— 它会构建进 internal/web/dist，go build 时被内嵌进二进制
cd frontend && npm install && npm run build && cd ..

# 2. 再编二进制
go build -o agenttape ./cmd/agenttape

# 3. 跑
./agenttape serve
# 打开 http://127.0.0.1:8787/viewer/
```

> ⚠️ **顺序很重要:Viewer 是编译期内嵌的。** 必须先 `npm run build` 再 `go build`。
> 跳过前端这步,`go build` 仍能过(仓库里有占位 index.html),但 `/viewer/` 只会显示
> 一句"还没构建前端"。改了前端代码,要重新 `npm run build` + `go build` 才生效。
>
> 前端开发时不必每次重编二进制:用 `-viewer` 指向磁盘上的 dist 跳过内嵌,或直接
> `cd frontend && npm run dev`(它把 `/api` 代理到本地 serve)。

---

## 方式三:go install(只装 CLI/服务,不含 Viewer)

```bash
go install <module-path>/cmd/agenttape@latest
```

`go install` 只编 Go 源码,**不会跑前端构建**,所以装出来的二进制里 Viewer 是占位页
(代理 / hook / API / `agenttape launch` 都正常,只是浏览器界面没内容)。要带 Viewer,请用
**方式一**(Release 二进制)或**方式二**(源码编译)。

---

## 方式四:Homebrew(规划中)

```bash
brew install <tap>/agenttape   # 待发布
```

brew 装的二进制不带 macOS 隔离标记,免去方式一的 Gatekeeper 步骤,且自动进 PATH——
mac 上最顺滑。出来后这里会更新。

---

## 跑起来之后

常用参数:

```bash
agenttape serve \
  -listen 127.0.0.1:8787 \   # 监听地址（默认只听本机）
  -data   agenttape-data \    # SQLite + 原始字节的目录
  -allow-launch=true         # 是否允许从界面起 agent 进程（默认开）

agenttape launch -kind cc|codex [-upstream URL] -- <透传给客户端的参数>
```

- 数据(库 + 抓到的原始字节)都在 `-data` 目录,默认 `./agenttape-data`。
- 怎么把 agent 接进来抓取,见 Launch 页,或 [`docs/modules/launch.md`](modules/launch.md)。

## 平台说明

- **Viewer / 代理 / 捕获:跨平台**(Go 跨平台编译 + 浏览器界面)。
- **"帮我启动"(开新终端拉起 agent):目前仅 macOS**。其它系统请用 Launch 页的
  **"自己运行"**复制命令——把你自己的 cc/codex 指向代理跑,一样能抓 HTTP。

## 相关文档

- 安全 / 凭证模型:[`docs/SECURITY.md`](SECURITY.md)
- 模块总览:[`docs/modules/`](modules/)
