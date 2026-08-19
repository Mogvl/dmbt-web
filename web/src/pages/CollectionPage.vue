<script setup lang="ts">
// Collection page mirroring pages/collection.$hash/route.tsx.
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { fetchCollection } from '../api/client';
import ResourcesTable from '../components/ResourcesTable.vue';
import { stringifySearchText } from '../utils/search';
import { SITE_TITLE } from '../utils/constants';

const route = useRoute();
const router = useRouter();

const hash = computed(() => String(route.params.hash));

const loading = ref(true);
const data = ref<any>(null);
const subjectNames = ref(new Map<number, string>());

async function load() {
  loading.value = true;
  const resp = await fetchCollection(hash.value);
  if (resp.ok) {
    data.value = resp;
    document.title = `${resp.name || '收藏夹'} | ${SITE_TITLE}`;
  } else {
    router.replace('/');
    return;
  }
  loading.value = false;
}

onMounted(load);
watch(() => route.fullPath, load);

const results = computed(() => data.value?.results ?? []);

const feedURL = computed(() => `/collection/${hash.value}/feed.xml`);

const filterText = (filter: any) => {
  if (!filter) return '';
  const params = new URLSearchParams();
  if (filter.types) filter.types.forEach((t: string) => params.append('type', t));
  if (filter.fansubs) filter.fansubs.forEach((f: string) => params.append('fansub', f));
  if (filter.publishers) filter.publishers.forEach((p: string) => params.append('publisher', p));
  if (filter.subjects) filter.subjects.forEach((s: number) => params.append('subject', String(s)));
  if (filter.search) filter.search.forEach((s: string) => params.append('search', s));
  if (filter.include) filter.include.forEach((s: string) => params.append('include', s));
  if (filter.keywords) filter.keywords.forEach((s: string) => params.append('keyword', s));
  if (filter.exclude) filter.exclude.forEach((s: string) => params.append('exclude', s));
  return stringifySearchText(params, subjectNames.value);
};
</script>

<template>
  <div class="w-full pt-13 pb-24">
    <div v-if="loading" class="py-24 flex justify-center">
      <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
    </div>

    <template v-else-if="data">
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ data.name || `收藏夹 ${hash}` }}</h1>
          <div class="mt-1 text-sm text-base-400">
            创建于 {{ new Date(data.createdAt).toLocaleString() }} · {{ results.length }} 个搜索条件
          </div>
        </div>
        <a
          :href="feedURL"
          target="_blank"
          class="inline-flex items-center gap-1 px-3 py-1.5 rounded-md border border-zinc-300 dark:border-zinc-600 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800 text-[#ee802f]"
        >
          📡 RSS
        </a>
      </div>

      <div class="space-y-8">
        <div v-for="(result, idx) in results" :key="idx">
          <h2 class="text-base font-bold mb-2 flex items-center gap-2">
            <span>{{ result.filter?.fansubs?.join(' ') || '搜索条件' + (idx + 1) }}</span>
            <span class="text-xs font-normal text-base-400">
              {{ filterText(result.filter) }}
            </span>
          </h2>
          <ResourcesTable
            :resources="result.resources ?? []"
            :complete="result.complete ?? true"
          />
        </div>
      </div>
    </template>
  </div>
</template>