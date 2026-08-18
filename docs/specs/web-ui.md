# AnimeGarden Web Frontend — Complete Specification for Vue 3 + TypeScript Reimplementation

Source analyzed: `/tmp/animegarden-orig/apps/web` (React + TanStack Router + TanStack Start + TanStack Query + TailwindCSS/UnoCSS + shadcn/ui + cmdk + sonner).
Target: a pixel-faithful reimplementation in Vue 3 + TypeScript.

> Conventions used below:
> - `FEED_HOST` (API host) default `api.animes.garden`; `APP_HOST` (site host) default `animes.garden`; `WEB_SERVER_URL` default `https://api.animes.garden/`; `KEEPSHARE_ID` default `gv78k1oi`.
> - All dates are rendered in the `Asia/Shanghai` timezone (`formatChinaTime`), default format `yyyy-MM-dd HH:mm`, filter-cards use `yyyy 年 M 月 d 日 HH:mm`.
> - The UI language is Simplified Chinese. `<html lang="zh-CN">`.
> - Every nav/click/copy error event is tracked via Umami (see §16).
> - Toast options used everywhere: `{ dismissible: true, duration: 3000, closeButton: true, position: 'top-right' }` (exceptions noted).

---

## 0. Global architecture & routing core

### 0.1 Router construction (`router.tsx`)
- Router: TanStack Router with the file-generated `routeTree.gen.ts`; context carries `{ queryClient, stores }` where `stores = createAppStores()`.
- `scrollRestoration: true`, `defaultPreload: 'intent'` (hover/near-intent preload), `defaultPendingComponent: PendingPage`.
- Query params are plain strings/string arrays (NOT type-safe search objects): `parseRouterSearch` converts `?a=1&a=2&b=3` into `{ a: ['1','2'], b: '3' }`; `stringifyRouterSearch` inverse (null/undefined skipped, arrays appended).
- QueryClient defaults: `staleTime: 1000 * 60` (1 min), `gcTime: 1000 * 60 * 10` (10 min).

### 0.2 Route table (complete, from `routeTree.gen.ts`)

| URL pattern | Route id | Kind |
|---|---|---|
| `/` | `/` | Home page |
| `/anime` | `/anime` | Redirects to `/calendar/{active-season}` |
| `/calendar/$season` | `/calendar/$season` | Anime calendar page |
| `/resources` | `/resources` | Layout (Outlet) |
| `/resources/` | `/resources/` | Redirect to `/resources/1{searchStr}` |
| `/resources/$page` | `/resources/$page` | Resources list page |
| `/subject/$subject` | `/subject/$subject` | Subject page |
| `/subject/$subject/$page` | `/subject/$subject/$page` | Redirect to `/subject/{subject}{searchStr}` |
| `/detail/$provider/$providerId` | `/detail/$provider/$providerId` | Resource detail page |
| `/collection/$hash` | `/collection/$hash` | Collection page |
| `/docs/api` | `/docs/api` | Swagger UI API docs |
| `/iframe` | `/iframe` | Embeddable resources widget |
| `/about` | `/about` | About (currently placeholder) |
| `/$` | `/$` | Catch-all splat → redirect to `/` |
| `/robots.txt` | `/robots.txt` | server-handled text |
| `/sitemap-index.xml` | `/sitemap-index.xml` | server-handled XML |
| `/sitemap-{$sitemap}.xml` | `/sitemap-{$sitemap}.xml` | server-handled XML |
| `/openapi.json` | `/openapi.json` | server-handled JSON |
| `/llms.txt` | `/llms.txt` | server-handled text |
| `/.well-known/api-catalog` | `/.well-known/api-catalog` | server-handled JSON |
| `/.well-known/mcp/server-card.json` | `/.well-known/mcp/server-card.json` | server-handled JSON |

### 0.3 URL query parameter names (single source of truth: `@animegarden/client` `parseURLSearch`/`stringifyURLSearch`)

Repeated params are collected with `getAll`; singular params with `get`.

| URL param | Repeatable | Type | Filter field |
|---|---|---|---|
| `fansub` | yes | string | `fansubs: string[]` |
| `publisher` | yes | string | `publishers: string[]` |
| `type` | yes | string | `types: string[]` |
| `subject` | yes | number | `subjects: number[]` |
| `search` | yes | string | `search: string[]` (fuzzy / multi-word mode) |
| `include` | yes | string | `include: string[]` (title-match mode) |
| `keyword` | yes | string | `keywords: string[]` |
| `exclude` | yes | string | `exclude: string[]` |
| `after` | no | date (ISO string or epoch-ms number) | `after: Date` |
| `before` | no | date | `before: Date` |
| `preset` | no | `bangumi` | `preset` |
| `provider` | no | `dmhy \| mikan \| moe \| ani` | `provider` |
| `duplicate` | no | boolean | `duplicate` |
| `page` | no | number (≥1, rounded) | pagination |
| `pageSize` | no | number 1..80 (web forces 30 on list loads) | pagination |

Parsing semantics (critical):
- Page < 1 or NaN → 1; pageSize out of 1..80 → default.
- If `filter.search` is present, `include` entries are *deleted* from the resolved filter (search mode wins).
- `stringifyURLSearch` always calls `params.sort()` — query strings are alphabetically sorted.

### 0.4 App shell (`routes/__root.tsx`)
- HTML: `<html lang="zh-CN" suppressHydrationWarning>`, `<body class="font-sans relative">` with inline CSS custom properties: `--nav-height: 66px`, `--search-top: 128px`, `--hero-height: 300px` (constants `NavHeight=66`, `SearchTop=128`, `HeroHeight=300` from `layouts/Layout.tsx`).
- Root head: charset utf-8; viewport `width=device-width, initial-scale=1, maximum-scale=1.0, user-scalable=no`; `msapplication-TileColor #FFFFFF`; `theme-color #ffffff`; `yandex-verification ff51c9d16e597b3c`; links: `rel=sitemap` → `/sitemap-index.xml`; icons: `/favicon.ico` (64x64 image/x-icon), `/favicon.svg` (image/svg+xml, any size), `/pwa-64x64.png` (64x64 png), apple-touch-icon `/apple-touch-icon-180x180.png` (180x180), mask-icon `/favicon.svg` color `#FFFFFF`; preconnects to `https://fonts.bunny.net` and `https://api.fontshare.com`; web font stylesheets (see §13).
- Inline script `WebFontLoadScript`: when `link[data-web-font-stylesheet]` loads, sets `media='all'` (async font swap); `<noscript>` renders plain `<link rel="stylesheet">` tags.
- Umami analytics script tags are injected in the `<head>` in **production only** (`!import.meta.env.DEV`) from `~analytics/scripts` (unplugin-analytics), deferred, with any `dataset` props turned into `data-*` kebab-case attributes.
- Body script: inline `scrollHandler` (`layouts/global.ts?inline-ts`), then `<Scripts/>`, `<Toaster/>` (sonner), and in dev TanStack Router devtools bottom-left + React Query devtools bottom-right.
- Root loader: appends the RFC 8288 `Link` header: `</openapi.json>; rel="service-desc", </sitemap-index.xml>; rel="sitemap"; type="application/xml", </llms.txt>; rel="alternate"; type="text/plain"; title="llms.txt"`.
- Root component: on every location change (`useRouterState` select `location.href`) calls `requestAnimationFrame(() => window.__animegardenLayoutController?.update())`.
- Root `errorComponent`: logs `[Route Error]`, renders `<RootDocument><div/></RootDocument>` (blank page shell — no visible error UI at root level).
- Global CSS loaded in order: `virtual:uno.css`, `@onekuma/preset.css`, `styles/main.css`, `styles/sonner.css`, `styles/layout.css`, `styles/sidebar.css`, `layouts/Search/cmdk.css`.

### 0.5 Server start middlewares (`start.ts`)
- CSRF middleware applies only to server functions.
- ETag middleware: non-HTML GET responses wrapped with `safeEtagResponse`.
- Markdown content-negotiation middleware: if `Accept` header contains `text/markdown` (and its `q` is not `0`), GET/HEAD requests to `/`, `/anime`, `/calendar/\d{4}-\d{2}`, `/resources`, `/resources/-?\d+(\.\d*)?`, `/detail/{p}/{id}`, `/subject/\d+`, `/collection/{hash}` are answered with Markdown responses (see §14.6).

---

## 1. Pages & routes

### 1.1 `/` — Home (`routes/index.tsx` + `pages/_index/route.tsx`)

- **Title (head meta)**: `Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站`
- **Description**: `Anime Garden 動漫花園資源網镜像站, 动漫花园动画 BT 资源聚合站`
- **Canonical**: `https://{APP_HOST}/`
- **API calls (loader, parallel)**:
  - `resourcesQueryOptions({ page: 1, pageSize: 30, types: ['动画', '合集'], preset: 'bangumi' })` with `timeout: 5000` (short 5 s fallback timeout).
  - `calendarQueryOptions()` (latest calendar).
- **Loader logic**: if resources fail (`!ok` or `resources.length === 0`) BUT calendar is ok and has at least one non-empty weekday (`calendar.calendar.some(day => day.length > 0)`), **redirect** (`throw redirect`) to `/calendar/{calendar.season}` (307; console.warn `[Home] redirect to calendar fallback` with `{path, season, resourcesOk, resourcesCount}`). If resources fail → SSR `setErrorResponse(500)`; success → `Cache-Control: public, max-age=30, s-maxage=60` (`ResponseCacheControl.List`).
- **Page layout**: `<Layout feedURL={getFeedURL(location.searchStr)} timestamp>` (feedURL = `https://{FEED_HOST}/feed.xml` with empty search here). Content wrapper `<div class="w-full pt-13 pb-24">`:
  - **Season header** (only when `latestSeason` present, `mb-12 lt-sm:mb-6`, flexible row `flex min-h-10 items-end justify-between gap-4 pl-4 lt-md:pl-0`, stacks on small screens): `<h1 class="text-3xl lt-sm:text-2xl font-bold ...">` containing a Link to `/calendar/{latestSeason}` with:
    - season emoji span (`class="anime-season-emoji text-2xl font-quicksand font-bold"`, aria-hidden) — emoji from §7 season map (e.g. 🌸 ☀️ 🍁 ❄️).
    - text: `{seasonTitle}放送中...` e.g. `2026 · 夏季新番放送中...` (season title from `getCalendarSeason`).
    - secondary link `→ 前往周历` (`class="text-link text-base"`) to the calendar.
  - **Content**: on ok, `<Resources resources={page 1 data} page={1} timestamp complete={false} link={(page) => getResourcesRouteLink(page, 'type=动画&type=合集&preset=bangumi')} />` (i.e. every pagination link is `/resources/{page}?type=动画&type=合集&preset=bangumi`). **No FilterCard on home.** On failure: `<Error tracking={{ error: renderError }}>` where `renderError = getTrackingError(error, 'index-render-failed')`; also `trackFetchResourcesError({ path, error: getTrackingError(error, 'index-fetch-failed') })` in an effect.
- Imports `pages/anime/anime.css` for the season-emoji classes.

### 1.2 `/resources` and `/resources/:page` — Resources list

- `/resources` → layout route rendering `<Outlet/>` (no UI of its own).
- `/resources/` (trailing slash) → loader `redirect('/resources/1' + location.searchStr)` (preserves query). Head if ever rendered: title `最新资源 | Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站`, description `最新资源`, canonical `/resources/1`.
- `/resources/$page` (`routes/resources/$page.tsx` + `pages/resources.($page)/route.tsx`):
  - `page = Math.floor(+(params.page ?? '1'))`; if `page <= 0` → redirect to same URL with path segment replaced by `/1`.
  - **API calls (parallel)**: `resourcesQueryOptions({...filter, ...pagination, page, pageSize: 30})` (parsed from URL via `parseURLSearch(url.searchParams, { pageSize: 80 })` — note URL parse uses default 80 but the outgoing request forces `pageSize: 30`); plus `calendarQueryOptions()`; plus `subjectQueryOptions(id)` for every `filter.subjects`.
  - **Deep pagination**: if `!ok && error.message.includes('Resources pagination is too deep.')` → redirect to `/calendar/{calendar.season}` (or `/anime` if no season). Other errors → `setErrorResponse(500)`; success → Cache-Control List.
  - **Head**: `title = generateTitleFromFilter(filter, subjects) + ' | Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站'` where `generateTitleFromFilter`:
    1. `filter.subjects` with resolved names → `names.join(' ') + ' 最新动画资源'`
    2. `filter.search` → `search.join(' ') + ' 最新动画资源'`
    3. `filter.include` → `include[0] + ' 最新动画资源'`
    4. single `filter.fansubs` → `fansubs[0] + ' 最新动画资源'`
    5. single `filter.publishers` → `publishers[0] + ' 最新动画资源'`
    6. single `filter.types` → `最新{type}资源`
    7. fallback `所有资源`
    - `description = '最新资源 ' + stringifySearchText(stringifyURLSearch(filter), subjects)` (the human-readable search-input text, §4.6).
    - `canonical = https://{APP_HOST}/resources/{page}?{search}`.
  - **Page body** (see §5 for full detail): `<Layout feedURL={getFeedURL(location.searchStr)} timestamp>`; content `<div class="w-full pt-13 pb-24">` with `FilterCard` (if ok) + `Resources` table with `page`, `complete={pagination?.complete}`, `link={(page) => getResourcesRouteLink(page, location.searchStr)}`.
  - `usePreferFansub(filter?.fansubs)` records visited fansubs (see §15).
  - Error tracking: `trackFetchResourcesError({ path, error: getTrackingError(error, 'resources-fetch-failed') })`; render error fallback `'resources-render-failed'`.

### 1.3 `/subject/:subject` — Subject page (`routes/subject/$subject/route.tsx` + `pages/subject.$subject.($page)/route.tsx`, `subject.tsx`, `utils.ts`)

