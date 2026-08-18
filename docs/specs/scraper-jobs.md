# AnimeGarden Scraper / Providers / Jobs / Push — Complete Specification for Go Reimplementation

Source of truth: `/tmp/animegarden-orig` (TypeScript monorepo, v0.5.4). This spec covers
`packages/scraper`, `packages/anipar`, `packages/shared`, `packages/client`,
`apps/server/src/{providers,resources,system,push,subjects,tags,utils,server/routes/admin,cli.ts}`,
and `apps/worker`.

AnimeGarden aggregates anime torrent resources from 4 upstream providers:

| id | Name | Source |
|----|------|--------|
| `dmhy` | 動漫花園 | https://share.dmhy.org |
| `mikan` | 蜜柑计划 | https://mikanani.kas.pub (mirror) |
| `moe` | 萌番组 | https://bangumi.moe |
| `ani` | ANi / ANi-ONE | https://api.ani.rip RSS + nyaa.si details |

`SupportProviders = ['dmhy', 'moe', 'mikan', 'ani']` (order = business priority,
`dmhy > moe > mikan > ani`). Provider seeds in DB (`drizzle/0001`, `0005`):
`('dmhy','动漫花园')`, `('moe','萌番组')`, `('ani','ANi')`, `('mikan','蜜柑计划')`.

---

## 1. Provider list

Each provider is implemented twice:

- `packages/scraper/src/{dmhy,moe,mikan,ani}/index.ts` — dumb fetch+parse functions
  `fetchXxxPage(ofetch, {page, retry})` / `fetchXxxDetail(ofetch, id|href, {retry})`.
  `ofetch` is an injected `fetch`-like function (the server passes global `fetch`;
  the CLI installs `EnvHttpProxyAgent` so outbound traffic honors `HTTP_PROXY/HTTPS_PROXY`).
- `apps/server/src/providers/scraper/{dmhy,mikan,moe,ani}.ts` — `Provider` subclasses
  that wire the scraper into the system (`fetchLatestResources`, `fetchResourcePages`,
  `fetchResourceDetail`, `getDetailURL`). Registered in `ScraperProviders` map keyed by id.

### 1.1 dmhy (動漫花園) — `packages/scraper/src/dmhy/index.ts`

- **List**: `GET https://share.dmhy.org/topics/list/page/{page}` (page starts at 1).
  No special headers. HTML parsed with JSDOM.
- **Error detection**: response `!ok` → `NetworkError`; HTML containing
  `.ui-state-error` element → `NetworkError`.
- **Row selector**: `#topic_list tbody tr`. For each row, `td` array:
  - `td[0]` — `span` text = raw publish time, e.g. `2025/05/07 13:00`.
    Parsed with `toShanghai(new Date(raw)).toISOString()`:
    `toShanghai` = `new Date(date.getTime() + (-480 - new Date().getTimezoneOffset()) * 60_000)`.
    Semantic: the raw wall time is **Asia/Shanghai**; the result is the correct UTC instant.
    In Go: parse the raw string as `Asia/Shanghai` wall-clock, convert to UTC ISO.
  - `td[1]` — category text. Mapping: `DisplayType[trad] ?? raw` then `SimpleType[...] ?? '动画'`.
    `DisplayType`: `動畫→动画, 季度全集, 音樂→音乐, 動漫音樂→动漫音乐, 同人音樂→同人音乐,
    流行音樂→流行音乐, 日劇→日剧, ＲＡＷ→RAW, 其他, 漫畫→漫画, 港台原版, 日文原版,
    遊戲→游戏, 電腦遊戲→电脑游戏, 電視遊戲→主机游戏, 掌機遊戲→掌机游戏, 網絡遊戲→网络游戏,
    遊戲周邊→游戏周边, 特攝→特摄`. `SimpleType` canonicalizes to one of:
    `动画, 合集, 音乐, 日剧, RAW, 其他, 漫画, 游戏, 特摄`.
  - `td[2]` — first `<a>` child = title text + `href`. Absolute detail URL =
    `'https://share.dmhy.org' + href`. Fansub: `span.tag a` → `{ id: href last segment, name: text }`.
  - `td[3]` — first `<a>` `href` = full magnet+tracks; `splitOnce(magnetFull, '&')`
    → `magnet` = text before first `&`, `tracker` = text **from the `&` inclusive**
    (i.e. `'&tr=...'`, may contain many `tr=` params).
  - `td[4]` — size text, e.g. `499.18 MB`.
  - `td[8]` — publisher `<a>` → `{ id: href last segment, name: text }`.
  - `providerId` = `/^(\d+)/.exec(lastHrefSegment)[1]` (leading digits of the URL tail,
    e.g. `12345`-75545 → `12345`). Stored `href` = the raw last segment (NOT the full URL).
  - **Title cleanups** (shared across providers, see §1.5):
    1. strip zero-width chars `[\u200B-\u200D\uFEFF]`; `[[`→`[`; `]]`→`]`;
    2. if fansub name `== 'ANi'`, collapse spaces then strip trailing
       `.torrent|.mp3|.MP3|.mp4|.MP4|.mkv|.MKV`; otherwise just collapse spaces;
    3. strip trailing `v2`;
    4. publisher `ANiTorrent` → `ANi`;
    5. fansub `悠哈C9字幕社` → `悠哈璃羽字幕社` (publisher name too);
    6. publisher `灼眼のシャナ` with id `110897` → publisher `ANi`/`747291`,
       fansub `ANi`/`816`, and strip leading `[搬運]` off the title.
- **Detail**: `GET {new URL(href, 'https://share.dmhy.org/topics/view/').href}`,
  i.e. `https://share.dmhy.org/topics/view/{id}` (or `{id}-xxx.html`). Selectors:
  - title `.topic-main>.topic-title>h3` — if missing → `undefined` (resource deleted).
  - type `.topic-main>.topic-title li:first-child a:last-of-type` (same mapping).
  - size `.topic-main>.topic-title li:nth-child(5) span`.
  - createdAt `li:nth-child(2) span` (same `toShanghai` parse).
  - publisher avatar `.topics_bk .avatar:first-child img` src,
    default `https://share.dmhy.org/images/defaultUser.png`;
    name/link `p:nth-child(2) a` (href last segment = user id).
  - fansub avatar `.topics_bk .avatar:nth-child(2) img` src,
    default `https://share.dmhy.org/images/defaultTeam.gif`;
    name/link `p:nth-child(2) a` (href last segment = team id).
  - description = innerHTML of `.topic-nfo`.
  - magnets (fixed array):
    `{name:'会员专用链接', url: #resource-tabs #tabs-1 p:nth-child(1) a href}` (prepend
    `https:` if it does not already start with `https://`),
    `{name:'磁力链接', url: #a_magnet href}`,
    `{name:'磁力链接 type II', url: #magnet2 href}`,
    `{name:'弹幕播放链接', url: #ddplay text}`.
  - files from `.file_list li`: size = first `SPAN` child text, name = `li.textContent`
    minus the size suffix; drop entries whose size is `種子可能不存在` or `Bytes` or with no
    name; if a file name matches `/More Than \d+ Files/` set `hasMoreFiles = true`.

### 1.2 moe (萌番组 / bangumi.moe) — `packages/scraper/src/moe/index.ts`

- **List**: `GET https://bangumi.moe/api/torrent/page/{page}` with UA
  `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0`.
  JSON response: `{ torrents: [...] }`.
- Torrent object fields used: `_id` (Mongo id), `title`, `magnet`, `size` (usually a
  human string, e.g. `499.18 MB`), `publish_time` (ISO-ish string), `uploader_id`,
  `team_id?`, `tag_ids: string[]`, `introduction` (detail description),
  `content: Array<[fileName, sizeStr]>` (detail file list).
- For every torrent the page fetcher resolves the uploader (and team when `team_id` is
  set) through POST JSON endpoints with the same UA:
  - `POST https://bangumi.moe/api/user/fetch` body `{"_ids":[id]}` →
    `[{username, emailHash}]`.
  - `POST https://bangumi.moe/api/team/fetch` body `{"_ids":[id]}` → `[{name, icon}]`.
  - Results are cached in module-level `Map`s (unbounded, no TTL) — reuse the same
    interning in Go.
- `tracker` = hard-coded constant (long `&tr=...` list of public trackers):
  `&tr=https%3A%2F%2Ftr.bangumi.moe%3A9696%2Fannounce&tr=http%3A%2F%2Ftr.bangumi.moe%3A6969%2Fannounce&tr=udp%3A%2F%2Ftr.bangumi.moe%3A6969%2Fannounce&tr=http%3A%2F%2Fopen.acgtracker.com%3A1096%2Fannounce&tr=http%3A%2F%2F208.67.16.113%3A8000%2Fannounce&tr=udp%3A%2F%2F208.67.16.113%3A8000%2Fannounce&tr=http%3A%2F%2Ftracker.ktxp.com%3A6868%2Fannounce&tr=http%3A%2F%2Ftracker.ktxp.com%3A7070%2Fannounce&tr=http%3A%2F%2Ft2.popgo.org%3A7456%2Fannonce&tr=http%3A%2F%2Fbt.sc-ol.com%3A2710%2Fannounce&tr=http%3A%2F%2Fshare.camoe.cn%3A8080%2Fannounce&tr=http%3A%2F%2F61.154.116.205%3A8000%2Fannounce&tr=http%3A%2F%2Fbt.rghost.net%3A80%2Fannounce&tr=http%3A%2F%2Ftracker.openbittorrent.com%3A80%2Fannounce&tr=http%3A%2F%2Ftracker.publicbt.com%3A80%2Fannounce&tr=http%3A%2F%2Ftracker.prq.to%2Fannounce&tr=http%3A%2F%2Fopen.nyaatorrents.info%3A6544%2Fannounce`
