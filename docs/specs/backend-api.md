# AnimeGarden Backend API Specification

**Source analyzed:** `/tmp/animegarden-orig` (pnpm monorepo, `apps/server` + `packages/client`), Hono 4.12.28 / drizzle-orm / postgres / ioredis.
**Purpose:** 1:1 reimplementation of the AnimeGarden public API in Go.

This document is exhaustive: every route, every query parameter, every response field, every header, every default value, every validation rule, the complete database schema, RSS/sitemap/MCP wire formats, and the exact filter/tokenizer semantics. Details that the original code leaves implicit (e.g. Zod coercion gotchas, Hono middleware ordering, typo'd query params) are called out explicitly.

---

## Table of contents

1. [Architecture & deployment context](#1-architecture--deployment-context)
2. [Global request/response behavior (middleware)](#2-global-requestresponse-behavior-middleware)
3. [HTTP endpoint reference](#3-http-endpoint-reference)
4. [Query-parameter / body parsing semantics](#4-query-parameter--body-parsing-semantics)
5. [Filter semantics & tokenization](#5-filter-semantics--tokenization)
6. [Database schema](#6-database-schema)
7. [RSS feed format (`/feed.xml`)](#7-rss-feed-format-feedxml)
8. [Sitemap format](#8-sitemap-format)
9. [MCP endpoint (`/mcp`)](#9-mcp-endpoint-mcp)
10. [Error format](#10-error-format)
11. [Collections](#11-collections)
12. [Admin & user endpoints, authentication model](#12-admin--user-endpoints-authentication-model)
13. [Appendix: shared types, seeds, constants](#13-appendix-shared-types-seeds-constants)

---

## 1. Architecture & deployment context

- **HTTP framework:** Hono 4.12.28 served by `@hono/node-server` (Node). Default listen host `0.0.0.0`, default port `3000` (`--port` / `PORT` env override).
- **Public API base URL (client default):** `https://api.animes.garden/`
- **Web site host:** configured via `--site` CLI flag or `APP_HOST` env; default fallback `animes.garden`. Used to build absolute URLs in feeds, detail links, MCP `href` fields, and the MCP server-card redirect.
- **Two process modes (same codebase):**
  - **server** (`profile: 'server'`, `cron: false`): serves the full HTTP API (all routes below) plus a *cron executor* that runs the fetch/sync cron jobs in-process and handles `resources.*` RPC invocations locally.
  - **cron** (`profile: 'cron'`, `cron: true`): runs scheduled jobs; if `--listen` it also serves the API. Admin endpoints dispatch `resources.fetch` / `resources.sync` RPC over Redis (`invoke-rpc` / `reply-rpc:<uuid>` channels) to the cron process. When no cron process is reachable, the RPC reply times out after **30 s** and the admin endpoint answers 503.
- **Cron schedule** (only in cron/executor mode, timezone `Asia/Shanghai`):
  - every `*/5 * * * *` → `resources.fetch` per provider,
  - every `0 * * * *` → `resources.sync {provider, start:1, end:10}` per provider,
  - every `17 * * * *` → subjects calendar update (`updateCalendar`).
- **Dependencies:** PostgreSQL (drizzle-orm + `postgres` driver), Redis (ioredis, optional). Without Redis the API still works (in-memory caches only).
- **Supported providers** (`SupportProviders`, order matters for duplicate winner selection):
  `['dmhy', 'moe', 'mikan', 'ani']` — enum `resources_provider`.
- **Supported presets** (`SupportPresets`): `['bangumi']`.

---

## 2. Global request/response behavior (middleware)

Middleware order in `registerHono` (outermost first):

1. **Request-ID / response-timestamp / content-type middleware** (`*`)
   - Request: `X-Request-Id` header value if present, else a new random UUID.
   - Response headers (set on **every** response, including 304s and errors):
     - `X-Request-Id: <requestId>`
     - `X-Response-Timestamp: <ISO-8601>` — value of the handler-set `responseTimestamp` (only `/` and `/health` set it, to the max provider `refreshedAt`), otherwise `new Date()` at response time.
     - If the response `Content-Type` starts with `application/json`, it is forced to `application/json; charset=utf-8`.
2. **CORS** (`*`): `origin: '*'`, `allowMethods: ['GET','HEAD','PUT','POST','DELETE','PATCH','OPTIONS']`. (Hono default `allowHeaders` reflects the request's `Access-Control-Request-Headers`; `Access-Control-Allow-Origin: *` on responses.)
3. **prettyJSON** (`*`): JSON bodies pretty-printed with 2-space indentation.
4. **Logger** (`*`): access log.
5. **Lazy system initialization gate** (`*`): for every path **except** `/health` and `/.well-known/mcp/server-card.json`, the first request triggers full system/module initialization (module load, DB warm caches) before the handler runs. `/health` and the server-card path skip this.
6. **Request timeout** (`*`): **60 seconds**; on expiry responds **408** with body:
   ```json
   { "status": "ERROR", "message": "Request timeout after waiting 60 seconds. Please try again later." }
   ```
7. Route handlers (see below). Many handlers wrap `safeEtag()` (custom ETag middleware):
   - Only applied when the response status is `200`; other statuses pass through untouched.
   - Computes a **strong ETag** `"<sha1-hex-of-response-body>"` (SHA-1 over the full response body).
   - If the request has `If-None-Match` and any comma-separated tag (with `W/` prefix stripped on both sides) equals the generated ETag → responds **304 Not Modified** with body empty and **only** these headers retained: `cache-control`, `content-location`, `date`, `etag`, `expires`, `vary` (plus `ETag` itself). Note: the outer middleware re-adds `X-Request-Id` / `X-Response-Timestamp` on the 304.
   - `safeEtag` never hashes an already-consumed body (skips ETag generation instead of 500).
8. **Error handler** (`app.onError`):
   - `HTTPException` → returns its own response unchanged (used by timeout → 408 and by `bearer-auth` → 400/401).
   - Resources query errors (`ResourcesDeepPaginationError` 400 / `ResourcesSlowQueryBusyError` 503 / `ResourcesSlowQueryTimeoutError` 504) — see [§10](#10-error-format). For paths ending in `/feed.xml`, the response is XML instead of JSON (with `Content-Type: application/xml; charset=UTF-8` and `Cache-Control: no-store`).
   - Any other error → `500` with body `{ "status": "ERROR" }` (no message).

Common JSON envelope: every endpoint returns an object with a `status` field, `"OK"` on success or `"ERROR"` on failure. Error bodies also carry `message` (see §10).

---

## 3. HTTP endpoint reference

### 3.0 Root / health

#### `GET /`
No auth. Response **200**:
```json
{
  "status": "OK",
  "timestamp": "2026-08-18T03:00:00.000Z",
  "providers": {
    "dmhy":  { "id": "dmhy",  "name": "动漫花园", "refreshedAt": "2026-08-18T03:00:00.000Z", "isActive": true },
    "moe":   { "id": "moe",   "name": "萌番组",   "refreshedAt": "2026-08-18T03:00:00.000Z", "isActive": true },
    "mikan": { "id": "mikan", "name": "蜜柑计划", "refreshedAt": "2026-08-18T03:00:00.000Z", "isActive": true },
    "ani":   { "id": "ani",   "name": "ANi",      "refreshedAt": "2026-08-18T03:00:00.000Z", "isActive": true }
  }
}
```
- `timestamp` = max `refreshedAt` over all providers (ISO-8601); also used as the response `X-Response-Timestamp`.
- No `Cache-Control` header is set. No ETag middleware.
- The client (`fetchStatus`) treats any non-OK response as service unavailable.

#### `GET /health`
Identical to `GET /` (status/timestamp/providers). Skips system-initialization gate; intended for load balancers.

### 3.1 Resources list

All of the following are **`.all()`** routes (any HTTP method: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS — the handler always runs identically; GET is the canonical use):

- `GET|POST|... /resources`
- `GET|POST|... /resources/`  (trailing slash variant)
- `GET|POST|... /resources/:page`  — `:page` is a path segment (ignored by the handler; pagination comes from query/body; the param is *not* validated or read)
- For each `provider` in `['dmhy','moe','mikan','ani']`:
  - `GET|POST|... /resources/{provider}`
  - `GET|POST|... /resources/{provider}/` (trailing slash variant)
  - `GET|POST|... /resources/{provider}/:page`
  - `GET /resource/{provider}/:id` (singular — detail by provider id; see §3.3)
  - `GET /detail/{provider}/:id` (see §3.3)

All list variants share one handler `listResources(ctx, sys, provider?)`:

1. `url.searchParams` are parsed together with an optional **JSON request body** (`await ctx.req.json().catch(() => undefined)` — a missing/invalid JSON body is silently treated as `undefined`). See §4 for the full parse/precedence/validation rules.
2. `assertResourcesPagination(pagination)` — deep-pagination guard: `(page-1)*pageSize + pageSize > 10000` → **400** `Resources pagination is too deep. Please keep offset + limit <= 10000.`
3. If a `provider` path segment was given, `filter.provider = provider` is set **after** parsing (overriding any parsed provider value).
4. `sys.modules.resources.query.find(filter, pagination)` executes the query.
5. Response-body stripping of `tracker` / `metadata` per query flag:
   - `tracker` query param: present **and** one of `true|yes|on` (case-insensitive) → keep `tracker` field; otherwise the `tracker` field is **deleted from every resource object**.
   - `metadata` query param: same rule → keep `metadata` field; otherwise deleted.
6. `Cache-Control: public, max-age=300` is set on the response.
7. ETag middleware is applied.

**Response 200** (all list variants):
```json
{
  "status": "OK",
  "complete": true,
  "resources": [
    {
      "id": 12345,
      "provider": "dmhy",
      "providerId": "123456",
      "title": "[LoliHouse] Re:Zero kara Hajimeru Isekai Seikatsu S3 - 01 [WebRip 1080p HEVC-10bit EAC3][简繁内挂字幕]",
      "href": "https://share.dmhy.org/topics/view/123456",
      "type": "动画",
      "magnet": "magnet:?xt=urn:btih:ABCDEF...",
      "tracker": "&tr=https://tracker.example/announce&tr=...",
      "size": 4831838208,
      "createdAt": "2026-02-01T08:00:00.000Z",
      "fetchedAt": "2026-02-01T08:05:00.000Z",
      "publisher": { "id": 1, "name": "LoliHouse", "avatar": "https://..." },
      "fansub":   { "id": 2, "name": "喵萌奶茶屋", "avatar": "https://..." },
      "subjectId": 456789,
      "metadata": { "anipar": { "title": "...", "episode": { "number": 1 }, "file": { "video": { "codec": "HEVC", "resolution": "1080p" } } } }
    }
  ],
  "pagination": { "page": 1, "pageSize": 100, "complete": true },
  "filter": {
    "preset": "bangumi",
    "provider": "dmhy",
    "duplicate": true,
    "publishers": ["LoliHouse"],
    "fansubs": ["喵萌奶茶屋"],
    "types": ["动画"],
    "before": "2026-02-28T15:59:59.000Z",
    "after": "2026-02-01T00:00:00.000Z",
    "subjects": [456789],
    "search": ["re zero"],
    "include": ["re:zero"],
    "keywords": ["1080p"],
    "exclude": ["合集"]
  }
}
```

Field semantics of a resource object:

| Field | Type | Notes |
|---|---|---|
| `id` | number | DB serial id |
| `provider` | string | one of `dmhy`/`moe`/`mikan`/`ani` |
| `providerId` | string | provider-scoped id |
| `title` | string | original title |
| `href` | string | absolute URL computed per provider: dmhy → `https://share.dmhy.org/topics/view/{href}`, mikan → `https://mikanani.me/Home/Episode/{href}`, moe → `https://bangumi.moe/torrent/{href}`, ani → `{href}` as-is; `''` if transform fails |
| `type` | string | e.g. `动画`, `合集`, `音乐`, `日剧`, `RAW`, `漫画`, `游戏`, `特摄`, `其他` |
| `magnet` | string | magnet URI (btih) without tracker params |
| `tracker` | string | `&tr=...` chain; **omitted unless `tracker` query flag enabled** |
| `size` | number | bytes (bigint) |
| `createdAt` / `fetchedAt` | string | ISO-8601 |
| `publisher` | object | `{ id, name, avatar? }` (avatar omitted when null) |
| `fansub` | object \| absent | `{ id, name, avatar? }`; **omitted entirely when resource has no fansub** |
| `subjectId` | number \| absent | bangumi subject id; omitted when null |
| `metadata` | object \| absent | `{ anipar?: <ParseResult> }`; **omitted unless `metadata` query flag enabled** |

`pagination`: `{ page, pageSize, complete }` — echoes the resolved request pagination; `complete = !hasMore` (see §4 for `complete` semantics).

`filter`: the **normalized** filter echoed back — publisher/fansub names resolved from ids, `before`/`after` as ISO strings, search/include/keywords/exclude as normalized (lowercased, punctuation-stripped) tokens. Only keys that were set appear. Note: `duplicate` only appears when truthy.

The top-level `complete` field is a legacy alias of `pagination.complete`.

**Errors:** 400 (deep pagination), 503 (slow-query busy), 504 (slow-query timeout) — see §10.

### 3.2 (No `/subjects/:id` route)

There is **no** per-subject endpoint. Only `GET /subjects` (below).

#### `GET /subjects`
ETag applied. `Cache-Control: public, max-age=86400`. **200**:
```json
{
  "status": "OK",
  "subjects": [
    { "id": 456789, "name": "Re:ゼロから始める異世界生活", "keywords": ["Re:Zero", "Re:ゼロから始める異世界生活", ...], "activedAt": "2026-01-07T16:00:00.000Z", "isArchived": false }
  ]
}
```
- Only **active** subjects (`isArchived = false`), loaded from the in-memory module cache (populated from the `subjects` table at startup / refresh).
- `id` = bangumi id (number), `name` = display name, `keywords` = normalized search keywords, `activedAt` = ISO date (Shanghai midnight stored as UTC instant), `isArchived` = boolean.

### 3.3 Resource detail

#### `GET /resource/{provider}/:id`
#### `GET /detail/{provider}/:id`
Both identical (aliases). `provider` must be one of the four; `:id` is the provider-scoped id (typically numeric, but treated as an opaque string — for dmhy it is the topic id, for moe the torrent id, for mikan the episode id, for ani the infoHash).

- ETag applied. `Cache-Control: public, max-age=86400` (24 h).
- Results are memoized in-process: key `"{provider}:{id}"`, TTL 60 min, max 10 000 entries (errors are cached too).

Handler flow (`findProviderDetail`):
1. Resolve the provider's canonical detail URL from `:id` (`provider.getDetailURL`). If unresolvable → **200** with:
   ```json
   { "status": "ERROR", "message": "Unknown detail id: {provider} {id}" }
   ```
2. Look up the resource row by `(provider, providerId)` in the `resources` table; if absent → **200** with `resource: undefined, detail: undefined, isDeleted: false, duplicatedId: undefined` (and no `message`).
3. Load the `details` row by `resource.id`. If the row is missing, or the resource is soft-deleted, or `now - detail.fetchedAt > DETAIL_EXPIRE` (7 days = `7*24*60*60` s), scrape the detail from the upstream provider (with error tolerance: failures just leave `detail` undefined) and upsert into `details` (Redis cache TTL 7 days, key `details:{provider}:{providerId}`).
4. Fire-and-forget "fix" jobs when the scraped detail carries data missing from the DB row (magnet/tracker split at first `&`, missing publisher/fansub avatars, resource update).

**Response 200:**
```json
{
  "status": "OK",
  "resource": {
    "id": 12345,
    "provider": "dmhy",
    "providerId": "123456",
    "title": "...",
    "href": "https://share.dmhy.org/topics/view/123456",
    "type": "动画",
    "magnet": "magnet:?xt=urn:btih:...",
    "tracker": "&tr=...",
    "size": 4831838208,
    "createdAt": "2026-02-01T08:00:00.000Z",
    "fetchedAt": "2026-02-01T08:05:00.000Z",
    "publisher": { "id": 1, "name": "LoliHouse", "avatar": "..." },
    "fansub": { "id": 2, "name": "喵萌奶茶屋", "avatar": "..." },
    "subjectId": 456789,
    "metadata": { "anipar": { ... } }
  },
  "detail": {
    "id": 12345,
    "description": "…HTML description…",
    "magnets": [ { "name": "magnet", "url": "magnet:?xt=urn:btih:...&tr=..." } ],
    "files": [ { "name": "ReZero_S3_01.mkv", "size": "1.2GB" } ],
    "hasMoreFiles": false,
    "fetchedAt": "2026-02-01T08:10:00.000Z"
  },
  "isDeleted": false,
  "duplicatedId": 0
}
```
- Note: unlike the list endpoint, detail responses **always** include `tracker` and `metadata` (no stripping here).
- `duplicatedId`: the id of the duplicate "winner" resource this row points to, or `0`/absent when the row is a root (`duplicatedId ?? undefined` — omitted when null via JSON `undefined` handling; in practice the property is serialized as `undefined` → dropped, or `0` when null).
- `detail` is omitted/`undefined` when not yet fetched or expired and the upstream scrape failed.
- `resource`/`detail` are `undefined` (dropped from JSON) when not found — the JSON then only carries `status`, `isDeleted`, `duplicatedId`.

#### `GET /detail/infohash/:hash`
Look up a resource by BTIH info hash (hex or base32), then resolve its detail.

- ETag applied. `Cache-Control: public, max-age=86400` on success; **`Cache-Control: no-store`** for invalid hash.
- Validation of `:hash` (after `trim()`):
  - uppercase, must match `^[0-9A-F]{40}$` (hex) **or** `^[A-Z2-7]{32}$` (base32).
  - Invalid / empty → **200** with:
    ```json
    {
      "status": "ERROR",
      "message": "Invalid info hash: {hash}",
      "resource": undefined,
      "detail": undefined,
      "isDeleted": false,
      "duplicatedId": undefined
    }
    ```
- Lookup: `SELECT ... FROM resources WHERE magnet ILIKE 'magnet:?xt=urn:btih:{UPPER(hash)}%' ORDER BY created_at DESC LIMIT 1` (magnet prefix match against the uppercase-normalized hash). If none → **200** with `status: "ERROR"`, `message: "Unknown detail info hash: {hash}"`, plus `resource`/`detail` `undefined`, `isDeleted: false`, `duplicatedId: undefined`.
- Otherwise the response equals the `getByProviderId` detail shape above with `status: "OK"` and the resource found by hash (the resource's own provider/providerId are used for the detail fetch).

### 3.4 Collections

#### `POST /collection` and `PUT /collection`
Create (or return the existing) collection. Body must be the collection JSON (§11). Both methods behave identically.

- No ETag. No Cache-Control.
- On invalid body → **400** `{ "status": "ERROR", "message": "Incorrect collection format" }`.
- On DB failure → **400** `{ "status": "ERROR", "message": "Failed generating collection" }`.
- On success → **200**:
  ```json
  { "status": "OK", "id": 12, "hash": "a1b2c3d4...40 hex chars", "createdAt": "2026-02-01T08:00:00.000Z" }
  ```
  (Idempotent: an existing collection with the same hash returns the stored row instead of inserting.)
- The client sends `PUT` with the collection body (filters stripped of `resources`/`complete`).

#### `GET /collection/:hash`
ETag applied. Returns the collection plus the query results of each filter.

- **200** on success:
  ```json
  {
    "status": "OK",
    "hash": "a1b2c3d4...",
    "name": "My List",
    "createdAt": "2026-02-01T08:00:00.000Z",
    "updatedAt": "2026-02-01T09:00:00.000Z",
    "filters": [
      {
        "name": "动画",
        "searchParams": "?type=%E5%8A%A8%E7%94%BB",
        "types": ["动画"],
        "fansubs": ["LoliHouse"]
      }
    ],
    "results": [
      {
        "resources": [ { ...resource objects, always with tracker & metadata... } ],
        "pagination": { "page": 1, "pageSize": 1000, "complete": true },
        "filter": { "types": ["动画"], "fansubs": ["LoliHouse"] }
      }
    ]
  }
  ```
  - `results` has one entry per filter, ordered as stored; each entry is the raw `query.find(filter, { page: 1, pageSize: 1000 })` response.
  - `filters` echoes the **stored** filter objects (empty arrays preserved, `before`/`after` as stored — ISO strings).
  - `updatedAt` is the time the response was produced (`new Date()`).
  - Internally cached (memo) keyed by hash, TTL 300 s, max 100 entries; cleared on refresh.
- Not found / DB failure → **400** `{ "status": "ERROR", "message": "Failed querying collection result" }`.

#### `GET /collection/:hash/feed.xml`
RSS feed of the collection (all filters' resources flattened). See §7. Missing hash → **400** `{ "status": "ERROR", "message": "Missing collection hash" }`; unknown hash → **400** `{ "status": "ERROR", "message": "Collection \"{hash}\" not found" }`.

### 3.5 Feed

#### `GET /feed.xml`
RSS 2.0 feed of resources matching the same filter/pagination query params as `/resources` (URL params + optional JSON body, §4) with `assertResourcesPagination` applied.

- ETag applied. `Cache-Control: public, max-age=3600`. `Content-Type: application/xml; charset=UTF-8`.
- **Tracker inclusion** is controlled by the (typo'd) query param **`trakcer`** (sic — not `tracker`):
  - absent → tracker **enabled** (default true);
  - present and one of `no|off|false` → disabled;
  - any other value → enabled.
  - The enclosure URL is `magnet + (trackerEnabled ? tracker : '')`.
- Error responses (400/503/504) are XML with `Cache-Control: no-store` — see §10.
- Exact XML structure in §7.

### 3.6 Users / teams (public)

#### `GET /users`
ETag applied. `Cache-Control: public, max-age=86400`. **200**:
```json
{
  "status": "OK",
  "users": [
    { "id": 1, "name": "anonymous", "avatar": "https://animes.garden/favicon.svg", "providers": {} },
    { "id": 2, "name": "LoliHouse", "avatar": "...", "providers": { "dmhy": { "providerId": "123", "avatar": "..." } } }
  ]
}
```
- **There are no signup/login/settings endpoints.** Users are publishers/video groups discovered from scraped resources; this endpoint merely lists them (from the in-memory `users` module cache). `providers` maps provider name → `{ providerId, avatar? }`.

#### `GET /teams`
ETag applied. `Cache-Control: public, max-age=86400`. **200**:
```json
{ "status": "OK", "teams": [ { "id": 2, "name": "喵萌奶茶屋", "avatar": "...", "providers": { "mikan": { "providerId": "42", "avatar": "..." } } } ] }
```
Fansub groups.

### 3.7 Sitemap data endpoints (JSON)

#### `GET /sitemaps/subjects`
No ETag, no Cache-Control. **200**:
```json
{
  "status": "OK",
  "subjects": [
    { "id": 456789, "activedAt": "2026-01-07T16:00:00.000Z", "isArchived": false }
  ]
}
```
All subjects (active **and** archived) sorted by `activedAt` descending.

#### `GET /sitemaps/:year/:month`
`:year` and `:month` are digits (`{[0-9]+}` route constraint). **200** when valid, otherwise **200** with error body:
- Valid range: `2020 <= year <= current year` and `1 <= month <= (year < currentYear ? 12 : currentMonth + 1)`.
- Valid response:
  ```json
  {
    "status": "OK",
    "count": 1234,
    "resources": [
      { "id": 12345, "provider": "dmhy", "providerId": "123456", "fetchedAt": "2026-02-01T08:05:00.000Z" }
    ]
  }
  ```
  Query: `resources` where `is_deleted = false AND duplicated_id IS NULL AND created_at >= <Shanghai first day of month> AND created_at < <Shanghai first day of next month>`, ordered by insertion (no explicit order), columns `id, provider, providerId, fetchedAt`. Shanghai month boundaries are computed as UTC instants `Date.UTC(y, m-1, 1) - 8h`.
- Invalid range → **200** `{ "status": "ERROR", "resources": [] }`.
- Month data memoized in-process (key `"{year}:{month}"`, TTL 1 h, max 100 entries).

### 3.8 Admin (Bearer-protected; see §12 for the auth quirk)

All admin routes require `Authorization: Bearer {secret}` (see §12 for the exact middleware semantics and the trailing-slash caveat). They return JSON.

#### `POST /admin/providers`
Re-read provider rows from DB. **200**:
```json
{
  "status": "OK",
  "providers": {
    "dmhy":  { "id": "dmhy",  "name": "动漫花园", "refreshedAt": "2026-08-18T03:00:00.000Z", "isActive": true },
    "moe":   { "id": "moe",   "name": "萌番组",   "refreshedAt": "...", "isActive": true },
    "mikan": { "id": "mikan", "name": "蜜柑计划", "refreshedAt": "...", "isActive": true },
    "ani":   { "id": "ani",   "name": "ANi",      "refreshedAt": "...", "isActive": true }
  }
}
```

#### `POST /admin/resources/{provider}`  (provider ∈ dmhy/moe/mikan/ani)
Queue a fetch job via RPC `resources.fetch { provider }`.
- **202** with ack:
  ```json
  { "status": "OK", "mode": "queued", "job": "fetch", "provider": "dmhy" }
  ```
  `mode` is `queued` (job started) or `already_running` (a job for this provider is still running; the request is not queued).
- **503** when the cron executor is unreachable: `{ "status": "ERROR", "message": "Cron service unavailable." }`

#### `POST /admin/resources/{provider}/sync`
Queue a sync job via RPC `resources.sync { provider, start, end }`.
- Query params: `start` (default `'1'`, coerced with unary `+` — `+'abc'` → NaN), `end` (default `'10'`). NaN values are passed through as NaN.
- **202** ack:
  ```json
  { "status": "OK", "mode": "queued", "job": "sync", "provider": "moe" }
  ```
- **503** when cron unavailable (same body as above).

### 3.9 MCP

#### `GET /.well-known/mcp/server-card.json`
**302** redirect to `https://{site}/.well-known/mcp/server-card.json` (site defaults to `animes.garden`). No body. Skips system-init gate.

#### `ALL /mcp`
Streamable HTTP MCP endpoint (any method; GET for SSE stream, POST for JSON-RPC messages). See §9.

---

## 4. Query-parameter / body parsing semantics

Implemented by `parseURLSearch(searchParams, body)` in `packages/client/src/resolver.ts`. It parses **both** the URL search params (into `res1`) and the optional JSON request body (into `res2`), then merges. **This is client code imported by the server** — the server literally uses the same parser.

### 4.1 Parameter inventory

**URL query params** (each `safeParse`'d; invalid values silently become `undefined`):

| Param | Cardinality | Zod schema | Notes |
|---|---|---|---|
| `provider` | single | `z.enum(['dmhy','moe','mikan','ani'])` | invalid provider → undefined (ignored) |
| `duplicate` | single | `z.coerce.boolean()` | **Zod gotcha:** coerced with `Boolean(value)` — for strings, only `""` is false; `"false"`/`"0"`/`"no"` all coerce to **true** |
| `page` | single | `z.coerce.number()` | |
| `pageSize` | single | `z.coerce.number()` | |
| `fansub` | **multi** (`getAll`) | `z.array(z.string())` | repeatable |
| `publisher` | **multi** | `z.array(z.string())` | repeatable |
| `type` | **multi** | `z.array(z.string())` | repeatable |
| `before` | single | `z.union([null, undefined, number→Date, Date])` (`dateLike`) | accepts timestamp number or date string |
| `after` | single | `dateLike` | |
| `subject` | **multi** | `z.array(z.coerce.number())` | repeatable; each coerced to number |
| `search` | **multi** | `z.array(z.string())` | repeatable (fuzzy search mode) |
| `include` | **multi** | `z.array(z.string())` | repeatable (title-match mode) |
| `keyword` | **multi** | `z.array(z.string())` | repeatable; maps to filter `keywords` |
| `exclude` | **multi** | `z.array(z.string())` | repeatable |
| `preset` | single | `z.enum(['bangumi'])` | |

**Request-body JSON fields** (valid only if the body parses as JSON; otherwise body ignored entirely):

| Field | Cardinality | Zod schema |
|---|---|---|
| `provider` | single | enum |
| `duplicate` | single | `z.coerce.boolean()` |
| `page` | single | `z.coerce.number()` |
| `pageSize` | single | `z.coerce.number()` |
| `fansub` | single | string → `[fansub]` |
| `fansubs` | array | string[] |
| `publisher` | single | string → `[publisher]` |
| `publishers` | array | string[] |
| `type` | single | string → `[type]` |
| `types` | array | string[] |
| `before` / `after` | single | `dateLike` (Date, number, date string) |
| `subject` | single | number → `[subject]` |
| `subjects` | array | number[] |
| `search` | single-or-array | string or string[] |
| `include` | single-or-array | string or string[] |
| `keywords` | single-or-array | string or string[] |
| `exclude` | single-or-array | string or string[] |
| `preset` | single | enum |

### 4.2 Merging & precedence

- **Pagination:** `page = res1.page ?? res2.page ?? 1`; `pageSize = res1.pageSize ?? res2.pageSize ?? 100` (URL wins over body).
- **preset:** body wins over URL.
- **provider:** body wins over URL. **When a provider is set (either source), `duplicate` defaults to `true`** (`res1.duplicate ?? res2.duplicate ?? true`) — i.e. filtering by a single provider includes duplicates by default. When **no** provider is set and no `duplicate` param given, `duplicate` stays `undefined` → duplicates are **excluded** (`duplicated_id IS NULL`).
- **fansubs:** body `fansub` (string) → body `fansubs` (array) → URL `fansub` (multi). Deduplicated with `Set`.
- **publishers / types / subjects:** same precedence pattern; `types`/`subjects` deduplicated.
- **before / after:** `body || url || undefined`.
- **search:** body first, then URL. Deduplicated.
- **include:** body first, then URL. Deduplicated.
- **keywords:** body `keywords` first, then URL `keyword`. Deduplicated.
- **exclude:** body first, then URL. Deduplicated.
- **search overrides include:** if `filter.search` is set, `filter.include` is **deleted**.

### 4.3 Pagination validation (exact)

After merging:

```
if (isNaN(page) || page < 1) page = 1;
else page = Math.round(page);

if (isNaN(pageSize) || pageSize < 1 || pageSize > 1000) pageSize = 100;
else pageSize = Math.round(pageSize);
```

- `DefaultPageSize = 100`, `MaxRequestPageSize = 1000`.
- **pageSize > 1000 is silently clamped to 100** (not an error).
- NaN (`?page=abc`), 0, negatives → defaults.
- Non-integer values are rounded (`Math.round`).

### 4.4 Deep pagination guard (server-side)

`assertResourcesPagination` (runs on `/resources*` and `/feed.xml`):

```
offset = (page - 1) * pageSize
if (offset + pageSize > 10000)  →  400 ResourcesDeepPaginationError
message: "Resources pagination is too deep. Please keep offset + limit <= 10000."
```

So e.g. page=100, pageSize=100 is fine (offset 9900 + 100 = 10000 ≤ 10000); page=101/pageSize=100 → 400.

### 4.5 What `complete` means

- Response `pagination.complete = !hasMore`.
- `hasMore` comes from the query execution: the normal path fetches from the in-memory Task cache and reports whether the underlying prefetch saw more rows; the DB fallback path queries `limit = pageSize + 1` and sets `hasMore = rows.length > pageSize`.
- The top-level `complete` field (legacy) equals `pagination.complete`.
- The client paginates by looping `page++` until `pagination.complete` is true or an empty page is returned.

### 4.6 `tracker` / `metadata` flags (list endpoints only)

```
isEnable(key) = query[key] !== undefined && ['true','yes','on'].includes(query[key].toLowerCase())
enableTracker  = isEnable('tracker')
enableMetadata = isEnable('metadata')
```
- `tracker` and `metadata` are **not** part of `ResolvedFilterOptions`; they only control response-body stripping.
- Default (param absent): both fields stripped from list responses.

### 4.7 Legacy flat params — mapping table

| Legacy/URL param | maps to filter | body equivalent |
|---|---|---|
| `fansub` (repeatable) | `fansubs: string[]` | `fansub` (string) / `fansubs` (array) |
| `publisher` (repeatable) | `publishers: string[]` | `publisher` / `publishers` |
| `type` (repeatable) | `types: string[]` | `type` / `types` |
| `subject` (repeatable) | `subjects: number[]` | `subject` / `subjects` |
| `search` (repeatable) | `search: string[]` | `search` (string or array) |
| `include` (repeatable) | `include: string[]` | `include` |
| `keyword` (repeatable) | `keywords: string[]` | `keywords` |
| `exclude` (repeatable) | `exclude: string[]` | `exclude` |
| `before` / `after` | `before` / `after` (Date) | same |
| `provider`, `duplicate`, `page`, `pageSize`, `preset` | same | same |

> **Note on the old `filter` JSON array param:** the version of AnimeGarden analyzed here does **not** accept a `filter=[{fansubId,...}]` query parameter anymore. The current contract is: flat query params **and/or** a JSON request body with the fields above. If the Go port must also accept the legacy `filter=` array (from much older API versions), it is not present in this codebase; the JSON body (`fansubs`/`types`/`include`/`keywords`/`exclude`/`subjects`/...) is the closest current equivalent.

---

## 5. Filter semantics & tokenization

### 5.1 Normalization pipeline (client → DB filter)

`normalizeDatabaseFilterOptions` transforms the resolved filter before querying:

- **search:** each term → `removePunctuations(term.trim())` (replace every Unicode punctuation/symbol char `[\p{P}\p{S}]` with a space) → drop empty → `normalizeTitle(term).toLowerCase()`.
- **include / keywords / exclude:** each term → `term.trim()` → drop empty → `normalizeTitle(term).toLowerCase()`.
- `normalizeTitle` = `fullToHalf(tradToSimple(title), { punctuation: true })` — traditional→simplified Chinese, full-width→half-width, punctuation normalization.
- **publishers / fansubs** (names) are resolved to DB **ids** via the in-memory `users`/`teams` modules; unknown names are dropped.
- `before` / `after` stay `Date`s; `subjects` stay numbers.

### 5.2 SQL conditions (`findFromDatabase`) — exact

Base: `is_deleted = false` always.

| Filter | SQL |
|---|---|
| `provider` | `provider = ?` |
| `!duplicate` (undefined/false) | `duplicated_id IS NULL` |
| `fansubs`/`publishers` (ids) | if only one list non-empty → `fansub_id = ?` or `publisher_id = ?` (single) / `fansub_id IN (...)` or `publisher_id IN (...)` (multi); if **both** lists present → `(fansub_id IN (...)) OR (publisher_id IN (...))` — **OR across the two groups** |
| `types` | `type = ?` (1) / `type IN (...)` |
| `subjects` | `subject_id = ?` (1) / `subject_id IN (...)` |
| `before` | `created_at <= ?` (inclusive) |
| `after` | `created_at >= ?` (inclusive) |
| `search` (tokenized, §5.4) | `title_search @@ to_tsquery('simple', 'tok1 & tok2 & ...')` |
| `include` | 1 term: `title_alt ILIKE '%term%'`; N terms: `(title_alt ILIKE '%t1%' OR title_alt ILIKE '%t2%' ...)` — **any-of** |
| `keywords` | **one `ILIKE '%k%'` per keyword, all AND-ed** (all-of) |
| `exclude` | **one `NOT ILIKE '%k%'` per keyword, all AND-ed** (none-of) |
| `preset = 'bangumi'` | additionally: `publisher_id NOT IN (<ids of banned publishers>)` and `fansub_id NOT IN (<ids of banned fansubs>)` |

Banned lists (`filter.ts`):
- `BANGUMI_BANNED_FANSUBS = ['Kirara Fantasia', '沸班亚马制作组', 'GMTeam']`
- `BANGUMI_BANNED_PUBLISHERS = ['Resona', '百度云盘', 'Lanborey']`

Order: always `ORDER BY created_at DESC` (plus id in the index order). No secondary sort by id in the SQL itself (index `resources_sort_by_created_at` is `(created_at DESC NULLS FIRST, id DESC NULLS FIRST)`).

### 5.3 In-memory task conditions (`buildFilterConds`) — used for the cached Task path

The server keeps per-filter prefetch tasks in memory; the cached slice is re-filtered in JS with equivalent semantics:

- provider: `r.provider === provider`
- duplicate excluded: `r.duplicatedId === null || undefined`
- title block (applies when include/keywords/exclude any non-empty): `title = normalizeTitle(r.title).toLowerCase()`; `include.some(i => title.includes(i))` (any-of) **AND** `keywords.every(k => title.includes(k))` (all-of) **AND** `exclude.every(e => !title.includes(e))` (none-of)
- subjects: `subjects.some(s => r.subjectId === s)`
- fansubs/publishers (ids): `publishers.some(p => r.publisherId === p) || fansubs.some(f => r.fansubId === f)` (OR)
- types: `types.some(t => r.type === t)`
- before: `r.createdAt.getTime() <= before.getTime()`
- after: `r.createdAt.getTime() >= after.getTime()`
- preset bangumi: banned publisher/fansub ids excluded

### 5.4 Search tokenization (jieba) — exact

- **Tokenizer:** `@node-rs/jieba` with the bundled default dictionary (`Jieba.withDict(dict)`).
- **Query time** (`findFromDatabase`): for each normalized search term `t`:
  ```
  tokens = jieba.cut(t, false)          // hmm=false (no HMM)
           .map(tok => tok.trim())
           .filter(tok => tok !== '')
  tsquery = tokens.flat().join(' & ')
  cond:   title_search @@ to_tsquery('simple', tsquery)
  ```
- **Index time** (`transform.ts`): on insert/update, `title_search` is a tsvector built as:
  ```
  setweight(to_tsvector('simple', <jieba tokens of anipar.title>), 'A')
  ||
  setweight(to_tsvector('simple', <jieba tokens of full titleAlt>), 'D')
  ```
  where `anipar` is the result of parsing the normalized title with `anipar` (if parse succeeds), `titleAlt = normalizeTitle(title)`. Tokens are `jieba.cut(text, false)` → trim → non-empty. `'simple'` config lowercases and strips punctuation.
- Postgres-side `@@` semantics: every token in the tsquery must be present (AND), tsvector-rank weights A/D only affect ranking, not matching.

**Important 1:1 note for Go:** to reproduce identical matching you need (a) the same dictionary + HMM-off tokenization for the query terms, and (b) the same token stream baked into the `title_search` tsvector at write time — the Go port must either replicate jieba (e.g. via a Go port of jieba with the same dict and `cut(..., false)` behavior) or persist the same token arrays. The `'simple'` text search config is Postgres built-in.

### 5.5 Duplicate handling

- `resources.duplicated_id`: rows pointing at the "winner" row of the same magnet (btih) are duplicates.
- Winner selection per magnet: candidates = all rows with the same btih (both hex and base32 magnet variants are normalized and compared), `is_deleted = false`; sorted by: provider order in `SupportProviders` (`dmhy` < `moe` < `mikan` < `ani`), then `created_at` asc, then `id` asc; first is winner, others get `duplicated_id = winner.id`.
- Maintenance runs after every upsert batch (`maintainDuplicatedResources`) and emits `attach`/`detach` notifications.
- API surface: `duplicate=true` includes duplicates in list results (rows with `duplicated_id IS NULL` condition dropped); sitemap month endpoints always exclude duplicates; detail endpoints report `duplicatedId`.

### 5.6 `preset = bangumi`

Applies the banned publisher/fansub exclusions on top of all other conditions (§5.2/§5.3). There is no other preset.

---

## 6. Database schema

PostgreSQL, tables created by drizzle migrations in `apps/server/drizzle/*.sql`. Final schema (migration 0000 → 0008).

### Enum

```sql
CREATE TYPE "public"."resources_provider" AS ENUM('dmhy', 'mikan', 'moe', 'ani');
```

### `resources`

| Column | Type | Constraints |
|---|---|---|
| `id` | serial | PK |
| `provider_name` | resources_provider | NOT NULL |
| `provider_id` | varchar(128) | NOT NULL |
| `title` | varchar(1024) | NOT NULL (original title) |
| `title_alt` | varchar(1024) | NOT NULL (normalized title: trad→simp, full→half width, punct) |
| `title_search` | tsvector | NOT NULL (weighted A/D tsvector, see §5.4) |
| `href` | text | NOT NULL (provider-relative href stored; absolute URL computed at API time) |
| `type` | varchar(64) | NOT NULL (e.g. `动画`) |
| `magnet` | varchar(256) | NOT NULL (btih magnet URI) |
| `tracker` | text | NOT NULL (`&tr=...` chain) |
| `size` | bigint | NOT NULL (bytes) |
| `created_at` | timestamptz | NOT NULL DEFAULT now() |
| `fetched_at` | timestamptz | NOT NULL DEFAULT now() |
| `indexed_at` | timestamptz | NOT NULL DEFAULT now() (added in 0008; backfilled from fetched_at) |
| `publisher_id` | integer | NOT NULL → `users.id` |
| `fansub_id` | integer | NULL → `teams.id` |
| `duplicated_id` | integer | NULL (winner row id) |
| `subject_id` | integer | NULL → `subjects.bangumi_id` |
| `metadata` | json | NULL, shape `{ "anipar": <ParseResult> }` |
| `is_deleted` | boolean | DEFAULT false (soft delete) |

**Indexes (final state after 0008):**
```sql
CREATE UNIQUE INDEX unique_resources_provider_id ON resources (provider_name, provider_id);
CREATE INDEX resources_magnet_index ON resources (magnet);
CREATE INDEX resources_publisher_id_index ON resources (publisher_id);
CREATE INDEX resources_fansub_id_index ON resources (fansub_id);
CREATE INDEX resources_subject_id_index ON resources (subject_id);
CREATE INDEX resources_sort_by_created_at ON resources (created_at DESC NULLS FIRST, id DESC NULLS FIRST);
CREATE INDEX resources_provider_created_at_id_index ON resources (provider_name, created_at DESC NULLS FIRST, id DESC NULLS FIRST) WHERE is_deleted = false;
CREATE INDEX resources_title_search_index ON resources USING gin (title_search);
CREATE INDEX resources_live_created_at_index ON resources (created_at DESC NULLS FIRST, id DESC NULLS FIRST) WHERE is_deleted = false AND duplicated_id IS NULL;
CREATE INDEX resources_live_title_alt_trgm_index ON resources USING gin (title_alt gin_trgm_ops) WHERE is_deleted = false AND duplicated_id IS NULL;
CREATE INDEX resources_live_subject_created_at_index ON resources (subject_id, created_at DESC NULLS FIRST, id DESC NULLS FIRST) WHERE is_deleted = false AND duplicated_id IS NULL AND subject_id IS NOT NULL;
CREATE INDEX resources_live_type_created_at_index ON resources (type, created_at DESC NULLS FIRST, id DESC NULLS FIRST) WHERE is_deleted = false AND duplicated_id IS NULL;
CREATE INDEX resources_live_fansub_created_at_index ON resources (fansub_id, created_at DESC NULLS FIRST, id DESC NULLS FIRST) WHERE is_deleted = false AND duplicated_id IS NULL AND fansub_id IS NOT NULL;
CREATE INDEX resources_live_publisher_created_at_index ON resources (publisher_id, created_at DESC NULLS FIRST, id DESC NULLS FIRST) WHERE is_deleted = false AND duplicated_id IS NULL;
CREATE INDEX resources_live_title_search_index ON resources USING gin (title_search) WHERE is_deleted = false AND duplicated_id IS NULL;
```
(`pg_trgm` extension required by the `gin_trgm_ops` index; `CREATE EXTENSION IF NOT EXISTS pg_trgm` in migration 0003.)

### `details`

| Column | Type | Constraints |
|---|---|---|
| `id` | integer | PK, **FK → resources.id** (no action on delete/update) |
| `description` | text | NOT NULL DEFAULT '' |
| `magnets` | json | NOT NULL DEFAULT '[]', items `{ name: string, url: string }` |
| `files` | json | NOT NULL DEFAULT '[]', items `{ name: string, size: string }` |
| `has_more_files` | boolean | NOT NULL DEFAULT false |
| `fetched_at` | timestamptz | NOT NULL DEFAULT now() |

One row per resource; `id` = resource id.

### `providers`

| Column | Type | Constraints |
|---|---|---|
| `id` | resources_provider | NOT NULL (no PK) |
| `name` | varchar(32) | NOT NULL |
| `refreshed_at` | timestamptz | NOT NULL |
| `is_active` | boolean | NOT NULL DEFAULT true |

### `subjects`

| Column | Type | Constraints |
|---|---|---|
| `bangumi_id` | integer | PK (bangumi subject id) |
| `name` | varchar(256) | NOT NULL |
| `keywords` | json | NOT NULL, `string[]` (search keywords) |
| `actived_at` | timestamptz | NOT NULL |
| `is_archived` | boolean | NOT NULL DEFAULT true |

### `users` (publishers) and `teams` (fansub groups)

Identical shape:
| Column | Type | Constraints |
|---|---|---|
| `id` | serial | PK |
| `name` | varchar(128) | NOT NULL, UNIQUE (`unique_users_name` / `unique_teams_name`) |
| `avatar` | text | NULL |
| `providers` | json | DEFAULT '{}', shape `Record<providerName, { providerId: string, avatar?: string }>` |

### `tags`

| Column | Type | Constraints |
|---|---|---|
| `id` | serial | PK |
| `name` | varchar(256) | NOT NULL, UNIQUE (`unique_tags_name`) |

(Currently unused by the API; a WIP anipar tag import exists.)

### `collections`

| Column | Type | Constraints |
|---|---|---|
| `id` | serial | PK |
| `hash` | varchar(64) | NOT NULL, UNIQUE (`unique_collections_hash`) — SHA-1 hex, 40 chars |
| `name` | varchar(64) | NOT NULL DEFAULT '' |
| `user` | varchar(64) | NOT NULL — column is literally named `user` (drizzle maps field `authorization` → column `user`) |
| `filters` | json | NOT NULL DEFAULT '[]' — `CollectionFilter[]` (§11) |
| `fetched_at` | timestamptz | NOT NULL DEFAULT now() (mapped as `createdAt`) |

### `telegram_messages`

| Column | Type | Constraints |
|---|---|---|
| `id` | serial | PK |
| `resource_id` | integer | NOT NULL |
| `publisher_id` | integer | NOT NULL |
| `fansub_id` | integer | NOT NULL (set NOT NULL in 0007) |
| `subject_id` | integer | NOT NULL |
| `episode` | varchar(128) | NOT NULL |
| `telegram_chat_id` | bigint | NULL |
| `telegram_message_id` | bigint | NULL |
| `status` | smallint | NOT NULL — `0=Pending, 1=Sending, 2=Sent, -1=Failed` |
| `sent_at` | timestamptz | NULL |
| `edited_at` | timestamptz | NULL |
| `updated_at` | timestamptz | NOT NULL DEFAULT now() |

Indexes: `unique_telegram_messages_publisher_subject_episode (publisher_id, subject_id, episode)`, `unique_telegram_messages_fansub_subject_episode (fansub_id, subject_id, episode)`, `telegram_messages_resource_id_index (resource_id)`, `telegram_messages_status_index (status)`.

### Seeds (migrations 0001 + 0005)

```sql
INSERT INTO providers VALUES ('dmhy','动漫花园',NOW(),true), ('moe','萌番组',NOW(),true), ('ani','ANi',NOW(),true);
INSERT INTO providers (id, name, refreshed_at, is_active) SELECT 'mikan','蜜柑计划',NOW(),true WHERE NOT EXISTS (...);
INSERT INTO users (name, avatar, providers) VALUES ('anonymous','https://animes.garden/favicon.svg','{}'), ('ANi','https://animes.garden/favicon.svg','{}');
INSERT INTO teams (name, avatar, providers) VALUES ('ANi','https://animes.garden/favicon.svg','{}');
```

---

## 7. RSS feed format (`/feed.xml`)

RSS 2.0 generated with fast-xml-parser `XMLBuilder` (compact output — no pretty-printing; `suppressEmptyNode: true` so empty nodes self-close).

### Channel-level

- `<title>`: from `generateTitleFromFilter(filter)`:
  - if `filter.subjects` non-empty → `<space-joined subject names> 最新动画资源` (names from the bundled bgmd subject table; fall back to skipping unknown ids; if none known falls through),
  - else if `filter.search` non-empty → `<space-joined search terms> 最新动画资源`,
  - else if `filter.include` non-empty → `<first include term> 最新动画资源`,
  - else if `filter.fansubs` has exactly 1 → `<fansub name> 最新动画资源`,
  - else if `filter.publishers` has exactly 1 → `<publisher name> 最新动画资源`,
  - else if `filter.types` has exactly 1 → `最新<type>资源`,
  - else → `所有资源`.
- `<description>`: `Anime Garden 是動漫花園資源網的第三方镜像站`
- `<link>`: `https://{site}/resources/1{url.search}` (site default `animes.garden`; `url.search` includes the leading `?` when params exist, appended verbatim — e.g. `https://animes.garden/resources/1?page=2&type=%E5%8A%A8%E7%94%BB`; trailing slash removed).

### Items (one per resource, in query order)

- `<title>`: resource title
- `<link>`: `https://{site}/detail/{provider}/{providerId}`
- `<guid isPermaLink="true">`: same URL as `<link>`
- `<pubDate>`: `toDate(resource.createdAt, { timeZone: 'Asia/Shanghai' }).toUTCString()` — i.e. the Shanghai wall-clock of `createdAt` rendered as a UTC string, e.g. `Sun, 01 Feb 2026 08:00:00 GMT`
- `<enclosure url="..." length="..." type="application/x-bittorrent"/>`:
  - `url` = `resource.magnet + (tracker enabled ? resource.tracker : '')`
  - `length` = `resource.size` (bytes, number)
  - `type` = `application/x-bittorrent`
- Tracker inclusion controlled by the `trakcer` query param (typo) — see §3.5.

### Verbatim example

```xml
<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>Re:Zero 最新动画资源</title><description>Anime Garden 是動漫花園資源網的第三方镜像站</description><link>https://animes.garden/resources/1?search=Re%3AZero</link><item><title>[LoliHouse] Re:Zero ...</title><link>https://animes.garden/detail/dmhy/123456</link><guid isPermaLink="true">https://animes.garden/detail/dmhy/123456</guid><pubDate>Sun, 01 Feb 2026 08:00:00 GMT</pubDate><enclosure url="magnet:?xt=urn:btih:ABCD...&amp;tr=https://tracker.example/announce" length="4831838208" type="application/x-bittorrent"/></item></channel></rss>
```

(Entity-escaped by the XML builder: `&` → `&amp;` etc.)

### `/collection/:hash/feed.xml` differences

- `<title>`: `${collection.name || '收藏夹 ' + hash}`
- `<description>`: `Anime Garden 是動漫花園資源網的第三方镜像站.` (with trailing period)
- `<link>`: `https://{site}/collection/{hash}`
- Items: the flattened `results[].resources` of the collection (§3.4), each item rendered identically (title/link/guid/pubDate/enclosure).

---

## 8. Sitemap format

### 8.1 What the API actually serves (JSON)

The server itself exposes **JSON** sitemap-data endpoints only (`GET /sitemaps/subjects`, `GET /sitemaps/:year/:month`, §3.7). There is **no** `/sitemap.xml` route in `apps/server`.

### 8.2 Exported (unmounted) XML helpers

`apps/server/src/server/sitemap/` exports generic middleware helpers (used by other deployments, not mounted by the API routes):

- `sitemapIndex({ getUrls })` → XML sitemap **index** via the `sitemap` npm package (`SitemapIndexStream`):
  ```xml
  <?xml version="1.0" encoding="UTF-8"?>
  <sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <sitemap><loc>https://example.com/sitemap-2026-02.xml</loc></sitemap>
  </sitemapindex>
  ```
- `sitemap({ sitemap, getURLs })` → XML urlset via `SitemapStream`:
  ```xml
  <?xml version="1.0" encoding="UTF-8"?>
  <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url><loc>https://example.com/detail/dmhy/123456</loc><lastmod>2026-02-01T08:00:00.000Z</lastmod></url>
  </urlset>
  ```
- Both respond `Content-Type: application/xml`; empty/undefined url lists → **500** with empty body; errors → 500 empty body.

The Go port should replicate the JSON endpoints exactly; the XML helpers are optional utilities.

---

## 9. MCP endpoint (`/mcp`)

### 9.1 Transport

- **Streamable HTTP** (`@hono/mcp` `StreamableHTTPTransport`):
  - `GET /mcp` — opens an SSE stream (`Accept: text/event-stream`).
  - `POST /mcp` — JSON-RPC 2.0 messages; the transport handles session management.
  - `DELETE /mcp` — terminates a session (per streamable-HTTP spec, handled by the transport).
- Session/version headers honored: `mcp-session-id`, `mcp-protocol-version` or `x-mcp-protocol-version`.
- If the request `Accept` header contains `text/html` → **400** `{ "status": "ERROR", "message": "Please connect /mcp with MCP client" }`.
- Unhandled transport errors → **500** `{ "status": "ERROR", "message": <error message or 'unknown mcp error'> }`.
- MCP server identity: `name: "animegarden"`, `version`: server package version, `websiteUrl: https://{site}`, icons: `https://animes.garden/favicon.svg`. Description enumerates capabilities and usage guidance.
- The `/.well-known/mcp/server-card.json` route **302-redirects** to the web site's server card.

### 9.2 Tools

#### `search_resources`

- Title: "Search anime torrent resources aggregated from 動漫花園, 蜜柑计划, 萌番组, ANi with Anime Garden".
- Description: documents AND-combination across groups, OR within fansubs/publishers/types/subjects/include/exclude, search taking priority over include, keywords all-of, exclude none-of.
- **Input schema** (JSON Schema generated from zod; all optional):
  ```json
  {
    "type": "object",
    "properties": {
      "fansubs":    { "type": "array", "items": { "type": "string" }, "description": "Fansub group names. Match ANY value (OR)." },
      "publishers": { "type": "array", "items": { "type": "string" }, "description": "Publisher names. Match ANY value (OR). Combined with fansubs in OR logic within this group." },
      "types":      { "type": "array", "items": { "enum": ["动画","合集","音乐","日剧","RAW","漫画","游戏","特摄","其他"] } },
      "before":     { "type": "string", "format": "date-time", "description": "Upper time bound (inclusive): createdAt <= before. Accepts date string or timestamp." },
      "after":      { "type": "string", "format": "date-time", "description": "Lower time bound (inclusive): createdAt >= after. Accepts date string or timestamp." },
      "subjects":   { "type": "array", "items": { "type": "number" }, "description": "Bangumi subject IDs. Match ANY value (OR)." },
      "search":     { "type": "array", "items": { "type": "string" }, "description": "Full-text query terms (tokenized search). Takes precedence over include." },
      "include":    { "type": "array", "items": { "type": "string" }, "description": "Title-contains terms. Match ANY value (OR). Only effective when search is not provided." },
      "keywords":   { "type": "array", "items": { "type": "string" }, "description": "Required title keywords. Title must contain ALL values (AND)." },
      "exclude":    { "type": "array", "items": { "type": "string" }, "description": "Blocked title keywords. Exclude resources containing ANY value." }
    }
  }
  ```
  (Note `before`/`after` use `z.coerce.date()` — the transport accepts ISO date strings or epoch timestamps.)
- **Execution:** body args → `parseURLSearch(undefined, args)` → `query.find(filter, { page: 1, pageSize: 30 })`.
- **Result:** `content: [{ type: 'text', text: <JSON pretty-printed (2-space) array> }]` plus `structuredContent: { resources: [...] }` where each item is:
  ```json
  {
    "id": 12345,
    "provider": "dmhy",
    "providerId": "123456",
    "title": "...",
    "uri": "animegarden://resources/dmhy/123456",
    "href": "https://animes.garden/detail/dmhy/123456",
    "type": "动画",
    "magnet": "magnet:?xt=urn:btih:...&tr=...",     // magnet + tracker concatenated
    "size": 4831838208,
    "createdAt": "2026-02-01T08:00:00.000Z",
    "publisher": "LoliHouse",
    "fansub": "喵萌奶茶屋"                             // omitted when no fansub
  }
  ```

### 9.3 Resources

- **`resource_detail`** resource template: `animegarden://resources/{provider}/{providerId}` (`list: undefined` → not advertised in `resources/list`), title "Anime Garden Resource Detail", mimeType `application/json`.
- Reading the resource:
  - `provider` must be one of the four supported; `providerId` is `decodeURIComponent`-decoded.
  - Invalid provider / empty id → content text JSON: `{ "error": "INVALID_RESOURCE_URI", "uri": "...", "message": "Expected URI format: animegarden://resources/{provider}/{providerId}, provider in [dmhy, moe, mikan, ani]." }`
  - Resource or detail not found → `{ "error": "RESOURCE_NOT_FOUND", "provider": "...", "providerId": "..." }`
  - Success → JSON text:
    ```json
    {
      "id": 12345,
      "provider": "dmhy",
      "providerId": "123456",
      "title": "...",
      "uri": "animegarden://resources/dmhy/123456",
      "href": "https://animes.garden/detail/dmhy/123456",
      "type": "动画",
      "magnet": "magnet:?xt=urn:btih:...&tr=...",
      "size": 4831838208,
      "createdAt": "2026-02-01T08:00:00.000Z",
      "publisher": "LoliHouse",
      "fansub": "喵萌奶茶屋",
      "description": "…",
      "files": [ { "name": "ReZero_S3_01.mkv", "size": "1.2GB" } ],
      "hasMoreFiles": false
    }
    ```
- No prompts are registered.

---

## 10. Error format

### 10.1 JSON error envelope

```json
{ "status": "ERROR", "message": "human readable message" }
```

- Unhandled internal errors: `500` with `{ "status": "ERROR" }` (no `message`).
- JSON responses always get `Content-Type: application/json; charset=utf-8` (enforced by middleware).
- HTTPException-derived responses (408 timeout; 400/401 from bearer-auth) are returned **verbatim** by the error handler — their bodies are whatever the middleware produced (see below).

### 10.2 Status codes per failure mode

| Code | Condition | Body |
|---|---|---|
| 400 | Deep pagination: `(page-1)*pageSize + pageSize > 10000` | `{status:'ERROR', message:'Resources pagination is too deep. Please keep offset + limit <= 10000.'}` |
| 400 | Invalid collection body | `{status:'ERROR', message:'Incorrect collection format'}` |
| 400 | Collection generation DB failure | `{status:'ERROR', message:'Failed generating collection'}` |
| 400 | `GET /collection/:hash` miss | `{status:'ERROR', message:'Failed querying collection result'}` |
| 400 | Collection feed missing hash | `{status:'ERROR', message:'Missing collection hash'}` |
| 400 | Collection feed unknown hash | `{status:'ERROR', message:'Collection "{hash}" not found'}` |
| 400 | MCP with `Accept: text/html` | `{status:'ERROR', message:'Please connect /mcp with MCP client'}` |
| 400 | bearer-auth malformed Authorization header | text `Unauthorized` + `WWW-Authenticate: Bearer error="invalid_request"` (see §12) |
| 401 | bearer-auth missing/wrong token | text `Unauthorized` + `WWW-Authenticate` header (see §12) |
| 408 | 60 s request timeout | `{status:'ERROR', message:'Request timeout after waiting 60 seconds. Please try again later.'}` |
| 500 | Unknown error | `{status:'ERROR'}` |
| 503 | Slow-query busy (concurrent slow query / lock held) | `{status:'ERROR', message:'Resources slow database query is busy. Please retry later.'}` |
| 503 | Admin RPC: cron unreachable/timeout | `{status:'ERROR', message:'Cron service unavailable.'}` |
| 504 | Slow-query statement timeout on extended lane | `{status:'ERROR', message:'Resources query exceeded both normal and extended database timeouts.'}` |

### 10.3 `"OK"`-statused error bodies (still HTTP 200)

- `GET /resource|detail/{provider}/{id}` with unresolvable detail URL → `{status:'ERROR', message:'Unknown detail id: {provider} {id}'}`
- `GET /detail/infohash/:hash` invalid hash → `{status:'ERROR', message:'Invalid info hash: {hash}', resource:null, detail:null, isDeleted:false, duplicatedId:null}`
- `GET /detail/infohash/:hash` not found → `{status:'ERROR', message:'Unknown detail info hash: {hash}', resource:null, detail:null, isDeleted:false, duplicatedId:null}`
- `GET /sitemaps/:year/:month` out of range → `{status:'ERROR', resources:[]}`

### 10.4 XML error for feeds

When a resources-query error (400/503/504) occurs on a path ending in `/feed.xml`:
```
Content-Type: application/xml; charset=UTF-8
Cache-Control: no-store

<?xml version="1.0" encoding="UTF-8"?><error><message>Resources pagination is too deep. Please keep offset + limit &lt;= 10000.</message></error>
```
Message is XML-escaped: `&` → `&amp;`, `<` → `&lt;`, `>` → `&gt;`, `"` → `&quot;`, `'` → `&apos;`.

---

## 11. Collections

### 11.1 Collection JSON (request/response)

```json
{
  "hash": "optional input, ignored on create",
  "name": "My List",
  "authorization": "some-user",
  "filters": [
    {
      "name": "动画",
      "searchParams": "?type=%E5%8A%A8%E7%94%BB",
      "preset": "bangumi",
      "provider": "dmhy",
      "duplicate": true,
      "types": ["动画"],
      "after": "2026-02-01T00:00:00.000Z",
      "before": "2026-02-28T23:59:59.000Z",
      "fansubs": ["LoliHouse"],
      "publishers": ["ANi"],
      "subjects": [456789],
      "search": ["Re:Zero"],
      "include": ["Re:Zero"],
      "keywords": ["1080p"],
      "exclude": ["合集"]
    }
  ]
}
```

Validation (zod `CollectionSchema`):
- `hash`: optional string (ignored; server computes it).
- `name`: coerced string, default `''`.
- `authorization`: **required** string (an arbitrary owner label — there is no auth check on read; this is the legacy "user" column).
- `filters`: array, **min 1, max 50**. Each item: `name` (coerced, default `''`), `searchParams` (**required** string — the query-string used to build the filter), optional `preset`/`provider`/`duplicate`/`types`/`after`/`before`/`fansubs`/`publishers`/`subjects`/`search`/`include`/`keywords`/`exclude`, `.passthrough()` (unknown keys are preserved).
- `after`/`before` coerced to Date.

### 11.2 Hash generation (exact, for 1:1 reproduction)

1. Copy the filters array; **sort by `searchParams.localeCompare(searchParams)`** (ascending, locale compare).
2. For each filter, delete the keys `name`, `searchParams`, `resources`, `complete`.
3. Serialize the resulting array with **`ohash` `serialize()`** — deterministic serialization:
   - object keys sorted alphabetically,
   - arrays `[a,b]`,
   - strings single-quoted `'...'` with JSON escaping,
   - numbers bare, booleans `true`/`false`,
   - `Date` → `Date(2026-02-01T00:00:00.000Z)`,
   - e.g. `[{exclude:['合集'],search:['Re:Zero']},{after:Date(2026-02-01T00:00:00.000Z),fansubs:['LoliHouse'],subjects:[123456],type:'动画',types:['动画']}]`
4. Hash the UTF-8 bytes of that string with **SHA-1**; lowercase hex, 40 chars → `hash`.

The server stores by that hash; creating an identical collection returns the existing row (`onConflictDoNothing` → select by hash).

### 11.3 What the server returns

- **Create** (`POST`/`PUT /collection`): `{ "status":"OK", "id": 12, "hash": "<40 hex>", "createdAt": "<ISO>" }`.
- **Read** (`GET /collection/:hash`): `{ status, hash, name, createdAt, updatedAt, filters (stored, as-is), results: [ { resources, pagination: { page:1, pageSize:1000, complete }, filter } ] }` — see §3.4.

The read path queries each filter with `query.find(filter, { page: 1, pageSize: 1000 })`; internally, before querying, each stored filter is normalized (empty arrays removed; `before`/`after` restored to Date). Resource objects in `results` always include `tracker` and `metadata` (no stripping in this path).

---

## 12. Admin & user endpoints, authentication model

### 12.1 Authentication

- **Secret source:** CLI `--secret` / env `ADMIN_SECRET` / env `SECRET`. If unset, a random 32-char password (charset `a-z0-9`) is generated and logged. Stored process-globally, exposed as `system.secret`.
- **Scheme:** HTTP Bearer token. Middleware: Hono `bearerAuth({ token: sys.secret })`, registered as `app.use('/admin/', auth)`.
  - Token pattern: `[A-Za-z0-9._~+/-]+=*`; header `Authorization: Bearer <token>` (prefix case-insensitive, requires a space after `Bearer`).
  - Missing header → **401**, body text/plain `Unauthorized`, `WWW-Authenticate: Bearer realm=""`.
  - Malformed header (no Bearer prefix) → **400**, body `Unauthorized`, `WWW-Authenticate: Bearer error="invalid_request"`.
  - Wrong token → **401**, body `Unauthorized`, `WWW-Authenticate: Bearer error="invalid_token"`.
  - **⚠ Trailing-slash quirk (verified against Hono 4.12.28):** `app.use('/admin/', ...)` matches **only the literal path `/admin/`**, not `/admin/*` descendants. As written upstream, the bearer middleware does **not** actually guard `/admin/providers`, `/admin/resources/*`, `/admin/resources/*/sync`. The Go port should decide: reproduce the (buggy) literal behavior, or apply Bearer auth to all `/admin/*` routes as clearly intended. The admin CLI and tests always send the `Authorization` header.
- **There are no user accounts, signup, login, sessions, or tokens for public endpoints.** `/users` and `/teams` are public read-only listings of publisher/fansub groups discovered from scraped resources (in-memory caches of the `users`/`teams` tables). `collections.authorization` is a free-form label, not a credential — `GET /collection/:hash` is public.

### 12.2 Admin endpoints (all POST)

| Route | Query params | Body | Success | Failure |
|---|---|---|---|---|
| `/admin/providers` | – | – | 200 `{status:'OK', providers:{id:{id,name,refreshedAt,isActive}}}` | – |
| `/admin/resources/{provider}` | – | – | 202 `{status:'OK', mode:'queued'\|'already_running', job:'fetch', provider}` | 503 `Cron service unavailable.` |
| `/admin/resources/{provider}/sync` | `start` (default 1, unary `+`), `end` (default 10, unary `+`) | – | 202 `{status:'OK', mode:'queued'\|'already_running', job:'sync', provider}` | 503 `Cron service unavailable.` |

- RPC round-trip: the API server publishes to Redis channel `invoke-rpc` and waits for a reply on `reply-rpc:<uuid>`; 30 s timeout → treated as unavailable → 503.
- `mode: 'already_running'` when a job for that provider is already in flight (coordinator per provider; a running provider cannot queue another job).

### 12.3 User-facing endpoints recap

`GET /users` (public, 24 h cache), `GET /teams` (public, 24 h cache) — full shapes in §3.6. No other user endpoints exist.

---

## 13. Appendix: shared types, seeds, constants

### 13.1 Constants

```ts
DefaultPageSize = 100
MaxRequestPageSize = 1000
DefaultBaseURL = 'https://api.animes.garden/'
SupportProviders = ['dmhy', 'moe', 'mikan', 'ani']
SupportPresets = ['bangumi']
MAX_RESOURCES_OFFSET_LIMIT = 10000            // deep pagination guard
MAX_DETAIL_CACHE_COUNT = 10000                // in-memory detail memo size
DETAIL_EXPIRE = 7 * 24 * 60 * 60 (seconds)    // detail freshness
MAX_RESOURCES_TASK_COUNT = 100                // in-memory query task cap
MAX_COLLECTION_COUNT = 100                    // collection memo cap
RESOURCES_TASK_PREFETCH_COUNT = 200
RESOURCES_TASK_PREFETCH_MAX_COUNT = 1000
NOTIFY_CHANNEL = 'notify-resources'           // Redis pub/sub
RPC_INVOKE_CHANNEL = 'invoke-rpc'
RPC_REPLY_CHANNEL = 'reply-rpc'
anonymous = 'anonymous'                       // fallback publisher name
```

### 13.2 `anipar` ParseResult (metadata.anipar) — top-level shape

```ts
{
  title: string;
  titles?: string[];
  fansub?: { name: string; alias?: string; collab?: string[]; tags?: string[] };
  season?: { number: number; title?: string };
  seasons?: SeasonInfo[];
  seasonsRange?: { from: number; to: number };
  part?: { number: number };
  type?: string;                       // e.g. "OVA"
  episode?: { number: number; numberSub?: number; type?: string; title?: string };
  volume?: { number: number };
  volumes?: VolumeInfo[];
  volumesRange?: VolumesRange;
  episodes?: EpisodeInfo[];
  episodesRange?: { from: number; fromSub?: number; to: number; toSub?: number; type?: string };
  version?: number;
  subtitle?: { format?: string; encoding?: string; encodings?: string[]; languages?: string[] };
  source?: string;                     // e.g. "WEB-DL"
  platform?: string;                   // e.g. "Baha"
  year?: number;
  month?: number;
  file?: {
    extension?: string;
    audio?: { channels?: string; codec?: string; language?: string; trackCount?: number };
    video?: { codec?: string; enhancement?: string; format?: string; frameRateMode?: string;
              quality?: string; resolution?: string; bitDepth?: string; fps?: string };
  };
  tmdbId?: string;
  tags?: string[];
  variants?: string[];
  search?: string[];
}
```

### 13.3 Size parsing (write path)

`size` strings like `"x KB"`, `"x MB"`, `"x GB"`, `"x TB"` (case-insensitive, optional space, optional `i`) → bytes (`1 KB = 1024 B`, `1 MB = 1024^2`, …); bare numbers parsed with `parseInt`; unparseable → `0`. Stored as bigint bytes; API returns the number.

### 13.4 Magnet / tracker conventions

- `resources.magnet` = `magnet:?xt=urn:btih:<hex|base32>` (no tracker params).
- `resources.tracker` = `&tr=<url>` (possibly multiple `&tr=`).
- Concatenated full magnet = `magnet + tracker` (used by feed enclosures and MCP).
- btih normalization for duplicate detection: `normalizeBtihToHex`/`normalizeBtihToBase32` convert between the two encodings (RFC 4648, no padding); `extractBtihFromMagnet` scans `xt=` params.

### 13.5 Cache-Control summary

| Endpoint | Cache-Control |
|---|---|
| `/resources*` (all variants) | `public, max-age=300` |
| `/resource/{p}/{id}`, `/detail/{p}/{id}`, `/detail/infohash/:hash` (valid) | `public, max-age=86400` |
| `/detail/infohash/:hash` (invalid) | `no-store` |
| `/feed.xml`, `/collection/:hash/feed.xml` | `public, max-age=3600` |
| feed XML errors | `no-store` |
| `/subjects`, `/users`, `/teams` | `public, max-age=86400` |
| `/`, `/health`, `/sitemaps/*`, `/collection*`, `/admin/*`, `/mcp` | none set |
| 304 responses | only `cache-control, content-location, date, etag, expires, vary` + `ETag` (+ outer `X-Request-Id`, `X-Response-Timestamp`) |

### 13.6 ETag scope

ETag middleware (`safeEtag`) applies to: `/resources*`, `/resource/{p}/{id}`, `/detail/{p}/{id}`, `/detail/infohash/:hash`, `/feed.xml`, `/collection/:hash/feed.xml`, `/collection/:hash`, `/subjects`, `/users`, `/teams`. SHA-1 strong ETag; `If-None-Match` honored (comma-separated, `W/` stripped); only 200 responses are etagged.

### 13.7 Request headers the server reads

- `X-Request-Id` (echoed back; else generated UUID)
- `If-None-Match` (ETag)
- `Authorization: Bearer <secret>` (admin only)
- `Accept` / `Content-Type` / `mcp-session-id` / `mcp-protocol-version` / `x-mcp-protocol-version` / `User-Agent` (MCP)

### 13.8 Response headers the server always sets

- `X-Request-Id`
- `X-Response-Timestamp` (ISO-8601)
- `Content-Type: application/json; charset=utf-8` for JSON responses
- `Access-Control-Allow-Origin: *` (+ CORS preflight handling for GET/HEAD/PUT/POST/DELETE/PATCH/OPTIONS)
- `ETag` where applicable; `Cache-Control` per table above.
