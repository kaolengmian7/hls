# 全站顶部登录入口设计

**日期：** 2026-06-19  
**需求：** 在每个页面右上角暴露登录/用户信息入口

---

## 背景

项目已有完整的微信扫码登录流程（`/login` 页 + `/auth/callback` 回调 + `useAuthStore`），但各页面没有入口引导用户登录，也没有展示已登录状态。本设计通过新增共享组件，将登录入口统一注入全站页面。

---

## 组件结构

### `src/shared/components/AuthWidget.vue`

读取 `useAuthStore()`，根据登录状态渲染两种形态：

- **未登录**：显示「登录」按钮，点击跳转 `/login`
- **已登录**：显示头像（复用 `Avatar.vue`）+ 昵称，点击展开下拉菜单，菜单包含「退出登录」

退出调用现有 `auth.logout()`（清除 localStorage token，跳转 `/login`）。

### `src/shared/components/AppHeader.vue`

全站统一顶部栏：

- 左侧：「✦ 花栗鼠俱乐部」品牌文字
- 右侧：`<AuthWidget />`
- 样式：`border-b border-white/5 px-6 py-4`，与现有 LandingPage nav 风格一致

---

## 页面改造范围

| 页面 | 处理方式 |
|------|----------|
| `LandingPage.vue` | 将现有 `<nav>` 替换为 `<AppHeader />`；现有「玩法」「微信服务号」链接保留在左侧区域 |
| `ShopHomePage.vue` | 在页面顶部新增 `<AppHeader />` |
| `RoomPage.vue` | 在页面顶部新增 `<AppHeader />` |
| `PlayPage.vue` | 在页面顶部新增 `<AppHeader />` |
| `TeachPage.vue` | 在页面顶部新增 `<AppHeader />` |
| `AuditionPage.vue` | 在页面顶部新增 `<AppHeader />` |
| `PokerPage.vue` | 在页面顶部新增 `<AppHeader />` |
| `LoginPage.vue` | 不改动（登录流程页） |
| `AuthCallbackPage.vue` | 不改动（OAuth 回调页） |

---

## 数据流与初始化

**问题：** `auth.user`（昵称、头像）只存在内存中，页面刷新后丢失，即使 token 仍在 localStorage。

**方案：** 在 `App.vue` 的 `onMounted` 中调用 `auth.init()`。该函数读取存储的 token，请求 `/auth/me` 接口恢复用户信息。

- `AuthWidget` 直接读取 `auth.user` 和 `auth.token`，响应式，无需手动传参
- `auth.init()` 已有防护：token 为空时直接返回，不发请求

---

## 不在本次范围内

- 手机号/邮箱注册登录（项目仅支持微信扫码）
- 个人资料页
- 路由鉴权（需要登录才能访问某些页面）