- **Type** from tag ids (`moe/tag.ts`) — first matching tag in array order wins, default `其他`:
  `549ef207fe682f7549f1ea90`→动画, `54967e14ff43b99e284d0bf7`→合集,
  `549eefebfe682f7549f1ea8c`→漫画, `549eef6ffe682f7549f1ea8b`→音乐,
  `549ff1db30bcfc225bf9e607`→日剧, `549ef015fe682f7549f1ea8d`→游戏,
  `549ef250fe682f7549f1ea91`→其他.
- Publisher: `{ id: uploader_id, name: user.name, avatar: team?.avatar ?? (emailHash ? `https://static.bangumi.moe/avatar/${emailHash}` : undefined) }`.
  Fansub: `{ id: team_id, name: team.name, avatar: team.icon }` (only when team present).
- Title hacks: same as §1.5; also `云光字幕组` treated like `ANi` (strip media-extension suffix).
- `href` = `_id`.
- **Detail**: `POST https://bangumi.moe/api/torrent/fetch` body `{"_id": id}` → single
  torrent object. Fields: description = `introduction`; magnets =
  `[{name:'磁力链接', url: torrent.magnet}]`; files = `content.map(([name,size]) => ({name,size}))`;
  `hasMoreFiles: false`; same publisher/fansub resolution; `href = https://bangumi.moe/torrent/{_id}`.

### 1.3 mikan (蜜柑计划) — `packages/scraper/src/mikan/index.ts`

- **Base** is the mirror `https://mikanani.kas.pub` (NOT mikanani.me — the display href
  uses `https://mikanani.me/Home/Episode/{id}`, see §5).
- **List**: `GET {BASE}/Home/Classic/{page}`. No headers. HTML parsed with JSDOM.
- Row selector `table.table tbody tr` (needs ≥4 `td`):
  - `td[0]` — date text. `parseMikanDate(text, now)`:
    - strip prefix `发布日期[:：]\s*`;
    - `^(今天|昨天)\s+(\d{1,2}):(\d{2})$` → today/yesterday at HH:MM in Asia/Shanghai;
    - `^(\d{4})[-/.年](\d{1,2})[-/.月](\d{1,2})(?:日)?(?:\s+(\d{1,2}):(\d{2}))?$`
      → absolute date (Shanghai);
    - `^(\d{1,2})[-/.月](\d{1,2})(?:日)?\s+(\d{1,2}):(\d{2})$` → yearless (assume current Shanghai year);
    - else `new Date(raw)`; unparseable → throws → the row is skipped.
    Conversion: `toShanghaiISOString(y,m,d,h=0,min=0)` = `Date.UTC(y, m-1, d, h-8, min)`.
  - `td[1]` — first `<a href*="/Home/PublishGroup/">` whose href matches
    `/^\/Home\/PublishGroup\/([^/?#]+)/` → `{ id, name }`; used for **both**
    `publisher` and `fansub` of the row.
  - `td[2]` — first `a[href*="/Home/Episode/"]` → title text + href;
    `providerId` = first path segment after `/Home/Episode/`;
    magnet from `[data-clipboard-text]` attribute → `splitOnce('&')`.
  - `td[3]` — size text.
  - `type` is always `'动画'`.
- Server-side `MikanProvider` additionally filters page results to
  `r.fansub && r.publisher` (rows without a publish group are dropped).
- **Detail**: `GET {BASE}/Home/Episode/{id}`.
  - title from `<title>` with suffix `' - Mikan Project'` stripped, falling back to
    `.episode-title` text; missing → `undefined`.
  - description = innerHTML of `.episode-desc` after removing any `div` containing
    `img[src*="/images/SSWJ/"]` or `a[href*="equity.tmall.com"]` (ads).
  - createdAt from a `.bangumi-info` whose text starts with `发布日期` (same `parseMikanDate`).
  - size from `.bangumi-info` starting with `文件大小`.
  - group = `parseFirstPublishGroup(document.querySelector('.leftbar-container') ?? document)`.
  - magnet from `.leftbar-nav a[href^="magnet:"]` (missing → `undefined`);
  - torrent file from `.leftbar-nav a[href$=".torrent"]` → magnet list:
    `[{name:'种子', url: resolveUrl(downloadHref, '/')}, {name:'磁力链接', url: fullMagnet}]`.
  - `files: []`, `hasMoreFiles: false`.

### 1.4 ani (ANi) — `packages/scraper/src/ani/index.ts`

- **List**: `GET https://api.ani.rip/ani-torrent.xml` (RSS) with the moe-style UA header.
  Parsed with `rss-parser`. Item requirements: `title`, `pubDate`, `enclosure?.length`,
  and `link` ending in `.torrent`.
- For each item, the .torrent file at `link` is downloaded (with UA, retry 5) and parsed
  with `parse-torrent`; magnet = `toMagnetURI(torrent)` split once at `&`
  (`tracker` is typically empty for ANi).
- `size` = `parseSize(enclosure.length)` formatting: `<1KB → 'n B'`, `<1MB → 'n.n KB'`,
  `<1GB → 'n.n MB'`, else `'n.n GB'` (1 decimal).
- `providerId`: if `link` starts with `https://tds.ani.rip/` → the btih hash
  (`magnet.slice('magnet:?xt=urn:btih:'.length)`), otherwise `filename` without its
  extension.
- Title: `removeExtraSpaces` then `replaceSuffix` `{'.torrent': '[MP4]', '.mp4': '[MP4]',
  '.MP4': '[MP4]', '.mkv': '[MKV]', '.MKV': '[MKV]'}`, then `transformAlias()` which runs
  `anipar.parse(title, {fansub:'ANi'})` and rewrites `A - B` alias pairs into
  `A / B` (joins duplicate-titled pairs with ` / `).
- `publisher = { id:'1', name:'ANi' }`, `fansub = { id:'1', name:'ANi' }` (fixed,
  no avatar); `type: '动画'`; `createdAt = item.pubDate`; `href = link`.
- `fetchResourcePages` ignores start/end — always fetches the single latest feed.
- **Detail**: `GET https://nyaa.si/view/{id}` with UA. Selectors:
  - title `.panel-heading .panel-title` (missing → `undefined`);
  - description = innerHTML of `#torrent-description`;
  - createdAt = `data-timestamp` (unix seconds) of `.panel-body .col-md-5:last-child`
    → `new Date(ts*1000).toISOString()`;
  - magnet = `.panel-footer a.card-footer-item` href split once at `&`;
  - size = `.panel-body .row:nth-child(4) .col-md-5` text;
  - files: `.torrent-file-list li` → name = `file.childNodes[1].textContent`,
    size = `.file-size` text with the outer parentheses stripped
    (`slice(1, length-1)`), empty size OK;
  - magnets: `[{name:'种子', url:'https://nyaa.si/download/{id}.torrent'},
    {name:'磁力链接', url: fullMagnet}]`; `hasMoreFiles: false`.

### 1.5 Shared title cleanups & common ScrapedResource shape

```ts
interface ScrapedResource {
  provider: string; providerId: string; title: string; href: string;
  type: string; magnet: string; tracker: string; size: string; // size is ALWAYS a string here
  publisher?: { id: string; name: string; avatar?: string };
  fansub?:    { id: string; name: string; avatar?: string };
  createdAt: string; // Date.toISOString()
}
interface ScrapedResourceDetail extends Omit<ScrapedResource,'magnet'|'tracker'> {
  description: string; files: Array<{name:string;size:string}>;
  magnets: Array<{name:string;url:string}>; hasMoreFiles: boolean;
}
```

Pre-cleanup pipeline (all providers): remove zero-width chars, `[[`→`[`, `]]`→`]`,
collapse whitespace (`removeExtraSpaces`), for ANi-tagged releases strip media extension
suffix, strip trailing `v2`.

HTTP client behavior: no rate limiting, no backoff — `retryFn` re-invokes immediately
`count+1` times (default 5) on thrown errors; only `NetworkError` (bad status) and mikan's
date-parse failures drive retry/abort behavior.

---

## 2. Scrape flow

### 2.1 Latest fetch (`fetchLatestResources`)

`apps/server/src/providers/scraper/base.ts` → `fetchLatestPages(sys, provider, fetchPage)`:

1. `visited = Map<providerId, ScrapedResource>` for this run.
2. Loop `page = 1, 2, 3, ...`:
   - `resp = fetch(page)` (each page function passes `retry: 5`);
   - `newRes = resp` minus ids already in `visited`;
   - query DB `getResourcesByProviderId(provider, newRes ids)` and build a set of
     existing ids;
   - `realNewRes = newRes` minus ids present in DB; add to `visited`;
   - **stop when a page yields 0 real-new resources**; also stop implicitly if a fetch throws.
3. Return `[...visited.values()]` (unordered map values; jobs sort by `createdAt` later).

ANi overrides: single RSS fetch then filter against DB only (no pagination).

Mikan: page results pre-filtered to rows with both fansub and publisher.

### 2.2 Sync fetch (`fetchResourcePages(sys, start, end)`)

`fetchResourcePages` in `base.ts`: loop `page = start..end`, each page deduped against the
run-local `visited` map only (no DB check, no early stop). ANi ignores the range (returns
latest feed).

### 2.3 Job wiring — `resources/jobs.ts`