- `/subject/$subject/$page` (any trailing page) → **redirect** to `/subject/{subject}{searchStr}` (page param is ignored entirely).
- **API calls (loader, parallel)**: `subjectQueryOptions(subjectId)`; `resourcesQueryOptions({ subject: subjectId, subjects: undefined, page: 1, pageSize: 1000, types: ['动画', '合集'] })` (up to 1000 resources!); `calendarQueryOptions()`.
  - Subject not found → SSR 404, returns `{ ok: false, subjectId, subject: undefined, resources: [], ... }` but does NOT redirect.
  - Resources are grouped client-side by `groupResourcesByFansub` (see §6.3).
- **Head**: title `{displayName} 最新资源 | Anime Garden …` (fallback `generateTitleFromFilter(...)` + suffix, or `最新动画资源 | …`); description: `{name}: {subject.summary collapsed to single spaces, truncated 120}` or `{name} | Anime Garden …`; og tags: `og:title` = `{name} 最新资源`, `og:url` = `https://{APP_HOST}/subject/{id}`, `og:type` = `video.episode`, `og:logo` = `/favicon.svg`, `og:image` = poster (if any). Canonical `/subject/{subject}`.
- **Page body**: `<Layout feedURL={computed} timestamp heading={false}>`; `feedURL` computed = `getFeedURL('?' + sorted search params with subject=<param> added)` — i.e. current URL search with `subject` set to the raw `subjectParam` string, sorted, appended to `https://{FEED_HOST}/feed.xml`.
  - `SubjectCard` (§6.2) followed by grouped sections; if `resources.length === 0`: centered orange-ish row (`h-20 text-2xl text-orange-700/80`) with `i-carbon-search` icon, text `暂时未索引到相应资源`, and a `前往搜索` link (`text-link`) to `/resources/1?include={all subject names}` (URL-encoded; `fallbackSearch = stringifyURLSearch({ include: getAllSubjectNames(subject) })`), clicking tracks `subject.fallback-search` with `{ subject: 'subject:' + id }`.
  - If not ok or no subject → `Error` with message `未找到 <a href="https://bgm.tv/subject/{id}">番剧 {id}</a>` (only when subject missing), tracking error `subject-not-found` or `renderError`.
  - Error tracking: `subject-fetch-failed` for fetch errors, `'subject-render-failed'` for render.

### 1.4 `/anime` → `/calendar/:season` — Anime calendar (`routes/anime/route.tsx`, `routes/calendar/$season.tsx`, `pages/anime/route.tsx` + `anime.css`)

- `/anime` loader: fetch `calendarsQueryOptions()`; redirect to `/calendar/{season}` of the first `is_active` calendar; if `!ok || !season` → SSR 500 and render nothing.
- `/calendar/$season` loader (parallel): `timestampQueryOptions()`, `calendarsQueryOptions()`, and if the season exists in the calendars list `calendarQueryOptions(season)` (else SSR 404 with empty calendar and the full calendars list). Cache-Control Calendar on success.
- **Head**: `getCalendarSeasonHead(season)` → title `${titlePrefix}动画周历 | Anime Garden …` where prefix is `${calendarSeason.title}动画周历` (e.g. `2026 · 夏季新番动画周历`) or `动画周历` when no season; description `${titlePrefix}, 动画每周播出时间表, Anime Garden …`. Canonical `/calendar/{season}`.
- **Page body** — §7 full detail: season header with two dropdowns (year & season), sticky weekday TOC, weekday sections of anime cards. On season change: `navigate({ to: '/calendar/$season', params: { season: nextSeason } })`.

### 1.5 `/collection/:hash` — Collection page (`routes/collection/$hash/route.tsx` + `pages/collection.$hash/route.tsx`)

- Loader: no hash → redirect `/`; `collectionQueryOptions(hash)` + `calendarQueryOptions()` parallel; if `!resp?.ok` → redirect `/` (a failed/unknown hash bounces home, no error page). Success → Cache-Control List.
- **Head**: title `{name} | Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站` (name prefix omitted when missing); description `Anime Garden 资源收藏夹, 動漫花園資源網镜像站, 动漫花园动画 BT 资源聚合站`; canonical `/collection/{hash}`.
- **Page body**: `<Layout timestamp={data.timestamp} feedURL={getCollectionFeedURL(hash)}>` where `getCollectionFeedURL(hash) = https://{FEED_HOST}/collection/{hash}/feed.xml`. If `!data` → `Error` tracking `collection-render-failed`. Otherwise renders one bordered rounded box (`py-4 rounded-md border drop-md`) **per filter entry**:
  - header (`mb-4 px-4 pb-4 border-b`) with `<h2 class="text-xl font-bold">` containing a Link to `/resources/1{searchParams}` showing the inferable item title (custom `name` else inferred title + ` 字幕组:{fansubs joined ' '}` else search-text, §4.10).
  - body `px-4`: `<ResourcesTable resources={item.resources} page={1} complete={item.complete} link={(page) => getResourcesRouteLink(page, filters[idx].searchParams)} />`.

### 1.6 `/detail/:provider/:providerId` — Resource detail (`routes/detail/$provider/$providerId/route.tsx` + `pages/detail.$provider.$providerId/route.tsx` + `FileTree.tsx` + `detail.css`)

- Loader: provider must be in `SupportProviders` (`['dmhy', 'mikan', 'moe', 'ani']`) and both params present, else redirect `/`. `resourceDetailQueryOptions(provider, providerId)` + `calendarQueryOptions()`; if `data?.ok && data?.resource` also `subjectQueryOptions(resource.subjectId)` when present. Error → redirect `/` (no detail error page). Success → Cache-Control Detail (`public, max-age=3600, s-maxage=86400`).
- **Head**:
  - title: `truncate(resource.title, 70)` or (no resource) `资源详情 | Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站`.
  - description: from normalized description — `description.summary` if it starts with the parsed anime title, else `{title}: {summary}`, else `{title}: {raw description}`.
  - `script:ld+json` (when `anipar.parse(title)` succeeds): `{ "@context": "http://schema.org", "@type": "TVEpisode", partOfTVSeries: { "@type": "TVSeries", name: info.title }, partOfSeason: { "@type": "TVSeason", seasonNumber: String(info.season?.number ?? 1) }, episodeNumber: info.episode?.number !== undefined ? String(info.episode.number) : undefined, datePublished: formatChinaTime(createdAt, 'yyyy-MM-dd'), url: 'https://{APP_HOST}/detail/{provider}/{providerId}' }`.
  - og: `og:title` = parsed title (`info?.title ?? resourceTitle`); `og:url`; `og:type` = `video.episode` for types 动画/合集/日剧/特摄, `music.song` for 音乐, else `website`; `og:logo` = `/favicon.svg`; `og:image` = first description image or subject poster.
- **Page body** — full detail in §8; rendered with `<Layout timestamp heading={false}>`.

### 1.7 `/docs/api` — API docs (`routes/docs/api/route.tsx` + `pages/docs.api/route.tsx` + `spec.ts` + `spec.json`)

- Loader: `timestampQueryOptions()` + `calendarQueryOptions()`; Cache-Control Docs on success, 500 otherwise.
- **Head**: title `API 文档 | Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站`; description `Anime Garden 动画 BT 资源开放接口文档`; canonical `/docs/api`.
- **Body**: `<Layout timestamp>` with `<div class="w-full pt-13 pb-24"><SwaggerUI spec={getPublicOpenApiSpec(version, license)} /></div>` — full-page **swagger-ui-react** rendered from the bundled `spec.json`, post-processed by `getPublicOpenApiSpec` (§10).

### 1.8 `/iframe` — Embeddable resources widget (`routes/iframe.tsx` + `pages/iframe/route.tsx`)

- Loader: parse the full URL search via `parseURLSearch`; query input = `{...filter, ...pagination, pageSize: 30}`; fetch `resourcesQueryOptions(queryInput)` + `calendarQueryOptions()` + subject queries for `filter.subjects`. Cache-Control List on ok, 500 otherwise.
- **Head**: title `{generateTitleFromFilter(...)} | Anime Garden 動漫花園資源網第三方镜像站` (note: 第三方镜像站 suffix differs from other pages); description `最新资源 {stringifySearchText(...)}`; canonical `https://{APP_HOST}/iframe?{search}`.
- **Body**: **no Layout, no header/footer/hero** — the root div is `<div class="w-full" :class="classNameParams" :style="inlineStyle">` where:
  - `className` query params (repeatable) are joined as classes (`searchParams.getAll('className')`).
  - `style` query params (repeatable) are joined with `;` and parsed with `style-to-object` into an inline style object.
  - Inside: ok → `<Resources resources page={pagination?.page ?? 1} complete timestamp link>` where link navigates back to `/iframe` with the same search params + `page` set (via `toRouterSearch`). Error → `Error` component (tracking `iframe-fetch-failed` / `'iframe-render-failed'`).

### 1.9 `/about` — About (`routes/about.tsx` + `pages/about/route.tsx`)

- Loader: `timestampQueryOptions()` + `calendarQueryOptions()`; Cache-Control List on ok.
- **Head**: title `关于 | Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站`; description `Anime Garden 動漫花園資源網镜像站, 动漫花园动画 BT 资源聚合站`; canonical `/about`.
- **Body**: `<Layout timestamp>` with `<div class="w-full pt-13 pb-24"><div class="h-[1000px]"></div></div>` — **the page is currently an empty 1000px-tall placeholder box** (no textual content). Reimplement as-is unless product decides otherwise.

### 1.10 Error / 404 behavior
- Unknown paths (`/$` splat) → **redirect to `/`** (no 404 page).
- `resources.($page)/Error.tsx` (shared by home/resources/subject/collection/iframe): renders `<div class="h-20 text-2xl text-red-700/80 flex items-center justify-center"><div><span class="mr2 i-carbon-error"/><span>发生错误</span>{message ? ': '+message : ''}</div>{children}</div>`; tracks `error.render` once per `{path, error}` key (dedup via ref).
- Root error component: logs and renders `<RootDocument><div/></RootDocument>`.
- Server-error statuses: pages call `setErrorResponse(500)` (default) or explicit 404; `Cache-Control: no-store` on error responses.

---

## 2. Header (fixed top nav)

`layouts/Header.tsx` — `memo` component. Markup:

```html
<header class="fixed z-13 pt-[1px] flex justify-center items-center w-full h-$nav-height pointer-events-none text-base-500">
  <nav class="main flex gap-1 [&>div]:(leading-$nav-height)">
    <div class="box-content w-[32px] pl3 lt-sm:pl1 text-2xl text-center font-quicksand font-bold pointer-events-auto">
      <Link to="/" @click="trackNavClick('home', { item: 'home' })">🌸</Link>
    </div>
    <AnimeDropdown/>
    <FansubsDropdown/>
    <TypesDropdown/>
    <div class="flex-auto pointer-events-none"></div>
    <div class="lt-md:hidden pointer-events-auto">   <!-- RSS button, hidden below md -->
      <a :href="feedURL" target="_blank" ...>
        <span><span class="i-mdi-rss text-sm mr1 relative top-[2px]"/><span>RSS</span></span>
      </a>
    </div>
    <div class="hidden lt-md:block"></div>
  </nav>
</header>
```

Behavior details:
- The whole `<header>` is `pointer-events-none`; each child is `pointer-events-auto`.
- **Logo**: 🌸 emoji, `font-quicksand font-bold`, click → `/` + `trackNavClick('home', {item:'home'})`.
- **RSS button** (only when `feedURL` prop provided; hidden `< md`): text `RSS` (with `i-mdi-rss` icon), color `text-[#ee802f]`; on hover the span gets `text-[#ff7800]!` + `border-b-2 border-b-[#ff7800]!`; opens `feedURL` in new tab; click tracks `feed.open` with `{href}` (`getOpenFeedTrackEvent`).
- Header children carry `data-nav-collision-target` attributes (`anime`, `fansubs`, `types`) used by the scroll-collision logic (§16.7).

### 2.1 AnimeDropdown (动画)
- Trigger: Link to `/calendar/{latest-season}` (or `<a href="/anime">` fallback), text `动画`; click → `trackNavClick('anime')`.
- Hover opens dropdown (CSS-only `[&:hover>.c-dropdown]:block`); trigger hover bg `bg-zinc-100! dark:bg-zinc-800!`.
- Dropdown menu (`DropdownMenu`): `mt-[-10px] w-[80px] max-h-[600px] lt-sm:max-h-[360px] rounded-md shadow-box divide-y bg-light-100 dark:bg-dark-100 leading-normal`.
  - First item: Link `周历` to the calendar route.
  - Then one `DropdownSubMenuItem` per weekday **ordered by `index`** (`周一`…`周日`, label text `周{text}`). Each row (`trigger` div `px2 py1 cursor-pointer`, last row `rounded-b-md`) opens a **submenu** (flyout, `DropdownSubMenu`) containing a `min-h-[100px] max-h-[min(500px,calc(100vh-120px-var(--offset)))] lt-sm:max-h-[360px] rounded-md shadow-box bg-light-100 dark:bg-dark-100 divide-y overflow-y-auto overscroll-none` list with inline style `--offset: {index*33}px`; submenu positioned `top-0 left-full` with `pt-[1px] pl-[6px] pb-[2px] pr-[2px]`).
  - Each bangumi row: Link to `/subject/{id}`, `block w-[360px] max-w-[calc(100vw-144px)] px2 py1 hover:bg-basis-100 dark:hover:bg-basis-800 whitespace-nowrap overflow-hidden text-ellipsis`, text `getSubjectDisplayName(bgm)`; click tracks `trackNavClick('anime', { item: name, group: '周{n}' })`.

### 2.2 FansubsDropdown (字幕组)
- Local state `fansubs` starts as the full static `fansubs` list (§13.8) then is **re-ordered** by the `preferFansubs` preference (localStorage `animegarden:fansubs`): preferred fansubs first (in preference order), then the rest in static order, no duplicates.
- Trigger: Link to `getResourcesRouteLink(1, { fansub: fansubs[0] })`, text `字幕组`; click → `trackNavClick('fansub')`.
- Dropdown: `mt-[-10px] w-[160px] max-h-[494px] overflow-y-auto rounded-md shadow-box divide-y bg-light-100 dark:bg-dark-100 leading-normal`. One row per fansub: Link `/resources/1?fansub={name}` (via `getResourcesRouteLink(1, { fansub })`), `block px2 py1 hover:bg-basis-100 dark:hover:bg-basis-800`, text = fansub name; click tracks `trackNavClick('fansub', { item: fansub })`.

