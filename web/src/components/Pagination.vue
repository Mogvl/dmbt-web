<script setup lang="ts">
// Pagination mirroring components/Resources/pagination.tsx.
import { computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';

const props = defineProps<{
  page: number;
  complete: boolean;
  timestamp?: Date;
}>();

const router = useRouter();
const route = useRoute();

const isPrev = computed(() => props.page > 1);
const isNext = computed(() => !props.complete);

const pages = computed(() =>
  isPrev.value
    ? [props.page - 1, props.page, props.page + 1, props.page + 2, props.page + 3]
    : [props.page, props.page + 1, props.page + 2, props.page + 3, props.page + 4]
);

function go(page: number) {
  const q: Record<string, string | string[]> = { ...route.query };
  q.page = String(page);
  router.push({ path: `/resources/${page}`, query: q });
}

// only navigable if the API allowed the target page (complete semantics)
const visiblePages = computed(() =>
  pages.value.filter((p) => !props.complete || p <= props.page)
);
</script>

<template>
  <div class="mt-4 flex lt-md:flex-col font-sm">
    <div class="flex-auto"></div>
    <div v-if="props.page > 1 || !props.complete" class="flex lt-md:(mt-4 justify-center) items-center gap-2 text-base-500">
      <button
        class="page-btn select-none cursor-pointer"
        :class="{ hidden: !isPrev }"
        @click="go(props.page - 1)"
      >
        <span>上一页</span>
      </button>
      <template v-if="props.page > 2">
        <button class="page-btn select-none cursor-pointer" @click="go(1)">
          <span>1</span>
        </button>
        <span class="select-none">…</span>
      </template>
      <template v-for="p in visiblePages" :key="p">
        <button
          class="page-btn select-none cursor-pointer"
          :class="p === props.page && 'page-btn-active'"
          @click="go(p)"
        >
          <span>{{ p }}</span>
        </button>
      </template>
      <span v-if="isNext" class="select-none">…</span>
      <button
        class="page-btn select-none cursor-pointer"
        :class="{ hidden: !isNext }"
        @click="go(props.page + 1)"
      >
        <span>下一页</span>
      </button>
    </div>
  </div>
</template>