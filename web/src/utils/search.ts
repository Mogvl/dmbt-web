// Search input parsing/serialization mirroring apps/web/src/layouts/Search/utils.ts.

import type { ResolvedFilterOptions } from '../api/client';
import { PRESET_DISPLAY_NAME } from './constants';

export interface ParsedSearchInput {
  search: string[];
  include: string[];
  keywords: string[];
  exclude: string[];
  subjects: string[];
  publishers: string[];
  fansubs: string[];
  types: string[];
  after?: Date;
  before?: Date;
  preset?: string;
}

// Tokenizer: splits on whitespace, respects quotes and backslash escapes.
export function tokenizeSearchInput(input: string): string[] {
  const tokens: string[] = [];
  let current = '';
  let quote: string | null = null;
  let escaped = false;
  for (const ch of input) {
    if (escaped) {
      current += ch;
      escaped = false;
      continue;
    }
    if (ch === '\\') {
      escaped = true;
      continue;
    }
    if (quote) {
      if (ch === quote) {
        quote = null;
      } else {
        current += ch;
      }
      continue;
    }
    if (ch === '"' || ch === "'" || ch === '“' || ch === '”') {
      quote = ch === '“' || ch === '”' ? '”' : ch;
      continue;
    }
    if (/\s/.test(ch)) {
      if (current) {
        tokens.push(current);
        current = '';
      }
      continue;
    }
    current += ch;
  }
  if (current) tokens.push(current);
  return tokens;
}

const fullWidthColon = (s: string) => s.replace(/：/g, ':');

function stripPrefix(word: string, prefixes: string[]): string | undefined {
  for (const p of prefixes) {
    if (word.startsWith(p)) return word.slice(p.length);
  }
  return undefined;
}

export function parseSearchInput(input: string): ParsedSearchInput {
  const result: ParsedSearchInput = {
    search: [],
    include: [],
    keywords: [],
    exclude: [],
    subjects: [],
    publishers: [],
    fansubs: [],
    types: []
  };

  for (const rawWord of tokenizeSearchInput(input)) {
    const word = fullWidthColon(rawWord);

    let value: string | undefined;

    if ((value = stripPrefix(word, ['subject:', '动画:'])) !== undefined) {
      result.subjects.push(value);
      continue;
    }
    if ((value = stripPrefix(word, ['title:', '标题:', '匹配:'])) !== undefined) {
      result.include.push(value);
      continue;
    }
    if (word.startsWith('+')) {
      result.keywords.push(word.slice(1));
      continue;
    }
    if ((value = stripPrefix(word, ['include:', '包含:'])) !== undefined) {
      result.keywords.push(value);
      continue;
    }
    if (word.startsWith('!') || word.startsWith('！') || word.startsWith('-')) {
      result.exclude.push(word.slice(1));
      continue;
    }
    if ((value = stripPrefix(word, ['exclude:', '排除:'])) !== undefined) {
      result.exclude.push(value);
      continue;
    }
    if ((value = stripPrefix(word, ['user:', 'publisher:', '发布:', '发布者:', '发布人:'])) !== undefined) {
      result.publishers.push(value);
      continue;
    }
    if ((value = stripPrefix(word, ['team:', 'fansub:', '字幕:', '字幕组:'])) !== undefined) {
      result.fansubs.push(value);
      continue;
    }
    if ((value = stripPrefix(word, ['after:', '开始:', '晚于:'])) !== undefined) {
      const t = new Date(value);
      if (!Number.isNaN(t.getTime())) result.after = t;
      continue;
    }
    if (word.startsWith('>=') || word.startsWith('>')) {
      const t = new Date(word.slice(word.startsWith('>=') ? 2 : 1));
      if (!Number.isNaN(t.getTime())) result.after = t;
      continue;
    }
    if ((value = stripPrefix(word, ['before:', '结束:', '早于:'])) !== undefined) {
      const t = new Date(value);
      if (!Number.isNaN(t.getTime())) result.before = t;
      continue;
    }
    if (word.startsWith('<=') || word.startsWith('<')) {
      const t = new Date(word.slice(word.startsWith('<=') ? 2 : 1));
      if (!Number.isNaN(t.getTime())) result.before = t;
      continue;
    }
    if ((value = stripPrefix(word, ['type:', '类型:'])) !== undefined) {
      result.types.push(value);
      continue;
    }
    if ((value = stripPrefix(word, ['preset:', '预设:'])) !== undefined) {
      // match display name or key
      for (const [key, display] of Object.entries(PRESET_DISPLAY_NAME)) {
        if (value === key || value === display) {
          result.preset = key;
          break;
        }
      }
      continue;
    }
    result.search.push(rawWord.startsWith('+') ? rawWord : rawWord);
  }

  // search words fold into include when include/keywords/exclude present
  if (result.include.length > 0 || result.keywords.length > 0 || result.exclude.length > 0) {
    result.include.push(...result.search);
    result.search = [];
  }

  return result;
}