### 2.3 TypesDropdown (资源)
- Trigger: Link `/resources/1` (no filters), text `资源`; click → `trackNavClick('type')`.
- Dropdown: `mt-[-10px] w-max overflow-y-auto rounded-md shadow-box divide-y ...`. One row per type from `types = ['动画','合集','音乐','日剧','RAW','漫画','游戏','特摄','其他']`:
  - Link: `动画` → `getResourcesRouteLink(1, { type: '动画', preset: 'bangumi' })`; every other type → `getResourcesRouteLink(1, { type })`.
  - Row: `flex items-center gap-2 px2 py1 hover:bg-basis-100 dark:hover:bg-basis-800` plus `DisplayTypeColor[type]` class; icon `DisplayTypeIcon[type]` (§13.6); label text.
  - Click → `trackNavClick('type', { item: type })`.

All three dropdown triggers get hover bg `bg-zinc-100! dark:bg-zinc-800!` via the Dropdown wrapper className.

**Header CSS collision (global.ts)**: at viewport width < 1440 px, when the fixed header rows overlap the "collision sources" (hero search box `[data-header-collision-source]` elements, which include the search wrapper and the hero title link), nav dropdown targets (`data-nav-collision-target` = anime → fansubs → types order) progressively get `display:none` injected by a dynamically generated `<style id="nav-collision-style">`, with `body.nav-collision-from-{id}` classes selecting which later targets to hide (see §16.7).

---

## 3. Sidebar (收藏夹 / collections)

`layouts/Sidebar/Sidebar.tsx`, `Collection.tsx`, styles in `styles/sidebar.css`. Rendered inside `Layout`'s `<div class="w-full flex main-with-sidebar|main-without-sidebar">` as the first child. Wrapped in `<ClientOnly>` (never SSR-rendered).

### 3.1 States
- **Closed** (default, `isOpenSidebarStore = false`, not persisted): renders `SidebarTrigger`:
  ```html
  <div class="sidebar-trigger font-quicksand font-500" @click="open + track('collection.open-sidebar')">
    <span class="i-carbon:bookmark text-sm relative top-[2px] mr1"/><span class="text-sm">收藏夹</span>
  </div>
  ```
  Style: `w-max pl2 pr3 py1 text-base-700 border-1 border-l-0 rounded-r-xl bg-layer-on z-5 drop-md hover:bg-layer-subtle relative top-[8px] select-none cursor-pointer` (a small pill peeking from the left edge; `.sidebar-root` has `width: 0` so the trigger overflows the page edge). When `.fix-hero` → `fixed top: calc(var(--nav-height) + 8px)`.
- **Open**: `SidebarContent` inside `.sidebar-wrapper`:
  - `w-[200px] lt-lg:w-[300px] bg-layer-on z-5 absolute border-r-1 flex-shrink-0 flex-grow-0`, `height: calc(100vh - var(--sidebar-pt))`, `left: 0`, `space-y-2`, padding top `mt-[8px]`. `--sidebar-pt` = `var(--hero-height)` normally; while scrolled inside the hero (`scrollY < HeroHeight - NavHeight`) it is set inline to `HeroHeight - scrollY` px by global.ts; when `.fix-hero` → `height: calc(100vh - var(--nav-height)); top: var(--nav-height); position: fixed` and `--sidebar-pt = var(--nav-height)!important`.
  - Header row: bookmark icon + bold `收藏夹` (left), spacer, collapse button (right): `h-[26px] w-auto rounded-md px-1 flex items-center cursor-pointer hover:bg-layer-muted` containing `i-fluent:panel-right-expand-16-regular` icon; click → `isOpenSidebarStore.setState(false)`.

### 3.2 QuickLinks (when a collection exists)
Rows (`ml1 mr2 px1 py2 cursor-pointer select-none text-sm text-base-700 flex items-center hover:bg-layer-subtle-overlay rounded-md`):
1. `动画周历` — Link to `/calendar/{latestSeason}` (fallback `<a href="/anime">`), icon `i-carbon-calendar`, active class `bg-layer-muted` when `match === 'anime'`, `resetScroll={false}`.
2. `所有资源` — Link `getResourcesRouteLink(1)` (→ `/resources/1`), icon `i-carbon-list`, active when `match === 'resources'`.
3. `高级搜索帮助` — external `<a href="https://docs.animes.garden/animegarden/search" target="_blank">`, icon `i-carbon-help` (never active).

