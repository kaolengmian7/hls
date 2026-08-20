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
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/pages/ProfilePage.vue'),
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('@/pages/AdminPage.vue'),
    },
    {
      path: '/order/:code',
      name: 'order-claim',
      component: () => import('@/pages/OrderClaimPage.vue'),
    },
    {
      path: '/review/:code',
      name: 'public-order-review',
      component: () => import('@/pages/PublicOrderReviewPage.vue'),
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

  if (to.path === '/profile' && !auth.token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.path.startsWith('/order/') && !auth.token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.path === '/admin') {
    if (!auth.token) return { name: 'login', query: { redirect: to.fullPath } }
    const allowed = ['owner', 'service', 'enforcer', 'customer']
    if (auth.user && !allowed.includes(auth.user.role)) return { name: 'home' }
  }
});
