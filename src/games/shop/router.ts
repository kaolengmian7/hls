import type { RouteRecordRaw } from 'vue-router';

const shopRoutes: RouteRecordRaw[] = [
  {
    path: '/shop',
    name: 'shop-home',
    component: () => import('./pages/ShopHomePage.vue'),
  },
  {
    path: '/shop/room/:code',
    name: 'shop-room',
    component: () => import('./pages/RoomPage.vue'),
    props: true,
  },
  {
    path: '/shop/room/:code/play',
    name: 'shop-play',
    component: () => import('./pages/PlayPage.vue'),
    props: true,
  },
];

export default shopRoutes;
