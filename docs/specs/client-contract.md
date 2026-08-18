# AnimeGarden API Client Contract — Complete Specification

> **Purpose**: This spec drives a Go + Vue reimplementation of the AnimeGarden public API with **identical behavior**.
> **Source**: `/tmp/animegarden-orig` (packages/client, apps/web, apps/server, packages/shared, examples, docs).
> **Package**: `@animegarden/client` **version 0.5.4** (from `packages/client/package.json`), license AGPL-3.0.
> **Production hosts**: Web site `https://animes.garden` — public API `https://api.animes.garden` (default `FEED_HOST`) — backend `animegarden-server-production.flycast` (internal).
> **Client default base URL**: `https://api.animes.garden/` (`DefaultBaseURL` in `packages/client/src/constants.ts`).

---

## 1. Full API request/response contract

### 1.0 Constants (client, `packages/client/src/constants.ts`)

```ts
export { version } from '../package.json';          // 0.5.4
export const DefaultPageSize = 100;                 // default pageSize
export const MaxRequestPageSize = 1000;             // max pageSize the CLIENT will ever request
export const DefaultBaseURL = 'https://api.animes.garden/';
export const SupportProviders = ['dmhy', 'moe', 'mikan', 'ani'] as const;
export const SupportPresets = ['bangumi'] as const;
```

Note the provider array order: **dmhy, moe, mikan, ani** (not alphabetical). `ProviderType = 'dmhy' | 'moe' | 'mikan' | 'ani'`, `PresetType = 'bangumi'`.

### 1.1 Types (`packages/client/src/types.ts`)

```ts
export interface Resource<T extends { tracker?: boolean; metadata?: boolean } = {}> {
  id: number;
  provider: string;
  providerId: string;
  title: string;
  href: string;           // relative provider resource id/href — see transformResourceHref (§1.10)
  type: string;           // e.g. "动画", "合集", ...
  magnet: string;         // magnet URI WITHOUT tracker suffix
  tracker: T['tracker'] extends true ? string : (string | null | undefined); // "&tr=..."
  size: number;           // bytes
  fansub?: { id: number; name: string; avatar?: string | null } | null | undefined;
  publisher: { id: number; name: string; avatar?: string | null | undefined };
  subjectId?: number | null | undefined;           // Bangumi subject id
  createdAt: Date;        // (string on the wire, parsed to Date by the client)
  fetchedAt: Date;        // (string on the wire, parsed to Date by the client)
  metadata?: { anipar?: ParseResult } | null | undefined; // only when metadata=true
}

export interface ResourceDetail {
  description: string;
  files: Array<{ name: string; size: string }>;   // size is a STRING (e.g. "1.2 GB")
  magnets: Array<{ name: string; url: string }>;
  hasMoreFiles: boolean;
}

export type FetchOptions = {
  fetch?: (request: RequestInfo, init?: RequestInit) => Promise<Response>;
  baseURL?: string;                       // default 'https://api.animes.garden/'
  retry?: number;                         // default 0
  timeout?: number;                       // ms; 0/undefined = no timeout
  signal?: AbortSignal;
  headers?: Record<string, string | ReadonlyArray<string>>;
  hooks?: {
    prefetch?: (path: string, init: RequestInit) => Promise<void> | void;
    postfetch?: (path: string, init: RequestInit, response: Response) => Promise<void> | void;
    timeout?: (signal?: AbortSignal) => Promise<void> | void; // default sleeps 100ms
  };
};

export type PaginationOptions = { page?: number /* default 1 */; pageSize?: number /* default 100 */ };
export type ResolvedPaginationOptions = Required<PaginationOptions>;
export type PresetOptions = { preset?: PresetType };
// FilterOptions — see §1.3 and §2
```

Body of the union-constrained `FilterOptions` (exactly one of each pair may be set):

```ts
export type FilterOptions = {
  provider?: string;
  duplicate?: boolean;        // default false
  after?: Date;               // createdAt >= after
  before?: Date;              // createdAt <= before
  search?: string | string[]; // full-text (fuzzy) search mode
  include?: string | string[];// title-contains mode ("include at least one of titles")
  keywords?: string | string[]; // "include all the keywords"
  exclude?: string | string[];  // "exclude keywords"
} & ( { type?: string; types?: null | undefined } | { type?: null | undefined; types?: string[] } )
  & { subject?: number; subjects?: number[] }
  & ( { fansub?: string; fansubs?: null | undefined } | { fansub?: null | undefined; fansubs?: string[] } )
  & ( { publisher?: string; publishers?: null | undefined } | { publisher?: null | undefined; publishers?: string[] } );

export type ResolvedFilterOptions = {
  preset?: PresetType;
  provider?: ProviderType;
  duplicate?: boolean;
  types?: string[];
  after?: Date;
  before?: Date;
  fansubs?: string[];
  publishers?: string[];
  subjects?: number[];
  search?: string[];
  include?: string[];
  keywords?: string[];
  exclude?: string[];
};
```

### 1.2 `fetchAPI` — the low-level transport (`packages/client/src/api/base.ts`)

Signature: `fetchAPI<T>(path: string, init?: RequestInit, options?: FetchOptions): Promise<FetchAPIResult<T>>`

Behavior — *exact*:

1. **URL construction**: `new URL(path.replace(/^\/+/g, ''), baseURL)`. Leading `/` chars are stripped; the path is resolved against `baseURL` (default `https://api.animes.garden/`). So `fetchAPI('resources?page=1', ...)` → `https://api.animes.garden/resources?page=1`. Query strings are simply part of the path string.
2. **Headers**: a fresh `Headers(options.headers)` is created; then:
   - `x-trace-id` is **always set** to `crypto.randomUUID()` (a new UUID per request).
   - `user-agent` is set to `animegarden@${version}` (`animegarden@0.5.4`) **only if not already present**.
3. **Timeout/signal**: if `options.timeout > 0` → `AbortSignal.any([AbortSignal.timeout(timeout), options.signal])` when a signal is given, else `AbortSignal.timeout(timeout)`. Otherwise the raw `options.signal`.
4. **Hooks**: `options.hooks.prefetch(url, payload)` awaited before fetch; `options.hooks.postfetch(url, payload, resp)` awaited after a response arrives.
5. **Success (`resp.ok`)**: `await resp.json()` returns `T`. `x-response-timestamp` header is parsed first (via `parseTimestamp`), else a top-level `timestamp` field in the JSON body. The parsed `Date` (or `undefined`) is merged into the result object as a `timestamp` property — i.e. `Object.assign(data, { timestamp })`. Every successful API response therefore carries a `timestamp: Date` at the client level. `parseTimestamp` handles: `Date` (NaN → undefined), `number` (epoch ms), numeric string (`/^-?\d+$/` → `new Date(Number(...))`), and any ISO date string; empty → undefined.
6. **HTTP errors**: non-ok response → if `resp.status === 429`, `await sleep(16 * 1000, { signal })` (16 s) **then** throw `AnimeGardenError.fromResponse(`${status} ${statusText} ${url}`, resp)` with the JSON/text body attached. ANY other status throws immediately.
7. **Network errors**: `AbortError` re-thrown as-is; `TimeoutError` → await `hooks.timeout(signal)` (default: sleep 100 ms) then rethrow; other errors become `AnimeGardenError.fromOriginalError(message, error)`. Note: after a `TimeoutError` the underlying fetch's signal is already aborted, so the sleep does not interfere.
8. **Retry**: wrapped in `retryFn(fn, { count: options.retry, signal: options.signal })` — retry count defaults to 0. Retries re-run the whole block (including hooks). `retryFn` lives in `packages/shared/src/retry.ts`.

### 1.3 `fetchResources` (`packages/client/src/api/resources.ts`)

Signature:
```ts
export async function fetchResources<T extends FetchResourcesOptions = FetchResourcesOptions>(
  options: T = {} as T
): Promise<FetchResourcesResult<T>>
```

`FetchResourcesOptions = PaginationOptions & FilterOptions & PresetOptions & FetchOptions & {
  count?: number;        // -1 = fetch ALL matched resources
  tracker?: boolean;     // sets tracker=true
  metadata?: boolean;    // sets metadata=true
  progress?: (delta: Resource[], payload: { url: string; searchParams: URLSearchParams; page: number }) => void | Promise<void>;
}`

