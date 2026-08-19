<script setup lang="ts">
// Sidebar mirroring apps/web/src/layouts/Sidebar (trigger + wrapper +
// QuickLinks + collection items with hover menu / tooltip / rename).
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { toast } from 'vue-sonner';
import { useSidebarStore, useCollectionsStore } from '../stores';
import { stringifySearchText } from '../utils/search';
import { formatChinaTime } from '../utils/date';
import { DisplayTypeColor } from '../utils/constants';

const sidebar = useSidebarStore();
const collectionsStore = useCollectionsStore();
const route = useRoute();

const subjectNames = ref(new Map<number, string>());
const activeMenu = ref<string | null>(null);
const activeTooltip = ref<string | null>(null);

const collection = computed(() => collectionsStore.currentCollection);

const activeSeason = ref<string | null>(null);

onMounted(async () => {
  await collectionsStore.load();
  try {
    const resp = await fetch('/bgmx/calendar');
    const data = await resp.json();
    if (data.ok && data.data?.seasons?.length > 0) {
      activeSeason.value = data.data.seasons[0];
    }
  } catch {
    // ignore
  }
  // subject names for collection item titles
  const subjectIds = new Set<number>();
  for (const f of collection.value?.filters ?? []) {
    const subjects = f.subjects ?? f.subject;
    if (Array.isArray(subjects)) subjects.forEach((s: any) => subjectIds.add(Number(s)));
    else if (subjects !== undefined) subjectIds.add(Number(subjects));
  }
  for (const id of subjectIds) {
    try {
      const resp = await fetch(`/bgmx/subject/${id}`);
      const data = await resp.json();
      if (data.ok && data.data) {
        const subj = data.data;
        subjectNames.value.set(id, subj.alias?.zh?.[0] || subj.title || String(id));
      }
    } catch {
      // ignore
    }
  }
});

// infer item title like useInferCollectionItemName
function inferItemTitle(item: any): string {
  if (item.name) return item.name;
  if (item.subjects && item.subjects.length === 1) {
    const id = Number(item.subjects[0]);
    const name = subjectNames.value.get(id);
    if (name) return name;
  }
  if (item.search?.length > 0) return item.search.join(' ');
  if (item.include?.length > 0) return item.include.join(' ');
  const params = new URLSearchParams(item.searchParams ?? '');
  return stringifySearchText(params, subjectNames.value);
}

function itemTitle(item: any): string {
  if (item.name) return item.name;
  const name = inferItemTitle(item);
  const fansubs = item.fansubs?.length ? item.fansubs.join(' ') : '';
  if (name && fansubs) return `${name} 字幕组:${fansubs}`;
  return name;
}

function itemLink(item: any) {
  return {
    path: '/resources/1',
    query: Object.fromEntries(new URLSearchParams(item.searchParams ?? ''))
  };
}

function isActive(item: any): boolean {
  return route.fullPath.includes(item.searchParams ?? '\u0000');
}

const activeTab = computed(() => {
  const pathname = route.path;
  if (pathname.startsWith('/resources/')) {
    for (const item of collection.value?.filters ?? []) {
      if (route.fullPath.includes(item.searchParams ?? '\u0000')) return item.searchParams;
    }
    return 'resources';
  }
  if (pathname === '/anime' || pathname.startsWith('/calendar/')) return 'anime';
  if (pathname === '/') return 'index';
  return undefined;
});

async function removeItem(item: any) {
  await collectionsStore.removeCollectionItem(item.searchParams);
  toast.success('删除成功', { dismissible: true, duration: 3000, closeButton: true });
}

async function copyItemRSS(item: any) {
  try {
    const url = `/feed.xml${item.searchParams ?? ''}`;
    await navigator.clipboard.writeText(url);
    toast.success('复制 RSS 订阅成功', { dismissible: true, duration: 3000, closeButton: true });
  } catch {
    toast.error('复制 RSS 订阅失败', { dismissible: true, duration: 3000, closeButton: true });
  }
}

