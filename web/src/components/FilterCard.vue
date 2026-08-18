<script setup lang="ts">
// FilterCard mirroring pages/resources.($page)/Filter.tsx + FilterOperations.
import { computed, ref, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { toast } from 'vue-sonner';
import type { Resource, ResolvedFilterOptions } from '../api/client';
import { stringifyURLSearch } from '../api/client';
import {
  resolveFilterOptions,
  generateCurlCode,
  generateJavaScriptCode,
  generatePythonCode,
  generateIframeCode
} from '../utils/code-generator';
import { DisplayTypeColor, PRESET_DISPLAY_NAME } from '../utils/constants';
import { formatChinaTime } from '../utils/date';
import { useCollectionsStore, useSidebarStore } from '../stores';

const props = defineProps<{
  filter?: ResolvedFilterOptions;
  feedURL?: string;
  resources?: Resource[];
  complete?: boolean;
}>();

const router = useRouter();
const route = useRoute();
const collectionsStore = useCollectionsStore();
const sidebar = useSidebarStore();

const dropdownOpen = ref(false);
const dropRef = ref<HTMLElement | null>(null);

const resolved = computed(() =>
  props.filter ? resolveFilterOptions(props.filter) : ({} as ResolvedFilterOptions)
);

const subjectNames = ref<Map<number, string>>(new Map());

const hasFilters = computed(() => {
  const f = props.filter;
  if (!f) return false;
  return (
    (f.types?.length ?? 0) > 0 ||
    (f.subjects?.length ?? 0) > 0 ||
    (f.fansubs?.length ?? 0) > 0 ||
    (f.publishers?.length ?? 0) > 0 ||
    (f.search?.length ?? 0) > 0 ||
    (f.include?.length ?? 0) > 0 ||
    (f.keywords?.length ?? 0) > 0 ||
    !!f.before ||
    !!f.after
  );
});

const showOperations = computed(() => {
  const f = props.filter;
  if (!f) return false;
  return (
    (f.search?.length ?? 0) !== 0 ||
    (f.include?.length ?? 0) !== 0 ||
    (f.keywords?.length ?? 0) !== 0 ||
    (subjectNames.value.size > 0 && (f.subjects?.length ?? 0) > 0)
  );
});

watch(
  () => props.filter?.subjects,
  async (subjects) => {
    const ids = subjects ?? [];
    for (const id of ids) {
      try {
        const resp = await fetch(`/subject/${id}`);
        const data = await resp.json();
        if (data.ok && data.data) {
          subjectNames.value.set(id, data.data.alias?.zh?.[0] || data.data.title || String(id));
        }
      } catch {
        // ignore
      }
    }
  },
  { immediate: true }
);

function removeQuote(text: string[]) {
  return text.map((t) => t.replace(/^"+|"+$/g, ''));
}

async function copyText(text: string, success: string, error: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success(success, { dismissible: true, duration: 3000, closeButton: true });
  } catch {
    toast.error(error, { dismissible: true, duration: 3000, closeButton: true });
  }
}

const copyRSS = async () => {
  if (!props.feedURL) return;
  await copyText(props.feedURL, '复制 RSS 订阅成功', '复制 RSS 订阅失败');
};

const copyAllMagnetLinks = async () => {
  const magnetLinks = (props.resources ?? [])
    .map((r) => r.magnet + (r.tracker ?? ''))
    .join('\n');
  if (magnetLinks.length === 0 || (props.resources?.length ?? 0) === 0) {
    toast.error('没有磁力链接', { dismissible: true, duration: 3000, closeButton: true });
    return;
  }
  await copyText(
    magnetLinks,
    `成功复制 ${props.resources!.length} 条磁力链接`,
    '没有磁力链接'
  );
};

const copyJSONData = async () => {
  const json = JSON.stringify({ filter: props.filter, resources: props.resources }, null, 2);
  await copyText(json, '复制 JSON 数据成功', '复制 JSON 数据失败');
};

const copyFetchCurl = async () => {
  const code = generateCurlCode({ filter: props.filter });
  await copyText(code, '复制 cURL 命令成功', '复制 cURL 命令失败');
};

const copyFetchJS = async () => {
  const code = generateJavaScriptCode({ filter: props.filter });
  await copyText(code, '复制 @animegarden/client JavaScript 代码成功', '复制 JavaScript 代码失败');
};

const copyFetchPython = async () => {
  const code = generatePythonCode({ filter: props.filter });
  await copyText(code, '复制 Python 代码成功', '复制 Python 代码失败');
};

