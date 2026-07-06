# 设计文档：游戏昵称统一使用用户个人资料昵称

**日期：** 2026-07-06  
**状态：** 已批准  

---

## 背景

神秘商店游戏原有一套独立的昵称系统：`usePlayerStore` 管理"名牌前缀（nicknamePreset）+ 自定义昵称（nicknameCustom）"，支持在准备阶段通过 `CHANGE_NICKNAME` intent 修改。引入用户账号系统后，用户已在个人资料页（`/profile`）维护统一昵称（`auth.user.nickname`），游戏内的独立昵称系统成了冗余的平行管理入口。

---

## 目标

- 游戏昵称统一读取 `authStore.user.nickname`
- 删除 `usePlayerStore` 中全部昵称相关代码
- 删除服务端 `CHANGE_NICKNAME` intent（游戏 UI 已无调用入口）

---

## 前端变更

### `src/shared/stores/player.ts`

删除以下导出和内部实现：
- `NICKNAME_PRESETS`、`NICKNAME_SEPARATOR`、`MAX_NICKNAME_LENGTH` 常量（如仅昵称系统使用）
- `composeNickname()`、`maxCustomNicknameLength()` 函数
- `STORAGE_KEY_NICKNAME`、`STORAGE_KEY_NICKNAME_PRESET`、`STORAGE_KEY_NICKNAME_CUSTOM` 常量
- `DEFAULT_NICKNAME` 常量
- `nicknamePreset`、`nicknameCustom`、`legacyNickname` ref
- `nickname` computed
- `hasCustomNickname` computed
- `persistNickname()`、`setNicknameCustom()`、`setNicknamePreset()` 方法
- `ensureIdentity()` 中所有昵称相关的 localStorage 读写逻辑

`nickname` 不再属于 playerStore 职责；调用方直接使用 `authStore.user?.nickname ?? ''`。

### `src/games/shop/pages/ShopHomePage.vue`

以下 3 处 `playerStore.nickname` 改为 `authStore.user?.nickname ?? ''`：
- `handleCreate()`：传给 `createRoom({ nickname: ... })`
- `handleCreate()`：传给 `roomStore.connect({ nickname: ... })`
- `handleJoin()`：传给 `roomStore.connect({ nickname: ... })`

同时引入 `useAuthStore`。

### `src/games/shop/stores/room.ts`

- `connect()` 参数 interface：`nickname` 字段保留（由调用方传入），调用方改为传 auth nickname
- `tryRestoreSession()` 中 `playerStore.nickname` 改为 `useAuthStore().user?.nickname ?? ''`

---

## 服务端变更

### `server/internal/protocol/intent.go`

删除：
- `IntentChangeNickname` 常量
- `ChangeNicknamePayload` struct

### `server/internal/room/room.go`

删除：
- `case protocol.IntentChangeNickname:` dispatch 分支
- `handleChangeNickname()` 函数及其注释

---

## 错误处理

- 若 `auth.user.nickname` 为空（用户未设置昵称），传给服务端的值为空字符串；服务端 JOIN 时昵称为空会拒绝（已有校验 `nickname required`）
- 用户应在进入游戏前通过 `/profile` 设置昵称；首页/路由守卫可根据需要添加提示（不在本次范围内）

---

## 不在范围内

- 路由守卫检查昵称是否已设置
- 游戏内实时同步 auth 昵称变更（进入房间后昵称固定，不随 profile 变化）
- `CHANGE_NICKNAME` 的服务端逻辑若被其他地方引用需单独评估（当前确认无 UI 入口）
