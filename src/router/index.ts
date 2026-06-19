import { createRouter, createWebHistory } from 'vue-router';
import shopRoutes from '@/games/shop/router';
import teachRoutes from '@/games/teach/router';
import auditionRoutes from '@/games/audition/router';
import pokerRoutes from '@/games/poker/router';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/pages/LandingPage.vue'),
    },
    ...shopRoutes,
    ...teachRoutes,
    ...auditionRoutes,
    ...pokerRoutes,
    { path: '/:catchAll(.*)', redirect: '/' },
  ],
});
