<script setup lang="ts">
import { useSidebarStore, useThemeStore } from '../stores';
import { formatChinaTime } from '../utils/date';

const props = defineProps<{
  timestamp?: Date;
  feedURL?: string;
}>();

const sidebar = useSidebarStore();
const theme = useThemeStore();

const timestamp = props.timestamp;
</script>

<template>
  <footer
    class="w-full relative flex justify-center border-t border-t-1 py-6 h-[252px] bg-hero"
    :class="sidebar.isOpen && 'main-with-sidebar'"
  >
    <div class="main w-full pl-4 lt-md:pl-0">
      <div class="lt-sm:text-sm">
        <div class="flex">
          <span class="text-base-900 font-bold select-none">状态</span>
          <span class="text-xl lt-sm:text-base text-base-900 font-bold select-none relative top-[2px]">›</span>
          <a href="https://uptime.animes.garden/status/animegarden" target="_blank" class="ml-2">
            监控
          </a>
          <span class="ml-4 lt-sm:ml-2">
            {{
              timestamp
                ? '数据更新于 ' + formatChinaTime(timestamp)
                : ' '
            }}
          </span>
        </div>
        <div class="flex mt-2">
          <span class="text-base-900 font-bold select-none">源站</span>
          <span class="text-xl lt-sm:text-base text-base-900 font-bold select-none relative top-[2px]">›</span>
          <a href="https://share.dmhy.org/" target="_blank" class="ml-2">動漫花園</a>
          <a href="https://mikanani.me/" target="_blank" class="ml-4">蜜柑计划</a>
          <a href="https://bangumi.moe/" target="_blank" class="ml-4">萌番组</a>
          <a href="https://open.ani.rip/" target="_blank" class="ml-4">ANi</a>
        </div>
        <div class="flex mt-2">
          <span class="text-base-900 font-bold select-none">关于</span>
          <span class="text-xl lt-sm:text-base text-base-900 font-bold select-none relative top-[2px]">›</span>
          <a href="https://github.com/yjl9903/AnimeGarden" target="_blank" class="ml-2">
            GitHub
          </a>
          <a href="https://docs.animes.garden" target="_blank" class="ml-4 lt-sm:ml-2">
            AnimeSpace 计划
          </a>
        </div>
        <div class="flex mt-2">
          <span class="text-base-900 font-bold select-none">开放</span>
          <span class="text-xl lt-sm:text-base text-base-900 font-bold select-none relative top-[2px]">›</span>
          <a
            href="https://github.com/yjl9903/AnimeGarden?tab=readme-ov-file#%E4%BD%BF%E7%94%A8-skills"
            target="_blank"
            class="ml-2 lt-sm:ml-2"
          >
            Agent Skills
          </a>
          <a
            href="https://github.com/yjl9903/AnimeGarden?tab=readme-ov-file#%E4%BD%BF%E7%94%A8-mcp"
            target="_blank"
            class="ml-4 lt-sm:ml-2"
          >
            MCP
          </a>
          <router-link to="/docs/api" class="ml-4 lt-sm:ml-2">API 文档</router-link>
        </div>
        <div class="flex mt-2">
          <span class="text-base-900 font-bold select-none">更多</span>
          <span class="text-xl lt-sm:text-base text-base-900 font-bold select-none relative top-[2px]">›</span>
          <a href="https://t.me/animegarden_dev" target="_blank" class="ml-2 lt-sm:ml-2">
            Telegram
          </a>
          <a href="https://t.me/animegarden_channel" target="_blank" class="ml-4 lt-sm:ml-2">
            Channel
          </a>
          <a href="https://github.com/yjl9903/AnimeGarden/issues" target="_blank" class="ml-4 lt-sm:ml-2">
            问题反馈
          </a>
          <a
            href="https://docs.animes.garden/animegarden/search"
            target="_blank"
            class="ml-4 lt-sm:ml-2"
          >
            帮助文档
          </a>
        </div>
        <div class="flex justify-between items-center mt-8">
          <div>
            <span>
              © 2022
              <a href="https://github.com/animegarden" target="_blank">Anime Space</a>.
            </span>
            <span> | </span>
            <span>
              <a :href="props.feedURL ?? '/feed.xml'">RSS</a>
            </span>
            <span> | </span>
            <span>
              <a href="/sitemap-index.xml">站点地图</a>
            </span>
          </div>
          <div>
            <button
              class="p-2 rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800"
              @click="theme.toggle()"
              :title="theme.current === 'dark' ? '切换到浅色模式' : '切换到深色模式'"
            >
              {{ theme.current === 'dark' ? '☀️' : '🌙' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </footer>
</template>
