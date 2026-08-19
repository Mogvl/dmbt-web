<script setup lang="ts">
// Search dialog mirroring apps/web/src/layouts/Search/Search.tsx (cmdk).
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import {
  parseSearchInput,
  toFilterOptions,
  matchDirectDetailURL,
  stringifySearchText
} from '../utils/search';
import { stringifyURLSearch, fetchResources, type Resource } from '../api/client';
import { useHistoriesStore } from '../stores';

const props = defineProps<{}>();

const router = useRouter();
const route = useRoute();
const histories = useHistoriesStore();

const input = ref('');
const searchActive = ref(false);
const selectedIndex = ref(-1);
const loading = ref(false);
const results = ref<Resource[]>([]);
const resultsEmpty = ref(false);

const inputEl = ref<HTMLInputElement | null>(null);

const parsed = computed(() => parseSearchInput(input.value));
const filterOptions = computed(() => toFilterOptions(parsed.value));

const isValidSearch = computed(() => {
  const p = parsed.value;
  return (
    p.search.length > 0 ||
    p.include.length > 0 ||
    p.keywords.length > 0 ||
    p.subjects.length > 0
  );
});

const isDirectDetail = computed(() => matchDirectDetailURL(input.value.trim()) !== undefined);

// keyboard shortcuts: s / / Ctrl/Cmd+K
const onKeydown = (e: KeyboardEvent) => {
  const target = e.target as HTMLElement;
  if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
    if ((e.key === 'Escape') && searchActive.value) {
      close();
    }
    return;
  }
  const isSlash = e.key === '/';
  const isS = e.key === 's' && !e.ctrlKey && !e.metaKey && !e.altKey;
  const isCmdK = (e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k';
  if (isSlash || isS || isCmdK) {
    e.preventDefault();
    open();
  }
  if (e.key === 'Escape' && searchActive.value) {
    close();
  }
};

const open = () => {
  searchActive.value = true;
  setTimeout(() => inputEl.value?.focus(), 10);
};

const close = () => {
  searchActive.value = false;
};

// sync input with the current route search when location changes
watch(
  () => route.fullPath,
  () => {
    if (route.path === '/resources/1') {
      const q = route.query;
      const params = new URLSearchParams();
      for (const [k, v] of Object.entries(q)) {
        if (Array.isArray(v)) v.forEach((item) => params.append(k, item));
        else if (v !== undefined) params.append(k, v);
      }
      input.value = stringifySearchText(params);
    }
  }
);

// fetch results with debounce
let timer: number | undefined;
watch(
  [input, isValidSearch],
  () => {
    if (!isValidSearch.value || isDirectDetail.value) {
      results.value = [];
      resultsEmpty.value = false;
      return;
    }
    loading.value = true;
    window.clearTimeout(timer);
    timer = window.setTimeout(async () => {
      const filter = filterOptions.value;
      const resp = await fetchResources({ ...filter, page: 1, pageSize: 5 });
      results.value = resp.resources;
      resultsEmpty.value = resp.ok && resp.resources.length === 0;
      loading.value = false;
    }, 150);
  },
  { immediate: true }
);

function selectGoToSearch(text: string, source: string) {
  const inputText = text.trim();
  if (!inputText) return;
  histories.push(inputText);
  const direct = matchDirectDetailURL(inputText);
  if (direct) {
    router.push(`/detail/${direct.provider}/${direct.providerId}`);
    close();
    return;
  }
  const filter = toFilterOptions(parseSearchInput(inputText));
  const params = stringifyURLSearch(filter);
  router.push({ path: '/resources/1', query: Object.fromEntries(params) });
  close();
}

// subject search suggestion
const subjectKeywords = computed(() => {
  const p = parsed.value;
  return [...p.search, ...p.include].map((s) => s.trim()).filter(Boolean);
});

const subjectSuggestions = ref<Array<{ id: number; name: string }>>([]);

watch(
  subjectKeywords,
  async (keywords) => {
    if (keywords.length === 0) {
      subjectSuggestions.value = [];
      return;
    }
    try {
      const resp = await fetch(
        `/bgmx/subjects?q=${encodeURIComponent(keywords.join(' '))}&limit=20&cursor=0`
      );
      const data = await resp.json();
      if (data.ok) {
        const list: any[] = Array.isArray(data.data) ? data.data : [];
        subjectSuggestions.value = list.slice(0, 3).map((s) => ({
          id: s.id,
          name: s.alias?.zh?.[0] || s.title || String(s.id)
        }));
      }
    } catch {
      subjectSuggestions.value = [];
    }
  },
  { immediate: true }
);

function goSubject(id: number, name: string) {
  histories.push(`动画:${name}`);
  router.push(`/subject/${id}`);
  close();
}

function openHelp() {
  window.open('https://docs.animes.garden/animegarden/search.html');
}

function appendSuffix(suffix: string) {
  input.value = (input.value + suffix).trimStart();
  inputEl.value?.focus();
}