Result union:
```ts
| { ok: true;
    resources: Resource<T>[];
    pagination: PaginationResult;          // { page, pageSize, complete } — REQUIRED on ok
    filter: ResolvedFilterOptions;         // echo of the resolved filter — REQUIRED on ok
    timestamp: Date;                       // REQUIRED on ok
    error: Error | any | undefined; }
| { ok: false;
    resources: Resource<T>[];              // partial results accumulated so far
    pagination: PaginationResult | undefined;
    filter: ResolvedFilterOptions | undefined;
    timestamp: Date | undefined;
    error: Error | any | undefined; }
```

**Request serialization (exact)**:
1. `searchParams = stringifyURLSearch(options)` (see §2) — encodes filters/pagination but NOT `tracker`/`metadata`.
2. `if (options.tracker) searchParams.set('tracker', 'true')` — only when truthy; no `tracker=false` is ever sent.
3. `if (options.metadata) searchParams.set('metadata', 'true')` — same.
4. `searchParams.sort()` already happened inside stringifyURLSearch — note that `tracker`/`metadata` are appended AFTER the sort, so they always appear **last** in the query string.

**Mode selection (exact ``count``/`once` semantics)**:
```ts
const { once, count } =
  options.count !== undefined && options.count !== null
    ? { count: options.count < 0 ? Number.MAX_SAFE_INTEGER : options.count, once: false }  // multi-page
    : { count: options.pageSize ?? DefaultPageSize, once: true };                          // single page
const startPage = options.page ?? 1;
if (!once) searchParams.set('pageSize', '' + MaxRequestPageSize);   // pageSize=1000 for multi-page mode
```
- `count` **undefined/null** → **once mode**: fetch exactly one page; `count` = `pageSize ?? 100`.
- `count` **≥ 0** → **multi-page mode**: fetch pages until `map.size >= count` or the server says `complete`; the wire `pageSize` is forced to **1000** (max). Note: `count` < requested-per-page just stops after one page.
- `count` **< 0** (i.e. `-1`) → fetch **all** matched resources (`Number.MAX_SAFE_INTEGER`).

