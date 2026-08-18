<script setup lang="ts">
// Home page mirroring pages/_index/route.tsx: season header + bangumi
// resources table.
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { fetchResources, type Resource } from '../api/client';
import ResourcesTable from '../components/ResourcesTable.vue';
import { getActiveSeason, getCalendarSeason } from '../utils/constants';
import { formatInTimeZone } from '../utils/date';

const router = useRouter();

const season = getActiveSeason();
const seasonMeta = getCalendarSeason(season);

const loading = ref(true);
const resources = ref<Resource[]>([]);
const complete = ref(false);
const timestamp = ref<Date | undefined>(undefined);
const failed = ref(false);

// current weekday string like "星期四"
const weekdayText = computed(() => {
  try {
    const date = new Date();
    const weekdays = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'];
    const weekday = formatInTimeZone(date, 'Asia/Shanghai', 'yyyy-MM-dd');
    const [y, m, d] = weekday.split('-').map(Number);
    return weekdays[new Date(Date.UTC(y, m - 1, d)).getUTCDay()];
  } catch {
    return '';
  }
});

async function load() {
  loading.value = true;
  failed.value = false;
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

onMounted(load);

function goCalendar() {
  router.push(`/calendar/${season}`);
}
</script>

<template>
  <div class="w-full pt-13 pb-24">
    <div class="mb-4 flex items-center justify-between">
      <div>
        <div class="text-2xl font-quicksand font-bold">
          {{ seasonMeta.emoji }} {{ season }} · {{ seasonMeta.name }} 放送中
        </div>
        <div class="mt-1 text-sm text-base-500">{{ weekdayText }}</div>
      </div>
      <button class="text-link text-sm" @click="goCalendar">动画周历 →</button>
    </div>

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