import type { RouteRecordRaw } from 'vue-router';

const auditionRoutes: RouteRecordRaw[] = [
  {
    path: '/audition',
    name: 'audition',
    component: () => import('./pages/AuditionPage.vue'),
  },
];

export default auditionRoutes;
