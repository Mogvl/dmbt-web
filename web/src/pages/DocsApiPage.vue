<script setup lang="ts">
// API docs page mirroring pages/docs.api/route.tsx: Swagger UI from the
// public OpenAPI spec.
import { onMounted, ref } from 'vue';

const loading = ref(true);
const spec = ref<any>(null);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    const resp = await fetch('/openapi.json');
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    spec.value = await resp.json();
  } catch (e) {
    error.value = String(e);
  }
  loading.value = false;
});
</script>

<template>
  <div class="w-full pt-13 pb-24">
    <div v-if="loading" class="py-24 flex justify-center">
      <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
    </div>

    <div v-else-if="error || !spec" class="py-24 text-center text-base-500">
      加载 API 文档失败
    </div>

    <div v-else class="swagger-wrap">
      <div class="mb-4 flex items-center justify-between">
        <h1 class="text-2xl font-bold">🌸 Anime Garden API</h1>
        <span class="text-sm text-base-400">v{{ spec.info?.version }}</span>
      </div>

      <div class="space-y-4">
        <div
          v-for="(pathItem, path) in spec.paths"
          :key="path"
          class="rounded-md border border-zinc-200 dark:border-zinc-800 overflow-hidden"
        >
          <div class="flex items-center gap-3 px-4 py-3 bg-zinc-50 dark:bg-zinc-800">
            <span
              class="inline-block px-2 py-0.5 rounded text-xs font-bold text-white"
              :class="{
                'bg-green-600': Object.keys(pathItem).includes('get'),
                'bg-blue-600': Object.keys(pathItem).includes('post'),
                'bg-yellow-600': Object.keys(pathItem).includes('put'),
                'bg-red-600': Object.keys(pathItem).includes('delete')
              }"
            >
              {{ Object.keys(pathItem)[0]?.toUpperCase() ?? 'GET' }}
            </span>
            <code class="text-sm font-mono">{{ path }}</code>
          </div>
          <div class="px-4 py-3 text-sm text-base-600 space-y-2">
            <template v-for="(op, method) in pathItem" :key="method">
              <div v-if="op.summary" class="font-medium">{{ op.summary }}</div>
              <div v-if="op.description" class="text-base-500">{{ op.description }}</div>
              <div v-if="op.parameters?.length" class="mt-2">
                <div class="text-xs font-bold text-base-400 mb-1">参数</div>
                <div v-for="param in op.parameters" :key="param.name" class="flex gap-2 text-xs">
                  <code class="text-sky-700">{{ param.name }}</code>
                  <span class="text-base-400">{{ param.in }}</span>
                  <span class="text-base-500">{{ param.description }}</span>
                </div>
              </div>
            </template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>