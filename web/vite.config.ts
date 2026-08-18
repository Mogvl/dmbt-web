import { defineConfig, type ProxyOptions } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';

// API paths proxied to the Go backend (port 9701).
// The original web app serves both the SPA and the proxied API from one
// origin and distinguishes them by the Accept header: browser navigations
// carry `Accept: text/html` and get the SPA, API fetches get the backend.
const apiPrefixes = [
  '/resources',
  '/resource',
  '/detail',
  '/subjects',
  '/users',
  '/teams',
  '/collection',
  '/feed.xml',
  '/sitemaps',
  '/health',
  '/bgmx',
  '/subject'
];

function proxyBypass(): ProxyOptions['bypass'] {
  return (req) => {
    const accept = req.headers.accept ?? '';
    if (typeof accept === 'string' && accept.includes('text/html')) {
      // SPA navigation -> serve index.html instead of proxying
      return req.url;
    }
    return undefined;
  };
}

function buildProxy() {
  const proxy: Record<string, ProxyOptions> = {};
  for (const prefix of apiPrefixes) {
    proxy[prefix] = {
      target: 'http://localhost:9701',
      changeOrigin: true,
      bypass: proxyBypass()
    };
  }
  return proxy;
}

export default defineConfig({
  plugins: [
    vue({
      template: {
        compilerOptions: {
          isCustomElement: (tag) => tag === 'search'
        }
      }
    }),
    tailwindcss()
  ],
  server: {
    port: 9700,
    host: true,
    proxy: buildProxy()
  },
  preview: {
    port: 9700,
    host: true,
    proxy: buildProxy()
  }
});