- `runFetchJob(sys, platform)`:
  1. `fetchedAt = new Date()`;
  2. `newResources = (await provider.fetchLatestResources(sys))` → `toNewResource` →
     sorted by `createdAt` **ascending**;
  3. `upsert = await reservation.upsertResources(newResources, { indexSubject: true })`;
  4. `duplicated = await maintainDuplicatedResources(upsert.changed)`;
  5. If anything changed (`inserted|updated|deleted|duplicated.attached|detached`):
     `providers.updateRefreshTimestamp(platform, fetchedAt)`,
     `notifyRefreshedResources(notification)`, then **fire-and-forget**:
     `push.enqueueResourceMessages(inserted ids)` and `push.enqueueFailedResourceMessages()`;
     else only `enqueueFailedResourceMessages()`.
  6. On `NetworkError`: `providers.updateActiveStatus(platform, false)` (marks provider
     inactive in DB and memory). Other errors are logged, job returns `undefined`.
- `runSyncJob(sys, platform, {start, end})`:
  1. Same upsert path as fetch;
  2. `markDeletedResources(platform, newResources)` — DB rows of this provider with
     `isDeleted = false` and `createdAt` strictly inside `(minCreatedAt, maxCreatedAt)`
     of the fetched batch that are NOT in the fetched set get `isDeleted = true`;
  3. `maintainDuplicatedResources([...upsert.changed, ...deleted ids])`;
  4. notify only. **Sync never triggers Telegram push** (by design; see docs).
- **Coordinator** (`ResourcesJobCoordinator`): per-provider single-flight —
  `running: Map<provider, 'fetch'|'sync'>`. If a provider already has a job, an ack
  `{status:'OK', mode:'already_running', job, provider}` is returned immediately.
  Otherwise the job is spawned as a promise and `{status:'OK', mode:'queued', ...}` returned.
- RPC handlers registered via `sys.rpc.define`:
  - `resources.fetch` payload `{provider}` → ack;
  - `resources.sync` payload `{provider, start, end}` → ack.

### 2.4 Dedup / insertion

- In-memory per-run dedup by `providerId` (fetch pages, and again by `provider:providerId`
  in `upsertResources`).
- DB uniqueness: `unique_resources_provider_id` on `(provider, provider_id)`; inserts use
  `ON CONFLICT DO NOTHING`, updates compare field-by-field (see §4).
- New-resource detection for jobs = ids in the fetched batch not present in DB
  (`getResourcesByProviderId`).

---

## 3. Jobs (schedules)

`apps/server/src/server/index.ts` — `Executor` (started by `cli cron`) uses `croner`
with `{ timezone: 'Asia/Shanghai', protect: true }` (no overlapping runs):

| # | Cron | Count | Action |
|---|------|-------|--------|
| 1 | `*/5 * * * *` (every 5 min) | 1 per provider (4) | `rpc.invoke('resources.fetch', {provider})` → `runFetchJob` |
| 2 | `0 * * * *` (top of every hour) | 1 per provider (4) | `rpc.invoke('resources.sync', {provider, start:1, end:10})` → `runSyncJob` |
| 3 | `17 * * * *` (hourly at :17) | 1 | `sys.modules.subjects.updateCalendar()` (bgmx calendar sync) |

(Note: the code maps over all 4 `SupportProviders` for both fetch and sync; older docs
mention only dmhy/moe/ani — trust the code.)

### Startup behavior

- `cli cron` (Executor, `profile:'cron'`, `cron:true`):
  1. `makeSystem` — connect Postgres (cron profile: `statement_timeout 60s`,
     `lock_timeout 5s`, idle-in-tx 30s), Redis (loose), register modules;
  2. `sys.initialize()` — module `initialize()`:
     - Providers: load `providers` table into memory;
     - Users/Teams: load `users`/`teams` tables into `byName` + `byId` maps;
     - Resources: `QueryManager.initialize()` (skips prewarm when cron), `DetailsManager`;
     - Subjects: `fetchSubjects()` (loads `subjects` table);
     - Push: if `TELEGRAM_TOKEN` set, create grammy `Bot`; **only** in cron/cli profile
       with `TELEGRAM_CHAT_ID` also starts the long-polling message listener
       (`allowed_updates: ['message']`) that auto-unpins forwarded channel messages;
     - Redis: subscribe `invoke-rpc` channel (cron role); server role subscribes
       `notify-resources` + its private reply channel `reply-rpc:<uuid>`.
  3. `sys.import()` (if `--import`, default true) → `SubjectsModule.import()` →
     `updateCalendar()` — immediate bangumi calendar sync at boot;
  4. `executor.start()` registers the 9 cron jobs;
  5. HTTP server listens (default `0.0.0.0:3000`), serving the same Hono routes incl.
     `/admin/*`, with `registerResourcesJobRpc` wired so `resources.fetch/sync` RPC
     invocations are handled **in-process** and replies published over Redis.
- `cli start` (Server, `profile:'server'`, `cron:false`): no cron jobs; subscribes to
  `notify-resources` + RPC reply channel; serves the public API. `QueryManager.initialize`
  prewarms the `bangumi` + `['动画']` filter cache (100 rows) and starts maintenance timers.
- Maintenance timers inside `QueryManager.initialize` (all profiles):
  - hourly: `clearDeadTasks()` (discard task caches >2h stale, LRU-evict down to
    `MAX_RESOURCES_TASK_COUNT = 100`, reset downgrade set);
  - 24h: clear `TitlePool`/`MagnetPool`/`TrackerPool` string interns;
  - memo caches (`findFromRedis`, `findFromAccurateQuery`) run their own TTL GCs
    (5 min TTL).

### Redis channels / cross-process

- `NOTIFY_CHANNEL = 'notify-resources'` — cron publishes `Notification` after changes;
  servers refresh `QueryManager` caches (`onNotifications`).
- `RPC_INVOKE_CHANNEL = 'invoke-rpc'` — servers publish `{channel,type,gid,payload}`;
  the cron process runs the handler and publishes the reply to `channel`
  (`reply-rpc:<uuid>` of the requester).
- RPC timeout 30 s; reply payload `undefined` on failure/timeout (HTTP wrappers then
  return 503).
- Redis is optional/loose: `connectRedis` uses `enableOfflineQueue:false`,
  `maxRetriesPerRequest:1`, retry every 10 s; channel subscriptions auto-retry.

Notification shape:

```ts
interface Notification {
  resources: { inserted: NotifiedResource[]; updated: NotifiedResource[];
               deleted: number[] /* ids */ };
  duplicated: { attached: number[]; detached: number[] }; // ids
}
NotifiedResource = { id, provider, providerId, title }
```

---

## 4. Normalization (`resources/transform.ts`, `resources/index.ts`)

### 4.1 Field mapping (ScrapedResource → NewResource → DB row `resources`)

```ts
NewResource { provider, providerId, title, href, type, magnet, tracker, size: string|number,
              createdAt: Date, fetchedAt?, publisher?: {providerId?,name,avatar?}, fansub?, isDeleted? }
```

DB columns (`schema/resources.ts`): `id serial PK`, `provider` (enum dmhy|moe|mikan|ani),
`provider_id varchar(128)`, `title varchar(1024)`, `title_alt varchar(1024)`,
`title_search tsvector` (NOT NULL), `href text`, `type varchar(64)`,
`magnet varchar(256)`, `tracker text`, `size bigint`, `created_at timestamptz`,
`fetched_at timestamptz`, `indexed_at timestamptz`, `publisher_id int NOT NULL`,
`fansub_id int NULL`, `duplicated_id int NULL`, `subject_id int NULL`,
`metadata jsonb NULL` (type `{anipar?: ParseResult}` but **always written as `{}`**),
`is_deleted bool default false`.

`transformNewResources(sys, res, {indexSubject=true})`:

1. **Validation errors** (resource dropped with error recorded):
   - provider not in `SupportProviders`;
   - `publisherName = res.publisher?.name ?? 'anonymous'` has no row in `users` module;
   - if `fansubName` present, no row in `teams` module.
2. `titleAlt = normalizeTitle(res.title)` = `fullToHalf(tradToSimple(title))` (traditional→
   simplified, full-width→half-width incl. punctuation).
3. `size = typeof res.size === 'string' ? Math.floor(parseSize(res.size)) : res.size`,
   NaN → 0. `parseSize` regexes (decimal allowed, optional spaces, `i`-case variants):
   - `^(\d+(?:\.\d+)?)\s*[Kk]i?[Bb]$` → bytes = n;
   - `... [Mm]i?[Bb]` → n × 1024;
   - `... [Gg]i?[Bb]` → n × 1024²;
   - `... [Tt]i?[Bb]` → n × 1024³;
   - fallback `parseInt(size)`; non-numeric → 0.
   (So moe's `size` numeric string from JSON still works; moe values are typically already
   byte numbers.)
4. `titleSearch`: run `anipar.parse(titleAlt)`; if it returned a result, `search1` =
   jieba-cut tokens of `anipar.title` (weight **A**); `search3` = jieba-cut tokens of
   `titleAlt` (weight **D**); built as
   `setweight(to_tsvector('simple', search1),'A') || setweight(to_tsvector('simple', search3),'D')`
   (each missing part falls back to the other; `simple` text search config).
5. `subjectId = indexSubject ? matchActiveSubjects(sys, titleAlt) : null` — see §4.4.
6. Row written: `metadata: {}`, `isDeleted: res.isDeleted ?? false`, `indexedAt = fetchedAt`.

### 4.2 Upsert (`upsertResources`)

1. Dedupe by `${provider}:${providerId}` (first occurrence wins).
2. `ensureParties` — bulk upsert publishers → `users`, fansubs → `teams` (name-unique;
   `providers` json maps provider → `{providerId, avatar?}`; avatar backfilled only when user
   has none).