// Convert a parsed input into ResolvedFilterOptions.
export function toFilterOptions(parsed: ParsedSearchInput): ResolvedFilterOptions {
  const filter: ResolvedFilterOptions = {};
  if (parsed.subjects.length) filter.subjects = parsed.subjects.map(Number).filter((n) => !Number.isNaN(n));
  if (parsed.search.length) filter.search = parsed.search;
  if (parsed.include.length) filter.include = parsed.include;
  if (parsed.keywords.length) filter.keywords = parsed.keywords;
  if (parsed.exclude.length) filter.exclude = parsed.exclude;
  if (parsed.publishers.length) filter.publishers = parsed.publishers;
  if (parsed.fansubs.length) filter.fansubs = parsed.fansubs;
  if (parsed.types.length) filter.types = parsed.types;
  if (parsed.after) filter.after = parsed.after;
  if (parsed.before) filter.before = parsed.before;
  if (parsed.preset) filter.preset = parsed.preset as any;
  return filter;
}

// quote a word when it contains whitespace
function quoteWord(word: string) {
  if (/\s/.test(word)) {
    return `"${word.replace(/"/g, '\\"')}"`;
  }
  return word;
}

// stringifySearchText: filter -> human readable search input text.
export function stringifySearchText(
  searchParams: URLSearchParams,
  subjectNames: Map<number, string> = new Map()
): string {
  const parts: string[] = [];
  const subjects = searchParams.getAll('subject');
  if (subjects.length === 1) {
    const id = Number(subjects[0]);
    const name = subjectNames.get(id);
    if (name) {
      parts.push(`动画:${/\s/.test(name) ? `"${name}"` : name}`);
    }
  }
  const search = searchParams.getAll('search');
  if (search.length > 0) {
    parts.push(...search.map(quoteWord));
  } else {
    const include = searchParams.getAll('include');
    if (include.length) parts.push(...include.map((w) => `标题:${quoteWord(w)}`));
    const keywords = searchParams.getAll('keyword');
    if (keywords.length) parts.push(...keywords.map((w) => `包含:${quoteWord(w)}`));
  }
  const exclude = searchParams.getAll('exclude');
  if (exclude.length) parts.push(...exclude.map((w) => `排除:${quoteWord(w)}`));
  const publishers = searchParams.getAll('publisher');
  if (publishers.length) parts.push(...publishers.map((p) => `发布者:${quoteWord(p)}`));
  const fansubs = searchParams.getAll('fansub');
  if (fansubs.length) parts.push(...fansubs.map((f) => `字幕组:${quoteWord(f)}`));
  const types = searchParams.getAll('type');
  if (types.length) parts.push(...types.map((t) => `类型:${quoteWord(t)}`));
  const after = searchParams.get('after');
  if (after) {
    const d = new Date(Number(after));
    if (!Number.isNaN(d.getTime())) {
      // a date exactly at T16:00:00.000Z prints as yyyy-MM-dd
      if (d.toISOString() === `${d.toISOString().slice(0, 10)}T16:00:00.000Z`) {
        parts.push(`开始:${d.toISOString().slice(0, 10)}`);
      } else {
        parts.push(`开始:${d.toISOString()}`);
      }
    }
  }
  const before = searchParams.get('before');
  if (before) {
    const d = new Date(Number(before));
    if (!Number.isNaN(d.getTime())) {
      if (d.toISOString() === `${d.toISOString().slice(0, 10)}T16:00:00.000Z`) {
        parts.push(`结束:${d.toISOString().slice(0, 10)}`);
      } else {
        parts.push(`结束:${d.toISOString()}`);
      }
    }
  }
  const preset = searchParams.get('preset');
  if (preset) {
    parts.push(`预设:${(PRESET_DISPLAY_NAME as any)[preset] ?? preset}`);
  }
  return parts.join(' ');
}

// removeQuote: strip surrounding quotes (used by the filter card display)
export function removeQuote(text: string[]) {
  return text.map((t) => t.replace(/^"+|"+$/g, ''));
}

// Direct detail URL matching (resolveSearchURL)
export function matchDirectDetailURL(input: string): { provider: string; providerId: string } | undefined {
  // dmhy: https://share.dmhy.org/topics/view/12345_name.html or bare id_name.html
  const dmhy = input.match(
    /^(?:https:\/\/share\.dmhy\.org\/topics\/view\/)?(\d+_[a-zA-Z0-9_\-]+\.html)$/
  );
  if (dmhy) return { provider: 'dmhy', providerId: dmhy[1] };
  const mikan = input.match(
    /^(?:(?:https?:\/\/mikanani\.kas\.pub\/Home\/Episode\/|\/Home\/Episode\/)?)([0-9a-fA-F]{40})$/
  );
  if (mikan) return { provider: 'mikan', providerId: mikan[1].toLowerCase() };
  return undefined;
}

export function isDirectDetailURL(input: string) {
  return matchDirectDetailURL(input.trim()) !== undefined;
}
