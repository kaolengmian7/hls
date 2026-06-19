import type { RouteRecordRaw } from 'vue-router';

const pokerRoutes: RouteRecordRaw[] = [
  {
    path: '/poker',
    name: 'poker',
    component: () => import('./pages/PokerPage.vue'),
  },
];

export default pokerRoutes;