3. Transform each resource; failures → `errors` list.
4. Batch queries existing rows by `(provider, providerId)`.
5. Inserts: `INSERT ... ON CONFLICT DO NOTHING RETURNING id,provider,providerId,title`.
6. Updates (per row, only when a compared field differs):
   `isDeleted, href, magnet, tracker, subjectId, publisherId, fansubId, title, titleAlt,
   titleSearch`. On any change also set `fetchedAt`.
   - subject overwrite rule `shouldOverwriteSubjectId(old, next, resetSubjectId)`:
     overwrite iff `old !== next` AND (`next != null` OR (`next == null` AND `resetSubjectId`)).
     I.e. a re-fetch that fails subject matching keeps the existing subject id unless
     explicitly reset.
7. Returns `{inserted, updated, changed: insertedIds+updatedIds, errors}`.

DB retries: writes wrapped in `retryDatabaseFn(fn, {count:5})` — retries everything except
Postgres codes `55P03` (lock not available) / `57014` (query canceled) and messages
containing `statement timeout`, `lock timeout`, `idle-in-transaction timeout`.

### 4.3 Duplicate maintenance (`maintainDuplicatedResources`)

For each changed resource id: read `{id, provider, magnet, createdAt, duplicatedId, isDeleted}`;
skip when no magnet. Compute magnet variants:
`[normalizeBtihToHex(magnet), normalizeBtihToBase32(magnet)]` (btih v1 hex ↔ base32
rewrites of the `magnet:?xt=urn:btih:<h>` value; invalid magnets pass through unchanged in
both variants — such entries are skipped because the two variants are equal, `length===2` fails).
For every magnet-with-two-variants: select all non-deleted rows whose magnet ∈ variants;
sort candidates by:
`provider` (index in `SupportProviders`, ascending ⇒ dmhy first = highest priority),
then `createdAt` ascending, then `id` ascending; first = **winner**
(`duplicated_id = NULL`), all others point `duplicated_id = winner.id`.
Result `{attached: ids newly attached, detached: ids newly made root}`.

### 4.4 Subject matching (`matchActiveSubjects`)

- Uses `sys.modules.subjects.activeSubjects` (in-memory `subjects` rows where
  `is_archived = false`), iterating in module order; for each subject, for each keyword:
  `titleAlt.toLowerCase().includes(keyword.toLowerCase())` → first hit wins, return
  `subject.id`; else `null`.
- Subject keywords are built during import (§8): `normalizeTitle` of
  `[zh alias[0] ?? title, title, all alias values, search.include]`, deduped.
- Historical backfill: `SubjectsModule.indexSubject` sets `subject_id` for resources where
  `isDeleted = false`, `subject_id IS NULL` (unless overwrite), `createdAt >= activedAt −
  30×24h` (offset default 30 days), and `titleAlt ILIKE '%keyword%'` for ANY keyword
  (SQL `OR`).

### 4.5 Type determination

Per-provider (`§1`); dmhy uses the trad→simple category maps, moe uses tag-id map, mikan and
ani are hardcoded `'动画'`. Canonical set: `动画, 合集, 音乐, 日剧, RAW, 其他, 漫画, 游戏, 特摄`.

### 4.6 `createdAt` parsing summary

- dmhy: extract `span` text, `toShanghai(parse)` (Shanghai wall time → UTC ISO).
- moe: raw `publish_time` string passed through `new Date` conversion at job level.
- mikan: `parseMikanDate` (relative 今天/昨天, absolute, yearless, else `new Date`; all in
  Asia/Shanghai, output UTC ISO).
- ani: RSS `pubDate` (RFC 822). Detail page: unix-seconds `data-timestamp`.

### 4.7 Avatars / parties

- Stored persistently in `users`/`teams`; `providers` json keeps per-provider ids.
- Missing avatars are repaired lazily by the detail path (`fixResourceWithDetail`):
  if the resource's publisher/fansub has no avatar but the scraped detail has one, upsert
  the party with the avatar; then re-run `updateResource` (re-normalize, re-index subject,
  re-maintain duplicates) for the affected resource.
- `transformDatabaseUser` renders parties for API output as `{id, name, avatar?}`.

### 4.8 String pools (memory optimization, optional in Go)

`TitlePool`/`MagnetPool`/`TrackerPool` intern repeated strings; `query.transform` and
cache hydration use pooled values; pools cleared every 24 h.

---

## 5. Detail fetching (`resources/details.ts`, `providers/scraper/*`)

### 5.1 Path resolution (`Provider.getDetailURL`)

| Provider | Behavior |
|----------|----------|
| `dmhy` | id must match `/^(\d+)/`. If `id === providerId`: look up stored `href` by `(provider, providerId)` from DB (missing → 404-ish `undefined`); else return `{providerId, href: id}` (the extra URL tail is kept as href). |
| `mikan` | `{providerId: id, href: id}` — fetched as `{BASE}/Home/Episode/{id}`. |
| `moe` | `{providerId: id, href: id}` — POST `/api/torrent/fetch`. |
| `ani` | `{providerId: id, href: id}` — `https://nyaa.si/view/{id}`. |

### 5.2 Display hrefs (API output, `packages/client/src/href.ts`)

`transformResourceHref`: dmhy → `https://share.dmhy.org/topics/view/{href}`;
mikan → `https://mikanani.me/Home/Episode/{href}`; moe →
`https://bangumi.moe/torrent/{href}`; ani → `href` as-is.
Publisher/fansub hrefs: dmhy `.../topics/list/user_id|team_id/{id}`; mikan
`https://mikanani.me/Home/PublishGroup/{id}`; moe `https://bangumi.moe/tag/{id}`;
ani → `https://aniopen.an-i.workers.dev/`.

### 5.3 Cache flow (`DetailsManager.getByProviderId`)

1. **Redis**: key `details:{provider}:{providerId}` (only when `publisherRedis` exists),
   value JSON `{resource, detail, isDeleted, duplicatedId}` with TTL = `DETAIL_EXPIRE`.
2. **DB resource** by `(provider, providerId)`; none → `{resource:undefined, ...}`.
3. **DB detail** row by resource id. Stale if: resource `isDeleted`, no detail row, or
   `now − detail.fetchedAt > DETAIL_EXPIRE * 1000` where
   `DETAIL_EXPIRE = 7*24*60*60` seconds (7 days). Then fetch via scraper:
   - success → `insertDetail` (`INSERT ... ON CONFLICT DO NOTHING`, falling back to UPDATE
     when the insert returned nothing; fields `description, magnets, files, hasMoreFiles,
     fetchedAt=now`) + `updateRedisCache(..., DETAIL_EXPIRE)` + async
     `fixResourceWithDetail` (see §4.7);
   - failure → log, return `{resource, detail: undefined, ...}`.
4. Fresh detail → return from DB/Redis without upstream fetch.

### 5.4 Info-hash lookup (`getByInfoHash`)

Validate 40-hex or 32-base32 (uppercase). Query
`resources WHERE magnet ILIKE 'magnet:?xt=urn:btih:{UPPER}%' ORDER BY created_at DESC LIMIT 1`,
then delegate to `getByProviderId(provider, providerId, () => providerInstance.fetchResourceDetail(sys, found.href))`.

### 5.5 HTTP surface & memo

`GET /resource/{provider}/:id` and `GET /detail/{provider}/:id` → `findProviderDetail`
memoized in-memory per process with key `provider:path`, TTL 1 h,
`MAX_DETAIL_CACHE_COUNT = 10_000` (evict-by-expiration LRU GC). Response
`Cache-Control: public, max-age=86400`. `GET /detail/infohash/:hash` likewise
(24 h headers, `no-store` on invalid hash). Also `GET /feed.xml` and
`GET /collection/:hash/feed.xml` (RSS; `enclosure.url = magnet (+ tracker unless
`trakcer=no|off|false`)`, `length = size`).

---

## 6. Anipar parser (`packages/anipar`)

Entry: `parse(title: string, options: {fansub?: string}): ParseResult | undefined`
(returns `undefined` on empty input, empty tokens, or no title parsed).

### 6.1 ParseResult schema (types.ts)

```ts
interface ParseResult {
  title: string;                    // primary title (after all tag/episode stripping)
  titles?: string[];                // other translations/aliases (deduped, excluding title)
  fansub?: { name: string; alias?: string; collab?: string[]; tags?: string[] };
  season?: { number: number; title?: string };
  seasons?: { number: number }[];                     // e.g. S3+S4
  seasonsRange?: { from: number; to: number };        // e.g. S1-S2
  part?: { number: number };
  type?: string;                    // TV, OVA, OAD, SP, 剧场版, 总集篇, 修正合集, Movie, ...
  episode?: { number: number; numberSub?: number; type?: string; title?: string }; // 12.5 = number 12 + numberSub 5
  episodes?: EpisodeInfo[];         // e.g. [16,17] → "16&17"
  episodesRange?: { from: number; fromSub?: number; to: number; toSub?: number; type?: string }; // "01-12 修正合集"
  volume?: { number: number };
  volumes?: VolumeInfo;
  volumesRange?: { from: number; to: number; type?: string };
  version?: number;                 // v2, (v1), Movie v2, "01-12全集v2"
  subtitle?: {
    format?: string;                // 内嵌字幕/内封字幕/外挂字幕/内挂字幕/软字幕/ASS字幕/SRT字幕/字幕
    encoding?: string;              // GB | BIG5 (single)
    encodings?: string[];           // ["GB","BIG5"] (canonical order)
    languages?: string[];           // subset of [简,繁,粤,日,英,泰] + unknown, canonical order
  };
  source?: string;                  // BD/BDRIP/BLURAY/BDREMUX/WebRip/WEB-DL/WEBRIP/TVRIP/HDTV/...
  platform?: string;                // Baha/Bili/Bilibili/CR/ABEMA/AT-X/Netflix/NF/ViuTV/AMZN/ADN/Sentai/B-Global
  year?: number; month?: number;    // 2024年10月番 / ★10月新番 / 2024.12.15 / 2024SP
  file?: {
    extension?: string;             // MP4/MKV/... (from trailing ".ext")
    audio?: { codec?: string; channels?: string; language?: string; trackCount?: number };
    video?: { codec?: string; bitDepth?: string; resolution?: string; fps?: string;
              enhancement?: string; format?: string; quality?: string; frameRateMode?: string };
  };
  tmdbId?: string;                  // "tmdbid=1406607"
  tags?: string[];                  // RAW,DUB,Fin,END,无水印,特典,修订版,仅限港澳台,... + unknown wrapped tags + 物语-class titles
  variants?: string[];              // 日配版/中配版/English Dub/dual-audio-ish
  search?: string[];                // 检索: A/B → ["A","B"]
}
```