const copyIframe = async () => {
  const code = generateIframeCode({ filter: props.filter });
  await copyText(code, '复制网页嵌入代码成功', '复制网页嵌入代码失败');
};

const addToCollection = async () => {
  if (!props.filter) return;
  const collection = collectionsStore.currentCollection;
  if (!collection) return;
  const params = '?' + stringifyURLSearch(resolved.value).toString();
  if (!collection.filters.find((i) => i.searchParams === params)) {
    await collectionsStore.addCollectionItem({ ...resolved.value, name: '', searchParams: params });
    toast.success(`成功添加到 ${collection.name}`, {
      dismissible: true,
      duration: 3000,
      closeButton: true
    });
  } else {
    toast.warning(`已添加到 ${collection.name}`, {
      dismissible: true,
      duration: 3000,
      closeButton: true
    });
  }
  sidebar.open();
};

const clickOutsideHandler = (e: MouseEvent) => {
  if (dropRef.value && !dropRef.value.contains(e.target as Node)) {
    dropdownOpen.value = false;
  }
};

watch(dropdownOpen, (open) => {
  if (open) {
    document.addEventListener('click', clickOutsideHandler);
  } else {
    document.removeEventListener('click', clickOutsideHandler);
  }
});
</script>

<template>
  <div v-if="props.filter && hasFilters" class="mb4 p-4 lt-sm:px-3 w-full bg-zinc-50 dark:bg-zinc-800 drop-shadow rounded-md space-y-2">
    <!-- preset -->
    <div v-if="resolved.preset" class="space-x-2 text-0">
      <span class="text-4 text-base-800 font-bold mr2 select-none keyword">预设</span>
      <span class="text-4 select-text">{{ (PRESET_DISPLAY_NAME as any)[resolved.preset] }}</span>
    </div>

    <!-- subjects -->
    <div v-if="(resolved.subjects ?? []).length > 0" class="space-x-2 text-0">
      <span class="text-4 text-base-800 font-bold mr2 select-none keyword">动画</span>
      <span v-for="sub in resolved.subjects" :key="sub" class="text-4 select-text text-base-900 text-link">
        <router-link :to="`/subject/${sub}`">{{ subjectNames.get(sub) ?? sub }}</router-link>
      </span>
    </div>

    <!-- types -->
    <div v-if="(resolved.types ?? []).length > 0" class="space-x-2 text-0">
      <span class="text-4 text-base-800 font-bold mr2 select-none keyword">类型</span>
      <span
        v-for="type in resolved.types"
        :key="type"
        class="text-4 select-text text-base-600"
        :class="DisplayTypeColor[type]"
      >
        {{ type }}
      </span>
    </div>

    <!-- publishers -->
    <div v-if="(resolved.publishers ?? []).length > 0" class="space-x-2 text-0">
      <span class="text-4 text-base-800 font-bold mr2 select-none keyword">发布者</span>
      <router-link
        v-for="publisher in resolved.publishers"
        :key="publisher"
        :to="{ path: '/resources/1', query: { publisher } }"
        class="text-4 select-text text-link"
      >
        {{ publisher }}
      </router-link>
    </div>

    <!-- fansubs -->
    <div v-if="(resolved.fansubs ?? []).length > 0" class="space-x-2 text-0">
      <span class="text-4 text-base-800 font-bold mr2 select-none keyword">字幕组</span>
      <router-link
        v-for="fansub in resolved.fansubs"
        :key="fansub"
        :to="{ path: '/resources/1', query: { fansub } }"
        class="text-4 select-text text-link"
      >
        {{ fansub }}
      </router-link>
    </div>

    <!-- after / before -->
    <div v-if="resolved.after" class="space-x-2 select-none text-0">
      <span class="text-4 text-base-800 font-bold mr2 keyword">搜索开始于</span>
      <span class="text-4 select-text">{{ formatChinaTime(resolved.after, 'yyyy 年 M 月 d 日 HH:mm') }}</span>
    </div>
    <div v-if="resolved.before" class="space-x-2 select-none text-0">
      <span class="text-4 text-base-800 font-bold mr2 keyword">搜索结束于</span>
      <span class="text-4 select-text">{{ formatChinaTime(resolved.before, 'yyyy 年 M 月 d 日 HH:mm') }}</span>
    </div>

    <!-- search -->
    <div v-if="(resolved.search ?? []).length > 0" class="space-x-2 text-0">
      <span class="text-4 select-none text-base-800 font-bold mr2 keyword">标题搜索</span>
      <span
        v-for="i in removeQuote(resolved.search ?? [])"
        :key="i"
        class="text-4 select-text underline underline-dotted underline-gray-500"
      >
        {{ i }}
      </span>
    </div>

    <!-- include -->
    <div v-if="(resolved.search ?? []).length === 0 && (resolved.include ?? []).length > 0" class="space-x-2 text-0">
      <span class="text-4 select-none text-base-800 font-bold mr2 keyword">标题匹配</span>
      <template v-for="(i, idx) in resolved.include ?? []" :key="idx">
        <span v-if="idx > 0" class="text-base-400 text-4 select-none mr2">|</span>
        <span class="text-4 select-text underline underline-dotted underline-gray-500">{{ i }}</span>
      </template>
    </div>

    <!-- keywords -->
    <div v-if="(resolved.search ?? []).length === 0 && (resolved.keywords ?? []).length > 0" class="space-x-2 select-none text-0">
      <span class="text-4 text-base-800 font-bold mr2 keyword">包含关键词</span>
      <template v-for="(i, idx) in resolved.keywords ?? []" :key="idx">
        <span v-if="idx > 0" class="text-base-400 text-4 select-none mr2">&</span>
        <span class="text-4 select-text underline underline-dotted underline-gray-500">{{ i }}</span>
      </template>
    </div>

    <!-- exclude -->
    <div v-if="(resolved.search ?? []).length === 0 && (resolved.exclude ?? []).length > 0" class="space-x-2 text-0">
      <span class="text-4 select-none text-base-800 font-bold mr2 keyword inline-block">排除关键词</span>
      <span v-for="i in resolved.exclude ?? []" :key="i" class="text-4 select-text">{{ i }}</span>
    </div>

    <!-- operations -->
    <div
      v-if="showOperations"
      class="flex items-center gap-4 lt-sm:gap-2 pt-4"
    >
      <button
        class="inline-flex items-center gap-1 px-3 py-1.5 text-sm rounded-md border border-zinc-300 dark:border-zinc-600 hover:bg-zinc-100 dark:hover:bg-zinc-700 add-collection"
        @click="addToCollection"
      >
        <span>🔖</span>
        <span>添加到收藏夹</span>
      </button>

      <div ref="dropRef" class="inline-flex w-fit divide-x rounded-md relative">
        <button
          class="inline-flex items-center gap-1 px-3 py-1.5 text-sm rounded-none rounded-s-md border border-zinc-300 dark:border-zinc-600 hover:bg-zinc-100 dark:hover:bg-zinc-700"
          @click="copyRSS"
        >
          <span>📋</span>
          <span>复制 RSS 订阅链接</span>
        </button>
        <button
          class="px-2 py-1.5 text-sm border border-zinc-300 dark:border-zinc-600 border-l-0 rounded-none rounded-e-md hover:bg-zinc-100 dark:hover:bg-zinc-700"
          @click="dropdownOpen = !dropdownOpen"
        >
          <span class="text-xl">⌄</span>
        </button>
        <div
          v-if="dropdownOpen"
          class="absolute right-0 top-full mt-1 z-50 w-[200px] rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-lg py-1"
        >
          <button class="w-full text-left px-3 py-2 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" @click="copyAllMagnetLinks">
            <span>🧲</span><span class="ml-1">复制所有磁力链接</span>
          </button>
          <button class="w-full text-left px-3 py-2 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" @click="copyJSONData">
            <span>{}ᵀ</span><span class="ml-1">复制 JSON 数据</span>
          </button>
          <button class="w-full text-left px-3 py-2 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" @click="copyFetchCurl">
            <span>⌘</span><span class="ml-1">复制为 cURL 命令</span>
          </button>
          <button class="w-full text-left px-3 py-2 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" @click="copyFetchJS">
            <span>JS</span><span class="ml-1">复制为 JavaScript 代码</span>
          </button>
          <button class="w-full text-left px-3 py-2 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" @click="copyFetchPython">
            <span>PY</span><span class="ml-1">复制为 Python 代码</span>
          </button>
          <button class="w-full text-left px-3 py-2 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" @click="copyIframe">
            <span>&lt;/&gt;</span><span class="ml-1">复制为网页嵌入代码</span>
          </button>
        </div>
      </div>

      <a
        href="https://docs.animes.garden/animegarden/search.html"
        target="_blank"
        class="lt-sm:hidden text-2xl text-link-active"
        title="搜索帮助"
      >
        ?
      </a>
    </div>
  </div>
</template>