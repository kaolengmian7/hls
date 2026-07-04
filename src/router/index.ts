import { createRouter, createWebHistory } from 'vue-router';
import shopRoutes from '@/games/shop/router';
import teachRoutes from '@/games/teach/router';
import auditionRoutes from '@/games/audition/router';
import pokerRoutes from '@/games/poker/router';
import { useAuthStore } from '@/stores/auth';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/pages/LandingPage.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
    },
    {
      path: '/auth/callback',
      name: 'auth-callback',
      component: () => import('@/pages/AuthCallbackPage.vue'),
    },
    ...shopRoutes,
    ...teachRoutes,
    ...auditionRoutes,
    ...pokerRoutes,
    { path: '/:catchAll(.*)', redirect: '/' },
  ],
});

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.path.startsWith('/shop') && !auth.token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
});