Normalization guarantees (context.ts):
- `subtitle.languages`: canonical order `[简,繁,粤,日,英,泰]` then unknown; `简`/`繁`/etc.
  detected from `CHS|ZH-HANS|簡|简体|简中`, `CHT|ZH-HANT|繁體|繁中|BIG5`, `YUE|粤|廣東話`,
  `JP|JPN|日|JAPANESE`, `EN|ENG|英|ENGLISH`, `TH|THA|泰|THAI`; `CN|CHINESE|中|国语中字`
  matches are dropped.
- `subtitle.format` normalization: `HARDSUB(S)`/内嵌→`内嵌字幕`, `SOFTSUB(S)`→`软字幕`,
  内封→`内封字幕`, 外挂→`外挂字幕`, 内挂→`内挂字幕`, `ASS(xN)`→`ASS字幕(×N)`, `SRT(xN)`→`SRT字幕(×N)`,
  `SUB|SUBBED|SUBTITLED|字幕`→`字幕`.
- `file.video.codec`: `H.264/X264/AVC`→`AVC`; `H.265/X265/HEVC2?/HVC1`→`HEVC`;
  `DIVX\d*`→`DivX`, `XVID`→`Xvid`, `HI10P`→`Hi10P`.
- `file.video.bitDepth`: `NNBIT(S)`→`NN-bit`.
- `file.video.fps`: `23.976FPS`→`23.976fps` (lowercase unit).
- `file.video.resolution`: `1080P`→`1080p`; `1920X1080/1920×1080`→`1920x1080`; `4K/2K` kept.
- `file.audio.codec`: `EAC3`→`E-AC-3`, `AC3`→`AC-3`, `OPUS`→`Opus`, `FLAC`,`AAC`,`DTS` kept,
  others lowercased.
- `file.audio.channels`: `2CH`→`2.0`, `5.1CH`→`5.1`.
- `file.audio.language`: `DUALAUDIO`/`DUAL AUDIO`→`dual audio`.

### 6.2 Parsing pipeline (parser.ts, tokenizer)

1. `parseFileExtension(title)` — trailing `.<ext>` where ext ∈
   `3GP|AVI|DIVX|FLV|M2TS|MKV|MOV|MP4|MPG|OGM|RM|RMVB|TS|WEBM|WMV` (case-insensitive); the
   extension is pre-seeded into `file.extension`.
2. `tokenize(rest)` — splits on bracket pairs `[] 【】 () （） {}` (nested wrappers allowed);
   each token records `text` plus optional `left/right` wrappers (`isWrapped`).
3. If `options.fansub` names a registered parser, run that exact pipeline; otherwise run the
   default pipeline: `parseFansub`, and if a fansub was discovered and HAS a registered
   parser, re-run `parse(title, {fansub})`.
4. Prefix tags: `parsePrefixWrappedTags` (each wrapped token matched against
   `matchSingleTag` / `matchEpiodes` / `matchMultipleTags`), `parsePrefixTextTags`
   (inline `★10月新番`, `★剧场版★`, `★老番★`).
5. Suffix tags: `parseSuffixWrappedTags` (drops unknown wrapped tags only at the very end
   or before a 检索/招募/ignore tail), `parseSuffixEpisodes` — wrapped tokens via
   `matchEpiodes`; inline text via `parseSuffixTextInlineEpisodes` (range `- 01-24 修正合集`,
   single ` - 01`, ` - 01.5`, `第01话`, `S01E01`, `- 特别篇`, ` - SP01`), plus
   `parseSuffixTextInlineSeason` (Parts N, 第N部分, S|Season N, Nth Season, 第N季, Chinese
   numerals, Vol.N, vN, ` 01` year-if-has-episode).
6. Titles: `splitMultipleTitles` on separators ` / ` / ` - ` (or without spaces for some
   fansubs, with the `Fate/` `命运/` and digit-boundary heuristics for `/`), strip inline
   suffix tags/seasons recursively; first part becomes `title`, remaining unique parts →
   `titles`.
7. `normalize()` — fansub tags merge, subtitle/file/variant normalization; returns
   `undefined` unless a non-empty `title` exists.

### 6.3 Fansub-specific pipelines (parser.ts)

`Kirara Fantasia` (suffix wrapped + suffix episodes + titles),
`ANi` (2-title swap: `title = titles[1]`, `titles = [titles[0]]`),
`LoliHouse` (`/`-split; postfix: trim trailing ` -`, strip `2`-suffix season duplicates,
`vN` tag → version),
`绿茶字幕组`, `桜都字幕组` (space:false, `/` separators),
`Prejudice-Studio` (space:false, `/`),
`喵萌奶茶屋` (prefix text tags first),
`雪飄工作室(FLsnow)` (prefix wrapped + suffix wrapped),
`三明治摆烂组` (trailing ` - ` hack, `06.5 总集篇(S00E01)` re-episode hack).

Registered Fansub enum: `Kirara_Fantasia='Kirara Fantasia'`, `ANi='ANi'`,
`LoliHouse='LoliHouse'`, `绿茶字幕组`, `桜都字幕组`, `Prejudice_Studio='Prejudice-Studio'`,
`沸班亚马制作组`, `喵萌奶茶屋`, `猎户发布组`, `爱恋字幕社`, `拨雪寻春`,
`雪飄工作室='雪飄工作室(FLsnow)'`, `幻樱字幕组`, `GMTeam`, `三明治摆烂组`, `星空字幕组`,
`北宇治字幕组`, `极影字幕社`, `MingYSub`, `黑白字幕组`, `S1百综字幕组`.

Fansub prefix tags: `個人製作合集`, `代发`, `羊圈个人译制` → `fansub.tags`;
collab separators `& ＆ · ，`; special `jibaketa`/`jibaketa合成&...`.

### 6.4 Keyword lists (keyword.ts, episodes.ts)

- **Audio codecs**: `DTS DTS-ES EAC3&AAC AAC QAAC AC3 EAC3 E-AC-3 FLAC FLAC/AC3 LOSSLESS
  MP3 WAV OGG VORBIS OPUS`.
- **Audio channels**: `2.0CH 2CH 5.1 5.1CH`; compound: `DTS5.1`, `TRUEHD5.1`,
  `AACX2|×2...×4`, `FLACX2..X4`.
- **Audio language**: `DUALAUDIO`, `DUAL AUDIO`.
- **Video codecs**: `HI10 HI10P HI444 HI444P HI444PP H264 H265 H.264 H.265 X264 X265 X.264
  AVC HEVC HEVC2 DIVX DIVX5 DIVX6 XVID`; compound: `AVC-8BIT HEVC-10BIT HEVC-10BIT-1440P
  HEVC-10BIT-2160P HEVC_10BIT HEVC-8BIT HEVC_8BIT HEVC_OPUS`.
- **Bit depths**: `8BIT 8-BIT 10BIT 10BITS 10-BIT 10-BITS`.
- **Video formats**: `AVI RMVB WMV WMV3 WMV9`.
- **Qualities**: `HDR HQ LQ`; resolution terms `HD SD`; resolutions `2K 4K`; regex
  `(AI)(\d{3,4}P)`, `(\d{3,5}(P|X\d{3,5}))@(\d+(\.\d+)?FPS)`, `(\d{3,4}P)(高帧率)`.
- **Frame rates**: `23.976FPS 24FPS 29.97FPS 30FPS 60FPS 120FPS`.
- **Sources (source)**: `BD BDRIP BLURAY BLU-RAY BDREMUX UHDBDRIP DVD DVD5 DVD9 DVD-R2J
  DVDRIP DVD-RIP R2DVD R2J R2JDVD R2JDVDRIP HDTV HDTVRIP TVRIP TV-RIP WEB WEBCAST WEBDL
  WEB-DL WEBRIP WEB-RIP WEB-MKV MASTERRIP DISC1..DISC9`.
- **Platforms**: `Baha Bili Bilibili BiliBili B-Global ABEMA CR AT-X AT-X版 ViuTV AMZN ADN
  Sentai Netflix NF`; platform+lang combos `ViuTV粵語`/`TVB粵語`.
- **Variants**: `日配版 中配版 日文配音 中文配音 Chinese Audio Japanese Audio JPN Audio
  Japanese Dub JP Dub English Audio English Dub`.
- **Subtitle format terms**: `ASS ASSX2..4 HARDSUB(S) SOFTSUB(S) SUB SUBBED SUBTITLED SRT
  SRTX2..4`; encodings `GB&BIG5 BIG5&GB 外挂GB/BIG5 GB/BIG5 GB BIG5`; language+format combos
  `代理商粵語 粵日雙語+內封繁體中文字幕 粵語+無對白字幕`.
- **Subtitle language terms**: `CN CHS CHT YUE JPN JP 简体 繁/體 简繁 國語中字 繁體 中日雙语
  简日双语 繁日双语 繁日雙語 HOY粵語 外挂CHS/CHT 外挂繁简日字幕 ...`; prefixes `简繁日双语,
  简繁日语, 简繁英日, 简日双语, 简/繁, 简繁英, 简繁泰, 中日英, 简日, 简繁, 簡繁, 简英, 繁體,
  繁体, 繁日, 繁英, 英文, 简体, 简, 繁, 英`; format suffixes `内嵌(字幕) 内封(字幕) 外挂(字幕)
  内挂 字幕`.
