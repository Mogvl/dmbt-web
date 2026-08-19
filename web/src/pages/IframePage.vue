<script setup lang="ts">
// Iframe embed page mirroring pages/iframe/route.tsx: shell-less resources
// list used for embedding.
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { parseURLSearch, fetchResources, type Resource } from '../api/client';
import { IFRAME_TITLE } from '../utils/constants';
import ResourcesTable from '../components/ResourcesTable.vue';

const route = useRoute();

const page = computed(() => {
  const p = route.query.page;
  const n = p ? Number(p) : 1;
  return Number.isNaN(n) || n < 1 ? 1 : n;
});

const loading = ref(true);
const resources = ref<Resource[]>([]);
const complete = ref(false);
const timestamp = ref<Date | undefined>(undefined);
const error = ref<any>(undefined);

async function load() {
  loading.value = true;
  const url = new URL(window.location.href);
  const { filter } = parseURLSearch(url.searchParams);
  const resp = await fetchResources({ ...filter, page: page.value, pageSize: 30, tracker: true });
  if (resp.ok) {
    resources.value = resp.resources;
    complete.value = resp.pagination?.complete ?? false;
    timestamp.value = resp.timestamp;
  } else {
    error.value = resp.error;
  }
  loading.value = false;
  document.title = `${title.value} | ${IFRAME_TITLE}`;
}

onMounted(load);
watch(() => route.fullPath, load);

const title = computed(() => {
  const f = route.query;
  if (f.search) return `${f.search} 最新动画资源`;
  if (f.include) return `${f.include} 最新动画资源`;
  if (f.fansub) return `${f.fansub} 最新动画资源`;
  if (f.type) return `最新${f.type}资源`;
  return '所有资源';
});
</script>

<template>
  <div class="w-full min-h-screen">
    <div class="p-4">
      <div class="mb-4 text-lg font-bold">{{ title }}</div>
      <div v-if="loading" class="py-12 flex justify-center">
        <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
      </div>
      <ResourcesTable
        v-else
        :resources="resources"
        :page="page"
        :complete="complete"
        :timestamp="timestamp"
      />
    </div>
  </div>
</template>