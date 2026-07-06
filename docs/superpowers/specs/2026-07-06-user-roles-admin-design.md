# 用户角色、后台管理与个人管理页面设计

**日期：** 2026-07-06  
**范围：** 用户角色系统 · 账密登录 · /admin 后台 · /profile 个人页

---

## 1. 背景与目标

当前系统使用手机短信验证码登录，用户表无角色字段，无后台管理界面。  
本期目标：
- 引入四种角色：店长(owner)、客服(service)、打手(enforcer)、客户(customer)
- 登录方式改为用户名 + 密码
- 新增 `/admin` 后台用户管理页（仅 owner 可访问）
- 新增 `/profile` 个人管理页（所有登录用户可访问）

微信扫码和短信登录的后端代码及路由保留，前端不再展示对应入口。

---

## 2. 数据库变更

在现有 `users` 表上追加三列：

```sql
ALTER TABLE users
  ADD COLUMN username      VARCHAR(64)  UNIQUE NULL,
  ADD COLUMN password_hash VARCHAR(255) NULL,
  ADD COLUMN role          ENUM('owner','service','enforcer','customer')
                           NOT NULL DEFAULT 'customer';
```

- `username` / `password_hash` 允许 NULL，兼容存量微信/手机用户
- `password_hash` 使用 bcrypt（cost=10）
- `role` 默认 `customer`

**第一个店长账号由运维手动执行 SQL：**

```sql
INSERT INTO users (username, password_hash, role, nickname, avatar_url)
VALUES ('admin', '<bcrypt_hash>', 'owner', '店长', '');
```

---

## 3. 后端 API

### 3.1 新增接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `POST` | `/api/auth/login` | 无 | 用户名+密码登录，返回 `{"token":"..."}` |
| `GET` | `/api/auth/me` | JWT | 已有，响应补充 `role` 字段 |
| `GET` | `/api/admin/users` | owner | 用户列表 |
| `POST` | `/api/admin/users` | owner | 创建用户 |
| `PUT` | `/api/admin/users/:id/password` | owner | 重置密码为 `123456` |
| `PUT` | `/api/admin/users/:id/nickname` | owner | 重置昵称为 `小小花栗鼠` |
| `PUT` | `/api/user/password` | JWT | 用户修改自己密码 |
| `PUT` | `/api/user/nickname` | JWT | 用户修改自己昵称 |

### 3.2 请求/响应结构

**POST /api/auth/login**
```json
// 请求
{ "username": "alice", "password": "secret" }
// 响应
{ "token": "<jwt>" }
```

**GET /api/auth/me（更新后）**
```json
{ "id": 1, "nickname": "Alice", "avatar_url": "", "role": "customer" }
```

**GET /api/admin/users**
```json
[
  { "id": 1, "username": "alice", "nickname": "Alice", "role": "customer" }
]
```

**POST /api/admin/users**
```json
// 请求
{ "username": "bob", "nickname": "Bob", "role": "service" }
// 密码自动设为 123456，昵称为空时默认 "小小花栗鼠"
```

**PUT /api/user/password**
```json
{ "old_password": "old", "new_password": "new" }
```

**PUT /api/user/nickname**
```json
{ "nickname": "新昵称" }
```

### 3.3 鉴权中间件

- `requireAuth(ctx)`：从 `Authorization: Bearer <jwt>` 提取并验证 userID，写入 request context
- `requireOwner(ctx)`：调用 `requireAuth` 后额外查询 role，非 `owner` 返回 403

### 3.4 现有路由

所有现有路由（短信、微信 QR）保持注册不变，后端代码不删除。

### 3.5 UserRepo 扩展

新增方法：
- `FindByUsername(ctx, username) (*User, error)`
- `CreateWithPassword(ctx, username, passwordHash, nickname, role) (*User, error)`
- `UpdatePassword(ctx, id, passwordHash) error`
- `UpdateNickname(ctx, id, nickname) error`
- `ListAll(ctx) ([]*User, error)`

User 结构体新增字段：`Username sql.NullString`、`PasswordHash sql.NullString`、`Role string`

---

## 4. 前端

### 4.1 路由变更

| 路由 | 组件 | 守卫 |
|------|------|------|
| `/login` | `LoginPage.vue`（改写） | 已登录跳 `/` |
| `/profile` | `ProfilePage.vue`（新建） | 需要登录，否则跳 `/login` |
| `/admin` | `AdminPage.vue`（新建） | 需要 owner 角色，否则跳 `/` |

### 4.2 LoginPage.vue

替换手机/验证码 UI 为：用户名输入框 + 密码输入框 + 登录按钮。  
调用 `POST /api/auth/login`，成功后存 token 并跳转。

### 4.3 ProfilePage.vue

两块功能区：

1. **修改昵称** — 输入框预填当前昵称，提交调用 `PUT /api/user/nickname`
2. **修改密码** — 旧密码 + 新密码 + 确认新密码，提交调用 `PUT /api/user/password`
3. **店长专属** — role === 'owner' 时显示「进入后台管理」按钮，跳转 `/admin`

### 4.4 AdminPage.vue

- 顶部「创建用户」按钮，弹出 Modal（填用户名、昵称可选、角色下拉）
- 用户列表表格：id / 用户名 / 昵称 / 角色 / 操作
- 操作列：「重置密码」「重置昵称」，点击前弹 ConfirmDialog 确认

### 4.5 AuthStore 更新

```typescript
interface AuthUser {
  id: number
  nickname: string
  avatar_url: string
  role: 'owner' | 'service' | 'enforcer' | 'customer'
}

// 新增方法
async function login(username: string, password: string): Promise<void>
```

### 4.6 AppHeader / AuthWidget

增加跳转 `/profile` 的入口（点击昵称或新增「个人」链接）。

### 4.7 路由守卫（router/index.ts）

```typescript
router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.path === '/profile' && !auth.token) return { name: 'login' }
  if (to.path === '/admin') {
    if (!auth.token) return { name: 'login' }
    if (auth.user?.role !== 'owner') return { name: 'home' }
  }
})
```

---

## 5. 文件变更清单

**后端（server/）：**
- `internal/repo/user.go` — 扩展 User 结构体和 UserRepo 方法
- `internal/handler/auth.go` — 新增 LoginHandler、更新 MeHandler 返回 role
- `internal/handler/admin.go` — 新建，含 ListUsersHandler / CreateUserHandler / ResetPasswordHandler / ResetNicknameHandler
- `internal/handler/user.go` — 新建，含 UpdatePasswordHandler / UpdateNicknameHandler
- `cmd/gameserver/main.go` — 注册新路由

**前端（src/）：**
- `stores/auth.ts` — 更新 AuthUser 类型，新增 login 方法
- `pages/LoginPage.vue` — 改写为账密表单
- `pages/ProfilePage.vue` — 新建
- `pages/AdminPage.vue` — 新建
- `router/index.ts` — 新增路由及守卫
- `shared/components/AppHeader.vue` 或 `AuthWidget.vue` — 增加 profile 入口

**数据库：**
- 提供 migration SQL（非代码文件，运维执行）

---

## 6. 不在本期范围内

- 角色权限细分（角色对游戏功能的影响）
- 用户名格式校验（本期只校验非空、唯一）
- 账号封禁/停用
- 操作日志审计