- **Types**: `GEKIJOUBAN MOVIE OAD OAV ONA OVA SPECIAL(S) TV 特别篇/特別篇/特別編/特别话/
  番外篇/剧场版/劇場版/总集篇/總集篇/广播剧/朗读剧/SP/ED/ENDING/NCED/NCOP/OP/OPENING/
  PREVIEW/PV/特别篇PV/合集/修正合集/开播纪念特别篇/开篇纪念特别篇`.
- **Other tags**: `RAW DUB DUBBED retake SNS 全歌曲特效 无水印 含副音轨 特典 LIVE纯享 无损重制
  广播剧_Dream☆Arch 国漫 Donghua 特別版 先行版(本) 正片先行版 正式版(本) 放送版 修订版/修訂版
  On-air version 年齡限制版 Ani-One 僅限港澳台(地區) 仅限港澳台(地区) 重播 End END FIN Fin
  TV + Movie Fin`; `xxx.ver`, `Bloomy_Cafe*`, `^(.*)(物语|物語)$`.
- **Search prefixes**: `检索:` `检索：` `檢索:` `檢索：` `检索用:` `检索用：` `檢索用:`
  `檢索用：` (value split on `/`). **Hiring**: `招募 急募 字幕社招人 字幕社招人`.
  **Ignores**: `务必查看bt站简介 请看bt站简介 添加日语 添加日語`.
- **Episode regexes (episodes.ts)** — single: `(TV|OVA|OAD|SP)?(\d+)(\.(\d))?(vV(\d+))?`
  (suffix `集/话/話` variants, `S(\d+)E(\d+)`); multi `\d+(\+\w+|,|&|、)\d+`; ranges
  `(type)?(\d+)(\.\d+)?[-~](\d+)(\.\d+)?(_?(.*))?`, `全NN集`; seasons `S(\d+) Fin|End`,
  `S1+S2`, `S1-S2`; volumes `Vol.?\s*(\d+)`, `Vol.N-N type`.
- `year/month` regexes: `^(\d{4})年(\d{1,2})月新?番$` (1949–2099), `^★?(\d{1,2})月新?番★?$`,
  `^(\d{4})\.(\d?\d)\.(\d?\d)$`, `^(\d{4})(SP)$`, `^[vV](\d+)$`.

### 6.5 Real examples (from test fixtures `packages/anipar/test/__assets__` + snapshots)

**E1 — ANi (asset `ani.csv` line 1):**
Input: `[ANi] AMAIM Warrior at the Borderline S2 -  境界戰機 第二部 - 18 [1080P][Baha][WEB-DL][AAC AVC][CHT][MP4]`
Output:
```json
{ "title": "境界戰機 第二部", "titles": ["AMAIM Warrior at the Borderline"],
  "fansub": {"name":"ANi"}, "season": {"number":2}, "episode": {"number":18},
  "platform": "Baha", "source": "WEB-DL",
  "subtitle": {"languages": ["繁"]},
  "file": {"extension":"MP4","video":{"codec":"AVC","resolution":"1080p"},"audio":{"codec":"AAC"}} }
```

**E2 — ANi (asset `ani.csv` line 2, no alias):**
Input: `[ANi] 愛在征服世界後[01][1080P][Baha][WEB-DL][AAC AVC][MP4]`
Output: `{ "title":"愛在征服世界後", "fansub":{"name":"ANi"}, "episode":{"number":1},
"platform":"Baha","source":"WEB-DL",
"file":{"extension":"MP4","video":{"codec":"AVC","resolution":"1080p"},"audio":{"codec":"AAC"}} }`

**E3 — ANi with dual subtitles (asset line 3):**
Input: `[ANi]  社畜想被幼女幽靈療癒。（僅限港澳台地區） - 04 [1080P][Bilibili][WEB-DL][AAC AVC][CHT CHS][MP4]`
Output: `{ "title":"社畜想被幼女幽靈療癒。（僅限港澳台地區）", "episode":{"number":4},
"platform":"Bilibili","source":"WEB-DL","subtitle":{"languages":["繁","简"]},
"file":{"extension":"MP4","video":{"codec":"AVC","resolution":"1080p"},"audio":{"codec":"AAC"}},
"tags":["僅限港澳台地區"] }` (shape: tags include 僅限港澳台地區 — see OtherTags/仅限港澳台
handling; `CHT CHS` are split by space separator `matchMultipleTags`.)

**E4 — LoliHouse collab + range (first lolihouse asset):**
Input: `[DHR动研字幕组&茉语星梦&LoliHouse] 在地下城寻求邂逅是否搞错了什么2 / DanMachi S2 [01-12合集][WebRip 1080p HEVC-10bit AAC][简繁内封字幕][Fin]`
Output:
```json
{ "title": "在地下城寻求邂逅是否搞错了什么", "titles": ["DanMachi"],
  "fansub": {"name":"DHR动研字幕组","collab":["茉语星梦","LoliHouse"]},
  "season": {"number":2}, "episodesRange": {"from":1,"to":12,"type":"合集"},
  "source": "WebRip", "subtitle": {"format":"内封字幕","languages":["简","繁"]},
  "tags": ["Fin"],
  "file": {"video":{"codec":"HEVC","bitDepth":"10-bit","resolution":"1080p"},"audio":{"codec":"AAC"}} }
```

**E5 — 桜都 (asset line 1):**
Input: `[桜都字幕组][戀愛中的小行星/戀愛小行星/Koisuru Asteroid][11][BIG5][1080P]`
Output: `{"title":"戀愛中的小行星","titles":["戀愛小行星","Koisuru Asteroid"],
"fansub":{"name":"桜都字幕组"},"episode":{"number":11},
"subtitle":{"encoding":"BIG5"},"file":{"video":{"resolution":"1080p"}}}`

**E6 — 雪飄工作室(FLsnow) multiple episodes (asset line 2):**
Input: `[FLsnow][Fate/Grand Order -绝对魔兽战线巴比伦尼亚- / Fate/Grand Order -絶対魔獣戦線バビロニア-][16&17][1080p][简繁外挂]`
Output: `{"title":"Fate/Grand Order -绝对魔兽战线巴比伦尼亚-",
"titles":["Fate/Grand Order -絶対魔獣戦線バビロニア-"],
"fansub":{"name":"雪飄工作室(FLsnow)","alias":"FLsnow"},
"episodes":[{"number":16},{"number":17}],
"subtitle":{"format":"外挂字幕","languages":["简","繁"]},
"file":{"video":{"resolution":"1080p"}}}`

**E7 — 三明治摆烂组 half episode + codec:**
Input: `[三明治摆烂组] 结缘甘神神社 / Amagami-san Chi no Enmusubi - 03 - [简体内封][H265 10bit 1080P]`
Output: `{"title":"结缘甘神神社","titles":["Amagami-san Chi no Enmusubi"],
"fansub":{"name":"三明治摆烂组"},"episode":{"number":3},
"subtitle":{"format":"内封字幕","languages":["简"]},
"file":{"video":{"codec":"HEVC","bitDepth":"10-bit","resolution":"1080p"}}}`

**E8 — 绿茶 subtitle combo:**
Input: `[绿茶字幕组] Silent Witch 沉默魔女的秘密 / Silent Witch - Chinmoku no Majo no Kakushigoto [09.5][WebRip][HEVC_AAC][1080p][简繁日内封]`
Output: `{"title":"Silent Witch 沉默魔女的秘密","titles":["Silent Witch - Chinmoku no Majo no Kakushigoto"],
"fansub":{"name":"绿茶字幕组"},"episode":{"number":9,"numberSub":5},
"source":"WebRip","subtitle":{"format":"内封字幕","languages":["简","繁","日"]},
"file":{"video":{"codec":"HEVC","resolution":"1080p"},"audio":{"codec":"AAC"}}}`

**E9 — Episode-key normalization used by push (from `push.ts`/tests):**
- `{episode:{number:12,numberSub:5}}` → `episode:12.5`
- `{episodesRange:{from:1,to:12}}` → `episodes_range:1-12`
- `{episodes:[{number:1},{number:2},{number:3}]}` → `episodes:1-2-3`

### 6.6 Server uses of anipar

1. `resources/transform.ts` — `parse(titleAlt)` only for jieba tokenization of
   `anipar.title` into `title_search` (weight A); the ParseResult itself is **not** stored
   (`metadata: {}` always at insert time).
2. `push/push.ts` — `parse(resource.title, {fansub: fansubName})` for episode extraction,
   priorities, and message rendering.
3. `push/message.ts` — renders parsed fields to the Telegram card.
4. `packages/scraper ani` — `parse(title, {fansub:'ANi'})` for `transformAlias`.
5. `TagsModule.importFromAnipar()` — empty stub (WIP).

---

## 7. Telegram push (`apps/server/src/push`)

### 7.1 Bot & channel configuration

- `SystemOptions.telegram = {token?, chatId?}` from `TELEGRAM_TOKEN` / `TELEGRAM_CHAT_ID`.
  Production channel: `-1003904271816` (fly.*.toml).
- `PushModule.initialize()` builds a grammy `Bot(token)` if token present.
- Message listener (long polling, `allowed_updates:['message']`) is started **only** when
  `chatId` is set AND profile is `cron` or `cli`. It handles auto-forwarded channel messages:
  if `message.is_automatic_forward` and `forward_origin.type === 'channel'` and the origin
  chat matches the configured channel (numeric id equal or `@username` match) →
  `unpinChatMessage(message.chat.id, message.message_id)` (unpins discussion group messages).
