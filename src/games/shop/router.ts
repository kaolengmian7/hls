import type { RouteRecordRaw } from 'vue-router';
import ShopLayout from './ShopLayout.vue';

const shopRoutes: RouteRecordRaw[] = [
  {
    path: '/shop',
    component: ShopLayout,
    children: [
      {
        path: '',
        name: 'shop-home',
        component: () => import('./pages/ShopHomePage.vue'),
      },
      {
        path: 'room/:code',
        name: 'shop-room',
        component: () => import('./pages/RoomPage.vue'),
        props: true,
      },
      {
        path: 'room/:code/play',
        name: 'shop-play',
        component: () => import('./pages/PlayPage.vue'),
        props: true,
      },
    ],
  },
];

export default shopRoutes;
