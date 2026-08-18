<script setup lang="ts">
// Dropdown components mirroring apps/web/src/components/Dropdown.
import { ref, onBeforeUnmount, watch } from 'vue';

const props = withDefaults(
  defineProps<{
    trigger: string;
    triggerClass?: string;
    menuClass?: string;
    align?: 'left' | 'right';
  }>(),
  { align: 'left' }
);

const open = ref(false);
const root = ref<HTMLElement | null>(null);

const onMouseEnter = () => (open.value = true);
const onMouseLeave = () => (open.value = false);

const onClickOutside = (e: MouseEvent) => {
  if (root.value && !root.value.contains(e.target as Node)) {
    open.value = false;
  }
};

onBeforeUnmount(() => {
  document.removeEventListener('click', onClickOutside);
});
</script>

<template>
  <div
    ref="root"
    class="relative inline-block"
    @mouseenter="onMouseEnter"
    @mouseleave="onMouseLeave"
  >
    <slot name="trigger" :open="open">
      <a class="cursor-pointer rounded-md p-2" :class="triggerClass">{{ trigger }}</a>
    </slot>
    <transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="opacity-0 scale-95"
      leave-active-class="transition duration-75 ease-in"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="open"
        class="absolute top-full z-50 mt-[-10px] min-w-max rounded-md shadow-box bg-white dark:bg-zinc-900 divide-y divide-zinc-100 dark:divide-zinc-800 leading-normal"
        :class="[
          menuClass,
          props.align === 'right' ? 'right-0' : 'left-0'
        ]"
        @click.stop
      >
        <slot name="menu" />
      </div>
    </transition>
  </div>
</template>
