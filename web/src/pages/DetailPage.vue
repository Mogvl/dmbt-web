<script setup lang="ts">
// Detail page mirroring pages/detail.$provider.$providerId/route.tsx with the
// file tree.
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { toast } from 'vue-sonner';
import { fetchResourceDetail, parseSize, type ResourceDetail } from '../api/client';
import { formatChinaTime } from '../utils/date';
import { getPikPakUrlChecker } from '../utils/constants';

const route = useRoute();
const router = useRouter();

const provider = computed(() => String(route.params.provider));
const providerId = computed(() => String(route.params.providerId));

const loading = ref(true);
const resource = ref<any>(null);
const detail = ref<ResourceDetail | null>(null);
const isDeleted = ref(false);
const error = ref<any>(undefined);

// file tree state
const expanded = ref<Set<string>>(new Set());
interface FileNode {
  name: string;
  size: string;
  children: FileNode[];
  path?: string;
}
const treeRoots = ref<FileNode[]>([]);

function buildTree(files: Array<{ name: string; size: string }>): FileNode[] {
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
        dir = { name: parts[i], size: '', children: [], path: path };
        dirs[path] = dir;
        node.push(dir);
      }
      node = dir.children;
    }
    node.push({ name: parts[parts.length - 1], size: file.size, children: [], path: file.name });
  }
  return root;
}

const magnetWithTracker = computed(() =>
  resource.value ? resource.value.magnet + (resource.value.tracker ?? '') : ''
);

async function load() {
  loading.value = true;
  error.value = undefined;
  const resp = await fetchResourceDetail(provider.value, providerId.value);
  if (resp.ok) {
    resource.value = resp.resource;
    detail.value = resp.detail;
    isDeleted.value = resp.isDeleted ?? false;
    if (resp.resource) {
      document.title = `${resp.resource.title} | Anime Garden 動漫花園資源網第三方镜像站`;
    }
    if (resp.detail) {
      treeRoots.value = buildTree(resp.detail.files);
    }
  } else {
    error.value = resp.error;
  }
  loading.value = false;
}

onMounted(load);
watch(() => route.fullPath, load);

const toggleDir = (path: string) => {
  const next = new Set(expanded.value);
  if (next.has(path)) next.delete(path);
  else next.add(path);
  expanded.value = next;
};

const isDir = (node: FileNode) => node.children.length > 0;

const copyMagnet = async () => {
  try {
    await navigator.clipboard.writeText(magnetWithTracker.value);
    toast.success('复制磁力链接成功', { duration: 3000 });
  } catch {
    toast.error('复制磁力链接失败', { duration: 3000 });
  }
};

// normalized description (mirrors scraper normalizeDescription)
const normalizedDescription = computed(() => {
  const desc = detail.value?.description ?? '';
  return desc.replace(/<br\s*\/?\s*>/gi, '\n').replace(/<[^>]+>/g, '');
});

