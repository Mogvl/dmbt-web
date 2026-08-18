<script setup lang="ts">
// Calendar page mirroring pages/anime/route.tsx: season dropdowns, sticky
// weekday TOC, scroll-spy, poster grid.
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { CalendarSeasonMonths, getCalendarSeason, getCalendarSeasonHead as getSeasonHead } from '../utils/constants';
import { getWeekday, formatInTimeZone } from '../utils/date';

interface CalendarSubject {
  id: number;
  title: string;
  alias?: Record<string, string[]>;
  display_title?: string | null;
  onair_date?: string | null;
  poster?: string;
  summary?: string;
  platform?: string;
  tags?: string[];
  bangumi?: { platform?: string; images?: { large?: string }; summary?: string; tags?: string[] };
}

interface CalendarData {
  seasons: string[];
  updated_at?: string;
  calendar: CalendarSubject[][];
  web?: CalendarSubject[];
}

const route = useRoute();
const router = useRouter();

const season = computed(() => String(route.params.season ?? ''));
const meta = computed(() => getCalendarSeason(season.value));

const data = ref<CalendarData | null>(null);
const loading = ref(true);

const activeDay = ref<number | null>(null);

async function load() {
  loading.value = true;
  try {
    const resp = await fetch('/calendar');
    const json = await resp.json();
    if (json.ok) {
      data.value = json.data;
    }
  } catch {
    data.value = null;
  }
  loading.value = false;
}

onMounted(load);
watch(season, load);

// The calendar response is global; the season param selects the day offset.
// The original groups by weekday with the 6am rotation rule.
const days = computed(() => {
  const raw = data.value?.calendar ?? [];
  if (raw.length === 0) return [];
  const shanghaiNow = formatInTimeZone(new Date(Date.now() - 6 * 60 * 60 * 1000), 'Asia/Shanghai', 'yyyy-MM-dd');
  const [y, m, d] = shanghaiNow.split('-').map(Number);
  const weekday = new Date(Date.UTC(y, m - 1, d)).getUTCDay();
  return raw.map((_, idx) => {
    const index = (idx + weekday) % 7;
    return {
      index: index + 1,
      text: ['一', '二', '三', '四', '五', '六', '日'][index],
      bangumis: raw[index] ?? []
    };
  });
});

const displayName = (bgm: CalendarSubject) => bgm.display_title || bgm.alias?.zh?.[0] || bgm.title || '';
const poster = (bgm: CalendarSubject) =>
  bgm.poster ||
  bgm.bangumi?.images?.large ||
  `https://bgm.animes.garden/bangumi/subject/${bgm.id}/poster.jpeg?quality=large`;
const onair = (bgm: CalendarSubject) => bgm.onair_date ?? '';
const platform = (bgm: CalendarSubject) => bgm.platform || bgm.bangumi?.platform || '';

const seasonOptions = computed(() => {
  const seasons = data.value?.seasons ?? [];
  if (seasons.length > 0) return seasons;
  const year = new Date().getFullYear();
  return CalendarSeasonMonths.map((m) => `${year}-${String(m).padStart(2, '0')}`);
});

const scrollToDay = (idx: number) => {
  activeDay.value = idx;
  document.getElementById(`day-${idx}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
};

const seasonHead = computed(() => getSeasonHead(season.value));
watch(seasonHead, (h) => {
  document.title = h.title;
}, { immediate: true });
</script>

<template>
  <div class="w-full pt-13 pb-24">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-quicksand font-bold">
        {{ meta.emoji }} {{ meta.title }} 动画周历
      </h1>
      <select
        v-model="season"
        class="rounded-md border border-zinc-300 dark:border-zinc-600 bg-white dark:bg-zinc-900 px-3 py-1.5 text-sm"
      >
        <option v-for="s in seasonOptions" :key="s" :value="s">
          {{ getCalendarSeason(s).title }}
        </option>
      </select>
    </div>

    <div v-if="loading" class="py-24 flex justify-center">
      <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
    </div>

    <div v-else-if="!data" class="py-24 text-center text-base-500">加载动画周历失败</div>

    <div v-else class="flex gap-6">
      <!-- weekday TOC -->
      <div class="sticky top-[80px] self-start w-[80px] shrink-0">
        <div class="space-y-1">
          <button
            v-for="day in days"
            :key="day.index"
            class="block w-full text-center py-1.5 rounded-md text-sm"
            :class="
              activeDay === day.index
                ? 'bg-pink-100 dark:bg-pink-900/30 text-pink-600 font-bold'
                : 'hover:bg-zinc-100 dark:hover:bg-zinc-800 text-base-600'
            "
            @click="scrollToDay(day.index)"
          >
            周{{ day.text }}
          </button>
        </div>
      </div>

      <!-- day groups -->
      <div class="flex-1 space-y-8">
        <div v-for="day in days" :id="`day-${day.index}`" :key="day.index">
          <h2 class="text-lg font-bold mb-3 flex items-center gap-2">
            <span>周{{ day.text }}</span>
            <span class="text-sm font-normal text-base-400">
              {{ day.bangumis.length }} 部
            </span>
          </h2>
          <div v-if="day.bangumis.length === 0" class="text-sm text-base-400 py-8 text-center">
            暂无动画
          </div>
          <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            <router-link
              v-for="bgm in day.bangumis"
              :key="bgm.id"
              :to="`/subject/${bgm.id}`"
              class="group block rounded-lg overflow-hidden border border-zinc-200 dark:border-zinc-800 hover:shadow-lg transition-shadow"
            >
              <div class="aspect-[3/4] overflow-hidden bg-zinc-100 dark:bg-zinc-800">
                <img
                  :src="poster(bgm)"
                  :alt="displayName(bgm)"
                  class="w-full h-full object-cover group-hover:scale-105 transition-transform"
                  loading="lazy"
                  @error="(e: Event) => ((e.target as HTMLImageElement).style.display = 'none')"
                />
              </div>
              <div class="p-2">
                <div class="text-sm font-medium line-clamp-2 min-h-[2.5em]">
                  {{ displayName(bgm) }}
                </div>
                <div v-if="onair(bgm)" class="mt-1 text-xs text-base-400">
                  {{ getWeekday(onair(bgm)) }} {{ onair(bgm) }}
                </div>
                <div v-if="platform(bgm)" class="text-xs text-base-400">{{ platform(bgm) }}</div>
              </div>
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>