// Stores mirroring apps/web/src/stores with Pinia.

import { defineStore } from 'pinia';

export type ThemeMode = 'light' | 'system' | 'dark';

const themeModeKey = 'animegarden:theme-mode';

function getSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function readThemeMode(): ThemeMode {
  try {
    return (JSON.parse(localStorage.getItem(themeModeKey) ?? '"system"') ?? 'system') as ThemeMode;
  } catch {
    return 'system';
  }
}

export const useThemeStore = defineStore('theme', {
  state: () => ({
    mode: readThemeMode() as ThemeMode
  }),
  getters: {
    current(state): 'light' | 'dark' {
      return state.mode === 'system' ? getSystemTheme() : state.mode;
    }
  },
  actions: {
    setMode(mode: ThemeMode) {
      this.mode = mode;
      localStorage.setItem(themeModeKey, JSON.stringify(mode));
    },
    toggle() {
      this.setMode(this.current === 'dark' ? 'light' : 'dark');
    }
  }
});

export const useSidebarStore = defineStore('sidebar', {
  state: () => ({ isOpen: false }),
  actions: {
    open() {
      this.isOpen = true;
    },
    close() {
      this.isOpen = false;
    },
    toggle() {
      this.isOpen = !this.isOpen;
    }
  }
});

const historiesKey = 'animegarden:histories';
const maxHistories = 10;

function readHistories(): string[] {
  try {
    const data = JSON.parse(localStorage.getItem(historiesKey) ?? '[]');
    return Array.isArray(data) ? data.filter((h) => typeof h === 'string') : [];
  } catch {
    return [];
  }
}

export const useHistoriesStore = defineStore('histories', {
  state: () => ({ histories: readHistories() }),
  actions: {
    push(text: string) {
      // keep only entries not contained in the new input, dedupe, prepend, cap 10
      const filtered = this.histories.filter((h) => !text.includes(h));
      this.histories = [text, ...filtered.filter((h) => h !== text)].slice(0, maxHistories);
      localStorage.setItem(historiesKey, JSON.stringify(this.histories));
    },
    remove(text: string) {
      this.histories = this.histories.filter((h) => h !== text);
      localStorage.setItem(historiesKey, JSON.stringify(this.histories));
    },
    clear() {
      this.histories = [];
      localStorage.setItem(historiesKey, JSON.stringify(this.histories));
    }
  }
});

const fansubsKey = 'animegarden:fansubs';

function readPreferFansubs(): string[] {
  try {
    const data = JSON.parse(localStorage.getItem(fansubsKey) ?? '[]');
    return Array.isArray(data) ? data.filter((f) => typeof f === 'string') : [];
  } catch {
    return [];
  }
}

export const useFansubsStore = defineStore('fansubs', {
  state: () => ({ preferFansubs: readPreferFansubs() }),
  actions: {
    add(fansub: string) {
      if (!this.preferFansubs.includes(fansub)) {
        this.preferFansubs.push(fansub);
        localStorage.setItem(fansubsKey, JSON.stringify(this.preferFansubs));
      }
    }
  }
});

export * from './collection';