- Guards: `isConfigured()` = bot exists AND chatId set; all push entry points no-op otherwise.

### 7.2 Trigger conditions (what gets pushed)

1. `runFetchJob` — only the **inserted** ids of the upsert are enqueued (updates are NOT
   pushed); enqueue is fire-and-forget (`void`), plus `enqueueFailedResourceMessages()`.
2. `updateCalendar()` — ids of resources **newly bound** to subjects during calendar sync
   (`indexSubject` with `indexResources:true` because the subject is new or its
   keywords/activedAt changed) are enqueued.
3. Manual CLI: `telegram push --resource dmhy:123` / `--subject 12345` / `--force`
   (server-side admin has no push endpoint).
4. `runSyncJob` never pushes.

Per-resource preconditions in `PushContext.prepare()`:
- `resource.type === '动画'` (`shouldSendTypeResource`);
- fansub/name allowlist (`shouldSendFansubResource(fansubName)` where
  `fansubName = resource.fansub?.name ?? resource.publisher.name`):
  `ANi, LoliHouse, 绿茶字幕组, 桜都字幕组, Prejudice-Studio, 喵萌奶茶屋, 雪飄工作室(FLsnow),
  三明治摆烂组` (exact `Fansub` enum values);
- `subjectId` set AND resolvable via `getSubjectById` (bgmd static data);
- `anipar.parse(title, {fansub: fansubName})` succeeds;
- an episode key can be normalized (`episode`/`episodes`/`episodesRange`);
- publisher and fansub entities present.
Anything failing these is skipped without writing `telegram_messages`.

### 7.3 Dedup & state machine (`telegram_messages` table)

- Unique index: `(publisher_id, subject_id, episode)` (fansub_id not in the index).
- Status: `Pending 0`, `Sending 1`, `Sent 2`, `Failed -1`.
- Lookup: `WHERE (publisher_id = ? OR fansub_id = ?) AND subject_id = ? AND episode = ? LIMIT 1`
  (note OR on publisher/fansub).
- Episode key forms: `episode:{n(.sub)?}` | `episodes:{n}-{n}-...` |
  `episodes_range:{from(.fromSub)?}-{to(.toSub)?}` (decimal kept, e.g. `episode:12.5`).
- Transitions:
  - no record → INSERT Pending → (queue) → Sending → Sent (`telegram_chat_id`,
    `telegram_message_id`, `sent_at`) | Failed.
  - existing Pending/Sending → compare priority with existing owner resource; lower→skip;
    higher→optimistic-lock UPDATE (`WHERE id=? AND resource_id=<old>`) to Pending with new
    owner, then proceed.
  - existing Failed → compare; whichever resource has higher priority retries (UPDATE to
    Pending first).
  - existing Sent → lower→skip; higher→UPDATE Pending then **edit** original Telegram
    message (`editMessageCaption`) if `telegram_chat_id`+`telegram_message_id` exist
    (else degrade to new send); success → `edited_at`.
  - `--force` re-push: if previous resource id === current id and record is Sent → re-push.
- Optimistic locking: every status write filters on the expected `resource_id`; a lost lock
  throws `TelegramMessageLockLostError` (task aborts) or, after sending, the newly sent
  message is deleted (`deleteMessage`) since the record was taken over mid-flight.

### 7.4 Priority (`PushContext.compare`, higher = better)

1. `parsed.version` (default 0) — bigger wins;
2. subtitle languages: `简` present → 2, else `繁` → 1, else 0;
3. provider index in `SupportProviders` (dmhy highest) — `SupportProviders.length - index`;
4. `createdAt` — newer wins.

### 7.5 Concurrency model

- `pendingResourceIds` Set dedupes per resource id in-process (no cross-process lock).
- `newQueue(1)` serializes only `sendResourceMessage` / `editResourceMessage` (the actual
  Telegram API calls); all DB prework is concurrent.
- Failure compensation `enqueueFailedResourceMessages()` (called after every fetch job):
  - `SELECT DISTINCT resource_id FROM telegram_messages WHERE status = Failed AND
    updated_at >= now() - 7d`;
  - UPDATE `Pending/Sending` with `updated_at <= now() - 6h` → `Failed` (returning
    resource_ids);
  - merge + enqueue both id sets for retry.

### 7.6 Message format (buildResourceCardMessage)

`sendPhoto(chatId, photo = subject.poster, caption, {parse_mode:'HTML'})`.
Caption lines (HTML-escaped):

```
<b>{SubjectDisplayName}{ · 第 x 集}</b>
#{FansubHashtags} · #{yyyy年M月新番}
<b>字幕:</b> {简中|繁中|日语|英语|...} · {内嵌|内封|外挂}字幕
<b>格式:</b> {1080p} · {AVC|HEVC}-{10-bit} · {fps} · {mp4|mkv} · {AAC|...}
<b>大小:</b> {499.18 MB | x.x GB | 未知}
<b>发布:</b> {yyyy 年 M 月 d 日 HH:mm}          (Asia/Shanghai)
<b>追踪:</b> #{SubjectName} #{Fansub}_{SubjectName}
<a href="https://{site}/detail/{provider}/{providerId}">查看详情</a> · <a href="https://keepshare.org/{keepshareId}/{magnet}">在线播放</a>
```

Details:
- `SubjectDisplayName = subject.alias.zh?.[0] ?? subject.title`.
- Episode line: `第 {n} 集` / `第 {n.sub} 集` (e.g. `第 12.5 集`), `第 1,2 集`,
  `第 1-12 集` per parsed structure.
- Fansub line: deduped union of `parsed.fansub.name` + `parsed.fansub.collab`, each mapped
  through `normalizeFansubName` (simplify with `tradToSimple`, compact
  `[a-z0-9]` comparison; `flsnow|雪飘工作室flsnow → 雪飘工作室`,
  `nekomoekissaten|喵萌奶茶屋 → 喵萌奶茶屋`).
- Quarter hashtag: derive from `subject.onair_date || resource.createdAt`, aligned: if the
  date falls within 7 days **before** a quarter start (Jan 1 / Apr 1 / Jul 1 / Oct 1 / next
  Jan 1), label that quarter; else label the quarter containing the date → `{yyyy}年{M}月新番`.
- Hashtag mangling: keep letters/digits/underscore, spaces → `_`, collapse `_+`, trim `_`.
- Size: `<1024 → 'n KB'`, `<1MiB → 'n.nn MB'`, else `'n.nn GB'`; 0/negative → `未知`.
- Links: detail `https://{site}/detail/{provider}/{providerId}` (site default
  `animes.garden`); play `https://keepshare.org/{keepshareId||'gv78k1oi'}/{encodeURIComponent(magnet)}`
  where magnet = `resource.magnet + resource.tracker` split at first `&` (keepshare.ts).
- Error handling in send/edit/delete/unpin: GrammyError 429 → sleep(retry_after+1)s then one
  retry; 400 `message is not modified` / message-to-unpin-not-found / message-is-not-pinned →
  ignore; else rethrow → record Failed.

The exact card produced for a test fixture
(`kirara_fantasia` asset with `(Baha 1920x1080 AVC AAC MP4)`, size 511160 B, createdAt
`2026-05-07T05:00:00Z`, subject `從前從前有隻貓！世界喵童話` onair `2025-10-01`):

```
<b>從前從前有隻貓！世界喵童話 · 第 1 集</b>
#Kirara_Fantasia · #2025年10月新番
<b>字幕:</b> 繁中 · 内封字幕
<b>格式:</b> 1080p · AVC · mp4 · AAC
<b>大小:</b> 499.18 MB
<b>发布:</b> 2026 年 5 月 7 日 13:00
<b>追踪:</b> #從前從前有隻貓世界喵童話 #Kirara_Fantasia_從前從前有隻貓世界喵童話
<a href="https://animes.garden/detail/dmhy/12345">查看详情</a> · <a href="https://keepshare.org/gv78k1oi/magnet%3A%3Fxt%3Durn%3Abtih%3A0123456789012345678901234567890123456789">在线播放</a>
```

---

## 8. Bangumi subjects (`apps/server/src/subjects/bgmd.ts`)

- **Data source**: external `bgmx` package reading the AnimeGarden Bangumi mirror
  **`bgm.animes.garden`** (calendar + full subject data). The `bgmd` package is a static
  JSON snapshot (imported as JSON) used only for basic subject metadata
  (`getSubjectById`/`getSubjectByName` in `utils/bgmd.ts` and `server/utils/subjects.ts`),
  notably `poster` for Telegram photos.
- `fetchCalendar({timeout:30s, retry:1})` returns
  `{ calendar: CalendarSubject[][], web: CalendarSubject[] }` — `calendar` is a weekday-grid
  (array of arrays); `web` is extra non-scheduled items. `CalendarSubject` shape (from tests):
  `{ id, title, alias: {ja?:[], zh?:[], en?:[]}, poster, onair_date?: 'YYYY-MM-DD'|null,
  search: {include: string[]}, bangumi?: { date, platform, images:{large}, summary, meta_tags, tags } }`.
- `importFromBgmd` (`cli import subjects`): stream `fetchSubjects()` (DatabaseSubject list,
  same shape w/ `bangumi.date`), transform all, sort by `activedAt` desc (then id desc),
  `clearAllSubjectIds()` (null out every resource subject_id), then insert with
  `indexResources:true, offset:30, overwrite:false`.
