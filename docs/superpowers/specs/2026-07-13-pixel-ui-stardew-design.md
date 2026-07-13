# 全站像素风 UI 改版设计（星露谷物语风格）

**日期：** 2026-07-13  
**状态：** 已确认，待实现

---

## 目标

将全站 UI 风格从现有 RPG/羊皮纸/金丝主题，统一改为「星露谷物语」式温暖像素游戏风格。范围覆盖所有页面，包括神秘商店游戏页面。

---

## 设计语言

### 风格特征

- **暖木色面板**：tan/parchment 暖色系卡片，搭配深棕木边框，营造像素木框感
- **深绿世界背景**：`#1a2e20` 深森林绿，带 24px 浅色像素网格底纹
- **四层嵌套边框**：外深棕 → 内高光线 → 内暗线 → 内容区，模拟 SDV 面板质感
- **像素阴影**：所有面板/按钮统一 `4px 4px 0 #0a0806` 硬阴影，无模糊
- **VT323 像素字体**：所有标题、标签、按钮文字；中文正文用 Noto Sans SC
- **24px 网格背景**：全局 `repeating-linear-gradient` 网格

### 颜色系统（CSS 变量）

| 变量名 | 色值 | 用途 |
|--------|------|------|
| `--sdv-bg` | `#1a2e20` | 页面背景 |
| `--sdv-panel-deep` | `#3a5030` | 导航栏、深色卡片 |
| `--sdv-panel-warm` | `#c8a870` | 表单卡片、对话框 |
| `--sdv-input-bg` | `#f0dfa0` | Input 背景 |
| `--sdv-border` | `#5c3d24` | 面板外边框 |
| `--sdv-border-inner` | `#e8c898` | 面板内高光线 |
| `--sdv-border-shadow` | `#a07840` | 面板内暗线 |
| `--sdv-gold` | `#f0e080` | Logo、标题高亮 |
| `--sdv-btn-green` | `#7ab848` | 主操作按钮 |
| `--sdv-btn-green-border` | `#2a5010` | 主按钮边框 |
| `--sdv-btn-brown` | `#8a6030` | 次要操作、返回 |
| `--sdv-btn-red` | `#c04030` | 删除、危险操作 |
| `--sdv-text-dark` | `#3a200e` | 面板内正文 |
| `--sdv-text-light` | `#f0e8c0` | 深色背景上文字 |
| `--sdv-text-nav` | `#c8d8a0` | 导航链接 |
| `--sdv-shadow` | `#0a0806` | 硬阴影色 |
| `--sdv-active-gold` | `#e8d060` | 选中项高亮 |

### 字体层级

| 级别 | 字体 | 大小 | 用途 |
|------|------|------|------|
| Display | VT323 | 28–36px | 页面大标题 |
| H1/Section | VT323 | 20–24px | 区块标题 |
| Label/Button | VT323 | 16–20px | 标签、按钮文字 |
| Body | Noto Sans SC | 14px | 中文正文说明 |
| Caption | Noto Sans SC | 12px | 辅助说明、错误提示 |

---

## 架构方案

全站统一主题，不需要路由隔离。直接替换全局样式和所有组件。

### 文件操作

| 文件 | 操作 |
|------|------|
| `src/shared/styles/main.css` | 替换全局 body 背景、scrollbar、selection、网格背景 |
| `tailwind.config.js` | 新增 SDV 颜色 token，加 VT323/monospace 字体族 |
| `src/shared/styles/sdv-components.css`（新建） | 定义全局可复用组件类 |
| 所有 `.vue` 文件 | 替换旧 RPG 类名为 SDV 类名 |

---

## 可复用组件类（`sdv-components.css`）

### `.sdv-panel`（暖色面板）
```css
background: var(--sdv-panel-warm);
border: 4px solid var(--sdv-border);
box-shadow:
  inset 0 0 0 2px var(--sdv-border-inner),
  inset 0 0 0 4px var(--sdv-border-shadow),
  4px 4px 0 var(--sdv-shadow);
```

### `.sdv-panel-deep`（深色面板）
```css
background: var(--sdv-panel-deep);
border: 4px solid #2a3820;
box-shadow: inset 0 0 0 2px #2a3820, 4px 4px 0 var(--sdv-shadow);
```

### `.sdv-panel-title`（面板标题栏）
```css
background: #a07040;
border-bottom: 3px solid var(--sdv-border);
font-family: 'VT323', monospace;
padding: 12px 16px 8px;
```

### `.sdv-btn`（主按钮 · 绿色）
```css
background: linear-gradient(180deg, #7ab848, #4a8828);
border: 3px solid var(--sdv-btn-green-border);
box-shadow: inset 0 1px 0 rgba(255,255,255,0.25), inset 0 -2px 0 rgba(0,0,0,0.3), 3px 3px 0 var(--sdv-shadow);
font-family: 'VT323', monospace;
font-size: 19px;
letter-spacing: 2px;
color: var(--sdv-text-light);
/* hover: translateY(-2px) */
```

### `.sdv-btn-secondary`（次要按钮 · 棕色）
```css
background: linear-gradient(180deg, #8a6030, #5c3d1a);
border-color: #3a2010;
```

### `.sdv-btn-danger`（危险按钮 · 红色）
```css
background: linear-gradient(180deg, #c04030, #7a2018);
border-color: #4a1808;
```

