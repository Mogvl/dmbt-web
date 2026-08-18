<script setup lang="ts">
// Resources table mirroring components/Resources/table.tsx.
import { computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import type { Resource } from '../api/client';
import { parseSize } from '../api/client';
import { formatChinaTime } from '../utils/date';
import { DisplayTypeColor, getPikPakUrlChecker } from '../utils/constants';
import Pagination from './Pagination.vue';

const props = defineProps<{
  resources: Resource[];
  page?: number;
  complete?: boolean;
  timestamp?: Date;
  displayFansub?: boolean;
}>();

const router = useRouter();
const route = useRoute();

const isDownloadable = (type: string) =>
  ['动画', '合集', '日剧', '特摄'].includes(type);

const typeIcon = (type: string) => {
  const icons: Record<string, string> = {
    动画: '🎬',
    合集: '📦',
    音乐: '🎵',
    日剧: '📺',
    RAW: '📼',
    漫画: '📚',
    游戏: '🎮',
    特摄: '🎥',
    其他: '📄'
  };
  return icons[type] ?? '📄';
};

// followSearch: keep current query and override one param
function followSearch(params: Record<string, string>) {
  const q = { ...route.query };
  for (const [k, v] of Object.entries(params)) q[k] = v;
  delete q.page;
  return q;
}

const magnetWithTracker = (r: Resource) => r.magnet + (r.tracker ?? '');

const firstPageLink = computed(() => {
  const path = route.fullPath.replace(/\/resources\/\d+/, '');
  return '/resources/1' + path;
});


</script>

<template>
  <div>
    <div class="overflow-y-auto w-full">
      <table class="resources-table border-collapse min-y-[1080px] w-full">
        <colgroup>
          <col class="text-left xl:min-w-[600px] lg:min-w-[480px]" />
          <col class="w-max whitespace-nowrap" />
          <col class="w-max whitespace-nowrap" />
        </colgroup>
        <thead class="resources-table-head border-b-2 text-lg lt-lg:text-base">
          <tr>
            <th class="py3 pl3 lt-sm:pl1 text-left xl:min-w-[600px] lg:min-w-[480px]">
              <div class="flex">
                <div class="flex-shrink-0 mr3 flex justify-center items-center w-[32px]">
                  <span class="i-carbon:categories text-xl"></span>
                </div>
                <div>资源</div>
              </div>
            </th>
            <th v-if="props.displayFansub !== false" class="py3 min-w-[60px]">发布者</th>
            <th class="py3 px2 text-center w-[72px]">播放</th>
          </tr>
        </thead>
        <tbody class="resources-table-body divide-y border-b text-base lt-lg:text-sm">
          <tr v-for="r in resources" :key="r.provider + '/' + r.providerId" class="">
            <td class="py2 pl3 lt-md:pl1">
              <div class="flex xl:min-w-[600px] lg:min-w-[480px] lt-lg:w-[calc(95vw-4px)]">
                <div class="flex-shrink-0 mr3 flex justify-center items-center">
                  <button
                    class="flex items-center justify-center h-[32px] w-[32px] rounded-full bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700"
                    :class="DisplayTypeColor[r.type]"
                    @click="router.push({ path: '/resources/1', query: followSearch({ type: r.type }) })"
                  >
                    {{ typeIcon(r.type) }}
                  </button>
                </div>
                <div>
                  <div class="flex items-center justify-start">
                    <div class="flex-1">
                      <span class="mr3">
                        <template v-if="isDownloadable(r.type)">
                          <a
                            :href="getPikPakUrlChecker(r.magnet)"
                            target="_blank"
                            class="text-link mr1"
                            :aria-label="`Go to download resource of ${r.title}`"
                          >
                            {{ r.title }}
                          </a>
                        </template>
                        <template v-else>
                          <router-link
                            :to="`/detail/${r.provider}/${r.providerId}`"
                            class="text-link"
                            :aria-label="`Go to resource detail of ${r.title}`"
                          >
                            {{ r.title }}
                          </router-link>
                        </template>
                      </span>
                    </div>
                  </div>
                  <div class="mt1 flex items-center gap-4">
                    <router-link
                      :to="`/detail/${r.provider}/${r.providerId}`"
                      class="text-link-secondary-hover-base text-xs text-zinc-400"
                    >
                      发布于 {{ formatChinaTime(new Date(r.createdAt)) }}
                    </router-link>
                    <a
                      :href="magnetWithTracker(r)"
                      class="text-link-secondary-hover-base text-xs text-zinc-400"
                      aria-label="Download resource"
                    >
                      大小 {{ parseSize(r.size) }}
                    </a>
                    <router-link
                      :to="`/detail/${r.provider}/${r.providerId}`"
                      class="text-link-secondary text-xs"
                      :aria-label="`Go to resource detail of ${r.title}`"
                    >
                      ↗ <span class="more">详情</span>
                    </router-link>
                  </div>
                </div>
              </div>
            </td>
            <td v-if="props.displayFansub !== false" class="py2 px2 lt-sm:px0">
              <div class="flex justify-center items-center">
                <template v-if="r.fansub">
                  <router-link
                    :to="{ path: '/resources/1', query: followSearch({ fansub: r.fansub.name }) }"
                    class="block w-max"
                    :aria-label="`Go to resources list of fansub ${r.fansub.name}`"
                  >
                    <span
                      class="text-xs inline-block px-2 py-1 rounded bg-gray-100 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700"
                    >
                      {{ r.fansub.name }}
                    </span>
                  </router-link>
                </template>
                <template v-else-if="r.publisher">
                  <router-link
                    :to="{ path: '/resources/1', query: followSearch({ publisher: r.publisher.name }) }"
                    class="block w-max"
                    :aria-label="`Go to resources list of publisher ${r.publisher.name}`"
                  >
                    <span
                      class="text-xs inline-block px-2 py-1 rounded bg-gray-100 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700"
                    >
                      {{ r.publisher.name }}
                    </span>
                  </router-link>
                </template>
              </div>
            </td>
            <td class="py2 px2 w-[72px]">
              <div class="flex gap1 items-center justify-start">
                <a
                  :href="getPikPakUrlChecker(r.magnet)"
                  target="_blank"
                  class="play text-xl text-base-500 hover:text-base-900"
                  aria-label="Play resource"
                  :title="r.title"
                >
                  ▶
                </a>
                <a
                  :href="magnetWithTracker(r)"
                  class="download text-xl text-base-500 hover:text-base-900"
                  aria-label="Download resource"
                  :title="r.title"
                >
                  ⬇
                </a>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- empty -->
    <div v-if="props.page !== undefined && resources.length === 0">
      <div class="h-20 text-2xl text-orange-600/80 flex items-center justify-center">
        <span class="mr-2">⚠</span>
        <span>没有搜索到匹配的资源</span>
      </div>
      <div class="flex items-center justify-center">
                <template v-if="!route.path.endsWith('/1')">
          <router-link :to="firstPageLink" class="text-link">
            第 1 页
          </router-link>
          <span>&nbsp;/&nbsp;</span>
        </template>
        <router-link to="/" class="text-link">主页</router-link>
      </div>
    </div>

    <!-- pagination -->
    <Pagination
      v-if="props.page !== undefined && !props.complete && resources.length > 0"
      :page="props.page"
      :complete="props.complete ?? false"
      :timestamp="props.timestamp"
    />
  </div>
</template>