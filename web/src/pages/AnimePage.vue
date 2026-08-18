<script setup lang="ts">
// /anime redirects to the active calendar season (mirrors the original).
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { getActiveSeason } from '../utils/constants';

const router = useRouter();

onMounted(() => {
  router.replace(`/calendar/${getActiveSeason()}`);
});
</script>

<template>
  <div class="py-24 flex justify-center">
    <div class="lds-ring"><div></div><div></div><div></div><div></div></div>
  </div>
</template>