`match` = `getActivePageTab(location, collection)`: `/resources/{page}` → the collection item whose `searchParams` exactly equals `location.searchStr` (returns that item's `searchParams`); else `'resources'`; `/anime` or `/calendar/*` → `'anime'`; `/` → `'index'`.

### 3.3 Collection block
Row: collection **name** — if `collection.hash` set, Link to `/collection/{hash}` (`block text-xs text-base-500 text-link-active`, name in `select-none` span); else `<a href="/collection/" class="...">` with `@click.prevent="createCollection()"`. Then a flex spacer containing a 1px `bg-zinc-200` divider, then **share button** (`h-[26px] w-auto rounded-md px-1 ... hover:bg-layer-muted`, icon `i-carbon-share`):
- `onClickShare`: if no hash, first `createCollection()` (generates). Then copies `{origin}/collection/{hash}` to clipboard → toast.success `复制 {collection.name} 分享链接成功` (or toast.warning `复制 {collection.name} 分享链接失败`); tracks `collection.share {hash}`.
- `createCollection`: if `!collection.authorization` assign `crypto.randomUUID()` and persist (invalidation `updateCollection`); POST the collection via `generateCollectionMutationOptions()` (mutation key `['api','collection']`, serverFn POST → `rawGenerateCollection`); on success store returned `hash` via `updateCollection(stores, collection, { hash })`.

Filter list (`collection.filters.length > 0`): `.collection-container py-[1px] pr-[1px] space-y-2 overflow-y-auto` (max-height formula in css: `calc(100vh - var(--sidebar-pt) - 34px - 36px - 36px - 36px - 32px - 26px - 24px)`; `mr-[8px] overscroll-none`). Each item = `CollectionItemContent` (below) with `active={match === item.searchParams}`.

Empty state (`filters.length === 0`): Link `getResourcesRouteLink(1, 'search=败犬女主太多了&type=动画')` with class `h-[80px] px2 flex items-center justify-center text-base-700 text-link-active` containing `<span class="text-sm">收藏一个搜索条件吧</span><span class="i-carbon:arrow-up-right"/>`.

Bottom divider: `mt2 px2` with a 1px `bg-zinc-200` line.

### 3.4 CollectionItemContent (one saved filter row)
- Link: `getResourcesRouteLink(1, item.searchParams)` → `/resources/1{searchParams}`; classes `collection-item hover:bg-layer-subtle-overlay rounded-md text-base-800 text-xs` + `bg-layer-muted` when active. While renaming, clicks are swallowed (`preventDefault/stopPropagation`).
- **Title** (span, `collection-item-title`): `item.name` if set; else inferred title (+ ` 字幕组:{fansubs joined ' '}` suffix when fansubs exist); else the text search representation (`stringifySearchText(new URLSearchParams(item.searchParams), subjectMap)` with names resolved from subject queries via `useInferCollectionItemName`; subject display-name for a single `subjects` entry, else `search.join(' ')`, else `include.join(' ')`).
- **Inline rename**: double-… no — “重命名” menu item toggles `contentEditable='plaintext-only'` on the title span (focus + select all the content; Enter commits; blur commits unless blur occurred <200ms after focus in which case refocuses). While editable the row gets `outline outline-1 outline-zinc-300 bg-transparent!`, the hover “…” button hides. Commit → `updateCollectionItem(stores, collection, { ...item, name: newTitle })` (name empty string is allowed → falls back to inferred title).
- **Hover “…” menu**: absolutely positioned `collection-item-op hidden absolute h-full top-0 right-[4px] py-[1px] justify-center items-center` (shown on row hover or when its DropdownMenu `[data-state=open]`), inner `w-[16px] ... hover:bg-layer-mask rounded-md` with `i-ant-design:more-outlined`; clicking prevents link navigation. Radix DropdownMenu (`modal=false`, `sideOffset=14`):
  1. `在新页面中打开` (icon `i-ant-design:link-outlined`) → `window.open('/resources/1' + item.searchParams)`; tracks `collection.open-resources { search }`.
  2. `复制 RSS 订阅链接` (icon `i-carbon-rss`) → copies `getFeedURL(item.searchParams)` → toast.success `复制 RSS 订阅成功` / toast.error `复制 RSS 订阅失败`; tracks `copy.feed` + `trackCopyFeed()`.
  3. separator.
  4. `重命名` (icon `i-ant-design:edit-outlined`) → startRename.
  5. separator.
  6. `删除` (icon `i-carbon-trash-can`, row hover `text-red-500! bg-red-100!`) → `deleteCollectionItem(stores, collection, item)`.
- **Tooltip** (Radix Tooltip, `side="right" sideOffset={20} align="start" alignOffset={-10}`, delayDuration 300, skipDelayDuration 100; suppressed while menu open or renaming): `CollectionItemFilter` — a small key/value list (labels in bold, `mr2 select-none`):
  - `条件别名` (custom name), `动画` (subject names resolved by id), `类型` (colored), `标题搜索` (search words, `|` separated), `标题匹配` (include, `|` separated), `包含关键词` (keywords, `&` separated), `排除关键词` (exclude), `字幕组` (fansub links to `/resources/1?fansub={name}`, `text-link mr2`), `搜索开始于` / `搜索结束于` (formatChinaTime `yyyy 年 M 月 d 日 HH:mm`).

### 3.5 Store mutations (stores/collection.ts) — see §9.4.

---

## 4. Search dialog (CMD-K style, `layouts/Search/Search.tsx` + `utils.ts` + `cmdk.css`)

### 4.1 Where it lives & how it opens
- Rendered inside the hero, `#hero-search` → `<search id="hero-search" class="w-full h-$nav-height z-12 flex items-center justify-center pointer-events-none">` → `<div data-header-collision-source class="vercel relative h-[44.4px] xl:w-[640px] lg:w-[600px] md:w-[500px] lt-md:w-[calc(100vw-116px)] pointer-events-auto">` → `<Search/>`.
- Two positioning modes: absolute at `top: var(--search-top)` (128 px); when `.fix-hero` (scrollY ≥ 128) → `fixed top-[0px]` (it stays above/behind the sticky header area).
- **Opening shortcut**: document `keypress` listener — keys `s`, `/`, or `k` with `metaKey`/`ctrlKey` focus the input `#animegarden-search input` (only if it isn't already the active element; `preventDefault+stopPropagation`). There is **no modal dialog** — the command input is always in the hero; results expand below it.
- It is **not a modal**, there are no "tabs" — sections (groups) stack vertically in the `Command.List` dropdown.

### 4.2 cmdk structure (`Command` from `cmdk`)
- Root: `<Command id="animegarden-search" label="Command Menu" should-filter="false">`; click intercepted (`preventDefault/stopPropagation`); Escape prevented (no default close).
- Input row (`relative`): `<Command.Input id="animegarden-search-input">` (placeholder color `var(--gray9)`; value controlled by `inputStore`; `onValueChange` syncs store; `searched` class when active adds bottom margin 16px).
  - When `input` non-empty: clear button `absolute right-[20px] top-0 h-[30px] flex items-center cursor-pointer` with `i-carbon-close text-xl text-base-500 hover:text-base-900` (mousedown → clear input+search); plus a `h-[22px] border-r` separator at `right-[20px] top-[4px]`.
  - Submit button `absolute right-0 top-[-1px] h-[30px]` with `i-fluent:arrow-enter-24-filled text-base-500 hover:text-base-900` (mousedown → `selectGoToSearch()`, source `'button'`).

### 4.3 Input→location sync
- On location change (unless `location.state.trigger === 'search'`): the current URL query is stringified back into the input text via `stringifySearchTextAsync` (subject names resolved through the query client; for `/subject/{numeric-id}` routes, param is injected as `subject=<id>`). If `state.trigger === 'search'` and `state.input` is a string, restore that input. Then `history.replaceState(null, '')` clears the state.

### 4.4 Behavior while the input is focused (`enable = activeElement === inputRef.current`)
Debounce 500 ms before updating the `search` state (with IME composition handling: `isComposing` input events don't trigger; composition end triggers).

Groups in order (shown only when focused):
1. **Go-to-search item** (when `input.trim()`): `Command.Item value="go-to-search-page"` — label: `前往 {input}` if `isDirectDetailURL(input)` else `在本页列出 {input} 的搜索结果...`. Select → `selectGoToSearch(undefined, 'command')`: pushes to histories, tracks `search.trigger {text, source}`, resolves URL via `resolveSearchURL` and navigates with `state: { trigger: 'search', input: text }`.
2. **动画 group** (`SearchSubject`, when input + search-term keywords): resolved keywords = `[...parseSearchInput(search).search, ...parseSearchInput(search).include]`; query `subjectSearchQueryOptions(keywords)` (each keyword searched server-side with limit 20, merged and ranked by hit-count then id desc, top 3). Items: subject display names; selecting puts `动画:{name}` into the input and navigates to `/subject/{id}` (no cleanup); tracks `search.suggestion.click {text, subjectId}` (windowed 300 ms dedup).
3. **搜索结果 group** (`SearchResult`, when input): filter = `parseSearchInput(input)`; disabled when the input is a direct detail URL or when no `include/keywords/search/subjects`; query `resourcesQueryOptions({...filter, page: 1, pageSize: 5})`.
   - Loading: `Command.Loading` with `lds-ring` spinner + `正在搜索 {input} ...`.
   - Empty: `Command.Loading`/empty text `没有搜索到任何匹配的资源.`
   - Items: resource `title`; select → navigate `/detail/{provider}/{providerId}`; tracks `search.result.click {text: search, resource: 'provider:providerId'}`.
   - More item: `展示更多 {input} 的搜索结果...` → full search navigation (source `'result-more'`).
4. **搜索历史 group** (only when `!input.trim()` and histories exist): heading row `搜索历史` + `清空` button (tracked `search.history.delete {action:'clear', count}`); items deduped — row: `h.replace(/"/g,'')` text + `i-carbon-close` remove button (tracked `search.history.delete {action:'remove', text}`). Clicking an item → executes search (`selectGoToSearch(h, 'history')` + `trackSearchHistoryClick(h)`); `onSelect` sets input to the item.
5. **高级搜索 group** (`SearchCompletion`, always): 8 items that append syntax to the input (`onInputChange(input + suffix)`):
   - `包含关键词` → ` 包含:`
   - `排除关键词` → ` 排除:`
   - `筛选字幕组` → ` 字幕组:`
   - `匹配标题` → ` 标题:`
   - `上传时间晚于` → ` 晚于:`
   - `上传时间早于` → ` 早于:`
   - `筛选资源类型` → ` 类型:`
   - `高级搜索帮助` → `window.open('https://docs.animes.garden/animegarden/search.html')`

Histories update on submit: keep only entries not contained in the new input, dedupe, prepend, cap at **10** → `historiesStore`.

### 4.5 parseSearchInput (search syntax → filter)
Tokenizer: splits on whitespace but respects quotes `"` `'` `“` `”` and backslash escapes (`\"`, `\\`).
Prefix handlers (all support full-width colon variants, e.g. `动画：`):
- `subject:` / `动画:` → `subjects` (raw names, resolved later)
- `title:` / `标题:` / `匹配:` → `include`
- `+` / `include:` / `包含:` → `keywords`
- `!` / `！` / `-` / `exclude:` / `排除:` → `exclude`
- `user:` / `publisher:` / `发布:` / `发布者:` / `发布人:` → `publishers`
- `team:` / `fansub:` / `字幕:` / `字幕组:` → `fansubs`
- `>=` / `>` / `after:` / `开始:` / `晚于:` → `after` (`new Date(word)`, last wins)
- `<=` / `<` / `before:` / `结束:` / `早于:` → `before` (last wins)
- `type:` / `类型:` → `types`
- `preset:` / `预设:` → `preset` (matched against `PRESET_DISPLAY_NAME` values or keys)
Any word not matched → `search` (with `+` escaped to `%2b`). If `include|keywords|exclude` non-empty, `search` words are folded into `include` and `search` cleared.
Returns `{ search, include, keywords, exclude, subjects, publishers, fansubs, after, before, types, preset }`.

### 4.6 stringifySearchText / stringifySearchTextAsync (URL → input text)
Order: `动画:{name}` (quoted `"…"` when the name contains a space; only for exactly 1 subject) → if `search` non-empty list them raw; else `标题:{w}`, `包含:{w}`, `排除:{w}` → `发布者:{f}` → `字幕组:{f}` → `类型:{t}` (each) → `开始:{iso}` (`结束:` likewise; a date exactly `T16:00:00.000Z` is printed as `yyyy-MM-dd` only) → `预设:{PRESET_DISPLAY_NAME[preset] ?? preset}`. Words containing spaces are wrapped in `"…"` with inner `"` escaped. Joined with single spaces.

### 4.7 resolveSearchURL (submit)
1. Input starts with `location.origin` or `location.host` → strip to a path.
2. `matchDirectDetailURL`: DMHY regex `(?:https://share.dmhy.org/topics/view/)?(\d+_[a-zA-Z0-9_\-]+\.html)` → `/detail/dmhy/{id}`; Mikan regex `(?:https?://mikanani\.kas\.pub/Home/Episode/|/Home/Episode/)?([0-9a-fA-F]{40})` → `/detail/mikan/{hex.lowercase()}`.
3. Otherwise parse input; resolve `subjects` names to ids via `subjectsByNameQueryOptions`. If the only serialized param is `subject` → `/subject/{id}`; else `/resources/1?{stringifyURLSearch(filter)}`.

### 4.8 cmdk visual tokens (cmdk.css, `.vercel` scope)
Vercel-gray scale CSS variables (`--gray1..#12`, `--grayA1..#12`, `--blue1..#12`) with a light + `.dark .vercel` override (dark bg `hsla(0,0%,9%,1)`); root `[cmdk-root]`: absolute top-0, `padding: 6px`, `border: 1px solid var(--gray6)`, `border-radius: 8px`, background #fff; `[cmdk-input]`: 16 px, `padding: 2px 2px 4px`, `padding-right: 44px`, bottom border `1px solid var(--gray6)`; `.searched` → `margin-bottom: 16px`; `[cmdk-list]`: `height: min(330px, var(--cmdk-list-height)); max-height: 400px; overflow: auto; overscroll-behavior: contain`; items `height: 48px; border-radius: 8px; font-size: 14px; padding: 0 16px; color: var(--gray11)`, selected `background: var(--grayA3); color: var(--gray12)`, active press `background: var(--gray4)`; items `margin-top: 4px` between; group heading 12px `color: var(--gray11), padding: 0 8px, margin-bottom: 8px`; loading uses the `lds-ring` CSS spinner.

---

## 5. Resources page in detail

### 5.1 Filter bar / FilterCard (`pages/resources.($page)/Filter.tsx`)
Rendered above the table when (a) a filter exists and (b) at least one of `types/subjects/fansubs/publishers/search/include/keywords/before/after` is non-empty. Isolated `preset` alone also renders (its own row). Container: `mb4 p-4 lt-sm:px-3 w-full bg-zinc-50 dark:bg-zinc-800 drop-shadow rounded-md space-y-2`.
Rows (each: bold label span `text-4 text-base-800 font-bold mr2 select-none keyword` + values):
- `预设` → `PRESET_DISPLAY_NAME[preset]` (value selectable).
- `动画` → each subject a Link to `/subject/{id}` (`text-4 select-text text-base-900 text-link`) with `getSubjectDisplayName`.
- `类型` → colored spans `text-4 select-text text-base-600 ${DisplayTypeColor[type]}` (no links).
- `发布者` → Links `/resources/1?publisher={p}` (`text-link`).
- `字幕组` → Links `/resources/1?fansub={f}` (`text-link`).
- `搜索开始于` / `搜索结束于` → `formatChinaTime(date, 'yyyy 年 M 月 d 日 HH:mm')`.
- `标题搜索` → `search` words, dotted-underline (`underline underline-dotted underline-gray-500`).
- `标题匹配` → `include` words (`|` separators) — only when `search` empty.
- `包含关键词` → `keywords` words (`&` separators) — only when `search` empty.
- `排除关键词` → `exclude` words.

**Operations row** (`FilterOperations`) — rendered only when `search.length !== 0 || include.length !== 0 || keywords.length !== 0 || realSubjects.length !== 0`; classes `flex items-center gap-4 lt-sm:gap-2 pt-4`:
1. **`添加到收藏夹`** button (shadcn `Button variant="outline" size="sm"`, class `add-collection`, icon `i-carbon:bookmark mr1`): builds `searchParams = '?' + stringifyURLSearch(resolved).toString()`; if the current collection has no filter with that searchParams → `addCollectionItem` (prepend) + toast.success `成功添加到 {collection.name}`; else toast.warning `已添加到 {collection.name}`; always opens the sidebar (`isOpenSidebarStore(true)`); tracks `collection.add`.
2. **CopyResourcesDropdown**: inline joined control `inline-flex w-fit divide-x rounded-md`: left `Button variant outline sm` `复制 RSS 订阅链接` (icon `i-carbon-copy mr1`; `rounded-none rounded-s-md`; copies `feedURL`) + right chevron `Button` (`i-carbon-chevron-down text-xl`, `rounded-none rounded-e-md`) opening a Radix DropdownMenu (`side="bottom" sideOffset={4} align="end"`, `w-[200px]`) with 6 items (each with icon + label):
   - `复制所有磁力链接` (`i-mage-magnet-up mr1`) — clipboard text = `resources.map(r => r.magnet + (r.tracker ?? '')).join('\n')`; toast.success `成功复制 {n} 条磁力链接` / toast.error `没有磁力链接`; tracks `copy.magnet-links`.
   - `复制 JSON 数据` (`i-mdi-code-json mr1`) — `JSON.stringify({ filter, resources }, null, 2)`; toast `复制 JSON 数据成功` / `复制 JSON 数据失败`; tracks `copy.json`.
   - `复制为 cURL 命令` (`i-fluent-key-command-20-filled mr1`) — `generateCurlCode({ filter })`; toast `复制 cURL 命令成功` / 失败; tracks `copy.fetch {language:'curl'}`.
   - `复制为 JavaScript 代码` (`i-proicons-javascript mr1`) — `generateJavaScriptCode({ filter })`; toast `复制 @animegarden/client JavaScript 代码成功` / 失败; tracks `copy.fetch {language:'javascript'}`.
   - `复制为 Python 代码` (`i-proicons-python mr1`) — `generatePythonCode({ filter })`; toast `复制 Python 代码成功` / 失败; tracks `copy.fetch {language:'python'}`.
   - `复制为网页嵌入代码` (`i-solar-code-bold mr1`) — `generateIframeCode({ filter })`; toast `复制网页嵌入代码成功` / 失败; tracks `copy.iframe`.
3. **SearchTooltip** (`lt-sm:hidden`): a link icon `i-carbon-help text-2xl text-link-active` → `https://docs.animes.garden/animegarden/search.html`.

**Code-generation formats (utils/code-generator.ts)** — resolved filter again (subjects may be overridden by a single subject id):

- cURL:
  ```
  curl "https://{FEED_HOST}/resources?{stringifyURLSearch(realFilter)}"
  ```
  (params sorted alphabetically, e.g. `curl "https://api.animes.garden/resources?fansub=%E6%A1%9C%E9%83%BD%E5%AD%97%E5%B9%95%E7%BB%84&search=%E8%91%AC%E9%80%81%E7%9A%84%E8%8A%99%E8%8E%89%E8%8E%B2&type=%E5%8A%A8%E7%94%BB"`)
- JavaScript (`@animegarden/client`):
  ```
  import { fetchResources } from '@animegarden/client';

  const resources = await fetchResources({
    preset: 'bangumi',
    types: ["动画"],
    subjects: [12345],
    publishers: ["..."],
    fansubs: ["桜都字幕组"],
    search: ["葬送的芙莉莲"],
    include: ["..."],
    keywords: ["..."],
    exclude: ["..."],
    after: new Date('2026-07-01T00:00:00.000Z'),
    before: new Date('2026-07-01T00:00:00.000Z')
  });
  ```
  (lines omitted when empty; two-space indent; keys appear in fixed order preset/types/subjects/publishers/fansubs/search/include/keywords/exclude/after/before)
- Python (requests):
  ```
  import requests

  url = "https://{FEED_HOST}/resources"
  params = {
    'preset': 'bangumi',
    'type': ["动画"],
    'subject': [12345],
    'publisher': ["..."],
    'fansub': ["桜都字幕组"],
    'search': ["葬送的芙莉莲"],
    'include': ["..."],
    'keyword': ["..."],
    'exclude': ["..."],
    'after': 1751328000000,
    'before': 1751328000000
  }

  response = requests.get(url, params=params)
  resources = response.json()
  ```
  (note singular key names `type/subject/publisher/fansub/keyword`; after/before as epoch **milliseconds** numbers)
- iframe:
  ```
  <iframe src="//{APP_HOST}/iframe?{stringifyURLSearch(realFilter)}" width="100%" height="600" frameborder="0" style="box-sizing:border-box;"></iframe>
  ```
- All four throw `new Error('没有筛选条件')` when `filter` is missing.

### 5.2 Table (`components/Resources/table.tsx`)
- Wrapper `.overflow-y-auto w-full`; `<table class="resources-table border-collapse min-y-[1080px] w-full">`; colgroup: first col `text-left xl:min-w-[600px] lg:min-w-[480px]`, then two `w-max whitespace-nowrap`.
- thead (`resources-table-head border-b-2 text-lg lt-lg:text-base`):
  - 资源 column: `py3 pl3 lt-sm:pl1 text-left`; inner flex with a fixed 32px icon slot (`flex-shrink-0 mr3 flex justify-center items-center w-[32px]`) holding the `CarbonTypes` icon (two chevrons) + label `资源`.
  - 发布者 column (`py3 min-w-[60px]`, only when `columns.fansub !== false`).
  - 播放 column (`py3 px2 text-center w-[72px]`).
- Rows (`resources-table-body divide-y border-b text-base lt-lg:text-sm`), key `provider/providerId`:
  - **Cell 1**: type-icon round button: `<Link to="/resources/$page" params="1" :search="followSearch(location, { type: r.type })">` — `flex items-center justify-center h-[32px] w-[32px] rounded-full bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700 ${DisplayTypeColor[r.type]}` + `DisplayTypeIcon[r.type]`; click tracks `resources.filter.click {filterType:'type', value:r.type}`. `followSearch` copies current URL params then `set(key, value)` (replaces!) and converts via `toResourcesRouterSearch` (keeps only resource params + `pageSize`).
  - **Title row** (`flex-1`): for types `动画|合集|日剧|特摄`, the title is an `<a href={getPikPakUrlChecker(r.magnet)} target="_blank">` (aria-label `Go to download resource of {title}`) — direct PikPak opening; otherwise a `<Link to="/detail/{provider}/{providerId}" class="text-link">` (aria-label `Go to resource detail of {title}`). All titles `class="text-link mr1"`.
    - **Meta row** (`mt1 flex items-center gap-4`, text `text-xs text-zinc-400`):
      - `发布于 {formatChinaTime(new Date(r.createdAt))}` — Link to detail (`text-link-secondary-hover-base`).
      - `大小 {parseSize(r.size)}` — `<a :href="magnet + (hydrated ? tracker : '')">` clicking calls `openMagnetLink(event, magnetHref, 'provider:providerId', 'size')` (tracks `download`), aria-label `Download resource`.
      - `详情` — Link to detail (`text-link-secondary text-xs`; icon `CarbonLaunch` inline + space + `<span class="more">详情</span>`); tracks `resources.detail.click {resource, type}`.
  - **Cell 2 发布者**: centered; if `r.fansub` → `<Tag text={fansub.name} class="text-xs hover:bg-gray-300 dark:hover:bg-gray-700">` linked to `/resources/1` + `followSearch(fansub=r.fansub.name)`, tracks `resources.filter.click {filterType:'fansub',...}`; else `r.publisher` → same with `publisher` param ({filterType:'publisher'}); else empty.
  - **Cell 3 播放**: `flex gap1 items-center justify-start`:
    - Play: `<a :href="getPikPakUrlChecker(r.magnet)" target="_blank" class="play text-xl text-base-500 hover:text-base-900" aria-label="Play resource">` + `CarbonPlay` icon; carries `data-umami-event="pikpak"` attrs (resource & source).
    - Download: `<a :href="magnetHref" class="download text-xl text-base-500 hover:text-base-900" aria-label="Download resource">` + `CarbonDownload` icon; click → `openMagnetLink(..., 'download')`.
  - The magnet `href` appends `r.tracker` **only after hydration** (`useHydrated`).
- **Empty state** (`page !== undefined && resources.length === 0`): `h-20 text-2xl text-orange-600/80` with `CarbonError` + `没有搜索到匹配的资源`; below, centered: if current path doesn't end with `/1` and link provided: `第 1 页` (link) ` / `, then `主页` (link to `/`).
- **Pagination** (`components/Resources/pagination.tsx`) rendered when `page !== undefined && !complete && resources.length > 0`:
  - `mt-4 flex lt-md:flex-col font-sm`; right-aligned row (`flex-auto` spacer; `flex lt-md:(mt-4 justify-center) items-center gap-2 text-base-500`).
  - Items: `上一页` (hidden when `page <= 1`), `1` (when `page > 2`), ellipsis `i-ant-design:ellipsis-outlined` (when `page > 2`), pages `[page-1..page+3]` when prev exists else `[page..page+4]`, filtered to `p <= page` when complete, ellipsis when next, `下一页` (hidden when complete).
  - Page item classes: `px-2 py-1 rounded-md hover:bg-gray-100 select-none cursor-pointer`; **current page `text-pink-600`**; `上一页/下一页` links `text-link-active`. With `link` prop → router `<Link>`; with `navigate` prop → clickable div calling `navigate(page)`; iframe page uses `navigate`; page ≤ 0 renders an empty div.

### 5.3 Tag component (`components/Resources/tag.tsx`)
`inline-flex items-center gap1 rounded-md px2 py1 font-mono transition transition-colors text-base-800 ${color}` with default color `bg-gray-200 dark:bg-gray-800`.

### 5.4 URL ⇄ filter parsing on the route (summary)
`parseURLSearch(url.searchParams, { pageSize: 80 })` → `{ filter, pagination }`; request sends `{...filter, ...pagination, page, pageSize: 30, tracker: true, metadata: true}` through the server-function proxy (which resolves string `subject` names server-side via bgmx search and drops unknown names). The `page` from the URL is authoritative for the pagination UI; the URL itself remains the sole state (no in-page filter editor — filters are changed via the search box, nav, and table links).
---

## 6. Subject page in detail

### 6.1 Data flow
- `subjectQueryOptions(subjectId)` → serverFn `fetchSubjectFn` → `getSubjectById` (bgmx `fetchSubject` with 10 s timeout / 1 retry, LRU in-memory cache keyed `subject:{id}` with 1 h TTL, fallback to bundled `bgmd/full` dataset) → `WebBgmSubject`.
- Resources: `resourcesQueryOptions({ subject: id, page: 1, pageSize: 1000, types: ['动画','合集'] })` (full list, no pagination UI).
- `WebBgmSubject` shape: `{ id, title, display_title, platform, onair_date, poster, summary, alias: {ja?, zh?, en?}, tags: string[], search: { include: string[] } }`. `display_title` = first `alias.zh` entry or `title`; `poster` = `https://bgm.animes.garden/bangumi/subject/{id}/poster.jpeg?quality=large`; `tags` = union of `meta_tags` + `tags`; `onair_date` = `onair_date || bangumi.date`.

### 6.2 SubjectCard (`pages/subject.$subject.($page)/subject.tsx`)
- Container: `mb-12 p-4 w-full bg-zinc-50 dark:bg-zinc-800 drop-shadow rounded-md flex gap-8 lt-md:flex-col`.
- Poster: if `subject.poster` → `<img :src="poster" :alt="displayName" class="rounded-md hover:drop-shadow">` inside `w-[300px] flex-shrink-0 lt-md:w-full`; else an empty `w-[300px] ...` div (keeps layout).
- Info (`info-box space-y-8`):
  - `<h1 class="text-2xl font-bold flex items-center gap-2 pr-2">` → Link to `/subject/{id}` (`text-link-active`) with display name; then **share button** (`h-[30px] px-1.5 py-1.5 w-auto rounded-md flex items-center cursor-pointer hover:bg-layer-muted` + `i-carbon-share text-sm`) → copies `{origin}/subject/{id}` → toast.success `复制 {displayName} 链接成功`.
  - Info article `grid grid-cols-1 gap-2`:
    - `放送日期` → `getWeekday(onair_date)` (e.g. `星期四`; computed in UTC from `yyyy-MM-dd`; weekdays `['星期日','星期一','星期二','星期三','星期四','星期五','星期六']`).
    - `放送开始` → raw `onair_date` string.
    - `外部链接` → `<a href="https://bgm.tv/subject/{id}" target="_blank" class="text-link inline-flex items-center gap-1 underline underline-offset-4">` with `i-mingcute-bilibili-line` icon + `Bangumi`.
  - `SubjectSummary` (`summary-box`): `leading-relaxed space-y-2 text-base text-base-600`; each `summary` line (`split('\n')`) its own `<p>`; then if `tags` non-empty a `mt-4` paragraph of `Tag` pills: `text-sm bg-zinc-200 hover:bg-zinc-300 dark:bg-zinc-600 text-base-800 px-2 py-1 rounded-md inline-flex items-center` (+ `cursor-default hover:bg-zinc-200!`), plain text (no href).

### 6.3 Grouped resource sections
`groupResourcesByFansub(resources)` (utils.ts): bucket by `resource.fansub.id` (fansub groups) and `resource.publisher.id` (no-fansub groups); then **sort by the hard-coded FansubNames order** (33 known names, see list in source: 雪飄工作室(FLsnow), 驯兽师联盟, 北宇治字幕组, LoliHouse, 喵萌奶茶屋, Prejudice-Studio, 三明治摆烂组, MoYuanCN, 拨雪寻春, SweetSub, S1百综字幕组, 星空字幕组, 桜都字幕组, 悠哈C9字幕社, 悠哈璃羽字幕社, MingYSub, 幻樱字幕组, 离谱Sub, 爱恋字幕社, 黑白字幕组, 绿茶字幕组, 霜庭云花Sub, 云光字幕组, 云歌字幕组, 千夏字幕组, 六四位元字幕組, 阿特拉斯字幕组, 晚街与灯, 极影字幕社, 猎户发布组, ANi, DBD制作组, VCB-Studio); known groups first by order, unknown ones after (sorted by `publisher.name.localeCompare`). Group = `{ publisher, fansub?, resources }`.
Each group section (flex flex-col gap-12):
- `<h2 class="text-2xl font-bold flex items-center gap-2 pr-2">`:
  - Link `getResourcesRouteLink(1, fansub ? { subject: String(id), fansub: name } : { subject: String(id), publisher: name })` with `text-link-active`; label = `group.fansub?.name ?? group.publisher.name`.
  - spacer + **RSS link**: `<a :href="getFeedURL('?subject={id}&fansub={name}')" target="_blank" class="flex items-center cursor-pointer text-base font-light text-[#ee802f] border-b-2 border-b-transparent hover:(text-[#ff7800] border-b-[#ff7800])">` with `i-mdi-rss text-sm mr-1 relative top-[1px]` + `RSS`; tracks `feed.open`.
- `<Resources resources={group.resources} :columns="{ fansub: false }">` (publisher column still shown, fansub column hidden; no pagination inside a group).
- When `!complete`: footer row `py-4 px-8 lt-xl:px-2 text-right border-b` with Link `搜索更多资源...` (`text-link`) to `/resources/1?subject={id}&fansub={name}` (or publisher variant).

### 6.4 Feed URL for the whole page
`getFeedURL('?' + sortedSearch)` where sortedSearch = current `location.searchStr` with `subject` set to the **raw path param string** (`subjectParam`), then `search.sort()` — so `/subject/330279?type=动画` → `https://api.animes.garden/feed.xml?subject=330279&type=动画`.

---

## 7. Anime / calendar page in detail (`pages/anime/route.tsx` + `anime.css`)

### 7.1 Data & navigation
- `/anime` → redirect to `/calendar/{is_active season}`; `/calendar/{season}` validates the season against the calendars list (404 status otherwise, empty calendar), fetches `calendarQueryOptions(season)`.
- `onSeasonChange(nextSeason)` → `navigate('/calendar/' + nextSeason)` (URL = state).
- Calendar data: `getCalendar(rawCalendar)` rotates 7 weekday buckets so **today is first**: weekday = ISO weekday of `now - 6h` in Asia/Shanghai minus 1; `index = (idx + Weekday) % 7 + 1`; `text` = `['一','二','三','四','五','六','日'][idx]`. Each bucket: filters out subjects without poster and **Chinese-origin productions** (tags containing any of `国创 国产 国产动画 国漫 中国`), sorted by Chinese-flag then `onair_date` desc.

### 7.2 Season model (`utils/calendar-season.ts`)
- `CalendarSeasonMonths = [1, 4, 7, 10]`; map: `1 → {emoji:'❄️', name:'冬季新番', label:'冬季'}`, `4 → {emoji:'🌸', name:'春季新番', label:'春季'}`, `7 → {emoji:'☀️', name:'夏季新番', label:'夏季'}`, `10 → {emoji:'🍁', name:'秋季新番', label:'秋季'}`; unknown month → `{emoji:'', name:'{M} 月新番', label:'{M} 月'}`.
- `getCalendarSeason('YYYY-MM')` → `{ season, year, month, emoji, name, label, title: '{year} · {name}' }` (e.g. `2026 · 夏季新番`); no season → `{ season:'', year:'', month:0, emoji:'', name:'新番', label:'选择季度', title:'新番' }`.

### 7.3 Layout
- `<Layout timestamp>` with wrapper `anime-page w-full pt-13 pb-24` (`padding-left: 1rem` on desktop, 0 on mobile).
- **Header** (`mb-12 flex items-end justify-between gap-4 lt-sm:flex-col lt-sm:items-start`): `<h1 class="text-3xl lt-sm:text-2xl font-bold leading-tight tracking-normal select-none">` with emoji span (`anime-season-emoji text-2xl font-quicksand font-bold`, aria-hidden, `width:1em; margin-right:16px`) + `{selectedSeason.title}`.
- **Season controls** (`flex gap-2`, disabled when no calendars):
  - Year dropdown: `Button variant="outline"` `h-10 w-[112px] justify-between rounded-md px-3 text-base font-bold` label `{year} 年` + `ChevronDown` (lucide, `ml-2 h-4 w-4 opacity-60`); Radix DropdownMenuContent `align="end"` width = trigger width; RadioGroup of years (from unique `season.split('-')[0]`); `updateSeason(year, month)` picks: exact month match → that season; else the latest season of that year with `month <= target`; else the first season of the year.
  - Season dropdown: `Button` label `<span class="mr-1.5">{emoji}</span>{label}` (e.g. `🌸 春季`); RadioGroup of the current year's seasons sorted by `CalendarSeasonMonths`; radio items `anime-season-menu-item` (`padding-left: 0.5rem`; hides the first child span; checked state bold `#18181b` / dark `#fafafa`).
- **Body** (`anime-layout` grid `minmax(0,1fr)`):
  - **TOC nav** (`anime-toc`, `aria-label="星期目录"`, sticky `top: calc(var(--nav-height) + 4rem)`, width 4.5rem, off-left aligned via `margin-left: calc((100vw - 100%) / -2 + 1.375rem)`, column gap 0.75rem, 14px, color `#71717a`; dark `#a1a1aa`): anchor links `星期{text}` per weekday (`href="#星期{text}"`); active (`is-active`) = current scroll-spy weekday → `font-weight:600; color:#18181b` (dark `#fafafa`). **Hidden below 768 px.**
  - **Day sections** (`anime-days space-y-14`): one `<section class="bgm-weekday" id="星期{text}">` per weekday — `scroll-margin-top: var(--nav-height); padding-top: 2rem; border-top: 1px solid rgb(187 187 187 / 20%)`; `<h2 class="mb-6 select-none"><a class="text-2xl lt-sm:text-xl font-bold leading-tight" href="#星期{text}">星期{text}</a></h2>`; then grid `anime-bgm-grid` (`repeat(auto-fill, minmax(132px, 1fr)); gap: 2rem 1.5rem`; mobile `minmax(112px, 1fr); gap: 1.5rem 1rem`).
  - **Anime card** (`anime-card group`, Link to `/subject/{id}`, click tracks `anime.calendar.click {subjectId, title, weekday:'星期{x}'}`): poster box `anime-poster` (`aspect-ratio: 2/3; overflow: hidden; rounded-md bg-layer-muted`) with `<img :src="poster" :alt="{name} poster">` (`object-fit: cover`; `group-hover:shadow-box` transition); title `anime-title` (`padding-top: 0.625rem; font-size: 0.875rem; font-weight: 700; line-height: 1.25rem; text-link-active`), 2-line clamp via `-webkit-line-clamp: 2`.
- **Scroll spy**: rAF-throttled scroll/resize listener; active weekday = last section whose `getBoundingClientRect().top <= min(window.innerHeight * 0.35, 180)`; default = first weekday's index.

---

## 8. Detail page in detail (`pages/detail.$provider.$providerId/`)

`<Layout timestamp heading={false}>` (no hero title, but hero banner stays as empty area). Wrapper `detail mt-4vh w-full space-y-4`, inside `w-full pt-13 pb-24`.

1. **Title**: `<h1 class="text-xl font-bold resource-title"><span>{resource.title}</span></h1>`.
2. **下载链接 card** (`download-link rounded-md shadow-box`):
   - Header `<h2 class="text-lg font-bold border-b px4 py2 flex items-center">` with PikPak link `下载链接` (`text-link-active underline underline-dotted underline-offset-6`, target _blank, tracks pikpak `{resource, source:'detail'}`).
   - Body `p4 space-y-1 overflow-auto whitespace-nowrap`; label column `w-[160px] min-w-[160px] lt-sm:w-[120px] lt-sm:min-w-[120px]` (`text-base-600 select-none inline-block`), values `inline-block flex-1 pr-4`, `lt-md:text-sm`:
     - Row: label `在线播放` (PikPak link) / value `使用 PikPak 播放` (PikPak link, `text-link`).
     - Magnet rows: if `detail?.magnets` exists → one row per magnet `{name, url}` (labels like `磁力链接`, or e.g. `[ANi] ... [磁力]`); else one fallback row: label `磁力链接`, value `resource.magnet + (resource.tracker ?? '')`. Value = `<a :href="magnet" class="download text-link">` showing `splitMagnetURL(url)` (everything before the first `&`); click → `openMagnetLink(event, url, 'provider:providerId', 'detail')` (preventDefault + track `download` + `window.location.assign(url)`).
     - Row: label `原链接` / value `<a :href="resource.href" target="_blank" class="text-link">{resource.href}</a>`.
   - `magnet` used for the PikPak checker = `resource.magnet || first detail magnet starting with 'magnet:'`; `pikpakUrl = getPikPakUrlChecker(magnet)` = `https://keepshare.org/{KEEPSHARE_ID}/{encodeURIComponent(magnetWithoutQuery)}`.
3. **Description**: `<div class="description" v-html="normalizedDescription">` where the raw `detail.description` first has the leading `<strong>簡介:</strong>...` block replaced with `<h2 className="text-xl font-bold">简介</h2>` (regex `<strong>)?簡介:(&nbsp;)*(</strong>)?(<br>)?(<hr>)?`); the description itself was server-normalized by `@animegarden/scraper` (`normalizeDescription`). CSS (`detail.css`): `.description h1..h6 {font-weight:bold}`, `p {margin: .5rem 0}`, `hr {margin: .5rem 0}`, `a {color: sky-700; hover: sky-500}`.
4. **发布者 / 字幕组** (`publisher`): `<h2 class="text-lg font-bold pb4">{resource.fansub ? '发布者 / 字幕组' : '发布者'}</h2>`; `flex gap8`; each entry: Link `getResourcesRouteLink(1, { publisher: name })` / `{ fansub: name }` with `<img class="inline-block w-[100px] h-[100px] rounded" :src="avatar ?? 'https://share.dmhy.org/images/defaultUser.png'">` (onerror swaps to the default user image) + `<span class="text-link block mt2">{name}</span>`. Publisher always shown; fansub card only when present.
5. **发布于**: `<span class="font-bold">发布于&nbsp;</span>` + `formatInTimeZone(createdAt, 'Asia/Shanghai', 'yyyy-MM-dd HH:mm')`.
6. **文件列表** (`FilesCard`, `pages/detail.$provider.$providerId/FileTree.tsx`): card `file-list rounded-md shadow-box`; `<h2 class="text-lg font-bold border-b px4 py2">文件列表</h2>`; body `mb4 max-h-[80vh] overflow-auto px4 py4 space-y-2`:
   - Tree built by splitting each `file.name` on `/`; leaves get the file `size`.
   - Row per node: `flex items-center gap4`; icon + name `text-sm text-base-600`; right-aligned size `text-xs text-base-400 select-none` for files only.
   - Icons: directory `i-ant-design-folder-outlined`; files by extension: `mp4`/`mkv` → `i-ant-design-play-circle-outlined`; `ass` → `i-fluent-subtitles-24-regular`; `rar`/`7z`/`zip` → `i-ant-design-file-zip-outlined`; else `i-ant-design-file-outlined`.
   - Children: `my-1 pl-4 py-1 space-y-2 border-l border-l-1` (indented, left border).
   - `files.length === 0` → centered red text `种子信息解析失败` (`py2 select-none text-center text-red-400`).
   - `hasMoreFiles` → trailing `...` (`text-base-400`).

---

## 9. Collection system (`stores/collection.ts`, sidebar, page)

### 9.1 Data model
```ts
type Collection = {
  hash?: string;            // server hash once generated
  name: string;             // default '收藏夹'
  authorization: string;    // client secret; crypto.randomUUID() when created
  filters: Array<{
    searchParams: string;   // e.g. '?type=动画&type=合集&preset=bangumi' (from stringifyURLSearch + leading '?')
    name?: string;          // custom display name ('' allowed → inferred)
    subjects?: number[]; types?: string[]; publishers?: string[]; fansubs?: string[];
    search?: string[]; include?: string[]; keywords?: string[]; exclude?: string[];
    after?: string; before?: string; preset?: string;
  }>;
};
```
Default: `{ hash: undefined, name: '收藏夹', authorization: '', filters: [] }`.

### 9.2 Stores
- `collectionsStore` (Record<name, Collection>), `currentCollectionNameStore` (string), `currentCollectionStore` (derived: collections[name] ?? `{hash:undefined, name, authorization:'', filters:[]}`).
- Persistence: current name → localStorage `animegarden:cur_collection_name` (JSON string); all collections → **IndexedDB** database `animegarden:collections` version 1, object stores `key-value` (key = collection name) and `_meta`; on init reads DB, validates entries (`isCollection`: object with string `name` and array `filters`), writes back if empty; every store change re-persists (clear + put all) with console.warn on failure.
- Mutations (all in-memory + auto-persist; hash reset to `undefined` on any filter change):
  - `setCurrentCollection(stores, c)` — sets name + upserts into map.
  - `addCollectionItem(stores, collection, value)` — **prepends** when `searchParams` not already present.
  - `updateCollectionItem(stores, collection, value)` — replaces by `searchParams` match (no-op if missing).
  - `deleteCollectionItem(stores, collection, value)` — removes by `searchParams` match.
  - `updateCollection(stores, collection, partial)` — shallow merge (used for `authorization` / `hash`).

### 9.3 Server sync
- `generateCollectionMutationOptions()` → serverFn `generateCollectionFn` (POST) → `rawGenerateCollection(collection, { baseURL, retry: 0 })` → response `{ hash, name?, authorization? }`-like; the web page stores `resp.hash` back into the local collection. **No pull-sync**: the shared collection page (`/collection/{hash}`) is read-only rendering of the server response (`collectionQueryOptions(hash)` → `rawFetchCollection`).
- Page feed: `https://{FEED_HOST}/collection/{hash}/feed.xml`.

### 9.4 Collection page rendering (recap)
One bordered box per filter (title = inferable name), each with an embedded ResourcesTable (`complete` → no pagination), plus the collection-level feedURL passed to Layout's footer/header RSS. If `data` is falsy (render error) → `Error` with tracking `collection-render-failed`.
---

## 10. API docs page (`/docs/api`)

- UI framework: **swagger-ui-react** (v5.32.8), rendered full-page inside the standard Layout (`<div class="w-full pt-13 pb-24"><SwaggerUI :spec="spec"/></div>`); imports `swagger-ui-react/swagger-ui.css`.
- `spec` = `getPublicOpenApiSpec(version, license)` (from `pages/docs.api/spec.ts`):
  - Base spec is the bundled `spec.json` (OpenAPI 3.1.0, 1899 lines).
  - `info.version` ← package version (`0.5.4`), `info.license.name` ← package license (`MIT`).
  - **Removes** `components.securitySchemes`, all `/admin/*` paths, and the `Admin` tag (public/agent-facing spec only).
- Spec info: title `🌸 Anime Garden API`; description (markdown) covering: third-party mirror of 動漫花園 + anime torrent aggregation; open API for developers; anime airing schedule; advanced search example `` `葬送的芙莉莲 +简体内嵌 字幕组:桜都字幕组 类型:动画` ``; custom RSS feeds (example link to `https://api.animes.garden/feed.xml?filter=[...]`); search-condition collections & aggregated RSS; AutoBangumi / AnimeSpace integration. `externalDocs` → GitHub repo. Server `https://api.animes.garden` (description `API 服务器`).
- Endpoints in the public spec (path → operationId):
  - `GET /` `getStatus` — status info (message, timestamp, providers dmhy/mikan/moe/ani) — tag `Status`.
  - `GET /users` `getUsers` — all publisher users — tag `Users`.
  - `GET /teams` `getTeams` — fansub teams — tag `Users`.
  - `GET /subjects` `getSubjects` — subjects — tag `Subjects`.
  - `GET /resources` `getResources`, `POST /resources` `postResources` — resource list — tag `Resources`.
  - `GET /resources/{provider}` `getProviderResources`, `POST /resources/{provider}` `postProviderResources` — tag `Resources`.
  - `GET /detail/{provider}/{id}` `getDetailAlias` — detail alias — tag `Resources`.
  - `POST /collection` `createCollection`, `PUT /collection` `upsertCollection`, `GET /collection/{hash}` `getCollection` — tag `Collections`.
  - `GET /feed.xml` `getRSSFeed`, `GET /collection/{hash}/feed.xml` `getCollectionRSSFeed` — tag `Feed`.
  - `GET /sitemaps/subjects` `getSubjectsSitemap`, `GET /sitemaps/{year}/{month}` `getResourcesSitemap` — tag `Sitemap`.
  - (Removed from public UI) `/admin/providers`, `/admin/resources/{provider}`, `/admin/resources/{provider}/sync` — tag `Admin`.
- Tags (public): `Status`, `Users`, `Subjects`, `Resources`, `Collections`, `Feed`, `Sitemap`.
- The same public spec is served at `/openapi.json` (server route, `Cache-Control: public, max-age=3600, s-maxage=86400`).
- Route loader caches the page with `ResponseCacheControl.Docs`.

---

## 11. Iframe page (`/iframe`)

- **Purpose**: embeddable “latest resources” widget for third-party sites; generated by “复制为网页嵌入代码” (§5.1).
- **URL params**: any of the standard resource-list query params (`fansub`, `publisher`, `type`, `subject`, `search`, `include`, `keyword`, `exclude`, `after`, `before`, `preset`, `provider`, `duplicate`, `page`, `pageSize` — pageSize is overridden to 30), plus:
  - `className` (repeatable) — extra CSS classes on the root wrapper.
  - `style` (repeatable, joined with `;`) — inline CSS parsed by `style-to-object`.
- **Rendering**: no app shell (no header/hero/footer/sidebar); root `<div class="w-full" :class="..." :style="...">` containing the Resources table + pagination (via `navigate`, URL `/iframe?{search}&page={n}`).
- Title/description/canonical from the filter (§1.8). Page fetch error → `Error` component.

---

## 12. About page (`/about`)

- Loader fetches timestamp + calendar (Cache-Control List). Title `关于 | Anime Garden …`; description `Anime Garden 動漫花園資源網镜像站, 动漫花园动画 BT 资源聚合站`.
- **Body is a placeholder**: `<Layout timestamp><div class="w-full pt-13 pb-24"><div class="h-[1000px]"></div></div></Layout>` — an empty 1000 px block. No text sections exist in the current source; the footer below carries the site's real “about” content.

### Footer (shared layout, `layouts/Footer.tsx`)
Rendered when `Layout footer !== false`; `<footer class="w-full relative flex justify-center border-t border-t-1 py-6 h-[252px] bg-hero" :class="isOpen && 'main-with-sidebar'">` → inner `.main w-full pl-4 lt-md:pl-0`. Content (all rows: bold label + `i-carbon:chevron-right` icon + links; `[&_a:hover]:underline`, `lt-sm:text-sm`):
- **状态**: `监控` → `https://uptime.animes.garden/status/animegarden`; then `数据更新于 {formatChinaTime(timestamp)}` or (when timestamp query failed) `服务器错误` (while loading: empty). Timestamp from route loader data or `timestampQueryOptions()` (serverFn → `fetchStatus`, 10 s timeout; `staleTime` 1 min).
- **源站**: `動漫花園` (https://share.dmhy.org/), `蜜柑计划` (https://mikanani.me/), `萌番组` (https://bangumi.moe/), `ANi` (https://open.ani.rip/).
- **关于**: `GitHub` (https://github.com/yjl9903/AnimeGarden), `AnimeSpace 计划` (https://docs.animes.garden).
- **开放**: `Agent Skills` (GitHub readme #使用-skills), `MCP` (GitHub readme #使用-mcp), `API 文档` (router Link `/docs/api`).
- **更多**: `Telegram` (https://t.me/animegarden_dev), `Channel` (https://t.me/animegarden_channel), `问题反馈` (GitHub issues), `帮助文档` (https://docs.animes.garden/animegarden/search).
- Bottom row `flex justify-between items-center mt-8`: `© 2022 <a href="https://github.com/animegarden">Anime Space</a>.` `|` RSS link (`resolvedFeedURL`, tracks `feed.open`) `|` `站点地图` (link `/sitemap-index.xml`); right side: `ThemeToggle`.
- All external links track `footer.link.click {section, label, href}`. `resolvedFeedURL = feedURL ?? getFeedURL()`.

---

## 13. Theme system & visual design language

### 13.1 Modes & persistence (`stores/theme.ts`)
- `ThemeMode = 'light' | 'system' | 'dark'`; persisted as **JSON** in localStorage key `animegarden:theme-mode` (e.g. `"dark"`, `"system"`; default `"system"`).
- `currentThemeStore` = `mode === 'system' ? getSystemTheme() : mode` where `getSystemTheme()` = `window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'` (SSR: `'light'`).
- `Layout` applies `document.documentElement.classList.toggle('dark', theme === 'dark')`; the inline `global.ts` script applies `light`/`dark` class to `<html>` on first paint from the same key (FOUC avoidance).

### 13.2 ThemeToggle (`layouts/ThemeToggle.tsx`)
- `<ClientOnly>` pill: `relative bg-gray-200 dark:bg-dark-800 rounded-full p-1 flex items-center` containing three 20×20 rounded-full icon buttons (`w-5 h-5 rounded-full flex items-center justify-center transition-all duration-200 cursor-pointer`), active one gets `bg-white dark:bg-transparent|dark:bg-dark-100 shadow-sm transform scale-105`, inactive `hover:bg-gray-100 dark:hover:bg-dark-300`; icons 12×12 `text-gray-800 dark:text-light-50`:
  1. **light** (sun: circle + 8 rays) → `handleModeChange('light')`
  2. **system** (monitor: rect + stand, with `mx-1`) → `handleModeChange('system')`
  3. **dark** (moon) → `handleModeChange('dark')`
- Click → `themeModeStore.setState(mode)` + `track('theme.switch', { mode })`.

### 13.3 Fonts
- `--font-sans`: **IBM Plex Sans + Noto Sans Simplified Chinese** (loaded from `https://fonts.bunny.net/css?family=ibm-plex-sans|noto-sans-simplified-chinese|input-mono&display=swap`); monospace: **Input Mono**; display/heading: **Quicksand** (`font-quicksand`) from `https://api.fontshare.com/v2/css?f[]=quicksand@1,300,400,500,600,700&display=swap`. Both stylesheets load with `media="print"` and swap to `all` on load (WebFontLoadScript). Used for the logo (🌸), hero title, sidebar trigger/header, pagination? (no) — specifically logo, hero, season emoji, sidebar.

### 13.4 Design tokens & colors
- Hero/background brand tint: `.bg-hero` = `#fef8f7` (light) / `#090201` (dark) — a warm off-white / near-black; hero banner bottom border `rgb(187 187 187 / 20%)`.
- Link colors: `.text-link` = `text-sky-700 hover:text-sky-500`; `.text-link-active` = `hover:text-sky-600`; `.text-link-secondary` = `text-sky-700/60 hover:text-sky-500/60`; `.text-link-secondary-hover-base` = `hover:text-sky-700/60`; `.text-link-secondary-hover-active` = `hover:text-sky-500/60`.
- Neutral ramp: Tailwind neutral scale via shortcuts `text-base-50..900` (`text-neutral-{n} dark:text-light-{n}`); cards `bg-zinc-50 dark:bg-zinc-800`; table hover/zebra via gray/zinc utilities; sidebar `bg-layer-on`, `bg-layer-muted`, `bg-layer-subtle-overlay`, `bg-layer-mask`, `bg-layer-subtle` (semantic layer tokens from @onekuma/shadcn presets).
- Brand accents: **RSS orange** `#ee802f` → hover `#ff7800`; **pink-600** current pagination page; **red-600 / #ff0000 / green-600 / purple-600 / blue-600 / #ffa500 / #0eb9e7 / #a52a2a** for type colors (§13.6); progress bar gradient `#fb923c → #fdba74`; `shadow-box` = `0 2px 3px rgb(10 10 10 / 10%), 0 0 0 1px rgb(10 10 10 / 10%)`; `drop-shadow`/`drop-md` used on cards.
- Spacing/layout: `.main` = `lg:max-w-[calc(100vw-232px)] lg:w-[80vw] md:w-[46rem] lt-md:w-[95vw]`; `.main-with-sidebar` = `lg:pl-[200px] transition-all`; page content wrapper `w-full pt-13 pb-24`; hero `h-[300px]` (`--hero-height`), header `h-[66px]` (`--nav-height`), search top `128px` (`--search-top`).
- Smooth scrolling when `prefers-reduced-motion: no-preference`; `html { overflow-y: scroll; scrollbar-gutter: stable; }`.

### 13.5 Visual language summary
Warm paper-like hero (off-white blush) with a soft emoji brand; clean white/dark cards with subtle drop shadows; sky-blue links; orange RSS affordances; colorful type badges; Quicksand for brand/headings; generous vertical rhythm (`pt-13 pb-24`), Chinese-first typography; fixed translucent header on scroll (backdrop blur `saturate(180%) blur(20px)` hero placeholder); the search bar doubles as the command palette; a bookmark sidebar for collections.

### 13.6 Icons & type colors
- `DisplayTypeColor`: 动画 `text-red-600`; 合集 `text-[#ff0000]`; 漫画 `text-green-600`; 音乐 `text-purple-600`; 日剧 `text-blue-600`; RAW `text-[#ffa500]`; 游戏 `text-[#0eb9e7]`; 特摄 `text-[#a52a2a]`; 其他 `text-base-800`.
- `DisplayTypeIcon` (inline SVG): 动画 `SolarTvLinear` (TV), 合集 `SolarFolderWithFilesOutline`, 漫画 `SolarNotebookMinimalisticLinear`, 音乐 `SolarMusicNote2Outline`, 日剧 `SolarVideocameraRecordOutline`, RAW `SolarFileLinear`, 游戏 `SolarGamepadBroken`, 特摄 `SolarVideocameraRecordOutline`, 其他 `SolarDocumentTextOutline`; plus `CarbonTypes` (chevrons) for the table header.
- Iconify class names used throughout (unplugin-icons): `i-carbon-*`, `i-mdi-*`, `i-ant-design-*`, `i-fluent-*`, `i-mage-*`, `i-proicons-*`, `i-solar-*`, `i-mingcute-*`, `i-custom-*` etc. In Vue reimplementation, either ship the same SVG set or use an icon library with identical glyphs.

### 13.7 shadcn/ui components (available & used)
`components.json`: shadcn "default" style, baseColor slate, CSS variables. Components in `components/ui/`: `button` (variants default/destructive/outline/secondary/ghost/link; sizes default `h-10 px-4`, sm `h-9 px-3`, lg `h-11 px-8`, icon `h-8 w-8 rounded-full`), `card` (Card/Header/Title/Description/Content/Footer), `dialog`, `dropdown-menu` (incl. radio/checkbox/separator/label/sub), `popover`, `scroll-area`, `skeleton`, `sonner` (Toaster wrapper, theme `'light'`), `tooltip`. Also `Dropdown` (CSS-hover custom dropdown used by the header nav), `Help` (TorrentTooltip/SearchTooltip), `Icons`, `Resources` (table/tag/pagination). `cn()` = plain `clsx` (tailwind-merge disabled).

### 13.8 Static constants (`utils/constants.ts`)
- `PRESET_DISPLAY_NAME = { bangumi: '番剧' }`.
- `types = ['动画','合集','音乐','日剧','RAW','漫画','游戏','特摄','其他']`.
- `fansubs` (46 entries, ordered by 6-month activity): ANi, LoliHouse, 喵萌奶茶屋, Prejudice-Studio, 绿茶字幕组, 桜都字幕组, 雪飄工作室(FLsnow), 三明治摆烂组, 北宇治字幕组, 魔星字幕团, 拨雪寻春, 爱恋字幕社, SweetSub, 樱桃花字幕组&sakura-hana, 千夏字幕组, 沸班亚马制作组, 动漫国字幕组, 天月動漫&發佈組, 猎户发布组, 天使动漫论坛, 幻樱字幕组, 六四位元字幕組, 悠哈璃羽字幕社, DBD制作组, TSDM字幕組, 百冬練習組, GMTeam, 银色子弹字幕组, VCB-Studio, 夜莺家族, 亿次研同好会, 丸子家族, 风之圣殿, 云光字幕组, 黑白字幕组, 驯兽师联盟, 星空字幕组, 极影字幕社, MingYSub, H-Enc, 离谱Sub, 云歌字幕组, PorterRAWS, 豌豆字幕组, 隣天使字幕组, Kirara Fantasia.
---

## 14. SEO & machine-readable endpoints

### 14.1 `/robots.txt` (exact content)
```
User-agent: *
Disallow: /api/

Content-Signal: ai-train=yes, search=yes, ai-input=yes

Sitemap: https://animes.garden/sitemap-index.xml
```
Headers: `Cache-Control: public, max-age=3600, s-maxage=86400` (Docs), `Content-Type: text/plain; charset=utf-8`.

### 14.2 `/sitemap-index.xml`
Index of: `sitemap-0.xml`, `sitemap-calendar.xml`, `sitemap-fansubs.xml`, `sitemap-subjects.xml` + monthly `sitemap-{YYYY}-{MM}.xml` for every month from `2020-01` through the current month, **reversed** (newest first). All URLs under `https://{APP_HOST}/`. GET and HEAD both handled; ETag via `safeEtagResponse`.

### 14.3 Sitemaps (`/sitemap-{$sitemap}.xml`)
- `sitemap-0.xml` (static): `https://{APP_HOST}/`, `/anime`, `/resources/1?preset=bangumi&type=动画`, `/resources/1?type=动画|合集|音乐|日剧|RAW|漫画|游戏|特摄|其他` (9 type URLs), `/docs/api`.
- `sitemap-calendar.xml`: `/calendar/{season}` for every calendar (sorted desc).
- `sitemap-fansubs.xml`: `/resources/1?fansub={name}` for every team from API `GET /teams` (baseURL `WEB_SERVER_URL`, retry 5).
- `sitemap-subjects.xml`: `/subject/{id}` for every subject from API `GET /sitemaps/subjects`.
- `sitemap-{YYYY}-{MM}.xml`: `/detail/{provider}/{providerId}` from API `GET /sitemaps/{year}/{month}`; valid only for 2020 ≤ year ≤ current year and month ≤ current month (else 404). `lastmodDateOnly: false`.
- Unknown/other sitemap paths → 404.

### 14.4 `/openapi.json`
`Response.json(getPublicOpenApiSpec(version, license))` with Cache-Control Docs (same spec as §10).

### 14.5 `/llms.txt` (exact template)
```
# Anime Garden

> Anime Garden is a third-party mirror site for 動漫花園 and an anime torrent resources aggregation platform for 動漫花園, 蜜柑计划, 萌番组, and ANi.

## Key Facts
- Primary language: Simplified Chinese
- Main site: https://{APP_HOST}
- Public API host: https://{FEED_HOST}
- Search supports anime resources, fansub groups, subjects, resource types, keywords, and publish time filters.
- Users can create custom RSS feeds from resource search filters.
- Anime Garden provides an MCP endpoint for AI clients.
- Site pages support Markdown negotiation: send `Accept: text/markdown` to receive `Content-Type: text/markdown; charset=utf-8`; normal browser requests still receive HTML.

## Important Links
- Resource search: https://{APP_HOST}/resources
- Anime calendar: https://{APP_HOST}/anime
- API documentation: https://{APP_HOST}/docs/api
- OpenAPI schema: https://{APP_HOST}/openapi.json
- MCP server card: https://{APP_HOST}/.well-known/mcp/server-card.json
- MCP endpoint: https://{FEED_HOST}/mcp
- Sitemap: https://{APP_HOST}/sitemap-index.xml
- GitHub repository: https://github.com/yjl9903/AnimeGarden

## API Entry Points
- Search resources: https://{FEED_HOST}/resources
- Subject sitemap data: https://{FEED_HOST}/sitemaps/subjects

## Contact
- Telegram group: https://t.me/animegarden_dev
- Telegram channel: https://t.me/animegarden_channel
```

### 14.6 Well-known endpoints
- `/.well-known/api-catalog` → `{ linkset: [{ anchor: 'https://api.animes.garden/', 'service-desc': [{ href: '/openapi.json', type: 'application/openapi+json' }] }] }` with `Content-Type: application/linkset+json`, Cache-Control Docs.
- `/.well-known/mcp/server-card.json` → MCP server card: `{ $schema: 'https://static.modelcontextprotocol.io/schemas/v1/server-card.schema.json', name: 'garden.animes/animegarden', version, title: 'Anime Garden MCP', description: 'Search Anime Garden torrent resources.', websiteUrl: 'https://{APP_HOST}', repository: { source:'github', url:'https://github.com/yjl9903/AnimeGarden', subfolder:'apps/server' }, icons: [{ src:'https://animes.garden/favicon.svg', mimeType:'image/svg+xml', sizes:['any'] }], remotes: [{ type:'streamable-http', url:'https://{FEED_HOST}/mcp' }], serverInfo:{ name:'animegarden', version }, transport:{ type:'streamable-http', endpoint:'/mcp', url }, capabilities:{ tools:true, resources:true, prompts:false }, _meta:{ 'garden.animes/primitives': { tools:['search_resources'], resources:['resource_detail'], prompts:[] } } }`.
- `/.well-known/agent-skills/index.json` + `/.well-known/agent-skills/animegarden/SKILL.md` are **emitted at build time** (`writeBundle` hook) from the monorepo `skills/animegarden` folder (schema `https://schemas.agentskills.io/discovery/0.2.0/schema.json`, digest sha256).
- The prod server (`server.mjs`, Hono) additionally serves `/.well-known/*` statically from the client build and answers `GET /health` with `{status:'OK'}` (CORS) and returns 404 `no-store` for `/api/*`.

### 14.7 Markdown content negotiation (per-page Markdown)
When `Accept: text/markdown` (q≠0) on GET/HEAD: pages answered as Markdown (`Content-Type: text/markdown; charset=utf-8`, `Vary: Accept`, `x-markdown-tokens: ceil(len/4)`):
- `/` → frontmatter(HomeHead) + `# Anime Garden` + description + `RSS 订阅：https://api.animes.garden/feed.xml` + `## 最新资源` list (title link, type, size, 字幕组?, 发布者, time — `- {title} · {type} · {size} · 发布者: {p} · {time}`).
- `/anime`, `/calendar/{season}` → `# {title}动画周历` + `## 星期{一..日}` lists (`- {name} - /subject/{id}`), stable Monday–Sunday order (not the UI rotation).
- `/resources` & `/resources/{page}` → frontmatter + `## 筛选条件` (formatFilter: `预设=…；来源=…；类型=…；动画=…；字幕组=…；发布者=…；搜索=…；标题=…；包含=…；排除=…` or `全部资源`) + `当前页：{page}` + RSS link + `## 资源列表` + `## 分页` (上一页/下一页 links); page ≤ 0 → 302 to `/resources/1`.
- `/detail/{p}/{id}` → frontmatter(with image) + `# title` + metadata list (类型/来源/大小/发布者/字幕组/关联动画/发布时间/页面/磁力链接) + `## 简介` + `## 图片` + `## 磁力链接` + `## 文件列表` (name · size; `更多文件请查看 HTML 页面。` when hasMoreFiles).
- `/subject/{id}` → frontmatter + `# name` + summary + per-fansub groups `## {fansub}字幕组 最新资源` (name ends with 字幕组 unless already) + resource lists.
- `/collection/{hash}` → `# {name|收藏夹}` + `收藏夹哈希：{hash}` + per-filter `## {filter name|筛选条件 N}` + filter + resources (`这个筛选条件还有更多资源，请查看 HTML 页面。` when incomplete).
- Failures → `errorMarkdown(title, message, status)` (frontmatter + `# title\n\nmessage`), 404/502 with `Cache-Control: no-store`.

### 14.8 Per-page head summary (table)
| Route | title | description | canonical |
|---|---|---|---|
| `/` | `Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站` | `Anime Garden 動漫花園資源網镜像站, 动漫花园动画 BT 资源聚合站` | `/` |
| `/resources/{p}?{q}` | `{generateTitleFromFilter} | Anime Garden …` | `最新资源 {stringifySearchText}` | `/resources/{p}?{q}` |
| `/subject/{id}` | `{name} 最新资源 | Anime Garden …` | `{name}: {summary(120)}` or `{name} | Anime Garden …` | `/subject/{id}` |
| `/calendar/{season}` | `{title}动画周历 | Anime Garden …` | `{title}, 动画每周播出时间表, Anime Garden …` | `/calendar/{season}` |
| `/detail/{p}/{id}` | `{truncate(title,70)}` / `资源详情 | …` | description text | `/detail/{p}/{id}` |
| `/collection/{hash}` | `{name} | Anime Garden …` | `Anime Garden 资源收藏夹, …` | `/collection/{hash}` |
| `/docs/api` | `API 文档 | Anime Garden …` | `Anime Garden 动画 BT 资源开放接口文档` | `/docs/api` |
| `/iframe?{q}` | `{title} | Anime Garden 動漫花園資源網第三方镜像站` | `最新资源 {text}` | `/iframe?{q}` |
| `/about` | `关于 | Anime Garden …` | `Anime Garden 動漫花園資源網镜像站, 动漫花园动画 BT 资源聚合站` | `/about` |

All titles use the full site suffix `Anime Garden 動漫花園資源網镜像站 动漫花园动画 BT 资源聚合站` (except iframe’s `…第三方镜像站`).

---

## 15. LocalStorage / storage keys

| Key | Kind | Format | Used by |
|---|---|---|---|
| `animegarden:theme-mode` | localStorage | JSON string: `"light"` \| `"system"` \| `"dark"` (default `"system"`) | theme store, global.ts first-paint script |
| `animegarden:histories` | localStorage | JSON array of search-input strings (≤ 10) | search histories |
| `animegarden:fansubs` | localStorage | JSON array of fansub names (preferred order) | FansubsDropdown ordering; `usePreferFansub` adds visited fansubs |
| `animegarden:cur_collection_name` | localStorage | JSON string (default `"收藏夹"`) | current collection selection |
| `animegarden:collections` | **IndexedDB** db name | version 1; object stores `key-value` (key=collection name, value=Collection) and `_meta` | collection persistence |
| `script_error_reloaded` | sessionStorage | `"true"` after one script-error reload | global.ts error guard |
| `__animegardenLayoutController` | `window` global | `{ update: () => void }` | layout scroll controller (also `window.updateHeroLayout?.(y)`) |

---

## 16. Other behaviors

### 16.1 Scroll restoration & navigation
- `scrollRestoration: true` on the router; sidebar quick links use `resetScroll={false}`; `window.scrollTo` is wrapped (global.ts) to re-run hero layout after programmatic scrolls.
- Route transitions show `#animegarden-progress` (see 16.4) and the default pending component `PendingPage` = `<Layout heading={false} footer={false}/>` (hero + search + header only).

### 16.2 Loading skeletons
- No skeleton components are used in page bodies. Loading states: the top progress bar (§16.4), the cmdk inline spinner (`lds-ring` + `正在搜索 …`), footer timestamp blank while loading, and `Skeleton`/`ScrollArea` primitives exist in `ui/` but are not wired into pages.

### 16.3 Error states
- Route-level: root errorComponent renders an empty document (console `[Route Error]`); page loaders return `ok:false`/SSR statuses (500/404) that pages turn into the `Error` component (`发生错误` + optional message, red, tracks `error.render`).
- Home/resources/iframe/subject fetch failures additionally track `error.fetch-resources` with `{path, error}`.
- Resources empty state (`没有搜索到匹配的资源` + `第 1 页` / `主页`), subject empty state (`暂时未索引到相应资源` + `前往搜索`), files empty (`种子信息解析失败`).

### 16.4 Progress bar (`#animegarden-progress`)
Two fixed 3 px bars (width 20%, `border-radius: 1px`, gradient `linear-gradient(90deg, #fb923c, #fdba74)`, z-index 10000, top 0; second delayed 0.8 s) animating `translateX(0 → 500%)` in an infinite 1 s loop. Rendered when hydrated and router status is `pending`.

### 16.5 Umami analytics events (complete list)
Programmatic `track(event, payload)` + `data-umami-event` attributes (only loaded in production):
`nav.click.home{item}`, `nav.click.anime{item,group?}`, `nav.click.fansub{item}`, `nav.click.type{item}`, `theme.switch{mode}`, `collection.open-sidebar`, `collection.add`, `collection.open{hash}`, `collection.share{hash}`, `collection.open-resources{search}`, `copy.feed`, `copy.magnet-links`, `copy.json`, `copy.fetch{language:curl|javascript|python}`, `copy.iframe`, `error.render{path,error}`, `error.fetch-resources{path,error}`, `search.trigger{text,source:button|command|history|result-more}`, `search.history.click{text}`, `search.history.delete{action:clear|remove,text?,count?}`, `search.suggestion.click{text,subjectId}`, `search.result.click{text,resource}`, `resources.detail.click{resource,type}`, `resources.filter.click{filterType:type|fansub|publisher,value}`, `pikpak{resource,source}` (data attrs on PikPak links), `feed.open{href}`, `footer.link.click{section,label,href}`, `anime.calendar.click{subjectId,title,weekday}`, `subject.fallback-search{subject}`, `download{resource,source}`. Umami script: `https://umami.animes.garden` with website id `bcff225d-6590-498e-9b39-3a5fc5c2b4d1`.

### 16.6 Keyboard shortcuts
- `s`, `/`, `Ctrl/Cmd+K` → focus search input (`#animegarden-search input`).
- Search input: `Home`/`End` move caret (and stop propagation); `Enter` (button) submits; `Escape` prevented (no close behavior — not a modal).
- Inline rename: `Enter` commits collection item rename.

### 16.7 Hero/header scroll behavior (global.ts — must be replicated in Vue)
- `updateHero()` on scroll (rAF-throttled), resize, body resize (ResizeObserver), DOMContentLoaded, and after route changes:
  - `y >= SearchTop (128)` → `#hero-search.fix-hero` (fixed, top 0).
  - `y >= HeroHeight - NavHeight (234)` → `#hero-placeholder.fix-hero` (fixed full-width blurred panel: `bg-[#fef8f7]/80 dark:bg-[#090201]/80`, `backdrop-filter: saturate(180%) blur(20px)`, border-bottom, transition-all), `header.fix-hero`, `sidebar-root.fix-hero`; otherwise `--sidebar-pt = (300 - y)px` inline.
  - Nav collision (viewport < 1440 px): header targets with `data-nav-collision-target` (anime, fansubs, types) overlap hero elements marked `[data-header-collision-source]` (search box, hero title); the first colliding target (leftmost by boundary) gets `body.nav-collision-from-{id}` which hides it **and all later targets** via an injected `<style id="nav-collision-style">` (`display: none`), i.e. right-side items are hidden progressively.
- Script-error guard: any window `error` from a `<script>` triggers one `location.reload()` guarded by sessionStorage `script_error_reloaded`.

### 16.8 Responsive behavior
- Breakpoints (UnoCSS): `lt-sm` (<640), `sm`, `lt-md` (<768), `md`, `lt-lg` (<1024), `lg`, `xl`, `lt-xl` (<1280). Key behaviors: RSS button hidden < md; sidebar 200 px (lg+) vs 300 px (<lg); `.main` 80vw/46rem/95vw; hero search width 640/600/500/`calc(100vw-116px)`; anime TOC hidden <768 px; grid min card 132 px (112 px mobile); footer stacks spacing with `lt-sm`; header logo `pl3 lt-sm:pl1`.
- Touch: tooltips open on touchstart (TorrentTooltip); magnet links open via `window.location.assign` after tracking.

### 16.9 PWA
- Icons shipped in `/public`: `favicon.ico`, `favicon.svg`, `pwa-64x64.png`, `pwa-192x192.png`, `pwa-512x512.png`, `maskable-icon-512x512.png`, `apple-touch-icon-180x180.png`, `twitter.jpg` (social card). The root head registers favicon/apple-touch/mask-icon links and `theme-color`.
- `vite-plugin-pwa` is a devDependency and `VIRTUAL_PWA_MODULE=true` is set, but **no `virtual:pwa-register`/manifest wiring exists in the source** — there is no generated `manifest.json`/service-worker registration in this web app. Reimplementation does not need a manifest to match current behavior.
- Other `/public` files: `261543988e684693a0dcbd4b9dad2857.txt` (Tencent verification token), `BingSiteAuth.xml` (`CD7F2B2E8843152DAFC93C13891922CF`), `google79f036f71a58993a.html` (`google-site-verification: google79f036f71a58993a.html`).

### 16.10 Toasts (sonner)
Default config: `position: 'top-right'`, `dismissible: true`, `duration: 3000`, `closeButton: true`. The `Toaster` is `theme={'light'}` with shadcn class overrides; toast markup CSS in `styles/sonner.css` (also defines rich color variants for success/info/warning/error, mobile layout <600 px, swipe-out, reduced-motion support).

### 16.11 Server-side caching (Cache-Control values)
`ResponseCacheControl`: List `public, max-age=30, s-maxage=60`; Detail/Subject/Calendar/Docs `public, max-age=3600, s-maxage=86400`; Error `no-store`. Query stale times: List 60 s, Detail/Subject/Calendar 1 h.

### 16.12 Env/build facts for reimplementation
- `~build/env` provides `APP_HOST` (`animes.garden`), `FEED_HOST` (`api.animes.garden`), `WEB_SERVER_URL` (`https://api.animes.garden/`), `KEEPSHARE_ID` (`gv78k1oi`); `~build/package` provides `version` (`0.5.4`) and `license` (`MIT`).
- Path aliases: `@/*` and `~/*` → `apps/web/src/*`; `@animegarden/client` → `packages/client/src/index.ts`.
- Build: Vite + TanStack Start plugin + React plugin + UnoCSS + unplugin-icons (jsx) + tsconfig paths + Inline + Info (env) + Analytics (umami) + build-time agent-skills emitter; target `es2022`; Hono Node server (`server.mjs`, port 3000) serving `/health`, static assets, `/.well-known/*` statically, then the SSR entry; `/api/*` returns 404 no-store.

---

## Appendix A — Data shapes (for API-typed Vue stores)

```ts
type Resource = {
  provider: 'dmhy' | 'mikan' | 'moe' | 'ani';
  providerId: string;
  type: string;              // one of the 9 types
  title: string;
  size: number;              // bytes
  magnet: string;            // magnet:?xt=urn:btih:... (without tracker)
  tracker?: string;          // &tr=... suffix (only when tracker:true requested)
  createdAt: string;         // ISO
  href: string;              // original page URL
  subjectId?: number;
  publisher: { id: number; name: string; avatar?: string };
  fansub?: { id: number; name: string; avatar?: string };
};
type ResourcesResponse = { ok: boolean; resources: Resource[]; pagination?: { page: number; pageSize: number; complete?: boolean }; filter?: ResolvedFilterOptions; timestamp: Date; error?: SerializedError };
type DetailResponse = { ok: boolean; resource?: Resource; detail?: { magnets: {name:string;url:string}[]; files: {name:string;size:string}[]; hasMoreFiles: boolean; description: string }; timestamp: Date; description?: DescriptionResult };
type CollectionResponse = { ok: boolean; hash: string; name: string; authorization?: string; filters: Filter[]; results: { filter: ResolvedFilterOptions; resources: Resource[]; complete: boolean }[]; timestamp: Date };
type CalendarResponse = { ok: boolean; calendar: WebBgmSubject[][]; season?: string };
type CalendarsResponse = { ok: boolean; calendars: { season: string; is_active: boolean }[] };
type SubjectResponse = { ok: boolean; subject?: WebBgmSubject };
type TimestampResponse = { ok: boolean; timestamp: Date };
```

## Appendix B — Source file map (original React files → spec sections)

| File | Section |
|---|---|
| `router.tsx`, `routeTree.gen.ts`, `client.tsx`, `start.ts`, `app-env.d.ts` | §0 |
| `routes/__root.tsx`, `routes/index.tsx` | §0.4, §1.1 |
| `layouts/Layout.tsx`, `Header.tsx`, `Footer.tsx`, `Sidebar/*`, `Search/*`, `ThemeToggle.tsx`, `Loading.tsx`, `global.ts` | §2, §3, §4, §13.2, §16.1/16.4/16.7, Footer in §12 |
| `pages/_index`, `pages/resources.($page)`, `pages/subject.$subject.($page)`, `pages/anime`, `pages/detail.*`, `pages/collection.$hash`, `pages/docs.api`, `pages/iframe`, `pages/about`, `pages/PendingPage` | §1, §5–§12 |
| `components/Resources/*`, `components/Dropdown`, `components/Help`, `components/Icons`, `components/ui/*` | §5.2/5.3, §2, §13.7 |
| `stores/*` | §3, §4, §9, §13.1, §15 |
| `query/*` | §0.3, §1, §16.11, Appendix A |
| `hooks/*`, `utils/*` | §4.5–4.7, §5.1, §6, §7, §8, §13, §16 |
| `styles/*`, `uno.config.ts`, `vite.config.ts`, `package.json`, `components.json` | §13, §16.12 |
| `routes/{robots,sitemap-*,openapi,llms,well-known,$}.tsx`, `sitemap/index.server.ts`, `markdown/*`, `utils/server/meta.ts` | §14 |
| `public/*` | §16.9 |
