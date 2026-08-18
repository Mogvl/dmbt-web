# 🌸 dmbt-web — AnimeGarden (Go + Vue)

A 1:1 reimplementation of [AnimeGarden](https://github.com/yjl9903/AnimeGarden) — 動漫花園
第三方便携站 / 动画 BT 资源聚合站 — built with **Go** (backend) and **Vue 3** (frontend),
with feature parity to the original (verified against the live API and the original source).

- 前端 Frontend: http://localhost:9700
- 后端 Backend API: http://localhost:9701

## Features

- ☁️ Open API with the same wire contract as `api.animes.garden`
  (`/resources`, `/subjects`, `/users`, `/teams`, `/collections`, `/feed.xml`, `/sitemaps/*`,
  `/admin/*`, `/mcp`, `/.well-known/mcp/server-card.json`)
- 📺 动画放送时间表 (`/anime`, `/calendar/:season`) powered by the same
  `bgm.animes.garden` mirror
- 🔖 Advanced search syntax: `葬送的芙莉莲 +简体内嵌 字幕组:桜都字幕组 类型:动画`
  (jieba tokenized, same filter semantics as the original)
- 📙 Custom RSS: `/feed.xml?search=...`, `/collection/:hash/feed.xml`
- ⭐ 搜索条件收藏夹 (localStorage + IndexedDB + server collection hashing SHA-1/ohash)
- 🔍 Resource detail with torrent file trees (`/detail/:provider/:id`, `/detail/infohash/:hash`)
- 📄 API 文档页, iframe 嵌入 (`/iframe`), robots/sitemap/llms.txt
- ✈️ 4 个数据源爬虫: 動漫花園 (dmhy) / 蜜柑计划 (mikan) / 萌番组 (moe) / ANi (ani)
- 🤖 MCP endpoint (`/mcp`) with `search_resources` tool + `animegarden://resources/...` resources
- 🛎️ 可选 Telegram 推送 (`TELEGRAM_TOKEN` / `TELEGRAM_CHAT_ID`)

## Architecture

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

## Development

### Backend (Go, port 9701)

```bash
cd server
go run ./cmd/server
```

Environment variables (all optional):

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `9701` | HTTP listen port |
| `DATA_DIR` | `<exe>/data` | SQLite data directory |
| `APP_HOST` | `animes.garden` | site host used in feed/detail URLs |
| `CRON` | `true` | enable the scheduler (fetch every 5 min, sync hourly, calendar hourly) |
| `TELEGRAM_TOKEN` / `TELEGRAM_CHAT_ID` | — | Telegram push |
| `SCRAPE_PAGE_DELAY_MS` | `800` | politeness delay between upstream pages |
| `MAX_FETCH_PAGES` | `200` | per-run fetch walk cap (history catch-up) |

On first boot the crawlers walk the upstream history page by page (the original
pre-seeds this data; here it happens in the background). The API works immediately
and fills up over time.

### Frontend (Vue, port 9700)

```bash
cd web
npm install
npm run dev      # dev server with HMR, proxies API to :9701
npm run build    # production build
npm run preview  # production preview (same port/proxy config)
```

## API examples

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

# MCP
curl -X POST http://localhost:9701/mcp -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## License

AGPL-3.0 (matching the original AnimeGarden license).