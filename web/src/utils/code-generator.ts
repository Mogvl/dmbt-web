// Code generation mirroring apps/web/src/utils/code-generator.ts.

import type { ResolvedFilterOptions } from '../api/client';
import { stringifyURLSearch } from '../api/client';

export const APP_HOST = 'localhost:9700';
export const FEED_HOST = 'localhost:9701';

export interface CodeGeneratorOptions {
  filter?: ResolvedFilterOptions;
  subject?: { id: number };
}

// Resolve filter options (dates -> Date, dedupe arrays)
export function resolveFilterOptions(filter: ResolvedFilterOptions): ResolvedFilterOptions {
  const types = [...new Set(filter.types ?? [])];
  const publishers = [...new Set(filter.publishers ?? [])];
  const fansubs = [...new Set(filter.fansubs ?? [])];
  return {
    preset: filter.preset ?? undefined,
    types: types.length > 0 ? types : undefined,
    subjects: filter.subjects ?? [],
    publishers: publishers.length > 0 ? publishers : undefined,
    fansubs: fansubs.length > 0 ? fansubs : undefined,
    before: filter.before ? new Date(filter.before) : undefined,
    after: filter.after ? new Date(filter.after) : undefined,
    search: filter.search ?? undefined,
    include: filter.include ?? undefined,
    keywords: filter.keywords ?? undefined,
    exclude: filter.exclude ?? undefined
  } as ResolvedFilterOptions;
}

export function generateCurlCode(options: CodeGeneratorOptions): string {
  const { filter, subject } = options;
  if (!filter) throw new Error('没有筛选条件');
  const resolved = resolveFilterOptions(filter);
  const realFilter = { ...resolved, subjects: subject ? [subject.id] : resolved.subjects };
  const searchParams = stringifyURLSearch(realFilter);
  const url = `https://${FEED_HOST}/resources?${searchParams.toString()}`;
  return `curl "${url}"`;
}

export function generateJavaScriptCode(options: CodeGeneratorOptions): string {
  const { filter, subject } = options;
  if (!filter) throw new Error('没有筛选条件');
  const resolved = resolveFilterOptions(filter);
  const realFilter = { ...resolved, subjects: subject ? [subject.id] : resolved.subjects };

  const filterOptions: string[] = [];
  if (realFilter.preset) filterOptions.push(`  preset: '${realFilter.preset}'`);
  if (realFilter.types && realFilter.types.length > 0)
    filterOptions.push(`  types: ${JSON.stringify(realFilter.types)}`);
  if (realFilter.subjects && realFilter.subjects.length > 0)
    filterOptions.push(`  subjects: ${JSON.stringify(realFilter.subjects)}`);
  if (realFilter.publishers && realFilter.publishers.length > 0)
    filterOptions.push(`  publishers: ${JSON.stringify(realFilter.publishers)}`);
  if (realFilter.fansubs && realFilter.fansubs.length > 0)
    filterOptions.push(`  fansubs: ${JSON.stringify(realFilter.fansubs)}`);
  if (realFilter.search && realFilter.search.length > 0)
    filterOptions.push(`  search: ${JSON.stringify(realFilter.search)}`);
  if (realFilter.include && realFilter.include.length > 0)
    filterOptions.push(`  include: ${JSON.stringify(realFilter.include)}`);
  if (realFilter.keywords && realFilter.keywords.length > 0)
    filterOptions.push(`  keywords: ${JSON.stringify(realFilter.keywords)}`);
  if (realFilter.exclude && realFilter.exclude.length > 0)
    filterOptions.push(`  exclude: ${JSON.stringify(realFilter.exclude)}`);
  if (realFilter.after) filterOptions.push(`  after: new Date('${realFilter.after.toISOString()}')`);
  if (realFilter.before) filterOptions.push(`  before: new Date('${realFilter.before.toISOString()}')`);

  return `import { fetchResources } from '@animegarden/client';

const resources = await fetchResources({
${filterOptions.join(',\n')}
});`;
}

export function generatePythonCode(options: CodeGeneratorOptions): string {
  const { filter, subject } = options;
  if (!filter) throw new Error('没有筛选条件');
  const resolved = resolveFilterOptions(filter);
  const realFilter = { ...resolved, subjects: subject ? [subject.id] : resolved.subjects };

  const params: string[] = [];
  if (realFilter.preset) params.push(`  'preset': '${realFilter.preset}'`);
  if (realFilter.types && realFilter.types.length > 0)
    params.push(`  'type': ${JSON.stringify(realFilter.types)}`);
  if (realFilter.subjects && realFilter.subjects.length > 0)
    params.push(`  'subject': ${JSON.stringify(realFilter.subjects)}`);
  if (realFilter.publishers && realFilter.publishers.length > 0)
    params.push(`  'publisher': ${JSON.stringify(realFilter.publishers)}`);
  if (realFilter.fansubs && realFilter.fansubs.length > 0)
    params.push(`  'fansub': ${JSON.stringify(realFilter.fansubs)}`);
  if (realFilter.search && realFilter.search.length > 0)
    params.push(`  'search': ${JSON.stringify(realFilter.search)}`);
  if (realFilter.include && realFilter.include.length > 0)
    params.push(`  'include': ${JSON.stringify(realFilter.include)}`);
  if (realFilter.keywords && realFilter.keywords.length > 0)
    params.push(`  'keyword': ${JSON.stringify(realFilter.keywords)}`);
  if (realFilter.exclude && realFilter.exclude.length > 0)
    params.push(`  'exclude': ${JSON.stringify(realFilter.exclude)}`);
  if (realFilter.after) params.push(`  'after': ${realFilter.after.getTime()}`);
  if (realFilter.before) params.push(`  'before': ${realFilter.before.getTime()}`);

  return `import requests

url = "https://${FEED_HOST}/resources"
params = {
${params.join(',\n')}
}

response = requests.get(url, params=params)
resources = response.json()`;
}

export function generateIframeCode(options: CodeGeneratorOptions): string {
  const { filter, subject } = options;
  if (!filter) throw new Error('没有筛选条件');
  const resolved = resolveFilterOptions(filter);
  const realFilter = { ...resolved, subjects: subject ? [subject.id] : resolved.subjects };
  const searchParams = stringifyURLSearch(realFilter);
  const url = `//${APP_HOST}/iframe?${searchParams.toString()}`;
  return `<iframe src="${url}" width="100%" height="600" frameborder="0" style="box-sizing:border-box;"></iframe>`;
}
