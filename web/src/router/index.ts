import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('../layouts/AppLayout.vue'),
    children: [
      {
        path: '',
        name: 'index',
        component: () => import('../pages/HomePage.vue')
      },
      {
        path: 'resources/:page?',
        name: 'resources',
        component: () => import('../pages/ResourcesPage.vue')
      },
      {
        path: 'subject/:subject/:page?',
        name: 'subject',
        component: () => import('../pages/SubjectPage.vue')
      },
      {
        path: 'anime',
        name: 'anime',
        component: () => import('../pages/AnimePage.vue')
      },
      {
        path: 'calendar/:season',
        name: 'calendar',
        component: () => import('../pages/CalendarPage.vue')
      },
      {
        path: 'collection/:hash',
        name: 'collection',
        component: () => import('../pages/CollectionPage.vue')
      },
      {
        path: 'detail/:provider/:providerId',
        name: 'detail',
        component: () => import('../pages/DetailPage.vue')
      },
      {
        path: 'docs/api',
        name: 'docs-api',
        component: () => import('../pages/DocsApiPage.vue')
      },
      {
        path: 'about',
        name: 'about',
        component: () => import('../pages/AboutPage.vue')
      }
    ]
  },
  {
    path: '/iframe',
    name: 'iframe',
    component: () => import('../pages/IframePage.vue')
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 };
  }
});
