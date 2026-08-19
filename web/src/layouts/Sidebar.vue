<script setup lang="ts">
// Sidebar mirroring apps/web/src/layouts/Sidebar: collection favorites list.
import { computed, onMounted, ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { toast } from 'vue-sonner';
import { useSidebarStore, useCollectionsStore } from '../stores';
import { stringifySearchText } from '../utils/search';
import { formatChinaTime } from '../utils/date';
import { generateCollection } from '../api/client';

const sidebar = useSidebarStore();
const collectionsStore = useCollectionsStore();
const router = useRouter();
const route = useRoute();

const subjectNames = ref(new Map<number, string>());

const collection = computed(() => collectionsStore.currentCollection);

onMounted(async () => {
  await collectionsStore.load();
  // load subject names for the collection filters
  const subjectIds = new Set<number>();
  for (const f of collection.value?.filters ?? []) {
    const subjects = f.subjects ?? f.subject;
    if (Array.isArray(subjects)) subjects.forEach((s: any) => subjectIds.add(Number(s)));
    else if (subjects !== undefined) subjectIds.add(Number(subjects));
  }
  for (const id of subjectIds) {
    try {
      const resp = await fetch(`/subject/${id}`);
      const data = await resp.json();
      if (data.ok && data.data) {
        const subj = data.data;
        subjectNames.value.set(
          id,
          subj.alias?.zh?.[0] || subj.title || String(id)
        );
      }
    } catch {
      // ignore
    }
  }
});

// infer item title like the original
function inferItemTitle(item: any): string {
  if (item.name) return item.name;
  const params = new URLSearchParams(item.searchParams);
  if (params.getAll('subject').length === 1) {
    const id = Number(params.getAll('subject')[0]);
    const name = subjectNames.value.get(id);
    if (name) return name;
  }
  const search = params.getAll('search');
  if (search.length > 0) return search.join(' ');
  const include = params.getAll('include');
  if (include.length > 0) return include.join(' ');
  const fansubs = params.getAll('fansub');
  if (fansubs.length > 0) return fansubs.join(' ');
  return '搜索条件';
}

function formatFilterText(item: any): string {
  const params = new URLSearchParams(item.searchParams ?? '');
  const text = stringifySearchText(params, subjectNames.value);
  return text;
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
}

async function syncCollection() {
  const resp = await collectionsStore.syncToServer();
  if (resp?.ok && resp.hash) {
    toast.success(`已同步到服务器，链接: /collection/${resp.hash}`, {
      dismissible: true,
      duration: 3000,
      closeButton: true
    });
    router.push(`/collection/${resp.hash}`);
  } else {
    toast.error('同步收藏夹失败', {
      dismissible: true,
      duration: 3000,
      closeButton: true
    });
  }
}

function goSearch(text: string) {
  router.push({ path: '/resources/1', query: { search: text } });
}

function itemLink(item: any) {
  return {
    path: '/resources/1',
    query: Object.fromEntries(new URLSearchParams(item.searchParams ?? ''))
  };
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
      收藏夹
    </div>

    <!-- content -->
    <div
      v-if="sidebar.isOpen"
      class="sidebar-wrapper overflow-y-auto"
    >
      <div class="flex items-center justify-between px-4 py-3">
        <span class="text-sm font-bold text-base-900">{{ collection?.name ?? '收藏夹' }}</span>
        <div class="flex items-center gap-2">
          <button class="text-xs text-link" @click="syncCollection">同步</button>
          <button
            class="text-xs text-zinc-400 hover:text-zinc-600"
            title="关闭"
            @click="sidebar.close()"
          >
            ✕
          </button>
        </div>
      </div>

      <div v-if="!collection || collection.filters.length === 0" class="px-4">
        <router-link
          to="/resources/1?search=动画&type=动画"
          class="h-[80px] px-2 flex items-center justify-center text-base-700 text-link-active"
        >
          <span class="text-sm">收藏一个搜索条件吧</span>
          <span>↗</span>
        </router-link>
      </div>

      <div v-else class="divide-y divide-zinc-100 dark:divide-zinc-800">
        <div
          v-for="item in collection.filters"
          :key="item.searchParams"
          class="px-4 py-3"
          :class="isActive(item) ? 'bg-zinc-100 dark:bg-zinc-800' : 'hover:bg-zinc-50 dark:hover:bg-zinc-900'"
        >
          <div class="flex items-start justify-between gap-2">
            <router-link
              :to="itemLink(item)"
              class="text-sm font-medium text-base-900 collection-item-title truncate"
            >
              {{ inferItemTitle(item) }}
            </router-link>
            <button
              class="text-xs text-zinc-400 hover:text-red-500 shrink-0"
              title="删除"
              @click="removeItem(item)"
            >
              ✕
            </button>
          </div>
          <div class="mt-1 text-xs text-zinc-400 truncate">{{ formatFilterText(item) }}</div>
        </div>
      </div>

      <div class="px-4 py-3 text-xs text-zinc-400">
        <div class="flex items-center justify-between">
          <span>搜索历史</span>
        </div>
      </div>
    </div>
  </div>
</template>