const completionItems = [
  { text: '包含关键词', suffix: ' 包含:' },
  { text: '排除关键词', suffix: ' 排除:' },
  { text: '筛选字幕组', suffix: ' 字幕组:' },
  { text: '匹配标题', suffix: ' 标题:' },
  { text: '上传时间晚于', suffix: ' 晚于:' },
  { text: '上传时间早于', suffix: ' 早于:' },
  { text: '筛选资源类型', suffix: ' 类型:' }
];

function goSearchHistory(h: string) {
  selectGoToSearch(h, 'history');
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown);
  window.clearTimeout(timer);
});
</script>

<template>
  <div class="relative h-full" v-show="true">
    <!-- input trigger -->
    <div
      class="relative cursor-text"
      @click="open"
    >
      <input
        ref="inputEl"
        v-model="input"
        type="text"
        :placeholder="'搜索资源...'"
        class="w-full h-[44.4px] px-5 pr-12 text-base rounded-full bg-white/55 dark:bg-white/5 border border-white/60 dark:border-white/10 backdrop-blur-xl shadow-soft outline-none placeholder:text-zinc-400 dark:placeholder:text-zinc-500 text-zinc-800 dark:text-zinc-100 transition-shadow focus:shadow-lift"
        @keydown.enter.prevent="selectGoToSearch(input, 'button')"
        @focus="searchActive = true"
      />
      <button
        class="absolute right-3 top-1/2 -translate-y-1/2 text-base-500 hover:text-accent transition-colors"
        title="搜索"
        @mousedown.prevent="selectGoToSearch(input, 'button')"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
      </button>
    </div>

    <!-- dropdown -->
    <div
      v-if="searchActive"
      class="glass-menu absolute top-[54px] left-0 right-0 z-50 overflow-hidden"
    >
      <!-- go-to-search -->
      <div
        v-if="input.trim()"
        class="px-4 py-3 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm"
        @click="selectGoToSearch(input, 'command')"
      >
        <span v-if="!isDirectDetail">在本页列出 {{ input }} 的搜索结果...</span>
        <span v-else>前往 {{ input }}</span>
      </div>

      <!-- subject suggestions -->
      <div v-if="input.trim() && subjectSuggestions.length > 0" class="border-t border-zinc-100 dark:border-zinc-800">
        <div class="px-4 pt-2 pb-1 text-xs text-zinc-400">动画</div>
        <div
          v-for="sub in subjectSuggestions"
          :key="sub.id"
          class="px-4 py-2 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm"
          @click="goSubject(sub.id, sub.name)"
        >
          {{ sub.name }}
        </div>
      </div>

      <!-- search results -->
      <div v-if="input.trim() && isValidSearch && !isDirectDetail" class="border-t border-zinc-100 dark:border-zinc-800">
        <div class="px-4 pt-2 pb-1 text-xs text-zinc-400">搜索结果</div>
        <div v-if="loading" class="px-4 py-3 text-sm text-zinc-400 flex items-center gap-2">
          <span class="lds-ring"><div></div><div></div><div></div><div></div></span>
          正在搜索 {{ input }} ...
        </div>
        <div v-else-if="resultsEmpty" class="px-4 py-3 text-sm text-zinc-400">
          没有搜索到任何匹配的资源.
        </div>
        <div v-else>
          <div
            v-for="r in results"
            :key="`${r.provider}/${r.providerId}`"
            class="px-4 py-2 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm truncate"
            @click="router.push(`/detail/${r.provider}/${r.providerId}`); close()"
          >
            {{ r.title }}
          </div>
          <div
            v-if="results.length > 0"
            class="px-4 py-2 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm text-link"
            @click="selectGoToSearch(input, 'result-more')"
          >
            展示更多 {{ input }} 的搜索结果...
          </div>
        </div>
      </div>

      <!-- histories -->
      <div
        v-if="!input.trim() && histories.histories.length > 0"
        class="border-t border-zinc-100 dark:border-zinc-800"
      >
        <div class="px-4 pt-2 pb-1 flex items-center justify-between text-xs text-zinc-400">
          <span>搜索历史</span>
          <button class="hover:text-red-500" @click="histories.clear()">清空</button>
        </div>
        <div
          v-for="h in histories.histories"
          :key="h"
          class="px-4 py-2 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm flex items-center justify-between group"
          @click="goSearchHistory(h)"
        >
          <span class="truncate">{{ h.replace(/"/g, '') }}</span>
          <button class="text-zinc-400 hover:text-red-500 opacity-0 group-hover:opacity-100" @click.stop="histories.remove(h)">✕</button>
        </div>
      </div>

      <!-- advanced search completions -->
      <div class="border-t border-zinc-100 dark:border-zinc-800">
        <div class="px-4 pt-2 pb-1 text-xs text-zinc-400">高级搜索</div>
        <div
          v-for="item in completionItems"
          :key="item.text"
          class="px-4 py-2 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm"
          @click="appendSuffix(item.suffix)"
        >
          {{ item.text }}
        </div>
        <div
          class="px-4 py-2 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm"
          @click="openHelp()"
        >
          高级搜索帮助
        </div>
      </div>
    </div>
  </div>
</template>