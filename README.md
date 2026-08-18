# 🌸 dmbt-web — AnimeGarden (Go + Vue)

A reimplementation of [AnimeGarden](https://github.com/yjl9903/AnimeGarden) — 動漫花園
third-party mirror & anime BT resource aggregator — built with **Go** (backend) and
**Vue 3** (frontend), with feature parity to the original.

- 前端 Frontend: http://localhost:9700
- 后端 Backend API: http://localhost:9701

## Features

- ☁️ Open API (`/resources`, `/subjects`, `/collections`, ...)
- 📺 动画放送时间表 (anime broadcast schedule / calendar)
- 🔖 Advanced search: `葬送的芙莉莲 +简体内嵌 字幕组:桜都字幕组 类型:动画`
- 📙 Custom RSS subscription (`/feed.xml?filter=[...]`)
- ⭐ Search-condition favorites (collections) + aggregated RSS
- 🔍 Resource detail with torrent file tree (`/detail/:provider/:id`)
- 📄 API docs page, iframe embed, sitemap, robots.txt
- 👷 AutoBangumi / AnimeSpace style integration API

## Development

### Backend (Go, port 9701)

```bash
cd server
go run ./cmd/server
```

### Frontend (Vue, port 9700)

```bash
cd web
npm install
npm run dev
```

## License

AGPL-3.0 (matching the original AnimeGarden license).
