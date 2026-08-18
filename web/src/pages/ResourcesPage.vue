<script setup lang="ts">
// Resources page mirroring pages/resources.($page)/route.tsx.
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { parseURLSearch, fetchResources, type Resource, type ResolvedFilterOptions } from '../api/client';
import FilterCard from '../components/FilterCard.vue';
import ResourcesTable from '../components/ResourcesTable.vue';

const route = useRoute();

const page = computed(() => {
  const p = route.params.page;
  const n = p ? Number(p) : 1;
  return Number.isNaN(n) || n < 1 ? 1 : n;
});

const loading = ref(true);
const error = ref<any>(undefined);
const resources = ref<Resource[]>([]);
const complete = ref(false);
const filter = ref<ResolvedFilterOptions | undefined>(undefined);
const timestamp = ref<Date | undefined>(undefined);
const feedURL = computed(() => `/feed.xml?${route.fullPath.split('?')[1] ?? ''}`);

async function load() {
  loading.value = true;
  error.value = undefined;
  const url = new URL(window.location.href);
  // The original forces pageSize 30 on the outgoing request
  const { filter: parsedFilter, pagination } = parseURLSearch(url.searchParams);
  const resp = await fetchResources({
    ...parsedFilter,
    page: page.value,
    pageSize: 30,
    tracker: true
  });
  if (resp.ok) {
    resources.value = resp.resources;
    complete.value = resp.pagination?.complete ?? false;
    filter.value = resp.filter;
    timestamp.value = resp.timestamp;
    document.title = `${generateTitle(filter.value)} | Anime Garden 動漫花園資源網第三方镜像站`;
  } else {
    error.value = resp.error;
  }
  loading.value = false;
}

function generateTitle(f: ResolvedFilterOptions | undefined): string {
  if (!f) return '所有资源';
  if (f.subjects?.length) return '动画 最新动画资源';
  if (f.search?.length) return f.search.join(' ') + ' 最新动画资源';
  if (f.include?.length) return f.include[0] + ' 最新动画资源';
  if (f.fansubs?.length === 1) return f.fansubs[0] + ' 最新动画资源';
  if (f.publishers?.length === 1) return f.publishers[0] + ' 最新动画资源';
  if (f.types?.length === 1) return `最新${f.types[0]}资源`;
  return '所有资源';
}

onMounted(load);
watch(() => route.fullPath, load);
</script>

<template>
  <div class="w-full pt-13 pb-24">
    <template v-if="!error">
      <FilterCard
        :filter="filter"
        :feed-url="feedURL"
        :resources="resources"
        :complete="complete"
      />
      <ResourcesTable
        :resources="resources"
        :page="page"
        :complete="complete"
        :timestamp="timestamp"
      />
    </template>
    <div v-else class="flex flex-col items-center justify-center py-32">
      <div class="text-6xl mb-4">❌</div>
      <div class="text-lg text-base-600">加载资源失败</div>
      <div class="mt-2 text-sm text-base-400">{{ String(error?.message ?? error) }}</div>
      <router-link to="/" class="mt-4 text-link">返回主页</router-link>
    </div>
  </div>
</template>