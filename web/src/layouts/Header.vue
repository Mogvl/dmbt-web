<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import Dropdown from '../components/Dropdown.vue';
import { fansubs as AllFansubs, types, DisplayTypeColor } from '../utils/constants';
import { useFansubsStore } from '../stores';
import { formatInTimeZone } from '../utils/date';

// Calendar types from the bgm mirror
interface CalendarSubject {
  id: number;
  title: string;
  display_title?: string | null;
  alias?: Record<string, string[]>;
  onair_date?: string | null;
  poster?: string;
  tags?: string[];
}

interface CalendarData {
  seasons: string[];
  calendar: CalendarSubject[][];
  web?: CalendarSubject[];
}

const props = defineProps<{ feedURL?: string }>();

const router = useRouter();
const fansubsStore = useFansubsStore();

const calendarData = ref<CalendarData | null>(null);

const loadCalendar = async () => {
  try {
    const resp = await fetch('/calendar');
    const data = await resp.json();
    if (data.ok) {
      calendarData.value = data.data;
    }
  } catch {
    // ignore
  }
};

onMounted(loadCalendar);

// weekday rotation mirroring getCalendar
const getCalendar = (rawCalendar: CalendarSubject[][], now = new Date()) => {
  let weekday =
    Number(formatInTimeZone(new Date(now.getTime() - 6 * 60 * 60 * 1000), 'Asia/Shanghai', 'i')) - 1;
  const isChina = (bgm: CalendarSubject) => {
    const cn = ['国创', '国产', '国产动画', '国漫', '中国'];
    return (bgm.tags ?? []).some((t) => cn.includes(t)) ? 1 : 0;
  };
  return rawCalendar.map((_, idx) => {
    const index = (idx + weekday) % 7;
    return {
      index: index + 1,
      text: ['一', '二', '三', '四', '五', '六', '日'][index],
      bangumis: (rawCalendar[index] ?? [])
        .filter((b) => !!b.poster)
        .filter((b) => !isChina(b))
        .sort((lhs, rhs) => {
          const lang = isChina(lhs) - isChina(rhs);
          if (lang !== 0) return lang;
          return new Date(rhs.onair_date!).getTime() - new Date(lhs.onair_date!).getTime();
        })
    };
  });
};

const calendar = computed(() =>
  getCalendar(calendarData.value?.calendar ?? [])
);

const getSeason = (data: CalendarData | null) => {
  const seasons = data?.seasons ?? [];
  return seasons.length > 0 ? seasons[0] : undefined;
};

const season = computed(() => getSeason(calendarData.value));

const displayName = (bgm: CalendarSubject) => bgm.display_title || bgm.title || '';

// fansubs ordering: preferred first
const orderedFansubs = computed(() => {
  const set = new Set<string>();
  const out: string[] = [];
  for (const f of fansubsStore.preferFansubs) {
    out.push(f);
    set.add(f);
  }
  for (const f of AllFansubs) {
    if (!set.has(f)) {
      out.push(f);
    }
  }
  return out;
});

const goResources = (page: number, search: Record<string, string> = {}) => {
  const params = new URLSearchParams();
  params.set('page', String(page));
  for (const [k, v] of Object.entries(search)) params.set(k, v);
  router.push({ path: '/resources/1', query: Object.fromEntries(params) });
};
</script>

<template>
  <header
    class="fixed z-13 pt-[1px] flex justify-center items-center w-full h-(--nav-height) pointer-events-none text-base-500"
  >
    <nav class="main flex gap-1 [&>div]:(leading-(--nav-height))">
      <div
        class="box-content w-[32px] pl3 lt-sm:pl1 text-2xl text-center font-quicksand font-bold pointer-events-auto"
      >
        <router-link to="/">🌸</router-link>
      </div>

      <!-- 动画 -->
      <Dropdown
        class="pointer-events-auto [&:hover>a]:bg-zinc-100! dark:[&:hover>a]:bg-zinc-800!"
        trigger="动画"
        :trigger-class="'rounded-md p-2'"
        menu-class="w-[80px] max-h-[600px] lt-sm:max-h-[360px] overflow-y-auto"
      >
        <template #menu>
          <router-link
            v-if="season"
            :to="`/calendar/${season}`"
            class="block px-2 py-1 rounded-t-md hover:bg-zinc-100 dark:hover:bg-zinc-800"
          >
            周历
          </router-link>
          <a v-else href="/anime" class="block px-2 py-1 rounded-t-md hover:bg-zinc-100 dark:hover:bg-zinc-800">
            周历
          </a>
          <template v-for="day in calendar" :key="day.text">
            <div class="relative group">
              <div
                class="px-2 py-1 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800"
              >
                周{{ day.text }}
              </div>
              <div
                class="hidden group-hover:block absolute left-full top-0 z-50 min-h-[100px] max-h-[min(500px,calc(100vh-120px))] lt-sm:max-h-[360px] rounded-md shadow-box bg-white dark:bg-zinc-900 divide-y divide-zinc-100 dark:divide-zinc-800 overflow-y-auto"
              >
                <router-link
                  v-for="bgm in day.bangumis"
                  :key="bgm.id"
                  :to="`/subject/${bgm.id}`"
                  class="block w-[360px] max-w-[calc(100vw-144px)] px-2 py-1 hover:bg-zinc-100 dark:hover:bg-zinc-800 whitespace-nowrap overflow-hidden text-ellipsis"
                >
                  {{ displayName(bgm) }}
                </router-link>
              </div>
            </div>
          </template>
        </template>
      </Dropdown>

      <!-- 字幕组 -->
      <Dropdown
        class="pointer-events-auto [&:hover>a]:bg-zinc-100! dark:[&:hover>a]:bg-zinc-800!"
        trigger="字幕组"
        menu-class="w-[160px] max-h-[494px] overflow-y-auto"
      >
        <template #menu>
          <router-link
            v-for="fansub in orderedFansubs"
            :key="fansub"
            :to="{ path: '/resources/1', query: { fansub } }"
            class="block px-2 py-1 hover:bg-zinc-100 dark:hover:bg-zinc-800 whitespace-nowrap overflow-hidden text-ellipsis"
          >
            {{ fansub }}
          </router-link>
        </template>
      </Dropdown>

      <!-- 资源 -->
      <Dropdown
        class="pointer-events-auto [&:hover>a]:bg-zinc-100! dark:[&:hover>a]:bg-zinc-800!"
        trigger="资源"
        menu-class="w-max overflow-y-auto"
      >
        <template #menu>
          <router-link
            v-for="type in types"
            :key="type"
            :to="{
              path: '/resources/1',
              query: type === '动画' ? { type: '动画', preset: 'bangumi' } : { type }
            }"
            class="flex items-center gap-2 px-2 py-1 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            :class="DisplayTypeColor[type]"
          >
            <span>{{ type }}</span>
          </router-link>
        </template>
      </Dropdown>

      <div class="flex-auto pointer-events-none"></div>
      <div class="lt-md:hidden pointer-events-auto">
        <a
          v-if="props.feedURL"
          :href="props.feedURL"
          target="_blank"
          class="inline cursor-pointer rounded-md p-2 text-[#ee802f] hover:!text-[#ff7800]"
        >
          <span class="text-sm mr-1">📡</span>
          <span class="text-sm">RSS</span>
        </a>
      </div>
    </nav>
  </header>
</template>
