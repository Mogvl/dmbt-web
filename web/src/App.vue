<script setup lang="ts">
import { watch } from 'vue';
import { useThemeStore } from './stores';

const theme = useThemeStore();

watch(
  () => theme.current,
  (current) => {
    document.documentElement.classList.toggle('dark', current === 'dark');
  },
  { immediate: true }
);
</script>

<template>
  <router-view />
  <Toaster richColors position="top-right" :duration="3000" />
</template>