const goInfoHash = (hash: string) => {
  router.push(`/detail/infohash/`).catch(() => {});
};
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
      <div class="mb-6">
        <div class="flex items-start justify-between gap-4">
          <div class="flex-1 min-w-0">
            <h1 class="text-xl font-bold leading-snug">{{ resource.title }}</h1>
            <div class="mt-2 flex items-center gap-4 text-sm text-zinc-400">
              <span>发布于 {{ formatChinaTime(new Date(resource.createdAt)) }}</span>
              <span>大小 {{ parseSize(resource.size) }}</span>
              <router-link
                :to="{ path: '/resources/1', query: { type: resource.type } }"
                class="text-link-secondary text-xs"
              >
                {{ resource.type }}
              </router-link>
            </div>
            <div class="mt-3">
              <div class="flex items-center gap-2">
                <a
                  :href="magnetWithTracker"
                  class="inline-flex items-center gap-1 px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-500 text-white text-sm"
                >
                  🧲 磁力链接
                </a>
                <a
                  :href="getPikPakUrlChecker(resource.magnet)"
                  target="_blank"
                  class="inline-flex items-center gap-1 px-3 py-1.5 rounded-md border border-zinc-300 dark:border-zinc-600 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800"
                >
                  ▶ 在线播放 (PikPak)
                </a>
                <button
                  class="inline-flex items-center gap-1 px-3 py-1.5 rounded-md border border-zinc-300 dark:border-zinc-600 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800"
                  @click="copyMagnet"
                >
                  📋 复制链接
                </button>
              </div>
            </div>
          </div>
          <div class="text-right shrink-0">
            <div class="flex items-center gap-2 justify-end">
              <img
                v-if="resource.publisher?.avatar"
                :src="resource.publisher.avatar"
                class="w-8 h-8 rounded-full object-cover"
                alt=""
              />
              <div class="text-sm">
                <div class="font-medium">{{ resource.publisher?.name }}</div>
                <div v-if="resource.fansub" class="text-base-400">
                  {{ resource.fansub.name }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- description -->
      <div v-if="normalizedDescription" class="mb-6">
        <h2 class="text-base font-bold mb-2">简介</h2>
        <pre class="whitespace-pre-wrap bg-zinc-50 dark:bg-zinc-800 rounded-md p-4 text-sm text-base-700 leading-relaxed overflow-x-auto">{{ normalizedDescription }}</pre>
      </div>

      <!-- magnets -->
      <div v-if="detail?.magnets?.length" class="mb-6">
        <h2 class="text-base font-bold mb-2">下载</h2>
        <div class="space-y-2">
          <div
            v-for="(m, idx) in detail.magnets"
            :key="idx"
            class="flex items-center justify-between gap-2 bg-zinc-50 dark:bg-zinc-800 rounded-md px-4 py-2"
          >
            <span class="text-sm font-medium shrink-0 w-[140px]">{{ m.name }}</span>
            <a
              :href="m.url"
              target="_blank"
              class="text-link text-sm truncate"
              :title="m.url"
            >
              {{ m.url }}
            </a>
          </div>
        </div>
      </div>

      <!-- file tree -->
      <div v-if="detail?.files?.length" class="mb-6">
        <h2 class="text-base font-bold mb-2 flex items-center gap-2">
          文件列表
          <span v-if="detail.hasMoreFiles" class="text-xs font-normal text-base-400">
            (仅显示部分文件)
          </span>
        </h2>
        <div class="bg-zinc-50 dark:bg-zinc-800 rounded-md p-2 text-sm">
          <div v-for="node in treeRoots" :key="node.path ?? node.name" class="file-node">
            <div
              v-if="isDir(node)"
              class="flex items-center gap-2 px-2 py-1 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700 rounded"
              @click="toggleDir(node.path)"
            >
              <span class="text-xs">{{ expanded.has(node.path) ? '▼' : '▶' }}</span>
              <span class="font-medium">📁 {{ node.name }}/</span>
            </div>
            <div
              v-else
              class="flex items-center justify-between gap-2 px-2 py-1 rounded hover:bg-zinc-100 dark:hover:bg-zinc-700"
            >
              <span class="truncate">📄 {{ node.name }}</span>
              <span v-if="node.size" class="text-xs text-base-400 shrink-0">{{ node.size }}</span>
            </div>
            <div v-if="isDir(node) && expanded.has(node.path)" class="ml-4">
              <div v-for="child in node.children" :key="child.path ?? child.name" class="file-node">
                <div
                  v-if="isDir(child)"
                  class="flex items-center gap-2 px-2 py-1 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-700 rounded"
                  @click="toggleDir(child.path)"
                >
                  <span class="text-xs">{{ expanded.has(child.path) ? '▼' : '▶' }}</span>
                  <span class="font-medium">📁 {{ child.name }}/</span>
                </div>
                <div
                  v-else
                  class="flex items-center justify-between gap-2 px-2 py-1 rounded hover:bg-zinc-100 dark:hover:bg-zinc-700"
                >
                  <span class="truncate">📄 {{ child.name }}</span>
                  <span v-if="child.size" class="text-xs text-base-400 shrink-0">{{ child.size }}</span>
                </div>
                <div v-if="isDir(child) && expanded.has(child.path)" class="ml-4">
                  <div
                    v-for="leaf in child.children"
                    :key="leaf.path ?? leaf.name"
                    class="flex items-center justify-between gap-2 px-2 py-1 rounded hover:bg-zinc-100 dark:hover:bg-zinc-700"
                  >
                    <span class="truncate">📄 {{ leaf.name }}</span>
                    <span v-if="leaf.size" class="text-xs text-base-400 shrink-0">{{ leaf.size }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="isDeleted" class="text-sm text-orange-600">
        ⚠ 该资源已被标记为删除
      </div>

      <div class="mt-8 text-xs text-zinc-400 space-y-1">
        <div>provider: {{ resource.provider }} · providerId: {{ resource.providerId }}</div>
        <div>fetchedAt: {{ formatChinaTime(new Date(resource.fetchedAt)) }}</div>
      </div>
    </template>
  </div>
</template>