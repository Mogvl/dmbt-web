<script setup lang="ts">
// Detail page mirroring pages/detail.$provider.$providerId/route.tsx +
// FileTree.tsx: download card (PikPak), magnet + original link, HTML
// description, publisher/fansub avatars, published time, file tree card.
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { fetchResourceDetail, type ResourceDetail } from '../api/client';
import { formatInTimeZone } from '../utils/date';
import { getPikPakUrlChecker, truncate } from '../utils/constants';

const route = useRoute();

const provider = computed(() => String(route.params.provider));
const providerId = computed(() => String(route.params.providerId));

const loading = ref(true);
const resource = ref<any>(null);
const detail = ref<ResourceDetail | null>(null);
const error = ref<any>(undefined);

// magnet = resource.magnet || first magnet: url of detail
const magnet = computed(() => {
  if (resource.value?.magnet) return resource.value.magnet;
  const m = detail.value?.magnets.find((m) => m.url.startsWith('magnet:'));
  return m?.url ?? '';
});
const pikpakUrl = computed(() => (magnet.value ? getPikPakUrlChecker(magnet.value) : ''));

// only one 磁力链接 entry: magnet + tracker
const magnetLine = computed(() =>
  magnet.value ? `${resource.value?.magnet ?? ''}${resource.value?.tracker ?? ''}` : ''
);

function splitMagnetURL(m: string) {
  return m?.split('&')[0] ?? '';
}

// description: strip 簡介: prefix like the original
const descriptionHtml = computed(() => {
  const desc = detail.value?.description ?? '';
  return desc.replace(
    /(<strong>)?簡介:(&nbsp;)*(<\/strong>)?(<br>)?(<hr>)?/,
    '<h2 class="text-xl font-bold">简介</h2>'
  );
});

async function load() {
  loading.value = true;
  error.value = undefined;
  const resp = await fetchResourceDetail(provider.value, providerId.value);
  if (resp.ok) {
    resource.value = resp.resource;
    detail.value = resp.detail;
    if (resp.resource) {
      document.title = truncate(resp.resource.title, 70);
    }
  } else {
    error.value = resp.error;
  }
  loading.value = false;
}

onMounted(load);
watch(() => route.fullPath, load);

const publishedText = computed(() =>
  resource.value
    ? formatInTimeZone(new Date(resource.value.createdAt), 'Asia/Shanghai', 'yyyy-MM-dd HH:mm')
    : ''
);

// --- file tree ---
interface FileNode {
  name: string;
  size: string;
  children: FileNode[];
}

function getDirTree(files: Array<{ name: string; size: string }>): FileNode[] {
  const root: FileNode[] = [];
  const dirs: Record<string, FileNode> = {};
  for (const file of files) {
    const parts = file.name.split('/');
    let node = root;
    let path = '';
    for (let i = 0; i < parts.length - 1; i++) {
      path += (path ? '/' : '') + parts[i];
      let dir = dirs[path];
      if (!dir) {
        dir = { name: parts[i], size: '', children: [] };
        dirs[path] = dir;
        node.push(dir);
      }
      node = dir.children;
    }
    node.push({ name: parts[parts.length - 1], size: file.size, children: [] });
  }
  return root;
}

const tree = computed(() => getDirTree(detail.value?.files ?? []));

function fileIcon(name: string) {
  const ext = name.split('.').pop()?.toLowerCase();
  switch (ext) {
    case 'mp4':
    case 'mkv':
      return '▶';
    case 'ass':
      return '字';
    case 'rar':
    case '7z':
    case 'zip':
      return '🗜';
    default:
      return '📄';
  }
}
</script>

