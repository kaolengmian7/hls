# 颜色系统重新设计

**日期：** 2026-07-14
**状态：** 已批准，待实施

## 背景

当前主题为 Stardew Valley 深色像素风（`sdv-*` token），体背景为深森林绿 `#1a2e20`。本次重新设计将系统切换为浅色系，同时将零散的命名体系升级为语义化 token 分层，提升可维护性。

## 设计决策

- **背景方向：** C · 米白奶油（`#faf6ed` 纯色，无渐变）
- **Token 方案：** 方案 B · 语义化分层，前缀 `sdv-*` 全部改为 `hls-*`
- **正文字体：** 思源黑体（Noto Sans SC），取代原有像素字体用于正文
- **像素字体保留：** VT323 继续用于 Logo、区块装饰标签、面板标题栏

---

## 一、颜色 Token 系统

所有 token 在 `tailwind.config.js` 的 `theme.extend.colors.hls` 下定义，同时以 CSS 变量形式在 `main.css` 的 `:root` 中声明。

### Background · 背景层

| Token | 色值 | 用途 |
|---|---|---|
| `hls-bg` | `#faf6ed` | 页面主背景，全局唯一纯色 |

### Surface · 内容面层

| Token | 色值 | 用途 |
|---|---|---|
| `hls-surface` | `#fffef9` | 卡片、面板、输入框默认背景 |
| `hls-surface-active` | `#fffdf0` | 选中 / 激活状态卡片背景 |
| `hls-surface-nav` | `#a07040` | 顶部导航栏背景（保留棕木深色） |

### Border · 边框

| Token | 色值 | 用途 |
|---|---|---|
| `hls-border` | `#d4b870` | 默认卡片、面板边框 |
| `hls-border-active` | `#c8a020` | 激活 / 焦点状态边框 |
| `hls-border-strong` | `#5c3d24` | 导航栏底边、按钮强边框 |
| `hls-border-subtle` | `rgba(160,112,64,0.3)` | 分隔线、Footer 分割线 |

### Text · 文字

| Token | 色值 | 用途 |
|---|---|---|
| `hls-text` | `#2a1408` | 主要正文，卡片标题 |
| `hls-text-secondary` | `#5c3820` | 副标题、Hero 描述文字 |
| `hls-text-muted` | `#9a7040` | 次级说明、卡片描述、placeholder |
| `hls-text-inverse` | `#fff8e8` | 深色背景上的文字（导航栏） |

### Accent · 金色强调

| Token | 色值 | 用途 |
|---|---|---|
| `hls-gold` | `#b88820` | 主标题、重要强调色 |
| `hls-gold-bright` | `#c89828` | 副标题装饰、动效闪烁 |

### Action · 操作色

| Token | 色值 | 用途 |
|---|---|---|
| `hls-action` | `#5a8828` | 主按钮、激活徽章背景 |
| `hls-action-border` | `#2a5010` | 主按钮边框 |
| `hls-danger` | `#c04030` | 危险操作按钮 |

### Shadow · 像素阴影

| Token | 色值 | 用途 |
|---|---|---|
| `hls-shadow` | `rgba(60,30,8,0.12)` | 卡片、按钮像素投影（2px 2px offset） |

---

## 二、字体系统

| 角色 | 字体 | 规格 | 用途 |
|---|---|---|---|
| `font-pixel` | VT323 | 像素等宽 | Logo、区块标签、面板标题栏、装饰文字 |
| `font-body` 粗 | Noto Sans SC 700 | 15px | 卡片主标题、按钮文字、徽章 |
| `font-body` 常规 | Noto Sans SC 400 | 14px | Hero 描述、正文段落 |
| `font-body` 次级 | Noto Sans SC 400 | 12px `hls-text-muted` | 卡片副描述、说明文字、Footer |

---

## 三、组件映射

所有组件类名前缀 `sdv-` → `hls-`，色值按上述 token 更新。

| 旧类名 | 新类名 | 变化说明 |
|---|---|---|
| `sdv-panel` | `hls-panel` | 背景 `#c8a870` → `hls-surface`；边框 `#5c3d24` → `hls-border`；阴影变浅 |
| `sdv-panel-deep` | （合并至 `hls-panel`） | 消除 deep/warm 区分，统一为单一面板类 |
| `sdv-panel-title` | `hls-panel-title` | 背景保留 `hls-surface-nav`；字体改 Noto Sans SC 700 |
| `sdv-panel-title-light` | （合并至 `hls-panel-title`） | 合并两种面板标题为一种 |
| `sdv-nav` | `hls-nav` | 背景保留 `hls-surface-nav`；链接字体改 Noto Sans SC |
| `sdv-btn` | `hls-btn` | 字体 VT323 → Noto Sans SC 500；阴影色用 `hls-shadow` |
| `sdv-btn-secondary` | `hls-btn-secondary` | 同上 |
| `sdv-btn-danger` | `hls-btn-danger` | 同上 |
| `sdv-btn-ghost` | `hls-btn-ghost` | 背景、边框适配浅色背景 |
| `sdv-input` | `hls-input` | 背景 `#f0dfa0` → `hls-surface`；边框 → `hls-border` |
| `sdv-menu-row` | `hls-menu-row` | 背景 `#d4b878` → `hls-surface`；激活态用 `hls-surface-active` + `hls-border-active` |
| `sdv-badge` | `hls-badge` | 字体改 Noto Sans SC 500；色值微调 |
| `sdv-badge-*` | `hls-badge-*` | 同上（gold / gray / red / teal / amber） |
| `sdv-toast` | `hls-toast` | 背景保留深色（通知气泡保持对比度） |
| `sdv-toast--*` | `hls-toast--*` | 同上 |
| `sdv-divider` | `hls-divider` | 颜色用 `hls-border-subtle` |
| `sdv-stat` | `hls-stat` | 背景 → `hls-surface`；边框 → `hls-border` |

---

## 四、实施范围

### 需修改的文件

| 文件 | 改动内容 |
|---|---|
| `tailwind.config.js` | `colors.sdv` → `colors.hls`，色值全部更新；字体 stack 不变 |
| `src/shared/styles/main.css` | body 背景改 `#faf6ed` 纯色；所有 `.sdv-*` → `.hls-*`；色值全面更新 |
| `src/shared/components/AppHeader.vue` | 类名替换 |
| `src/shared/components/AuthWidget.vue` | 类名替换 |
| `src/shared/components/Avatar.vue` | 类名替换 |
| `src/shared/components/ConfirmDialog.vue` | 类名替换 |
| `src/shared/components/SliderCaptcha.vue` | 类名替换 |
| `src/pages/*.vue` | LandingPage、LoginPage、ProfilePage、AdminPage 等类名替换 |
| `src/games/**/*.vue` | shop、teach、audition、poker 下所有组件类名替换 |

### 迁移步骤

1. 更新 `tailwind.config.js`：重命名 `sdv` → `hls`，更新所有色值
2. 更新 `main.css`：body 背景 + 所有组件类重写
3. 全局批量替换 `sdv-` → `hls-`（覆盖所有 `.vue` 文件）
4. 逐页视觉验证，处理硬编码色值（如 `#1a2e20`、`#3a5030` 等内联样式）

### 不在本次范围内

- 深色模式（未来可在此系统基础上扩展）
- 游戏内动效颜色（如抽奖动画粒子色）单独处理
