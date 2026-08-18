<script setup lang="ts">
// API docs page mirroring pages/docs.api/route.tsx: Swagger UI from the
// public OpenAPI spec.
import { nextTick, onMounted, ref } from 'vue';
import SwaggerUIBundle from 'swagger-ui-dist/swagger-ui-bundle.js';
import 'swagger-ui-dist/swagger-ui.css';

const loading = ref(true);
const error = ref<string | null>(null);
const specRef = ref<any>(null);
const container = ref<HTMLElement | null>(null);

onMounted(async () => {
  try {
    const resp = await fetch('/openapi.json');
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const spec = await resp.json();
    specRef.value = spec;
    loading.value = false;
    await nextTick();
    if (container.value && specRef.value) {
      SwaggerUIBundle({
        spec: specRef.value,
        domNode: container.value,
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
        layout: 'BaseLayout'
      });
    }
  } catch (e) {
    error.value = String(e);
  }
  loading.value = false;
});
</script>

<template>
  <div class="w-full pt-13 pb-24" data-docspage="true">
    <div v-if="loading" class="py-24 flex justify-center">
      <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
    </div>
    <div v-else-if="error" class="py-24 text-center text-base-500">
      加载 API 文档失败: {{ error }}
    </div>
    <div v-else ref="container" class="swagger-ui-wrap"></div>
  </div>
</template>