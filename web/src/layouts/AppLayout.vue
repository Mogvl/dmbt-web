<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { useSidebarStore } from '../stores';
import Header from './Header.vue';
import Sidebar from './Sidebar.vue';
import Footer from './Footer.vue';
import SearchDialog from './SearchDialog.vue';
import { initLayoutController } from './global';

const props = withDefaults(
  defineProps<{
    heading?: boolean;
    footer?: boolean;
  }>(),
  { heading: true, footer: true }
);

const route = useRoute();
const sidebar = useSidebarStore();

const feedURL = computed(() => {
  const search = window.location.search;
  return `/feed.xml${search}`;
});

const timestamp = computed(() => undefined);

onMounted(() => {
  // hero scroll pinning + header collision hiding (port of global.ts)
  initLayoutController();
});
</script>

<template>
  <div>
    <!-- hero search -->
    <search
      id="hero-search"
      class="w-full h-(--nav-height) z-12 flex items-center justify-center pointer-events-none"
    >
      <div
        data-header-collision-source
        class="vercel relative h-[44.4px] xl:w-[640px] lg:w-[600px] md:w-[500px] lt-md:w-[calc(100vw-116px)] pointer-events-auto"
      >
        <SearchDialog />
      </div>
    </search>
    <Header :feedURL="feedURL" />
    <div id="hero-banner" class="w-full h-(--hero-height) bg-hero">
      <template v-if="props.heading">
        <h1
          class="lg:z-12 lt-lg:z-10 w-full pt-5rem pb-3rem text-4xl font-quicksand font-bold text-center select-none outline-none pointer-events-none"
        >
          <router-link to="/" data-header-collision-source class="pointer-events-auto cursor-pointer">
            <span>🌸 Anime Garden</span>
          </router-link>
        </h1>
      </template>
      <template v-else>
        <div
          class="lg:z-12 lt-lg:z-10 w-full pt-5rem pb-3rem text-4xl font-quicksand font-bold text-center select-none outline-none pointer-events-none"
        >
          <router-link to="/" data-header-collision-source class="pointer-events-auto cursor-pointer">
            <span>🌸 Anime Garden</span>
          </router-link>
        </div>
      </template>
    </div>
    <div id="hero-placeholder" class="w-full h-(--nav-height) hidden z-1"></div>

    <div class="w-full flex" :class="sidebar.isOpen ? 'main-with-sidebar' : 'main-without-sidebar'">
      <Sidebar />
      <div
        class="flex-auto flex items-center justify-center min-h-[calc(100vh-300px-220px)]"
      >
        <main class="main">
          <router-view :key="route.fullPath" />
        </main>
      </div>
    </div>
    <Footer v-if="props.footer" :timestamp="timestamp" :feedURL="feedURL" />
  </div>
</template>