- `updateCalendar()` (hourly cron + boot):
  1. fetch calendar; `onair = [...calendar.flat(), ...web]`;
  2. `insertMap` = fetched ids; every local active subject not in it → `archiveSubjects(id)`
     (sets `is_archived = true`);
  3. `transformSubjects`: for each bgm — `onairDate = onair_date || bangumi.date`;
     `activedAt = toShanghai(onairDate)` = `Date.UTC(y,m-1,d) − 8h` (i.e. midnight
     Asia/Shanghai as UTC instant); `title = alias.zh[0] ?? title`;
     `keywords = dedupe(normalizeTitle([title, bgm.title, ...alias values, ...search.include]))`;
     skip (record as error) when missing id or activedAt;
  4. per subject: `insertSubject(sub, {indexResources: shouldIndex, offset:30, overwrite:false})`
     — `shouldIndex` = subject absent locally OR (`keywords` or `activedAt` changed);
     INSERT ... ON CONFLICT (id) DO UPDATE name/activedAt/keywords/is_archived;
     `indexResources` only if `activedAt >= 2000-01-01`, offset 30 days; then
     `indexSubject` update (see §4.4);
  5. collect all newly matched resource ids → dedupe →
     `push.enqueueResourceMessages(resourceIds)` (subject-bound compensation push);
  6. reload `subjects` into memory.
- API: `GET /subjects` returns
  `{status:'OK', subjects: activeSubjects}` (rows: `{bangumi_id, name, keywords[],
  actived_at, is_archived}`), `Cache-Control: public, max-age=86400`.
- Local row → `BasicSubject` used by push comes from **bgmd JSON** (`poster`, `alias.zh`,
  `onair_date`, `title`), keyed by the same bangumi id.

---

## 9. Environment variables

Read in `apps/server/src/cli.ts` (`dotenv/config` + `process.env`) and Docker/fly config:

| Variable | Default | Purpose |
|----------|---------|---------|
| `SECRET` / `ADMIN_SECRET` | random 32-char (`[a-z0-9]`) if unset (logged at boot) | Bearer token for all `/admin/*` routes (`bearerAuth`); `ADMIN_SECRET` wins |
| `POSTGRES_URI` / `DATABASE_URI` | — (required; SystemError if missing) | Postgres connection string |
| `REDIS_URI` | — (optional, loose) | Redis for notify pub/sub, cross-process `resources.fetch/sync` RPC, detail + query caches |
| `APP_HOST` | — | Site hostname (`SystemOptions.site`), used for detail URLs and feed `link`; default fallback `animes.garden` |
| `KEEPSHARE_ID` | `gv78k1oi` (default in code; set in Dockerfile/fly) | KeepShare id for `在线播放` URLs |
| `TELEGRAM_TOKEN` | — | Telegram bot token (push + unpin listener) |
| `TELEGRAM_CHAT_ID` | — | Channel id/username, e.g. `-1003904271816` |
| `HOST` | `0.0.0.0` | HTTP listen host (`start`, `cron`) |
| `PORT` | `3000` | HTTP listen port |
| `TZ` | `Asia/Shanghai` (Dockerfile) | container timezone (parsing helpers also fix Shanghai explicitly) |
| `NODE_ENV`, `CI` | `production`, `true` (Dockerfile) | node runtime flags |
| `HTTP_PROXY` / `HTTPS_PROXY` | — | honored for outbound scrapes via `undici` `EnvHttpProxyAgent` in `cli.ts` |
| `S3_SECRET_KEY` | — | `upload.mjs` only: R2 heapdump upload on manager.sh failure |
| `APP_HOST`, `FEED_HOST`, `NODE_VERSION` | `animes.garden`, `api.animes.garden` | Docker build args (web + feed) |

`apps/server/.example.env`: `PORT=8080 SECRET=123456 POSTGRES_URI=postgres://root:example@0.0.0.0:5432/animegarden REDIS_URI=redis://0.0.0.0:6379`.

Worker (`apps/worker/src/types.ts`) declares `Env` with `DATABASE_HOST`,
`DATABASE_USERNAME`, `DATABASE_PASSWORD`, KV binding `animegarden`, D1 binding `database`
(declared but unused at runtime — the worker is a pure redirector).

---

## 10. Admin HTTP endpoints (`server/routes/admin.ts`)

Auth: `bearerAuth({token: sys.secret})` applied to everything under `/admin/`.

| Method | Path | Auth | Behavior |
|--------|------|------|----------|
| `POST` | `/admin/providers` | Bearer | Reload `providers` from DB; returns `{status:'OK', providers:{id: row}}` |
| `POST` | `/admin/resources/:provider` | Bearer | `rpc.invoke('resources.fetch', {provider})`; `202` with ack `{status:'OK', mode:'queued'\|'already_running', job:'fetch', provider}`; `503 {status:'ERROR', message:'Cron service unavailable.'}` if no cron process replies (or RPC times out after 30 s) |
| `POST` | `/admin/resources/:provider/sync?start=&end=` | Bearer | `rpc.invoke('resources.sync', {provider, start: +(query.start ?? '1'), end: +(query.end ?? '10')})`; `202` ack (`job:'sync'`) or `503` |

The cron `--listen` mode exposes the same routes, so `rpc.invoke` executes in-process.
Wire-compatible CLI: `animegarden-server admin fetch {provider} [--url]`,
`admin sync {provider} --start --end [--url]`, `telegram push --resource provider:id ...
[--force] | --subject <bgmId> ... [--force]`, `migrate`, `fetch {dmhy|moe|mikan} --start
--end --retry --out-dir`, `fetch ani --retry --out-dir`, `import tags`, `import subjects`,
`import resources <dir> --start --end --batch-size`, `detail {dmhy|moe|mikan|ani} <id|url>`.

Other public HTTP endpoints (context): `GET /` and `GET /health` (status + providers +
timestamp), `GET/POST /resources[...]` (+ `/resources/{provider}`, `/resources/{page}`),
`GET /resource/{provider}/:id`, `GET /detail/{provider}/:id`, `GET /detail/infohash/:hash`,
`GET /feed.xml`, `GET /collection/:hash/feed.xml`, `GET /subjects`, users/collections
routes, sitemaps, MCP (`/mcp`, `/.well-known/mcp/server-card.json` redirect). Request
timeout 60 s; `X-Request-Id`/`X-Response-Timestamp` headers; CORS open.

---

## 11. Cloudflare worker (`apps/worker/src/index.ts` + `legacy.ts`)

A legacy API-compatibility redirector (301) that maps old query shapes onto the current
deployment. No KV/D1 usage at runtime despite `Env` types.

Routing:

1. `pathname.startsWith('/api')` → rewrite host to `api.animes.garden`, keep path/query.
2. `pathname === '/feed.xml'` → host `api.animes.garden`, with **legacy filter
   translation**:
   - `filter` query param is a JSON string (default `{page:1, pageSize:1000}`); invalid
     JSON → `400` with `{status:400, detail:{url, filter, message}}`.
   - Validate with `FilterSchema`/`ManyFilterSchema` (zod, see `legacy.ts`):
     `provider` (single or array), `duplicate` bool, `page ≥1 default 1`,
     `pageSize 1..1000 default 100`, `fansubId[]`, `fansubName[]`, `publisherId[]`,
     `type`, `before`/`after` (numeric ts or date), `search[]`, `include[]`,
     `keywords[]`, `exclude[]` (each string-or-array, JSON-or-comma-tolerant).
   - Translation: `fansubName → fansub` (rename); `fansubId → fansub` (map provider id →
     name via `teams.json`); `publisherId → publisher` (via `users.json`); drop the old
     keys; serialize with `stringifyURLSearch` (page/pageSize/duplicate/after/before/
     provider/fansub/publisher/type/subject/search|include+keyword+exclude, sorted params).
   - Only the **first** filter of multi-filter arrays is applied.
3. Otherwise → host `animes.garden`, translating query params:
   `fansubName → fansub`, `fansubId → fansub` (name lookup from `teams.json`),
   `publisherId → publisher` (name lookup from `users.json`).

Data files (`apps/worker/src/teams.json`, `users.json`) are exported-as-JSON snapshots:
`{status:'OK', teams|users:[{id, name, avatar, providers:{<dmhy|moe|mikan>: {providerId}}}]}`
(also `users[].providers` may be `{}` for `anonymous`). Lookup only covers
`LEGACY_PROVIDER_IDS = ['dmhy','mikan','moe']` (**ani excluded** — ANi has no legacy
provider id mapping). The `id` in these files is the DB `users.id`/`teams.id`, NOT the
provider id; maps are built `provider → (providerId → team/user record)` and the team/user
`name` is emitted in the translated param.

---

## Key implementation notes for Go port

- Keep per-provider fetch functions byte-for-byte faithful: selectors, type maps, date
  hacks (`Asia/Shanghai` wall-time → UTC), magnet `splitOnce('&')` semantics (tracker keeps
  the leading `&`), title cleanup pipeline, and ANi btih-prefix provider id rule.
- `dmhy` must fetch `page` until a page contains zero DB-new resources; `ani` is a single
  RSS feed filtered against the DB; `mikan` additional fansub+publisher filter.
- Keep the per-provider single-flight coordinator, the 5-minute fetch / hourly sync / hourly
  calendar crons (Asia/Shanghai, no overlap), and the Redis RPC/notify channels unchanged —
  they are the cross-process contract.
- Size parsing: regex-based KB/MB/GB/TB with 1024 base, `Math.floor`, invalid → 0.
- Duplicates: magnet btih hex↔base32 variants, winner = provider order + createdAt + id.
- Subject matching: substring on lowercased normalized title vs active subject keyword list,
  first hit; subject id never cleared unless explicitly reset.
- Telegram: photo caption card (HTML), per-(publisher,subject,episode) dedup with
  optimistic resource_id locking, serialized API calls, 7-day/6-hour compensation windows,
  fansub allowlist, exact priority order (version → 简/繁 → provider → createdAt).