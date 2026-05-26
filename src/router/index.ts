import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: () => import('@/pages/HomePage.vue'),
  },
  {
    path: '/room/:code',
    name: 'room',
    component: () => import('@/pages/RoomPage.vue'),
    props: true,
  },
  {
    path: '/room/:code/play',
    name: 'play',
    component: () => import('@/pages/PlayPage.vue'),
    props: true,
  },
  { path: '/:catchAll(.*)', redirect: '/' },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});
