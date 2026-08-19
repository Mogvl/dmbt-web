<script setup lang="ts">
// Subject page mirroring pages/subject.$subject.($page)/route.tsx.
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { fetchResources, type Resource, type ResolvedFilterOptions } from '../api/client';
import ResourcesTable from '../components/ResourcesTable.vue';
import { fansubs, SITE_TITLE } from '../utils/constants';
import { stringifyURLSearch } from '../api/client';

const route = useRoute();
const router = useRouter();

const subjectId = computed(() => Number(route.params.subject));
const page = computed(() => {
  const p = route.params.page;
  const n = p ? Number(p) : 1;
  return Number.isNaN(n) || n < 1 ? 1 : n;
});

const subject = ref<any>(null);
const loading = ref(true);
const resources = ref<Resource[]>([]);
const complete = ref(false);
const timestamp = ref<Date | undefined>(undefined);
const error = ref<any>(undefined);
const subjectNames = ref<string[]>([]);

const displayName = computed(() => subject.value?.alias?.zh?.[0] || subject.value?.title || '');
const poster = computed(() =>
  subject.value?.poster ||
  subject.value?.bangumi?.images?.large ||
  `https://bgm.animes.garden/bangumi/subject/${subjectId.value}/poster.jpeg?quality=large`
);

async function load() {
  loading.value = true;
  error.value = undefined;
  try {
    const sResp = await fetch(`/subject/${subjectId.value}`);
    const sData = await sResp.json();
    if (sData.ok && sData.data) {
      subject.value = sData.data;
      const names = new Set<string>([sData.data.title]);
      if (sData.data.alias?.zh?.[0]) names.add(sData.data.alias.zh[0]);
      (sData.data.search?.include ?? []).forEach((n: string) => names.add(n));
      subjectNames.value = [...names];
      document.title = `${displayName.value} 最新资源 | ${SITE_TITLE}`;
    } else {
      subject.value = null;
    }
  } catch {
    subject.value = null;
  }

  const filter: ResolvedFilterOptions = { subjects: [subjectId.value] };
  const resp = await fetchResources({ ...filter, page: page.value, pageSize: 30, tracker: true });
  if (resp.ok) {
    resources.value = resp.resources;
    complete.value = resp.pagination?.complete ?? false;
    timestamp.value = resp.timestamp;
  } else {
    error.value = resp.error;
  }
  loading.value = false;
}

onMounted(load);
watch(() => route.fullPath, load);

// grouped sections by the 33 hard-coded fansub names (original behavior)
const grouped = computed(() => {
  const groups: { fansub: string; list: Resource[] }[] = [];
  const rest: Resource[] = [];
  for (const fansub of fansubs) {
    const list = resources.value.filter((r) => r.fansub?.name === fansub);
    if (list.length > 0) groups.push({ fansub, list });
  }
  for (const r of resources.value) {
    if (!fansubs.includes(r.fansub?.name ?? '')) rest.push(r);
  }
  if (rest.length > 0) groups.push({ fansub: '其他', list: rest });
  return groups;
});

function goFallbackSearch() {
  const params = stringifyURLSearch({ include: subjectNames.value });
  router.push(`/resources/1?${params.toString()}`);
}
</script>

<template>
  <div class="w-full pt-13 pb-24">
    <div v-if="loading" class="py-24 flex justify-center">
      <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
    </div>

    <div v-else-if="!subject" class="py-24 text-center">
      <div class="text-2xl text-base-500">未找到该动画</div>
    </div>

    <template v-else>
      <!-- subject card -->
      <div class="mb-6 flex gap-6 p-4 bg-zinc-50 dark:bg-zinc-800 rounded-md">
        <img
          :src="poster"
          :alt="displayName"
          class="w-[140px] h-[190px] object-cover rounded-md shrink-0 bg-zinc-100 dark:bg-zinc-900"
          loading="lazy"
        />
        <div class="flex-1 min-w-0">
          <h1 class="text-xl font-bold">{{ displayName }}</h1>
          <div v-if="subject?.bangumi?.date" class="mt-2 text-sm text-base-500">
            开播日期: {{ subject.bangumi.date }}
          </div>
          <div v-if="subject?.bangumi?.platform" class="mt-1 text-sm text-base-500">
            平台: {{ subject.bangumi.platform }}
          </div>
          <p v-if="subject?.bangumi?.summary" class="mt-3 text-sm text-base-600 line-clamp-4">
            {{ subject.bangumi.summary }}
          </p>
          <div class="mt-3 text-xs text-base-400">
            #{{ subjectId }} · Bangumi
          </div>
        </div>
      </div>

      <!-- fallback search when empty -->
      <div v-if="resources.length === 0 && !error">
        <div class="h-20 text-2xl text-orange-700/80 flex items-center justify-center gap-2">
          <span>🔍</span>
          <span>暂时未索引到相应资源</span>
        </div>
        <div class="flex items-center justify-center">
          <button class="text-link" @click="goFallbackSearch">前往搜索</button>
        </div>
      </div>

      <!-- grouped table -->
      <div v-for="group in grouped" :key="group.fansub" class="mb-8">
        <h2 class="text-lg font-bold mb-3 flex items-center gap-2">
          <span>{{ group.fansub }}</span>
          <span class="text-sm font-normal text-base-400">{{ group.list.length }} 条</span>
        </h2>
        <ResourcesTable
          :resources="group.list"
          :page="page"
          :complete="complete"
          :timestamp="timestamp"
        />
      </div>
    </template>
  </div>
</template>