**Pagination loop (exact)**:
```ts
for (let page = startPage; map.size < count && !pagination?.complete; page++) {
  // 1. if options.signal?.aborted -> aborted = true; break
  // 2. resp = await fetchPage(page, searchParams, options)  — sets searchParams.set('page', '' + page)
  //    - if !resp -> aborted = true; break   (fetchPage threw/returned falsy)
  //    - first successful response snapshots timestamp / pagination / filter
  //    - if resp.resources.length === 0 -> break   ("no new resources" stop)
  // 3. dedupe by r.href into a Map (later pages' duplicates skipped)
  //    delta = newly-seen resources; await options.progress?.(delta, { url: searchParams.toString(), searchParams, page })
  // 4. AbortError/TimeoutError -> aborted; error; break | other errors -> failed; error; break
  if (once) break;
}
```
- **Page is appended to the QUERY STRING** (`searchParams.set('page', '...')`), not to the path. `fetchPage` calls `fetchAPI('resources?' + searchParams.toString(), ...)`.
- Multi-page mode stops as soon as `pagination.complete === true` (server says no more data).
- `progress` receives only the *new* (dedup'd) resources of that page plus `{ url, searchParams, page }` (the same shared `URLSearchParams` object mutated across pages).

**Date fix-up**: on every page, `createdAt`/`fetchedAt` of each resource are converted `new Date(...)`, and `filter.before`/`filter.after` likewise — so the returned `resources`/`filter` carry real `Date` objects.

**Validation**: if a page response lacks `r.timestamp` → throws `AnimeGardenError.fromOriginalError('Invalid response /resources?...', r)` and (unless abort/timeout) the whole call returns `ok: false`.

**Final result**:
- `ok: true` when neither aborted nor failed: resources dedup'd by `href`, then **sorted by `createdAt` desc** (`rhs.createdAt.getTime() - lhs.createdAt.getTime()`).
- `ok: false` returns partial resources (also dedup'd + sorted), plus pagination/filter/timestamp if any page succeeded, and the captured error.

**Wire request example** (multi-page, count=250, tracker):
`https://api.animes.garden/resources?after=1751155200000&page=1&pageSize=1000&tracker=true` (page increments per iteration; note `pageSize=1000` forced).

### 1.4 `fetchResourceDetail` (`packages/client/src/api/detail.ts`)

```ts
export async function fetchResourceDetail(
  provider: ProviderType, href: string, options: FetchResourceDetailOptions = {}
): Promise<FetchResourceDetailResult>
```
- URL: **`detail/${provider}/${href}`** — `href` is **not** URL-encoded (raw interpolation into the path).
- `FetchResourceDetailResult = { ok: boolean; resource: Resource<{tracker:true;metadata:true}> | undefined; detail: ResourceDetail | undefined; timestamp: Date | undefined }`.
- `ok` is true iff the response exists **and** `resp.resource !== undefined` **and** `resp.timestamp !== undefined`.
- Any fetch error is swallowed (`.catch(() => undefined)`) → returns `{ok:false, resource:undefined, detail:undefined, timestamp:undefined}`.
- The wire response is `{ status: 'OK'|'ERROR', resource?, detail?, isDeleted?, duplicatedId?, timestamp? }` (see server §1.12).

### 1.5 `fetchResourceDetailByInfoHash` (same file)

```ts
export async function fetchResourceDetailByInfoHash(infoHash: string, options: FetchResourceDetailOptions = {})
```
- URL: **`detail/infohash/${encodeURIComponent(hash)}`** — here the hash IS `encodeURIComponent`-encoded (unlike provider detail).
- `hash = infoHash.trim()`; empty string → immediate `{ok:false,...}` without any request.
- Same `ok` semantics and swallowing as §1.4.

### 1.6 `generateCollection` (`packages/client/src/api/collection.ts`)

```ts
export async function generateCollection(
  collection: Collection<true>, options: FetchOptions = {}
): Promise<CollectionResult<true, false> | undefined>
```
- `Collection<true>` requires `hash?`, `name`, `authorization`, `filters: CollectionFilter<true,false>[]` where each filter has `name` and `searchParams` (see §2.5) plus resolved filter fields.
- **Request**: `PUT /collection` with body
  ```json
  {
    "name": "...",
    "authorization": "...",
    "filters": [ { "name": "", "searchParams": "?after=...&fansub=..." } ]
  }
  ```
  Each filter is spread as `{ ...f, resources: undefined, complete: undefined }` — i.e. `resources` and `complete` keys are set to `undefined` (and thus dropped by `JSON.stringify`). The whole object is `JSON.stringify({ ...collection, filters: [...] })` — no explicit content-type header is set (fetch defaults to `text/plain;charset=UTF-8`; the server only reads the JSON body).
- **Response**: `{ status: 'OK', id: number, hash: string, createdAt: string }` (note: no `name`/`filters` echo on the wire).
- **Result**: on success returns `{ ok: true, ...collection, hash: resp.hash, createdAt: resp.createdAt, timestamp: resp.timestamp }` — `timestamp` is the client-merged response timestamp (Date).
- Errors swallowed → returns `undefined`.

### 1.7 `fetchCollection` (same file)

```ts
export async function fetchCollection(
  hash: string, options: FetchOptions = {}
): Promise<CollectionResourcesResult<true, false, { tracker: true }> | undefined>
```
- **Request**: `GET /collection/${hash}` (hash raw, not encoded).
- **Result** on success: `{ ok: true, ...resp, timestamp }` where the wire body is
  ```json
  {
    "status": "OK",
    "hash": "...", "name": "...", "createdAt": "ISO",
    "results": [
      { "resources": [ Resource... ], "complete": boolean, "filter": ResolvedFilterOptions }
    ],
    "timestamp": "ISO"
  }
  ```
- If `resp.timestamp` is missing → returns `undefined` (treated as failure). Errors swallowed → `undefined`.
- `updatedAt` is present on the server response but not typed in the client result (extra property passes through spread).

### 1.8 `fetchStatus` (`packages/client/src/api/status.ts`)

```ts
export async function fetchStatus(options: FetchOptions = {})
```
- **Request**: `GET /` (root path).
- **Response type**:
  ```ts
  {
    timestamp?: Date;
    providers: Record<ProviderType, { id: ProviderType; name: string; refreshedAt: string; isActive: boolean }>;
  }
  ```
- **Result**: `{ ok: true, timestamp, providers }` on success; `{ ok: false, timestamp: undefined, providers: undefined }` on any error (swallowed).
- Server also returns `message` on the wire (`status`, `message`, `timestamp`, `providers`) — see §8.

### 1.9 `generateCollection` hash algorithm (`packages/client/src/collections.ts`)

`hashCollection(collection)` (used server-side to derive the collection hash; the client exports it too):
1. Copy filters, sort them by `searchParams.localeCompare(rhs.searchParams)` (lexicographic).
2. For each filter: delete `name`, `searchParams`, `resources`, `complete`; keep the rest (`preset`, `provider`, `duplicate`, `types`, `after`, `before`, `fansubs`, `publishers`, `subjects`, `search`, `include`, `keywords`, `exclude`, and any passthrough keys).
3. `body = serialize(filters)` where `serialize` is **`ohash`** (https://github.com/unjs/ohash) — deterministic object serialization.
4. `hashHex = SHA-1(UTF-8(body))` hex digest (40 lowercase hex chars, e.g. `RUi8lloFMts8DjeI97lIGfFE5zz6MM9a-Qek7xaEY78` — note that example is not SHA-1 hex; actual client/server produce 40-hex SHA-1).

`parseCollection(collection)` validates with a zod schema: `hash?`, `name` coerced to string (default `''`), `authorization` required string, `filters` array length **1..50**, each filter `{ name (default ''), searchParams: string, ...ResolvedFilterOptions fields }` with `.passthrough()`.

### 1.10 href helpers (`packages/client/src/href.ts`)

```ts
transformResourceHref(provider, href?)  // provider-specific resource URL
  dmhy  -> `https://share.dmhy.org/topics/view/${href}`
  mikan -> `https://mikanani.me/Home/Episode/${href}`
  moe   -> `https://bangumi.moe/torrent/${href}`
  ani   -> href (as-is)
  else  -> undefined

transformPublisherHref(provider, publisherId?)
  dmhy  -> `https://share.dmhy.org/topics/list/user_id/${publisherId}`
  mikan -> `https://mikanani.me/Home/PublishGroup/${publisherId}`
  moe   -> `https://bangumi.moe/tag/${publisherId}`
  ani   -> 'https://aniopen.an-i.workers.dev/'
  else  -> undefined

transformFansubHref(provider, fansubId?)   // identical shape to publisher
```

### 1.11 `makeResourcesFilter` (`packages/client/src/filter.ts`) — client-side in-memory filter

`makeResourcesFilter(filter: Omit<ResolvedFilterOptions,'duplicate'>)` returns `(res: Resource) => boolean`. Semantics (mirrors the server but is client-side):
- `provider` equality;
- title condition (normalized via `normalizeTitle(title).toLowerCase()`, `simptrad` full→half + trad→simplified, punctuation normalized):
  - `include` — **ANY** term must be a substring (OR);
  - `keywords` — **ALL** terms must be substrings (AND);
  - `exclude` — **NO** term may be a substring (AND of negatives);
- `subjects` — `r.subjectId` in list (OR);
- `search` — **no-op** (TODO in client);
- `publishers`/`fansubs` — `(publisher.name in publishers) OR (fansub?.name in fansubs)` — i.e. the two groups are **OR** of each other;
- `types` — `r.type` in list (OR);
- `before` — `createdAt <= before`; `after` — `createdAt >= after`.
All conditions ANDed. `normalizeTitle` lives in `packages/client/src/utils.ts` (same code as `packages/shared/src/title.ts`).

### 1.12 Server route behaviors (what the Go backend must reproduce)

**`GET|POST /resources` and `/resources/` and `/resources/:page`** (all methods! the Hono route uses `.all(...)`), plus **`/resources/{provider}`, `/resources/{provider}/`, `/resources/{provider}/:page`**:
- Parses query params via `parseURLSearch(url.searchParams, await ctx.req.json().catch(() => undefined))` — the JSON body is optional; for GET there is no body, for POST the body carries `ResourcesRequest` (§6.4). Query params take **precedence** over the body (see §2.4 resolution order).
- `assertResourcesPagination`: rejects `offset + limit > 10000` (`offset = (page-1)*pageSize`) → HTTP 400 `{status:'ERROR', message:'Resources pagination is too deep. Please keep offset + limit <= 10000.'}`.
- `/resources/:page` — a page path segment acts in addition to query args; the route handler does not read it explicitly (client never uses it).
- `tracker` / `metadata` truthy check: `['true','yes','on'].includes(value.toLowerCase())`; if not enabled the corresponding fields are deleted from each resource.
- Response: `{ status: 'OK', complete: pagination.complete /*legacy*/ , resources, pagination: {page,pageSize,complete}, filter, timestamp? }` — note `complete` at top level echoes `pagination.complete`.
- `Cache-Control: public, max-age=300` (5 min) set by the route; the global middleware also sets `X-Request-Id` and `X-Response-Timestamp`.
- `pagination.complete` = `!hasMore` (server-side task logic).
- `filter` echo: `before`/`after` serialized as **ISO strings**; `fansubs`/`publishers` echoed as **display NAMES** (ids resolved to names); `subjects` as numbers.
- `href` in each resource is `transformResourceHref(provider, href) ?? ''`.

**`GET /resource/{provider}/:id` and `GET /detail/{provider}/:id`** (aliases, idempotent):
- Detail lookup memoized 1 h, `Cache-Control: public, max-age=86400`.
- Response on found resource: `{ status: 'OK', resource, detail, isDeleted, duplicatedId }`; unknown id → `{ status: 'ERROR', message: 'Unknown detail id: ${provider.name} ${path}' }` (HTTP 200 with ERROR status!).

**`GET /detail/infohash/:hash`**:
- Hash validation: 40-hex (`/^[0-9A-F]{40}$/i`) or 32 base32 (`/^[A-Z2-7]{32}$/i`); invalid → HTTP 200 `{ status:'ERROR', message:'Invalid info hash: ...', resource: undefined, detail: undefined, isDeleted: false, duplicatedId: undefined }` with `Cache-Control: no-store`.
- Not found in DB → `{ status:'ERROR', message:'Unknown detail info hash: ...', resource: undefined, detail: undefined, isDeleted: false, duplicatedId: undefined }`.
- Found → `{ status:'OK', resource, detail, isDeleted, duplicatedId }`.

**`POST|PUT /collection`**: parse with `parseCollection`; invalid → 400 `{status:'ERROR', message:'Incorrect collection format'}`; generation failure (`generateCollection` returned undefined) → 400 `{status:'ERROR', message:'Failed generating collection'}`; success → 200 `{status:'OK', id, hash, createdAt}`.

**`GET /collection/:hash`**: found → 200 `{status:'OK', hash, name, createdAt, updatedAt, filters, results}` (each result `{resources, complete, filter}` with `page:1, pageSize:1000` queries per filter); missing/hash empty → 400 `{status:'ERROR', message:'Failed querying collection result'}` (or `'Missing collection hash'`); collection memoized 300 s, max 100 entries.

**`GET /users`** → 200 `{status:'OK', users:[User...]}`; **`GET /teams`** → `{status:'OK', teams:[Team...]}`; **`GET /subjects`** → `{status:'OK', subjects:[Subject...]}`; all `Cache-Control: public, max-age=86400`.

**`GET /feed.xml` / `GET /collection/:hash/feed.xml`** → see §6.4. `isTrackerEnabled` reads **`trakcer`** (sic, typo is real): `get('trakcer') === null ? true : ['no','off','false'].includes(value) ? false : true` — default true.

**Error envelope** (global `onError`): `HTTPException` passthrough (e.g. 408 timeout); query errors (`ResourcesDeepPaginationError`→400, `ResourcesSlowQueryBusyError`→503, `ResourcesSlowQueryTimeoutError`→504) → `{status:'ERROR', message}` (XML for feed.xml: `<?xml version="1.0" encoding="UTF-8"?><error><message>...</message></error>` with `Cache-Control: no-store`); anything else → 500 `{status:'ERROR'}`.

**`/` and `/health`** → `{status:'OK', timestamp, providers}` (see §8). Server middleware: `X-Request-Id`, `X-Response-Timestamp`, JSON `charset=utf-8`, CORS `*`, pretty JSON, 60 s timeout.

---

## 2. Filter → query param serialization

### 2.1 `stringifyURLSearch` (`packages/client/src/resolver.ts`) — exact param mapping

Input: `PaginationOptions & (FilterOptions | Jsonify<FilterOptions>) & PresetOptions`.

Rules in order (all appends dedupe via `new Set(...)`):

| Option field(s) | Output param(s) | Notes |
|---|---|---|
| `preset` | `preset` (single, `params.set`) | value `'bangumi'` |
| `page` | `page` (single) | `'' + page` |
| `pageSize` | `pageSize` (single) | `'' + pageSize` |
| `duplicate` | `duplicate=true` (single) | **only when truthy**; never `duplicate=false` |
| `after` | `after=<epochMillis>` (single) | Date → `getTime()`; string → `new Date(v).getTime()` — **always a millisecond number** |
| `before` | `before=<epochMillis>` (single) | same |
| `provider` | `provider` (single) | |
| `subject` (singular) | `subject=<n>` (single) | takes precedence over `subjects` |
| `subjects` (array) | `subject=<n>` **repeated** | one param per id, `params.append` |
| `search[]` | `search=<word>` repeated | only if `search.length > 0` |
| `search[]` + `keywords[]` | `search` repeated **then** `keyword=<w>` repeated | |
| `search[]` + `exclude[]` | `search`, then `exclude=<w>` repeated | |
| `include[]` (no search) | `include=<word>` repeated | only if `include.length > 0` |
| `include[]` + `keywords[]` | `include` repeated, then `keyword` repeated | |
| `include[]` + `exclude[]` | `include`, then `exclude` | |
| only `keywords[]` (no search/include) | `keyword=<w>` repeated | |
| only `exclude[]` | `exclude=<w>` repeated | |
| `type` (singular) | `type` (single) | precedence over `types` |
| `types` (array) | `type=<t>` repeated | one param per type |
| `fansub` (singular) | `fansub` (single) | precedence over `fansubs` |
| `fansubs` (array) | `fansub=<f>` repeated | |
| `publisher` (singular) | `publisher` (single) | precedence over `publishers` |
| `publishers` (array) | `publisher=<p>` repeated | |

Then at the very end: **`params.sort()`** — all params are alphabetically sorted (name only, stable order within equal names is the append order). This is why query strings from the client look like `after=1751155200000&fansub=%E6%A1%9C%E9%83%BD%E5%AD%97%E5%B9%95%E7%BB%84&keyword=%E7%AE%80%E4%BD%93&page=1&subject=528438`.

**Array serialization is REPEATED params** (`?fansub=a&fansub=b`), never JSON and never comma-joined. Singular field wins over plural array field. Falsy/empty values produce **no** param. `duplicate:false` produces nothing.

The three exclusive title-matching modes (a reimplementation MUST reproduce this priority):
1. `search` present → emits `search` params (AND-ed full-text tokens server-side) + `keyword` + `exclude`.
2. else `include` present → emits `include` params (OR-ed) + `keyword` + `exclude`.
3. else → only `keyword` + `exclude`.

### 2.2 `parseURLSearch(params?, body?)` — parsing (server + web share this)

Two zod schemas validate URLSearchParams (plural names from `getAll`) and a JSON body (singular names win, e.g. `fansub`/`fansubs`, `publisher`/`publishers`, `type`/`types`, `subject`/`subjects`, `keywords` — note **`keyword` is pluralized to `keywords`** for the body, and the body uses **`search`/`include`/`exclude` as string-or-array**).

URL params schema (each read with `params.get` unless noted):
`provider` (enum), `duplicate` (coerced bool), `page` (coerced number), `pageSize` (coerced number), `fansub` (string[]) via `getAll('fansub')`, `publisher` (string[]) via getAll, `type` (string[]) via getAll, `before`/`after` (dateLike: null | undefined | coerced-number-ms | coerced date), `subject` (number[]) via getAll, `search` (string[]) via getAll, `include` (string[]) via getAll, `keyword` (string[]) via getAll, `exclude` (string[]) via getAll, `preset` (enum).

Body schema: `provider`, `duplicate`, `page`, `pageSize`, `fansub` (single) / `fansubs` (array), `publisher`/`publishers`, `type` (single)/`types` (array), `before`/`after`, `subject` (single number)/`subjects` (number[]), `search`/`include`/`keywords`/`exclude` each string-or-array (`stringArray`), `preset`.

**Resolution order (exact)** — this is what makes query-vs-body precedence and singular-vs-plural work:
1. `pagination = { page: res1?.page ?? res2?.page ?? 1, pageSize: res1?.pageSize ?? res2?.pageSize ?? 100 }`.
2. Clamp: `page` NaN/null/<1 → 1, else `Math.round(page)`; `pageSize` NaN/null/<1/>1000 → 100, else `Math.round(pageSize)`.
3. `preset`: body first, then URL.
4. `provider` + `duplicate`: if body provider → `duplicate = res1?.duplicate ?? res2?.duplicate ?? true` (!); if URL provider → `duplicate = res2?.duplicate ?? res1?.duplicate ?? true`. **Note the default is `true` when a provider is given!**
5. `fansubs`: body `fansub` singular → `[fansub]`; else body `fansubs` (if non-empty); else URL `fansub[]`. Dedupe with Set.
6. `publishers`: same pattern with `publisher`/`publishers`.
7. `types`: `type` singular → `[type]`; else `types`; else URL `type[]`. Dedupe.
8. `before`/`after`: `res2?.before || res1?.before || undefined`.
9. `subjects`: body `subject` → `[subject]`; else `subjects`; else URL `subject[]`.
10. `search`: body `search` (array-ified) if non-empty, else URL `search[]`. Dedupe.
11. `include`: body `include` if non-empty, else URL `include[]`. Dedupe.
12. `keywords`: body `keywords` if non-empty, else URL `keyword[]`. Dedupe.
13. `exclude`: body `exclude` if non-empty, else URL `exclude[]`. Dedupe.
14. **`if (filter.search) delete filter.include;`** — search mode **discards** include.

Server-side SQL semantics produced from the resolved filter (`apps/server/src/resources/query.ts` + `filter.ts`):
- Always `is_deleted = false`; when `!duplicate` also `duplicated_id IS NULL`.
- `provider` equality; `fansubs`/`publishers` → ids (names resolved through users/teams modules) with `fansub OR publisher` groups joined by OR.
- `types` → `type IN (...)`; `subjects` → `subject_id IN (...)`; `before` → `created_at <=`, `after` → `created_at >=`.
- `search[]` → jieba tokenized, joined with ` & ` → Postgres `title_search @@ to_tsquery('simple', ...)` full-text.
- `include[]` → `title_alt ILIKE '%term%'` (OR across terms); `keywords[]` → ANDed `ILIKE`; `exclude[]` → ANDed `NOT ILIKE`.
- `preset='bangumi'` → excludes banned publishers & fansubs (by name lists `BANGUMI_BANNED_PUBLISHERS`/`BANGUMI_BANNED_FANSUBS`).

### 2.3 Examples (verbatim from `examples/api.http` and README)

```
GET https://api.animes.garden/resources?page=2
GET https://api.animes.garden/resources?page=2&pageSize=20
GET https://api.animes.garden/resources?type=动画&fansub=LoliHouse&publisher=LoliHouse&after=2023-04-16T13:00:00.000Z&before=1681653600000
GET https://api.animes.garden/resources?fansub=ANi&fansub=爱恋字幕社
GET https://api.animes.garden/resources?type=动画&include=%E9%97%B4%E8%B0%8D%E8%BF%87%E5%AE%B6%E5%AE%B6
```

Body-based (POST) example payloads (schemas in §6.4 `ResourcesRequest`):
```json
{ "include": ["機動戰士鋼彈 水星的魔女"] }
{ "include": ["间谍过家家"] }
{ "include": ["复仇者"], "exclude": ["东京复仇者"] }
{ "include": ["机动战士高达", "水星的魔女"], "keywords": ["第二季", "ANi"] }
{ "search": ["我推的孩子", "简体"] }
```

### 2.4 `filter` JSON param (legacy feed worker / RSS feed)

`apps/worker` (Cloudflare worker proxy for `animes.garden/feed.xml`) parses the **`filter` query param**:

- `filter` = URL-encoded JSON. Either a single object or an **array of objects** (first object wins: `filter.data[0]`).
- Schema (`apps/worker/src/legacy.ts` `FilterSchema`): `provider` (single or array, enum), `duplicate` (bool), `page` (number, default 1, ≥1), `pageSize` (number, default 100, 1..1000), `fansubId` (array of number|string), `fansubName` (string[]), `publisherId` (string[]), `type` (string), `before`/`after` (number ms or ISO date), `search`/`include`/`keywords`/`exclude` (string or string[]).
- Mapping to the real API: `fansubName` → `fansubs`; `fansubId` → resolved to **names** via a bundled `teams.json` map (provider-specific ids) → `fansubs`; `publisherId` → names → `publishers`. Then `stringifyURLSearch(options)` rewrites the URL.
- A 400 error shape (worker): `{ "status": 400, "detail": { "url", "filter", "message" } }`.
- README example (verbatim, URL-decoded):
  ```json
  [{"fansubId":["619"],"type":"动画","include":["葬送的芙莉莲"],"keywords":["简体内嵌"]}]
  ```
  i.e. the documented public `filter` param shape is `fansubId`, `type`, `include`, `keywords`, `exclude`, `subject`, `after`, `before`. (The current server feed route itself reads classic query params via `parseURLSearch`; `filter` JSON is the legacy worker contract and is what the docs pages advertise.)
- The **collection** creation flow stores each filter's URL as `searchParams = '?' + stringifyURLSearch(resolved).toString()` (`apps/web/src/pages/resources.($page)/Filter.tsx`, `addToCollection`).

### 2.5 Collection filter object shape (`CollectionFilter`)

```ts
type CollectionFilter<S, R, T> = ResolvedFilterOptions & {
  name: string;
  searchParams: S extends true ? string : undefined;
} & (R extends true ? { resources: Resource<T>[]; complete: boolean } : {});
```
On the wire (GET /collection/:hash) every entry also carries `resources` + `complete` because the server resolves each filter against `page:1, pageSize:1000`. `before`/`after` inside a stored filter are ISO strings; empty arrays are deleted before re-query (`collections/index.ts`).

---

## 3. Sort options

**There is NO user-facing sort parameter in the AnimeGarden API.** Neither the client (`stringifyURLSearch`, `parseURLSearch`, resolver schemas) nor the server routes accept a `sort` key.

The only ordering, everywhere, is **`createdAt` DESCENDING (newest first)**:
- Server DB query: `.orderBy(desc(resources.createdAt)).offset(offset).limit(limit)` (`apps/server/src/resources/query.ts` `executeResourcesQuery`).
- Client final de-duplication: `[...map.values()].sort((lhs, rhs) => rhs.createdAt.getTime() - lhs.createdAt.getTime())` (`api/resources.ts` `uniq`).
- Collection in-memory insert keeps `createdAt` desc order.
- Info-hash lookup: `.orderBy(desc(resources.createdAt)).limit(1)`.

Implementation guidance for Go: the default and only sort is `created_at DESC` (+ stable tie-break by `id DESC` in the indexed queries: composite index `resources_sort_by_created_at` on `(isDeleted?, createdAt desc nullsFirst, id desc nullsFirst)` per `schema/resources.ts`). A reimplementation should hard-code this ordering; do not expose a sort param.

---

## 4. Code generator (`apps/web/src/utils/code-generator.ts`)

The web resources page lets users copy API snippets. All generators call `resolveFilterOptions(filter)` (see §4.4) then `stringifyURLSearch(realFilter)` with `subjects` replaced by `[subject.id]` when a subject page is active. `FEED_HOST` default `api.animes.garden`, `APP_HOST` default `animes.garden`.

### 4.1 cURL

```ts
const searchParams = stringifyURLSearch(realFilter);
const url = `https://${FEED_HOST}/resources?${searchParams.toString()}`;
return `curl "${url}"`;
```
Template (verbatim shape):
```
curl "https://api.animes.garden/resources?after=1751155200000&fansub=%E6%A1%9C%E9%83%BD%E5%AD%97%E5%B9%95%E7%BB%84&keyword=%E7%AE%80%E4%BD%93&subject=528438"
```

### 4.2 JavaScript (`@animegarden/client`)

Options assembled in this exact order: `preset`, `types`, `subjects`, `publishers`, `fansubs`, `search`, `include`, `keywords`, `exclude`, `after` (`new Date('<ISO>')`), `before` (`new Date('<ISO>')`); each emitted as a 2-space-indented line and joined with `',\n'`.

Template (verbatim):
```
import { fetchResources } from '@animegarden/client';

const resources = await fetchResources({
  preset: 'bangumi',
  types: ["动画"],
  subjects: [528438],
  publishers: ["LoliHouse"],
  fansubs: ["桜都字幕组"],
  search: ["我推的孩子"],
  include: ["葬送的芙莉莲"],
  keywords: ["简体内嵌"],
  exclude: ["合集"],
  after: new Date('2025-06-29T00:00:00.000Z'),
  before: new Date('2025-06-30T00:00:00.000Z')
});
```

### 4.3 Python (`requests`)

Params assembled in this exact order with **singular names**: `preset`, `type`, `subject`, `publisher`, `fansub`, `search`, `include`, `keyword`, `exclude`, `after` (epoch ms **number**, `getTime()`), `before` (epoch ms number); joined with `',\n'`.

Template (verbatim):
```
import requests

url = "https://api.animes.garden/resources"
params = {
  'preset': 'bangumi',
  'type': ["动画"],
  'subject': [528438],
  'publisher': ["LoliHouse"],
  'fansub': ["桜都字幕组"],
  'search': ["我推的孩子"],
  'include': ["葬送的芙莉莲"],
  'keyword': ["简体内嵌"],
  'exclude': ["合集"],
  'after': 1751155200000,
  'before': 1751241600000
}

response = requests.get(url, params=params)
resources = response.json()
```
(requests sends list values as repeated params: `?subject=528438&fansub=桜都字幕组&...`, matching the API.)

### 4.4 iframe embed

```
const searchParams = stringifyURLSearch(realFilter);
const url = `//${APP_HOST}/iframe?${searchParams.toString()}`;
return `<iframe src="${url}" width="100%" height="600" frameborder="0" style="box-sizing:border-box;"></iframe>`;
```
Example (verbatim README):
```html
<iframe src="//animes.garden/iframe?subject=477825" width="100%" height="600" frameborder="0"></iframe>
```

### 4.5 `resolveFilterOptions` (used by all generators; `apps/web/src/pages/resources.($page)/Filter.tsx`)

```ts
export function resolveFilterOptions(filter) {
  return {
    preset: filter.preset ?? undefined,
    types: types.length > 0 ? types : undefined,       // dedup'd
    subjects: filter.subjects ?? [],
    publishers: publishers.length > 0 ? publishers : undefined,  // dedup'd
    fansubs: fansubs.length > 0 ? fansubs : undefined,            // dedup'd
    before: filter.before ? new Date(filter.before) : undefined,
    after: filter.after ? new Date(filter.after) : undefined,
    search: filter.search ? removeQuote(filter.search) : undefined,
    include: filter.include ?? undefined,
    keywords: filter.keywords ?? undefined,
    exclude: filter.exclude ?? undefined
  };
}
```
`removeQuote(words) = words.map(w => w.replace(/^(\+|-)?"([^"]*)"$/, '$1$2'))` (strips wrapping quotes from quoted search words).

---

## 5. Examples directory (`/tmp/animegarden-orig/examples/`)

### 5.1 `api.http` — full request list (verbatim, with expected responses)

| # | Block comment | Request | Expected response |
|---|---|---|---|
| 1 | List the first page of resources | `GET https://api.animes.garden/resources` | 200 `ResourcesResponse` (100 items, page 1) |
| 2 | List the first page with full magnet + tracker | `GET https://api.animes.garden/resources?tracker=true` | 200; each resource has `tracker` string (`&tr=...`) |
| 3 | Get resource detail (dmhy) | `GET https://api.animes.garden/detail/dmhy/674983` | 200 `{status:'OK', resource, detail, ...}` |
| 4 | Get resource detail by id (alias) | `GET https://api.animes.garden/detail/dmhy/674983` | same as #3 |
| 5 | Second page | `GET https://api.animes.garden/resources?page=2` | 200; `pagination.page=2` |
| 6 | 21st–40th items | `GET https://api.animes.garden/resources?page=2&pageSize=20` | 200; 20 items, `pageSize=20` |
| 7 | More filters (type/fansub/publisher/after/before) | `GET https://api.animes.garden/resources?type=动画&fansub=LoliHouse&publisher=LoliHouse&after=2023-04-16T13:00:00.000Z&before=1681653600000` | 200; only matching resources (note `after` ISO and `before` epoch ms both accepted) |
| 8 | Multiple fansubs | `GET https://api.animes.garden/resources?fansub=ANi&fansub=爱恋字幕社` | 200; resources from either fansub (OR) |
| 9 | Search (body) | `GET https://api.animes.garden/resources` + body `{"include":["機動戰士鋼彈 水星的魔女"]}` | 200; titles containing the term (OR) |
| 10 | Only type 动画 search | `GET https://api.animes.garden/resources?type=动画` + body `{"include":["间谍过家家"]}` | 200; 动画 only |
| 11 | Search with exclude | `GET .../resources` + body `{"include":["复仇者"],"exclude":["东京复仇者"]}` | 200; includes 复仇者 minus 东京复仇者 |
| 12 | Complicated search | `GET .../resources?type=动画` + body `{"include":["机动战士高达","水星的魔女"],"keywords":["第二季","ANi"]}` | 200; type=动画 AND (title contains ANY include) AND (contains ALL keywords) |
| 13 | Full-text search | `GET .../resources?type=动画` + body `{"search":["我推的孩子","简体"]}` | 200; full-text index search (both terms) |
| 14 | All users | `GET https://api.animes.garden/users` | 200 `{status:'OK', users:[User]}` |
| 15 | All teams | `GET https://api.animes.garden/teams` | 200 `{status:'OK', teams:[Team]}` |
| 16 | Get collection | `GET https://api.animes.garden/collection/RUi8lloFMts8DjeI97lIGfFE5zz6MM9a-Qek7xaEY78` | 200 `{status:'OK', hash, name, createdAt, results:[{resources,complete,filter}]}` |

(The `Content-Type: application/json` headers in the notes are for the body-bearing requests; blocks 9–13 are GETs with JSON bodies, which the server accepts via `ctx.req.json().catch(() => undefined)`.)

### 5.2 `embed.html`

Minimal HTML page embedding the iframe:
```html
<iframe src="//animes.garden/iframe?fansub=%E6%A1%9C%E9%83%BD%E5%AD%97%E5%B9%95%E7%BB%84&subject=528438" width="100%" height="600" frameborder="0" style="box-sizing:border-box;padding:12px;border-radius:8px;"></iframe>
```
Demonstrates the iframe embed with `fansub` + `subject` query params (protocol-relative `//`).

### 5.3 `fetch.py`

PEP 723 script (dependencies: `requests`):
```python
import requests
url = "https://api.animes.garden/resources"
params = {
  'subject': [528438],
  'fansub': ["桜都字幕组"],
  'keyword': ["简体"],
  'after': 1751155200000
}
response = requests.get(url, params=params)
resources = response.json()
print(resources)
```
Demonstrates repeated-param lists (`subject`, `fansub`) and epoch-ms `after`.

### 5.4 `fetch.ts`

```ts
import { fetchResources } from '@animegarden/client';
const resources = await fetchResources({
  subjects: [528438],
  fansubs: ["桜都字幕组"],
  keywords: ["简体"],
  after: new Date('2025-06-29T00:00:00.000Z')
});
console.log(resources);
```
Demonstrates the plural client option names (`subjects`, `fansubs`, `keywords`, Date `after`).

---

## 6. OpenAPI spec — `GET /openapi.json` and the docs page

### 6.1 Served document

- Route: `apps/web/src/routes/openapi[.]json.ts` — TanStack Start server handler, `GET /openapi.json` → `Response.json(getPublicOpenApiSpec(version, license), { 'Cache-Control': 'public, max-age=3600, s-maxage=86400' })`.
- `version`/`license` come from the web build's package metadata (`~build/package`).
- `getPublicOpenApiSpec(version, license)` (`apps/web/src/pages/docs.api/spec.ts`):
  - deep-copies `spec.json`, deletes `components.securitySchemes`;
  - sets `info.version = version`, `info.license.name = license` (keeps license `url`);
  - filters out every path starting with `/admin/`;
  - drops the `Admin` tag from `tags`.
  - The base spec files: `openapi: 3.1.0`.
- The server ALSO exposes `/mcp`; the API server route list includes admin endpoints (stripped from the public doc).

### 6.2 Info block (verbatim from `spec.json`)

```json
{
  "openapi": "3.1.0",
  "info": {
    "title": "🌸 Anime Garden API",
    "description": "🌸 Anime Garden 是 [动漫花园](https://share.dmhy.org/) 第三方镜像站以及动画 BT 资源聚合站。\n\n+ ☁️ 为开发者准备的开放 API 接口\n+ 📺 查看动画放送时间表来找到你喜欢的动画\n+ 🔖 支持丰富的高级搜索，例如：`葬送的芙莉莲 +简体内嵌 字幕组:桜都字幕组 类型:动画`\n+ 📙 自定义 RSS 订阅链接，例如：[葬送的芙莉莲](https://api.animes.garden/feed.xml?filter=%5B%7B%22fansubId%22:%5B%22619%22%5D,%22type%22:%22%E5%8B%95%E7%95%AB%22,%22include%22:%5B%22%E8%91%AC%E9%80%81%E7%9A%84%E8%8A%99%E8%8E%89%E8%8E%B2%22%5D,%22keywords%22:%5B%22%E7%AE%80%E4%BD%93%E5%86%85%E5%B5%8C%22%5D%7D%5D)\n+ ⭐ 搜索条件收藏夹和生成聚合的 RSS 订阅链接\n+ 👷‍♂️ 支持与 AutoBangumi 和 AnimeSpace 集成\n",
    "version": "0.0.0",
    "license": { "name": "MIT", "url": "https://github.com/yjl9903/AnimeGarden/blob/main/LICENSE" }
  },
  "externalDocs": { "description": "GitHub Repository", "url": "https://github.com/yjl9903/AnimeGarden" },
  "servers": [ { "url": "https://api.animes.garden", "description": "API 服务器" } ]
}
```

### 6.3 Paths (public, after admin filter)

`/`, `/users`, `/teams`, `/subjects`, `/resources` (GET+POST), `/resources/{provider}` (GET+POST), `/detail/{provider}/{id}` (GET), `/collection` (POST, PUT — "upsert"), `/collection/{hash}` (GET), `/feed.xml` (GET), `/collection/{hash}/feed.xml` (GET), `/sitemaps/subjects` (GET), `/sitemaps/{year}/{month}` (GET). (Stripped admin: `/admin/providers`, `/admin/resources/{provider}`, `/admin/resources/{provider}/sync`.)

Parameter components (`components.parameters`):
- `PageParam` `page` integer min 1 default 1;
- `PageSizeParam` `pageSize` integer min 1 max 1000 default 100;
- `ProviderParam` `provider` enum `["dmhy","mikan","moe","ani"]`;
- `SearchParam`/`IncludeParam`/`KeywordsParam`/`ExcludeParam`/`TypeParam`/`SubjectParam`/`FansubParam`/`PublisherParam` — array, `style: form, explode: true` (repeated params; docs text says Swagger users may comma-separate: `?search=foo,bar`);
- `AfterParam`/`BeforeParam` — string date-time (the spec advertises ISO, the client always emits epoch ms; server accepts both);
- `DuplicateParam` boolean default false;
- `TrackerParam`/`MetadataParam` boolean default false.

### 6.4 Schemas (`components.schemas`, verbatim key facts)

- **Provider**: `{ id: enum[dmhy,mikan,moe,ani], name, refreshedAt: date-time, isActive: boolean }` all required.
- **User / Team**: `{ id: integer, provider: string, providerId: string, name: string, avatar: string|null }` required `[id, provider, providerId, name]` (avatar nullable).
- **Subject**: `{ id: integer (Bangumi subject ID), name: string, keywords: string[], activedAt: date-time, isArchived: boolean }` all required.
- **Resource**: `{ id, provider, providerId, title, href, type, magnet, tracker (string|null, only when tracker=true), size (integer), fansub ({id,name,avatar?} null), publisher ({id,name,avatar?}), subjectId (integer|null), createdAt: date-time, fetchedAt: date-time, metadata ({anipar: object}|null, only when metadata=true) }` required `[id, provider, providerId, title, href, type, magnet, size, publisher, createdAt, fetchedAt]`.
- **ResourceDetail**: `{ description: string, files: [{name,size}], magnets: [{name,url}], hasMoreFiles: boolean }` — file `size` is **string**.
- **FilterOptions** (resolved echo): `preset (enum [bangumi]), provider (enum), duplicate, types[], after, before, fansubs[], publishers[], subjects[], search[], include[], keywords[], exclude[]`.
- **PaginationInfo**: `{ page, pageSize, complete }` all required.
- **ResourcesRequest** (POST body): `page` (1), `pageSize` (1..1000, 100), `provider` enum, `duplicate` bool default false, `after`/`before` date-time, `search`/`include`/`keywords`/`exclude` (string | string[]), `type` string, `types` string[], `subject` int, `subjects` int[], `fansub` string, `fansubs` string[], `publisher` string, `publishers` string[], `preset` enum [bangumi].
- **ResourcesResponse**: `{ status: enum[OK], complete: boolean (legacy, == pagination.complete), resources: Resource[], pagination: PaginationInfo, filter: FilterOptions, timestamp: date-time }` all required.
- **CollectionRequest**: `{ name: string, authorization: string, filters: [{ name, searchParams }] }` required `[name, authorization, filters]`, filter items `additionalProperties: true`, only `name` required.
- **ErrorResponse**: `{ status: enum[ERROR], message: string }` both required.

### 6.5 Docs page rendering

- Page: `apps/web/src/pages/docs.api/route.tsx` — imports `swagger-ui-react` + `swagger-ui-react/swagger-ui.css`, renders **`<SwaggerUI spec={spec} />`** inside the site Layout. **The library is Swagger UI React** (default config; no custom `url`, `docExpansion` etc. — spec passed directly as a prop).
- Path: `/docs/api` (`apps/web/src/routes/docs/api/route.tsx`).

---

## 7. Torrent utilities (`packages/shared/src/torrent.ts`)

btih(v1) (BitTorrent info-hash v1) Base32 ⇄ Hex conversions — RFC 4648 alphabet `ABCDEFGHIJKLMNOPQRSTUVWXYZ234567`, magnet-style **no padding** (`=` stripped), uppercase output.

Functions (verbatim signatures + behavior):

```ts
function base32ToBytes(b32: string): Uint8Array
// trim, upper-case, strip trailing '='s; throws Error(`Invalid base32 char: ${ch}`)
// bit-stream decode, 5-bit groups → bytes.

function bytesToBase32(bytes: Uint8Array): string
// 8-bit stream → 5-bit groups; leftover bits padded with zeros; NO '=' padding.

function hexToBytes(hex: string): Uint8Array
// trim + lowercase; throws Error('Invalid hex string') unless /^[0-9a-f]+$/ and even length.

function bytesToHex(bytes: Uint8Array): string   // lowercase, zero-padded pairs.

export function btihBase32ToHex(btihB32: string): string
// base32ToBytes then require exactly 20 bytes, else throw `btih(v1) must be 20 bytes, got ${n}`; returns lowercase hex.

export function btihHexToBase32(btihHex: string): string
// hexToBytes, require 20 bytes, return base32 (uppercase, no padding).

export function extractBtihFromMagnet(magnetUrl: string): { format: 'hex' | 'base32'; value: string } | null
// trim; must start with 'magnet:?' (case-insensitive) else null.
// Parses query via URLSearchParams; iterates ALL `xt` params; first matching
// /^urn:btih:([a-zA-Z0-9]+)$/ wins.
// 40-hex -> { format:'hex', value: lowercased }; 32 [A-Z2-7] -> { format:'base32', value: UPPERCASED }.
// Anything else (non-standard length) -> null.

export function normalizeBtihToHex(magnetUrl: string): string
// extract; if null or already hex -> returns the input magnetUrl unchanged;
// else 'magnet:?xt=urn:btih:' + btihBase32ToHex(value).

export function normalizeBtihToBase32(magnetUrl: string): string
// extract; if null or already base32 -> input unchanged;
// else 'magnet:?xt=urn:btih:' + btihHexToBase32(value).
```

Key detail: **hex output is lowercase, base32 output is uppercase**, `=`
padding is never emitted, and both normalize functions return the original magnet unchanged when the btih cannot be extracted or is already in the target format.

---

## 8. Status endpoint

- **Path**: `GET /` (root) — also `GET /health` returns the identical body.
- **Shape** (server, `apps/server/src/server/index.ts`):
  ```json
  {
    "status": "OK",
    "timestamp": "2026-01-01T08:00:00.000Z",
    "providers": {
      "dmhy": { "id": "dmhy", "name": "動漫花園", "refreshedAt": "ISO", "isActive": true },
      "mikan": { "id": "mikan", "name": "蜜柑计划", "refreshedAt": "ISO", "isActive": true },
      "moe":   { "id": "moe",   "name": "萌番组",   "refreshedAt": "ISO", "isActive": true },
      "ani":   { "id": "ani",   "name": "ANi",      "refreshedAt": "ISO", "isActive": true }
    }
  }
  ```
  The client `fetchStatus` only surfaces `{ ok, timestamp, providers }` (see §1.8); the spec.json documents `message: "Anime Garden 動漫花園 镜像站 / 动画 BT 资源聚合站"` as an additional property but the current server response does not include it (required fields in spec: `message`, `timestamp`, `providers` — a small spec/impl drift to be aware of).
- `X-Response-Timestamp` header is set to the same providers timestamp (`c.set('responseTimestamp', timestamp)`), and the client's merged `timestamp` therefore equals the body `timestamp`.

---

## 9. MCP public docs

### 9.1 README usage (verbatim, `README.md` / `README.en.md`)

Anime Garden MCP 服务端点: `https://api.animes.garden/mcp`.

```json
{
  "mcpServers": {
    "animegarden": {
      "url": "https://api.animes.garden/mcp"
    }
  }
}
```

### 9.2 `/llms.txt` (web route, `apps/web/src/routes/llms[.]txt.ts`)

Advertises: main site, public API host (`api.animes.garden`), MCP endpoint `https://api.animes.garden/mcp`, OpenAPI schema `https://animes.garden/openapi.json`, MCP server card `https://animes.garden/.well-known/mcp/server-card.json`, sitemap, GitHub repo. Served with `Cache-Control: public, max-age=3600, s-maxage=86400`, `Content-Type: text/plain; charset=utf-8`. Site pages additionally support Markdown negotiation via `Accept: text/markdown`.

### 9.3 MCP server card (`/.well-known/mcp/server-card.json`, web route)

```json
{
  "$schema": "https://static.modelcontextprotocol.io/schemas/v1/server-card.schema.json",
  "name": "garden.animes/animegarden",
  "version": "<web pkg version>",
  "title": "Anime Garden MCP",
  "description": "Search Anime Garden torrent resources.",
  "websiteUrl": "https://animes.garden",
  "repository": { "source": "github", "url": "https://github.com/yjl9903/AnimeGarden", "subfolder": "apps/server" },
  "icons": [ { "src": "https://animes.garden/favicon.svg", "mimeType": "image/svg+xml", "sizes": ["any"] } ],
  "remotes": [ { "type": "streamable-http", "url": "https://api.animes.garden/mcp" } ],
  "serverInfo": { "name": "animegarden", "version": "<version>" },
  "transport": { "type": "streamable-http", "endpoint": "/mcp", "url": "https://api.animes.garden/mcp" },
  "capabilities": { "tools": true, "resources": true, "prompts": false },
  "_meta": { "garden.animes/primitives": { "tools": ["search_resources"], "resources": ["resource_detail"], "prompts": [] } }
}
```
The API server also serves `GET /.well-known/mcp/server-card.json` as a **302 redirect** to the web site's card.

### 9.4 MCP server implementation (`apps/server/src/server/mcp/`)

- **Transport**: `StreamableHTTPTransport` from `@hono/mcp`, mounted at `app.all('/mcp')` on the API server. HTML `Accept` → 400 `{status:'ERROR', message:'Please connect /mcp with MCP client'}`.
- **Server metadata**: name `animegarden`, version from `apps/server/package.json`, description lists capabilities and recommended usage (search first; use detail only when description/file list needed; providerId preferred; provider must be one of `[dmhy, moe, mikan, ani]`).
- **Tool `search_resources`** input schema (`searchResourcesInputSchema`): `fansubs` (string[], OR), `publishers` (string[], OR, OR-combined with fansubs), `types` (enum `['动画','合集','音乐','日剧','RAW','漫画','游戏','特摄','其他']`, OR), `before`/`after` (coerce date — accepts date string or timestamp), `subjects` (coerced int[], OR), `search` (string[], full-text; takes precedence over include), `include` (string[], OR; only effective without search), `keywords` (string[], AND), `exclude` (string[], block).
  - Handler: `parseURLSearch(undefined, args)` → `query.find(filter, { page: 1, pageSize: 30 })`; each result maps to `{ id, provider, providerId, title, uri: 'animegarden://resources/{provider}/{providerId}', href: 'https://animes.garden/detail/{provider}/{providerId}', type, magnet: magnet+tracker, size, createdAt, publisher: publisher.name, fansub: fansub?.name }`; returned as `content[0].text` (pretty JSON) + `structuredContent.resources`.
- **Resource `resource_detail`**: template `animegarden://resources/{provider}/{providerId}`; invalid provider/uri → `{error:'INVALID_RESOURCE_URI', uri, message}`; not found → `{error:'RESOURCE_NOT_FOUND', provider, providerId}`; found → JSON with `{ id, provider, providerId, title, uri, href, type, magnet (magnet+tracker), size, createdAt, publisher, fansub, description, files, hasMoreFiles }` (mimeType `application/json`).
- `buildResourceUri(provider, providerId) = 'animegarden://resources/' + provider + '/' + encodeURIComponent(providerId)`; `decodeURIComponentSafe` guards decoding.

---

## Appendix A — exact URL construction recipe (for Go reimplementation)

1. `baseURL` (default `https://api.animes.garden/`) + path with leading slashes removed:
   - `resources` + `'?' + searchParams.toString()` (client `fetchPage`); page is a query param.
   - `detail/${provider}/${href}` (href raw); `detail/infohash/${encodeURIComponent(hash)}`.
   - `collection` (PUT, JSON body); `collection/${hash}` (GET).
   - `` (empty path → base URL root) for status.
2. Search params built by `stringifyURLSearch` (§2.1) then `params.sort()`; `tracker`/`metadata` appended after sort in `fetchResources`.
3. Headers: `x-trace-id` = new UUID; `user-agent` = `animegarden@0.5.4` unless user-set.
4. Every 2xx JSON response is merged with a `timestamp` Date resolved from `X-Response-Timestamp` header first, then body `timestamp`.
5. 429 → sleep 16 s then throw; other non-2xx → throw `AnimeGardenError` with status/statusText/url message and parsed body; TimeoutError → 100 ms hook sleep then throw; AbortError → rethrow.
6. Multi-page fetch: force `pageSize=1000`, iterate `page` from `options.page ?? 1`, stop on `map.size >= count` (count<0 → ∞), `pagination.complete`, empty page, abort, or error; dedupe by `href`; final sort `createdAt` desc.

## Appendix C — internal `docs/` directory (repository engineering docs, not public contract)

All files under `/tmp/animegarden-orig/docs/` were read. They are **internal engineering/audit docs**, not part of the public API contract, but several contain behavioral facts worth keeping:

| File | Relevance to the reimplementation |
|---|---|
| `docs/README.md` | Doc index; contributor conventions. |
| `docs/server/architecture-overview.md` | Service roles: `server` (public API + MCP + /mcp), `cron` (fetch/sync jobs), feed service; Hono routes bound: `/`, `/health`, `users`, `subjects`, `resources`, `collections`, `feed`, `admin`, `sitemaps`, `/mcp`; error→JSON/XML conversion; 60 s timeout; Redis publisher for cache invalidation. |
| `docs/server/deployment-topology.md` | Domains: `animes.garden` (web), `api.animes.garden` (public API/feed), internal `animegarden-server-production.flycast`. |
| `docs/server/resources-write-flow.md` | Cron-driven write path (`/admin/resources/:provider` every 5 min, `/sync` hourly, pages 1–10); `fansubId`/`publisherId` resolution; detail backfill. |
| `docs/server/resources-query-task-review.md` | Query `Task` prefetch tiers (search → include → subjects → types → fansubs/publishers → fallback), `RESOURCES_TASK_PREFETCH_MAX_COUNT=1000`, deep-pagination guard `offset+limit > 10000` rejected. |
| `docs/server/resources-index-plan.md` | Hot-path partial SQL indexes on `is_deleted=false AND duplicated_id IS NULL`, ordered `created_at DESC`; confirms the fixed sort. |
| `docs/server/telegram-push-flow.md` | Telegram push state machine (out of scope for the client contract). |
| `docs/web/architecture-overview.md` | Web no longer proxies `/api/*` or `/feed.xml`; uses `@animegarden/client` against `FEED_HOST`; `/llms.txt`, `/openapi.json`, `/.well-known/*` are web routes; cache headers per page type. |
| `docs/web/resources-anchor-cursor-cache-plan.md` | **Draft (not implemented)** anchor+cursor pagination plan — page/offset pagination remains current. |
| `docs/web/tanstack-start-migration-performance-notes.md`, `tanstack-start-parity-test-paths.md` | Migration-era notes; the parity doc lists live expectations: e.g. local `?type=动画&type=合集&preset=bangumi`, full-text search results, magnet+tracker behavior. |
| `docs/web/umami-tracking.md` | Analytics events (`download`, `pikpak`, `feed.open`, ...) — frontend only. |
| `docs/anipar/README.md`, `parse-result-title-audit.md`, `parse-result-metadata-audit.md` | Title/metadata parsing audits for `packages/anipar` (the `metadata.anipar` payload producer) — relevant if metadata fidelity must match. |

## Appendix B — web app page → API mapping (for Vue reimplementation)

| Web page | API call(s) |
|---|---|
| `/resources/:page` | `fetchResources({...parsedFilter, ...pagination, page, pageSize:30, tracker:true, metadata:true})` |
| `/iframe` | same with `pageSize:30` |
| `/detail/:provider/:providerId` | `fetchResourceDetail(provider, providerId)` |
| `/collection/:hash` | `fetchCollection(hash)` |
| collection creation | `generateCollection(collection)` (PUT /collection) |
| `/subjects` data | server `GET /subjects` (web uses bgmx/bgm.animes.garden for calendar + subject metadata) |
| RSS links | `https://api.animes.garden/feed.xml<search>` and `https://api.animes.garden/collection/{hash}/feed.xml` |