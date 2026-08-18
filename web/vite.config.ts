import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    port: 9700,
    host: true,
    proxy: {
      // API requests proxied to the Go backend (port 9701)
      '/resources': 'http://localhost:9701',
      '/resource': 'http://localhost:9701',
      '/detail': 'http://localhost:9701',
      '/subjects': 'http://localhost:9701',
      '/users': 'http://localhost:9701',
      '/teams': 'http://localhost:9701',
      '/collection': 'http://localhost:9701',
      '/feed.xml': 'http://localhost:9701',
      '/sitemaps': 'http://localhost:9701',
      '/health': 'http://localhost:9701',
      '/bgmx': 'http://localhost:9701',
      '/subject': 'http://localhost:9701',
    },
  },
  preview: {
    port: 9700,
    host: true,
    proxy: {
      '/resources': 'http://localhost:9701',
      '/resource': 'http://localhost:9701',
      '/detail': 'http://localhost:9701',
      '/subjects': 'http://localhost:9701',
      '/users': 'http://localhost:9701',
      '/teams': 'http://localhost:9701',
      '/collection': 'http://localhost:9701',
      '/feed.xml': 'http://localhost:9701',
      '/sitemaps': 'http://localhost:9701',
      '/health': 'http://localhost:9701',
      '/bgmx': 'http://localhost:9701',
      '/subject': 'http://localhost:9701'
    }
  }
});