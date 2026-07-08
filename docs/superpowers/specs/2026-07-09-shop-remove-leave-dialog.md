# 神秘商店：移除离开确认弹窗

**日期：** 2026-07-09

## 目标

点击"离开"按钮时直接执行离开操作，不再弹出确认对话框。

## 涉及文件

- `src/games/shop/pages/PlayPage.vue`
- `src/games/shop/pages/RoomPage.vue`

## 设计

### PlayPage.vue

- 将"离开"按钮的 `@click` 从 `showLeaveConfirm = true` 改为直接调用 `confirmLeave()`
- 删除 `showLeaveConfirm` ref
- 删除对应的 `ConfirmDialog`（离开确认，第一个）
- 保留"商店已关闭" `ClosedDialog`（第二个，由服务端事件触发）

### RoomPage.vue

- 将"离开商店"按钮的 `@click` 从 `showLeaveConfirm = true` 改为直接调用 `confirmLeave()`
- 删除 `showLeaveConfirm` ref
- 删除对应的 `ConfirmDialog`（离开确认）
- 保留"房主已关闭商店" `HostClosedDialog`

## 不变的部分

两页中由服务端关闭事件触发的提示弹窗（`showClosedDialog` / `showHostClosedDialog`）保持不变。
