# 🌸 dmbt-web

[![CI](https://github.com/Mogvl/dmbt-web/actions/workflows/ci.yml/badge.svg)](https://github.com/Mogvl/dmbt-web/actions/workflows/ci.yml)
[![Docker Images](https://github.com/Mogvl/dmbt-web/actions/workflows/docker-images.yml/badge.svg)](https://github.com/Mogvl/dmbt-web/actions/workflows/docker-images.yml)

[English](/README.en.md) | [简体中文](/README.md)

[動漫花園](https://share.dmhy.org/) 第三方 [镜像站](https://github.com/yjl9903/AnimeGarden) 以及 [动画 BT 资源聚合站](https://github.com/yjl9903/AnimeGarden) 的 **Go + Vue 重新实现**，API 线格式与原版完全兼容。

- 前端 Frontend: http://localhost:9700
- 后端 Backend API: http://localhost:9701

+ ☁️ 为开发者准备的开放 [API 接口](http://localhost:9700/docs/api)，与原版 `api.animes.garden` 线格式一致
+ 📺 查看 [动画放送时间表](http://localhost:9700/anime) 来找到你喜欢的动画
+ 🔖 支持丰富的高级搜索，例如: `葬送的芙莉莲 +简体内嵌 字幕组:桜都字幕组 类型:动画`
+ 📙 自定义 RSS 订阅链接, 例如: [葬送的芙莉莲](http://localhost:9700/feed.xml?search=%E8%91%AC%E9%80%81%E7%9A%84%E8%8A%99%E8%8E%89%E8%8E%B2)
+ ⭐ 搜索条件收藏夹和生成聚合的 RSS 订阅链接
+ ✈️ 四个数据源爬虫: [動漫花園](https://share.dmhy.org/)、[蜜柑计划](https://mikanani.me/)、[萌番组](https://bangumi.moe/)、[ANi](https://open.ani.rip/)

[![home](./assets/home.jpeg)](http://localhost:9700/resources/1)

## 使用 Skills

**Anime Garden skill**: 用于从 Anime Garden 上检索动画资源（本仓库自带，API 指向你的自建实例）.

**Yuc's Anime List skill**: 用于从 yuc.wiki 上检索每个季度的新番列表.

两个 skill 随本仓库提供（`skills/animegarden`、`skills/yuc`）。使用 Vercel skills CLI 添加：

```bash
npx skills add https://github.com/Mogvl/dmbt-web --skill animegarden
npx skills add https://github.com/Mogvl/dmbt-web --skill yuc
```

OpenClaw 等客户端也可以直接把本仓库的 `skills/animegarden`、`skills/yuc` 目录挂载/克隆到本地 skills 目录使用.

**Design taste skill**：前端设计风格系列技能（内置自 [taste-skill](https://github.com/Leonxlnx/taste-skill)，MIT）：

```bash
npx skills add https://github.com/Mogvl/dmbt-web --skill design-taste-frontend   # 默认 v2
npx skills add https://github.com/Leonxlnx/taste-skill                            # 官方源全部技能
```

本仓库 `skills/taste-skill/` 下内置了全部 14 个技能（brandkit / brutalist / gpt-taste / image-to-code / imagegen-frontend-web / imagegen-frontend-mobile / minimalist / output / redesign / soft / stitch / taste-skill v1+v2），安装名见各 `SKILL.md` 的 `name:` 字段。

`animegarden` skill 的 API 地址通过环境变量 `ANIMEGARDEN_API_BASE` 配置（默认 `http://localhost:9701`；Docker Compose 部署后改为你的服务器地址，例如 `https://anime.example.com:9701`），首次使用请确认你的实例已爬取到数据（打开 http://localhost:9700 应能看到资源列表）。

## 使用 MCP

Anime Garden MCP 服务端点: `http://localhost:9701/mcp`.

你只需要将如下配置放入你的 MCP Client 即可.

```json
{
  "mcpServers": {
    "animegarden": {
      "url": "http://localhost:9701/mcp"
    }
  }
}
```

## 使用开放 API

```bash
curl "http://localhost:9701/resources?page=1&pageSize=10"
```

你可以在[这里](http://localhost:9700/docs/api)找到交互式的 Open API 文档, 以及原版仓库的 [examples/api.http](https://github.com/yjl9903/AnimeGarden/blob/main/examples/api.http) 文件内查看到更多 API 用例.

你也可以直接使用网站, 在资源列表页直接复制生成的 cURL、JavaScript 和 Python 的 API 请求代码.

## 使用 npm 包

API 与原版 [@animegarden/client](https://www.npmjs.com/package/animegarden) 线格式兼容，将 `baseURL` 指向自建实例即可：

```bash
npm i @animegarden/client
```

```ts
import { fetchResources } from '@animegarden/client'

// Fetch the first page of your Anime Garden instance
const resources = await fetchResources({ baseURL: 'http://localhost:9701' })

// Fetch all the resources which match some filter conditions
const sakurato = await fetchResources({ baseURL: 'http://localhost:9701', count: -1, fansub: 'ANi' })
```

使用时, 你需要保证你的程序环境中有内置的 [Fetch](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch) 函数. 如果没有, 你可以安装使用 [undici](https://github.com/nodejs/undici) 或者 [ofetch](https://github.com/unjs/ofetch) 进行 polyfill.

## 使用内嵌代码

你可以从资源搜索页复制出网页嵌入代码，放到你的博客等各种页面中.

```html
<iframe src="//localhost:9700/iframe?type=动画" width="100%" height="600" frameborder="0"></iframe>
```

## 一键部署 (Docker Compose)

镜像自动构建并发布到 GHCR（每次 push main 由 GitHub Actions 触发），支持 docker compose 一键启动：

`docker-compose.yml` 完整内容：

```yaml
# dmbt-web — AnimeGarden (Go + Vue) 一键部署
#
#   docker compose up -d
#
# 前端 http://localhost:9700 （nginx 同源代理 /resources /feed.xml /bgmx 等到后端）
# 后端 http://localhost:9701 （容器内部网络，如需直连可取消注释 ports）
#
# 数据保存在 ./data（SQLite），修改 volume 可将数据迁移到任意目录。

services:
  dmbt-web:
    image: ghcr.io/mogvl/dmbt-web/web:latest
    container_name: dmbt-web
    ports:
      - "9700:80"
    environment:
      - TZ=Asia/Shanghai
    depends_on:
      - dmbt-server
    restart: unless-stopped

  dmbt-server:
    image: ghcr.io/mogvl/dmbt-web/server:latest
    container_name: dmbt-server
    environment:
      - PORT=9701
      - DATA_DIR=/data
      - CRON=true
      - APP_HOST=animes.garden
      - TZ=Asia/Shanghai
    volumes:
      - ./data:/data
    # 如需直接暴露 API 端口:
    # ports:
    #   - "9701:9701"
    restart: unless-stopped
```

使用方法：

```bash
# 1. 拉取并启动（前端 :9700，后端容器内部 :9701）
curl -O https://raw.githubusercontent.com/Mogvl/dmbt-web/main/docker-compose.yml
docker compose up -d

# 2. 打开 http://localhost:9700
# 数据持久化在 ./data（SQLite）
```

服务说明：

| 服务 | 镜像 | 端口 | 说明 |
|---|---|---|---|
| `dmbt-web` | `ghcr.io/mogvl/dmbt-web/web:latest` | `9700:80` | nginx 托管前端，同源代理 API/RSS 到后端 |
| `dmbt-server` | `ghcr.io/mogvl/dmbt-web/server:latest` | 内部 9701 | Go API + 爬虫调度，`DATA_DIR=/data` 持久化 |

后端环境变量（`docker-compose.yml` 中可覆盖）：`PORT`（默认 9701）、`DATA_DIR`（默认 `/data`，SQLite 数据库位置，必须指向挂载卷）、`CRON`（默认 true）、`APP_HOST`（feed/detail 链接用域名）、`TELEGRAM_TOKEN` / `TELEGRAM_CHAT_ID`（可选推送）、`SCRAPE_PAGE_DELAY_MS` / `MAX_FETCH_PAGES`（爬虫节流）。

> 容器以非 root 用户运行；启动入口会自动 `chown` 数据目录，宿主机新建的
> `./data`（root 属主）也能直接写入，无需手动改权限。
> 镜像为 amd64 + arm64 双架构。

本地构建镜像：

```bash
docker build -t ghcr.io/mogvl/dmbt-web/server:latest ./server
docker build -t ghcr.io/mogvl/dmbt-web/web:latest ./web
```

## 本地开发

### 后端 (Go, 端口 9701)

```bash
cd server
go run ./cmd/server
```

环境变量（均可选）：

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `9701` | HTTP listen port |
| `DATA_DIR` | `data` | SQLite data directory (relative to the working directory) |
| `APP_HOST` | `animes.garden` | site host used in feed/detail URLs |
| `CRON` | `true` | enable the scheduler (fetch every 5 min, sync hourly, calendar hourly) |
| `TELEGRAM_TOKEN` / `TELEGRAM_CHAT_ID` | — | Telegram push |
| `SCRAPE_PAGE_DELAY_MS` | `800` | politeness delay between upstream pages |
| `MAX_FETCH_PAGES` | `200` | per-run fetch walk cap (history catch-up) |

首次启动时爬虫会在后台逐页补全历史数据（原版是预置数据）；API 立即可用并随时间填满。

### 前端 (Vue, 端口 9700)

```bash
cd web
npm install
npm run dev      # dev server with HMR, proxies API to :9701
npm run build    # production build
npm run preview  # production preview (same port/proxy config)
```

## 架构

```
server/   Go 1.25 backend (port 9701)
  cmd/server            entrypoint
  internal/api          HTTP handlers (Hono-route parity), bgm mirror proxy
  internal/scraper      dmhy / moe / mikan / ani fetchers + parsers
  internal/anipar       anime title parser (Go port, 12,356 fixture tests pass)
  internal/search       jieba tokenizer (vendored jiebago + embedded dict)
  internal/zh           traditional→simplified + fullwidth→halfwidth (simptrad port)
  internal/query        filter → SQL (SQLite port of the Postgres schema)
  internal/resources    upsert / dedup / details store
  internal/subjects     bangumi calendar sync (bgm.animes.garden)
  internal/collections  collection store + ohash/SHA-1 hashing
  internal/rss,sitemap, mcp, push, jobs, db, config, model, filter

web/      Vue 3 + Vite + Tailwind v4 frontend (port 9700)
  src/api/client.ts     @animegarden/client port (parse/stringify search params)
  src/utils             search syntax, code generators, dates, constants
  src/stores            theme / sidebar / histories / fansubs / collections (IndexedDB)
  src/layouts           hero + header + cmdk-style search + sidebar + footer
  src/pages             home / resources / subject / anime calendar / collection /
                        detail (file tree) / docs api / iframe / about
```

API 用例：

```bash
# latest resources
curl "http://localhost:9701/resources?page=1&pageSize=30"

# advanced search: anime containing 葬送的芙莉莲 with 简体内嵌, fansub 桜都字幕组
curl "http://localhost:9701/resources?search=%E8%91%AC%E9%80%81%E7%9A%84%E8%8A%99%E8%8E%89%E8%8E%B2&keyword=%E7%AE%80%E4%BD%93%E5%86%85%E5%B5%8C&fansub=%E6%A1%9C%E9%83%BD%E5%AD%97%E5%B9%95%E7%BB%84"

# schedule
curl "http://localhost:9701/subjects"

# RSS
curl "http://localhost:9701/feed.xml?search=%E8%91%AC%E9%80%81%E7%9A%84%E8%8A%99%E8%8E%89%E8%8E%B2"

# detail with torrent file tree
curl "http://localhost:9701/detail/dmhy/725145"
```

## 设计取舍 (Out of scope)

- 原版的 Cloudflare Worker（旧版 301 跳转器 + legacy filter 参数翻译）是独立部署层的
  兼容层；本项目前后端同源部署，无需该组件。
- Redis（查询/详情缓存、跨进程 RPC）在本实现中以进程内缓存替代（SQLite 单机部署）。
- 页面的 `Accept: text/markdown` 内容协商需要 SSR 层；本 SPA 实现由静态
  `/llms.txt` 提供同等信息（站点结构、API 入口、关键事实）。

## 相关项目

+ [AnimeGarden](https://github.com/yjl9903/AnimeGarden): 本项目复刻的原版
+ [AnimeSpace](https://github.com/yjl9903/AnimeSpace): Keep following your favourite anime
+ [anipar](https://github.com/yjl9903/AnimeGarden/tree/main/packages/anipar): Parse structure metadata from resource's title.

## 鸣谢

+ [AnimeGarden](https://github.com/yjl9903/AnimeGarden) — 原版项目与设计参考
+ [動漫花園](https://share.dmhy.org/)
+ [蜜柑计划](https://mikanani.me/)
+ [萌番组](https://bangumi.moe/)
+ [ANi](https://open.ani.rip/)
+ [Bangumi 番组计划](https://bgm.tv/)
+ [bangumi-data](https://github.com/bangumi-data/bangumi-data)

## 开源协议

AGPL-3.0 License © 2025 [Mogvl](https://github.com/Mogvl)（基于 [AnimeGarden](https://github.com/yjl9903/AnimeGarden) AGPL-3.0 原版重构）