function openInNewTab(item: any) {
  window.open(`/resources/1${item.searchParams ?? ''}`);
}

// rename state
const renaming = ref<string | null>(null);
const renameInput = ref<HTMLInputElement | null>(null);

function startRename(item: any) {
  renaming.value = item.searchParams;
  activeMenu.value = null;
  setTimeout(() => renameInput.value?.focus(), 50);
}

async function commitRename(item: any) {
  const value = renameInput.value?.value ?? '';
  renaming.value = null;
  if (value && value !== item.name) {
    await collectionsStore.updateCollectionItem({ ...item, name: value });
  }
}

// filter tooltip content
function filterLines(item: any): Array<{ label: string; values: string[]; colored?: boolean }> {
  const lines: Array<{ label: string; values: string[]; colored?: boolean }> = [];
  if (item.name) lines.push({ label: '条件别名', values: [item.name] });
  if (item.subjects?.length) {
    lines.push({
      label: '动画',
      values: item.subjects.map((s: number) => subjectNames.value.get(s) ?? String(s))
    });
  }
  if (item.types?.length) lines.push({ label: '类型', values: item.types, colored: true });
  if (item.search?.length) lines.push({ label: '标题搜索', values: item.search });
  if (item.include?.length) lines.push({ label: '标题匹配', values: item.include });
  if (item.keywords?.length) lines.push({ label: '包含关键词', values: item.keywords });
  if (item.exclude?.length) lines.push({ label: '排除关键词', values: item.exclude });
  if (item.fansubs?.length) lines.push({ label: '字幕组', values: item.fansubs });
  if (item.after) {
    lines.push({
      label: '搜索开始于',
      values: [formatChinaTime(new Date(item.after), 'yyyy 年 M 月 d 日 HH:mm')]
    });
  }
  if (item.before) {
    lines.push({
      label: '搜索结束于',
      values: [formatChinaTime(new Date(item.before), 'yyyy 年 M 月 d 日 HH:mm')]
    });
  }
  return lines;
}
</script>