### `.sdv-input`（输入框）
```css
background: var(--sdv-input-bg);
border: 3px solid var(--sdv-border);
box-shadow: inset 2px 2px 0 var(--sdv-border-shadow), inset -1px -1px 0 var(--sdv-border-inner);
font-family: 'VT323', monospace;
font-size: 18px;
/* focus: outline 2px solid #f0c840 */
```

### `.sdv-badge`（状态徽章）
```css
background: var(--sdv-btn-green);
border: 2px solid var(--sdv-btn-green-border);
box-shadow: 1px 1px 0 var(--sdv-shadow);
font-family: 'VT323', monospace;
```

### `.sdv-menu-row`（菜单行）
```css
border: 2px solid rgba(122,80,48,0.6);
background: #d4b878;
box-shadow: inset 0 1px 0 #e8c888, 2px 2px 0 #1a1208;
/* hover: translateX(3px) */
/* active: background #e8d060, border-color #c89020 */
```

---

## 页面改动清单

### 共享组件

**`AppHeader.vue`**
- 背景改为 `--sdv-panel-deep`（`#3a5030`）
- Logo 文字用 VT323 + `--sdv-gold`
- 导航链接用 VT323 + `--sdv-text-nav`
- 底部边框 `4px solid #2a3820` + 下阴影 `3px 0 #1a2810`

**`AuthWidget.vue`**
- 登录/用户按钮改为 `.sdv-btn-secondary`
- 用户昵称/角色标签改为 `.sdv-badge`

**`ConfirmDialog.vue`**
- 对话框容器改为 `.sdv-panel`
- 标题栏改为 `.sdv-panel-title`
- 确认按钮 `.sdv-btn`，取消按钮 `.sdv-btn-secondary`

### 普通页面

**`LandingPage.vue`**
- 全局 Hero 区：大标题 VT323 Display，副标题 Noto Sans SC
- 游戏卡片改为 `.sdv-menu-row` 风格（选中：神秘商店高亮金色，未开放：半透明 + `.sdv-badge sdv-badge-gray`）
- 页脚改为深绿背景 + VT323 徽章

**`LoginPage.vue`**
- 整体居中 `.sdv-panel`（暖木色）
- 标题栏用 `.sdv-panel-title`
- 账号/密码用 `.sdv-input`
- 登录按钮 `.sdv-btn`

**`ProfilePage.vue`**
- 页面背景继承全局深绿
- 每个设置区块改为 `.sdv-panel`（修改昵称 / 修改密码 / 后台管理）
- 保存按钮 `.sdv-btn`，改密码按钮 `.sdv-btn-secondary`

**`AdminPage.vue`**
- 用户列表改为 `.sdv-panel` 包裹的 `.sdv-menu-row` 列表
- 操作按钮（重置密码/删除）分别用 `.sdv-btn-secondary` / `.sdv-btn-danger`
- 创建用户表单用 `.sdv-panel` + `.sdv-input`

**`TeachPage.vue` / `AuditionPage.vue` / `PokerPage.vue`**
- 深绿深色面板居中，图标放大
- 标题 VT323 Display
- "即将推出"用 `.sdv-badge sdv-badge-gray`
- 返回按钮 `.sdv-btn-secondary`

### 神秘商店页面

**`ShopHomePage.vue`**
- 顶部 AppHeader（共享组件已统一）
- 房间号输入框改为 `.sdv-input`
- 进入/创建房间按钮改为 `.sdv-btn`

**`RoomPage.vue`**
- 房间信息卡片改为 `.sdv-panel-deep`
- 玩家槽位（PlayerSlot）改为 `.sdv-menu-row` 风格
- 角色选择（护航/老板）改为 `.sdv-menu-row.active`（选中金高亮）
- 操作按钮统一 `.sdv-btn` 系列

**`PlayPage.vue`**
- 主游戏区改为 `.sdv-panel-deep`
- 子模块（LotteryPanel、EquipmentShop、FeedStream）各自包裹 `.sdv-panel`
- 抽奖/结算按钮改为 `.sdv-btn`（主）/ `.sdv-btn-secondary`（次）

**`PlayerSlot.vue`**
- 玩家卡片改为 `.sdv-menu-row` 风格，空槽位用灰色透明

**`LotteryPanel.vue`**
- 转盘/奖品区外框 `.sdv-panel`
- 奖品标签 `.sdv-badge`

**`EquipmentShop.vue`**
- 装备列表行改为 `.sdv-menu-row`
- 购买按钮 `.sdv-btn`

**`FeedStream.vue`**
- 消息流行改为轻量 `.sdv-menu-row`（无硬阴影）

**`SettlementModal.vue` / `SettlementRecordModal.vue`**
- 模态框容器改为 `.sdv-panel`，标题栏 `.sdv-panel-title`
- 结算明细行改为 `.sdv-menu-row`（金色高亮赢家行）

**`LotteryRecordModal.vue`**
- 同上，模态框 `.sdv-panel`，记录行 `.sdv-menu-row`

**`ToastContainer.vue`**
- Toast 改为 `.sdv-badge` 风格（绿/红/金三种状态）

---

## 字体加载

在 `index.html` 或 `main.css` 中引入：

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=VT323&family=Noto+Sans+SC:wght@400;700&display=swap" rel="stylesheet">
```

Tailwind 字体配置：
```js
fontFamily: {
  pixel: ['"VT323"', 'monospace'],
  body: ['"Noto Sans SC"', '"PingFang SC"', '"Microsoft YaHei"', 'sans-serif'],
}
```

---

## 不在本次范围内

- 游戏逻辑、状态管理、网络层（无改动）
- 路由结构（无改动）
- 后端 API（无改动）
