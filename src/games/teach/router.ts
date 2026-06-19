import type { RouteRecordRaw } from 'vue-router';

const teachRoutes: RouteRecordRaw[] = [
  {
    path: '/teach',
    name: 'teach',
    component: () => import('./pages/TeachPage.vue'),
  },
];

export default teachRoutes;
