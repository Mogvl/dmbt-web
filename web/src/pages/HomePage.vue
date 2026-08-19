<script setup lang="ts">
// Home page mirroring pages/_index/route.tsx: season header + bangumi
// resources table.
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { fetchResources, type Resource } from '../api/client';
import ResourcesTable from '../components/ResourcesTable.vue';
import { getActiveSeason, getCalendarSeason, SITE_TITLE } from '../utils/constants';


const router = useRouter();

const season = ref(getActiveSeason());
const seasonMeta = computed(() => getCalendarSeason(season.value));
const title = computed(() => SITE_TITLE);

const loading = ref(true);
const resources = ref<Resource[]>([]);
const complete = ref(false);
const timestamp = ref<Date | undefined>(undefined);
const failed = ref(false);

async function load() {
  loading.value = true;
  failed.value = false;
  // season comes from the bgm calendar mirror (like the original home loader)
  try {
    const resp = await fetch('/bgmx/calendar');
    const data = await resp.json();
    if (data.ok && data.data?.seasons?.length > 0) {
      season.value = data.data.seasons[0];
    }
  } catch {
    // keep the computed active season
  }
  const resp = await fetchResources({
    preset: 'bangumi',
    types: ['动画'],
    page: 1,
    pageSize: 30,
    tracker: true
  });
  if (resp.ok) {
    resources.value = resp.resources;
    complete.value = resp.pagination?.complete ?? false;
    timestamp.value = resp.timestamp;
  } else {
    failed.value = true;
  }
  loading.value = false;
}

onMounted(() => {
  document.title = title.value;
  load();
});

</script>

<template>
  <div class="w-full pt-13 pb-24">
    <header class="mb-12 lt-sm:mb-6 flex min-h-10 items-end justify-between gap-4 pl-4 lt-md:pl-0 lt-sm:flex-col lt-sm:items-start">
      <div>
        <div class="flex items-baseline gap-4 lt-sm:flex-col lt-sm:items-start lt-sm:gap-2">
          <h1 class="text-3xl lt-sm:text-2xl font-bold leading-tight tracking-normal select-none">
            <router-link :to="`/calendar/${season}`" class="text-inherit text-link-active">
              <span class="anime-season-emoji text-2xl font-quicksand font-bold" aria-hidden="true">{{ seasonMeta.emoji }}</span>{{ seasonMeta.title }}放送中...
            </router-link>
          </h1>
          <router-link :to="`/calendar/${season}`" class="text-link text-base">→ 前往周历</router-link>
        </div>
      </div>
    </header>

    <div v-if="loading" class="py-24 flex justify-center">
      <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
    </div>

    <div v-else-if="failed" class="py-24 flex flex-col items-center">
      <div class="text-base-500 mb-4">加载失败，正在前往动画周历...</div>
      <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
    </div>

    <template v-else>
      <ResourcesTable :resources="resources" :page="1" :complete="complete" :timestamp="timestamp" />
    </template>
  </div>
</template>