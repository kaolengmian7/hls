# 神秘商店背景图片方案设计

**日期：** 2026-06-26  
**范围：** 神秘商店所有页面（ShopHomePage、RoomPage、PlayPage）

---

## 背景与问题

项目中有一张 `shop-background.png`（19MB，5504×3072 PNG），需要作为神秘商店所有页面的背景图。网站部署在 Vercel，大陆用户访问亚太节点，19MB 的原始图片加载极慢。

**根本原因**：图片是原始分辨率的未压缩 PNG，体积远超 Web 背景图所需。压缩后大陆访问 Vercel 亚太节点的延迟不再是瓶颈。

---

## 方案选择

选择**方案 A：本地压缩 + 路由级布局层**，理由：

- 不引入额外服务（CDN 账号、运维、费用）
- 压缩后预期体积 200~400KB，Vercel 亚太节点完全够用
- 实现最简，不改动现有页面组件

---

## 设计

### 1. 图片压缩（一次性离线操作）

编写 `scripts/convert-shop-bg.js`，使用 `sharp` 库：

- **输入**：项目根目录 `shop-background.png`（19MB，5504×3072）
- **输出**：`public/shop-bg.webp`
- **处理**：
  - 缩放至最大宽度 1920px，保持原始宽高比
  - 格式 WebP，quality 85
- **预期输出体积**：200~400KB

脚本只需运行一次，输出文件提交到 git。`public/` 目录下的文件 Vite 原样部署，不做额外处理。

`shop-background.png` 原始文件不放入 `public/` 也不提交到 git（`.gitignore` 排除）。

### 2. ShopLayout.vue

新增 `src/games/shop/ShopLayout.vue`，作为所有商店路由的父级包装层。

**层级结构：**

```
App.vue (min-h-screen flex flex-col)
  └── ShopLayout.vue (flex-1 flex flex-col，背景图固定在此)
        └── <router-view>  ← ShopHomePage / RoomPage / PlayPage
```

**背景 CSS：**

```css
background-image:
  linear-gradient(rgba(10, 5, 2, 0.45), rgba(10, 5, 2, 0.45)),
  url('/shop-bg.webp');
background-size: cover;
background-position: center;
background-attachment: fixed;
```

- 半透明深色叠层（45%）保留 RPG 沉浸感，保证文字可读性
- `background-attachment: fixed` 与 `body` 现有背景行为一致，滚动时背景不动
- 该 div 通过 `flex-1` 填满 `App.vue` 的剩余高度，视觉上覆盖 `body` 的渐变背景

**无需修改** `main.css` 或任何现有页面组件。

### 3. 路由更新

修改 `src/games/shop/router.ts`，将三个商店路由改为 `ShopLayout` 的子路由：

```ts
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
```

路由名称（`shop-home`、`shop-room`、`shop-play`）不变，不影响任何现有的 `router.push` 调用。

---

## 文件变更清单

| 操作 | 文件 |
|------|------|
| 新增 | `scripts/convert-shop-bg.js` |
| 新增 | `public/shop-bg.webp`（脚本生成） |
| 新增 | `src/games/shop/ShopLayout.vue` |
| 修改 | `src/games/shop/router.ts` |
| 修改 | `.gitignore`（排除 `shop-background.png`） |

---

## 非目标

- 不做 blur-up 渐进加载（体积已足够小，不必要）
- 不接入 CDN（300KB 级别 Vercel 够用）
- 不修改其他游戏（poker、audition、teach）的任何文件