<template>
  <div class="sidebar-root">
    <!-- trigger -->
    <div
      v-if="!sidebar.isOpen"
      class="sidebar-trigger font-quicksand font-medium"
      title="打开收藏夹"
      @click="sidebar.open()"
    >
      <span class="mr-1">🔖</span>
      <span class="text-sm">收藏夹</span>
    </div>

    <!-- content -->
    <div v-if="sidebar.isOpen" class="sidebar-wrapper space-y-2">
      <!-- header row -->
      <div
        class="mt-[8px] px-2 py-1 text-base-700 select-none font-medium font-quicksand flex items-center"
      >
        <div class="block">
          <span class="mr-1">🔖</span>
          <span class="text-sm font-bold">收藏夹</span>
        </div>
        <div class="flex-auto"></div>
        <div
          class="h-[26px] w-auto rounded-md px-1 flex items-center cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800"
          title="关闭侧边栏"
          @click="sidebar.close()"
        >
          <span class="text-sm">✕</span>
        </div>
      </div>

      <!-- quick links -->
      <template v-if="collection">
        <router-link
          :to="activeSeason ? `/calendar/${activeSeason}` : '/anime'"
          class="ml-1 mr-2 px-1 py-2 cursor-pointer select-none text-sm text-base-700 flex items-center hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-md"
          :class="activeTab === 'anime' && 'bg-zinc-100 dark:bg-zinc-800'"
        >
          <span class="mr-1">📅</span>
          <span>动画周历</span>
        </router-link>
        <router-link
          to="/resources/1"
          class="ml-1 mr-2 px-1 py-2 cursor-pointer select-none text-sm text-base-700 flex items-center hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-md"
          :class="activeTab === 'resources' && 'bg-zinc-100 dark:bg-zinc-800'"
        >
          <span class="mr-1">📋</span>
          <span>所有资源</span>
        </router-link>
        <a
          href="https://docs.animes.garden/animegarden/search"
          target="_blank"
          class="ml-1 mr-2 px-1 py-2 cursor-pointer select-none text-sm text-base-700 flex items-center hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-md"
        >
          <span class="mr-1">❓</span>
          <span>高级搜索帮助</span>
        </a>
      </template>

      <!-- collection items -->
      <div v-if="collection && collection.filters.length > 0" class="px-1 pb-2">
        <div
          v-for="item in collection.filters"
          :key="item.searchParams"
          class="collection-item group relative hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-md text-base-800 text-xs"
          :class="isActive(item) && 'bg-zinc-100 dark:bg-zinc-800'"
          @mouseenter="activeTooltip = item.searchParams"
          @mouseleave="activeTooltip = null"
        >
          <router-link :to="itemLink(item)" class="block w-full pl-2 pr-6 py-1">
            <template v-if="renaming !== item.searchParams">
              <span class="block truncate">{{ itemTitle(item) }}</span>
            </template>
            <input
              v-else
              ref="renameInput"
              :value="item.name"
              class="w-full bg-transparent outline outline-1 outline-zinc-300 dark:outline-zinc-600 rounded px-1"
              @keydown.enter.prevent="commitRename(item)"
              @blur="commitRename(item)"
            />
          </router-link>

          <!-- hover menu -->
          <div
            v-if="renaming !== item.searchParams"
            class="hidden group-hover:flex absolute h-full top-0 right-[4px] py-[1px] items-center"
          >
            <button
              class="w-[16px] h-full flex items-center justify-center hover:bg-zinc-200 dark:hover:bg-zinc-700 rounded-md font-bold"
              @click="activeMenu = activeMenu === item.searchParams ? null : item.searchParams"
            >
              ⋯
            </button>
            <div
              v-if="activeMenu === item.searchParams"
              class="absolute right-0 top-full z-50 w-[180px] rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-lg py-1 text-sm"
              @click.stop
            >
              <button
                class="w-full text-left px-3 py-1.5 hover:bg-zinc-100 dark:hover:bg-zinc-800"
                @click="openInNewTab(item)"
              >
                在新页面中打开
              </button>
              <button
                class="w-full text-left px-3 py-1.5 hover:bg-zinc-100 dark:hover:bg-zinc-800"
                @click="copyItemRSS(item)"
              >
                复制 RSS 订阅链接
              </button>
              <div class="border-t border-zinc-100 dark:border-zinc-800 my-1"></div>
              <button
                class="w-full text-left px-3 py-1.5 hover:bg-zinc-100 dark:hover:bg-zinc-800"
                @click="startRename(item)"
              >
                重命名
              </button>
              <div class="border-t border-zinc-100 dark:border-zinc-800 my-1"></div>
              <button
                class="w-full text-left px-3 py-1.5 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
                @click="removeItem(item)"
              >
                删除
              </button>
            </div>
          </div>

          <!-- filter tooltip -->
          <div
            v-if="activeTooltip === item.searchParams && renaming !== item.searchParams"
            class="hidden group-hover:block absolute left-full top-0 z-50 ml-2 w-[220px] rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-lg p-3 space-y-1 text-xs pointer-events-none"
          >
            <div v-for="line in filterLines(item)" :key="line.label" class="flex gap-2">
              <span class="font-bold select-none shrink-0">{{ line.label }}</span>
              <span
                v-for="v in line.values"
                :key="v"
                class="select-text"
                :class="line.colored ? DisplayTypeColor[v] : 'text-base-600'"
              >
                {{ v }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- empty state -->
      <div v-if="collection && collection.filters.length === 0" class="px-2">
        <router-link
          to="/resources/1?search=动画&type=动画"
          class="h-[80px] px-2 flex items-center justify-center text-base-700 text-link-active"
        >
          <span class="text-sm">收藏一个搜索条件吧</span>
          <span>↗</span>
        </router-link>
      </div>
    </div>
  </div>
</template>