<template>
  <div class="w-full pt-13 pb-24">
    <div v-if="loading" class="py-24 flex justify-center">
      <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
    </div>

    <div v-else-if="error || !resource" class="py-24 text-center">
      <div class="text-2xl text-base-500 mb-3">资源不存在或已删除</div>
      <div v-if="error" class="text-sm text-base-400">{{ String(error?.message ?? error) }}</div>
      <router-link to="/" class="text-link mt-4 inline-block">返回主页</router-link>
    </div>

    <template v-else>
      <div class="detail mt-4vh w-full space-y-4">
        <h1 class="text-xl font-bold resource-title">
          <span>{{ resource.title }}</span>
        </h1>

        <!-- download card -->
        <div class="download-link rounded-md shadow-box">
          <h2 class="text-lg font-bold border-b px-4 py-2 flex items-center">
            <a
              :href="pikpakUrl"
              target="_blank"
              class="play text-link-active underline underline-dotted underline-offset-6"
            >
              下载链接
            </a>
          </h2>
          <div class="p-4 space-y-1 overflow-auto whitespace-nowrap">
            <div class="flex">
              <span class="text-base-600 select-none inline-block w-[160px] min-w-[160px] lt-sm:w-[120px] lt-sm:min-w-[120px]">
                <a
                  :href="pikpakUrl"
                  target="_blank"
                  class="play text-link-active underline underline-dotted underline-offset-6"
                >
                  在线播放
                </a>
              </span>
              <a :href="pikpakUrl" target="_blank" class="play text-link inline-block flex-1 pr-4">
                使用 PikPak 播放
              </a>
            </div>
            <div class="flex">
              <span class="text-base-600 select-none inline-block w-[160px] min-w-[160px] lt-sm:w-[120px] lt-sm:min-w-[120px]">
                磁力链接
              </span>
              <a :href="magnetLine" class="download text-link inline-block flex-1 pr-4">
                {{ splitMagnetURL(magnetLine) }}
              </a>
            </div>
            <div class="flex">
              <span class="text-base-600 select-none inline-block w-[160px] min-w-[160px] lt-sm:w-[120px] lt-sm:min-w-[120px]">
                原链接
              </span>
              <a :href="resource.href" target="_blank" class="text-link inline-block flex-1 pr-4">
                {{ resource.href }}
              </a>
            </div>
          </div>
        </div>

        <!-- description -->
        <div
          v-if="descriptionHtml"
          class="description"
          v-html="descriptionHtml"
        ></div>

        <!-- publisher / fansub -->
        <div class="publisher">
          <h2 class="text-lg font-bold pb-4">{{ resource.fansub ? '发布者 / 字幕组' : '发布者' }}</h2>
          <div class="flex gap-8">
            <div>
              <router-link
                :to="{ path: '/resources/1', query: { publisher: resource.publisher?.name ?? '' } }"
                class="block text-left"
              >
                <img
                  :src="resource.publisher?.avatar || 'https://share.dmhy.org/images/defaultUser.png'"
                  :alt="`${resource.publisher?.name} avatar`"
                  class="inline-block w-[100px] h-[100px] rounded"
                  @error="(e: Event) => ((e.target as HTMLImageElement).src = 'https://share.dmhy.org/images/defaultUser.png')"
                />
                <span class="text-link block mt-2">{{ resource.publisher?.name }}</span>
              </router-link>
            </div>
            <div v-if="resource.fansub">
              <router-link
                :to="{ path: '/resources/1', query: { fansub: resource.fansub.name } }"
                class="block w-auto text-left"
              >
                <img
                  :src="resource.fansub.avatar || 'https://share.dmhy.org/images/defaultUser.png'"
                  :alt="`${resource.fansub.name} avatar`"
                  class="inline-block w-[100px] h-[100px] rounded"
                  @error="(e: Event) => ((e.target as HTMLImageElement).src = 'https://share.dmhy.org/images/defaultUser.png')"
                />
                <span class="text-link block mt-2">{{ resource.fansub.name }}</span>
              </router-link>
            </div>
          </div>
        </div>

        <div>
          <span class="font-bold">发布于&nbsp;</span>
          <span>{{ publishedText }}</span>
        </div>

        <!-- files card -->
        <div class="file-list rounded-md shadow-box">
          <h2 class="text-lg font-bold border-b px-4 py-2">文件列表</h2>
          <div class="mb-4 max-h-[80vh] overflow-auto px-4 py-4 space-y-2">
            <div v-for="item in tree" :key="item.name">
              <div class="flex items-center gap-4">
                <div class="flex items-center gap-1">
                  <span>{{ item.children.length > 0 ? '📁' : fileIcon(item.name) }}</span>
                  <div class="text-sm text-base-600">{{ item.name }}</div>
                </div>
                <div class="flex-auto"></div>
                <div v-if="item.children.length === 0" class="text-xs text-base-400 select-none">
                  {{ item.size }}
                </div>
              </div>
              <div v-if="item.children.length > 0" class="my-1 pl-4 py-1 space-y-2 border-l border-l-1">
                <div v-for="child in item.children" :key="child.name">
                  <div class="flex items-center gap-4">
                    <div class="flex items-center gap-1">
                      <span>{{ child.children.length > 0 ? '📁' : fileIcon(child.name) }}</span>
                      <div class="text-sm text-base-600">{{ child.name }}</div>
                    </div>
                    <div class="flex-auto"></div>
                    <div v-if="child.children.length === 0" class="text-xs text-base-400 select-none">
                      {{ child.size }}
                    </div>
                  </div>
                  <div v-if="child.children.length > 0" class="my-1 pl-4 py-1 space-y-2 border-l border-l-1">
                    <div v-for="leaf in child.children" :key="leaf.name">
                      <div class="flex items-center gap-4">
                        <div class="flex items-center gap-1">
                          <span>{{ fileIcon(leaf.name) }}</span>
                          <div class="text-sm text-base-600">{{ leaf.name }}</div>
                        </div>
                        <div class="flex-auto"></div>
                        <div class="text-xs text-base-400 select-none">{{ leaf.size }}</div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="(detail?.files ?? []).length === 0" class="py-2 select-none text-center text-red-400">
              种子信息解析失败
            </div>
            <div v-if="detail?.hasMoreFiles" class="text-base-400">...</div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>