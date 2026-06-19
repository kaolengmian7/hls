# 微信登录用户系统设计文档

**日期：** 2026-06-19  
**状态：** 已确认，待实现

---

## 背景

花栗鼠俱乐部当前无用户系统，玩家仅凭昵称进入游戏房间，身份完全临时。本期目标是实现一个独立的用户注册/登录系统，不与游戏房间逻辑耦合，为后续账号体系打基础。

**范围外（本期不做）：**
- 用户身份与游戏房间的绑定
- 公众号 H5 OAuth（需域名备案，后续补）
- Refresh Token / 多设备管理

---

## 技术决策

| 项目 | 决策 |
|------|------|
| 登录方式 | 微信开放平台网站应用（PC + 移动端统一扫码） |
| Token 策略 | 单个 JWT，7 天有效期，存 localStorage |
| 数据库 | MySQL，自建于 DMIT 服务器，Docker Compose 管理 |
| DB 连接 | localhost，零网络延迟，无额外费用 |
| 数据备份 | 每日 `mysqldump` cron job，上传腾讯云 COS |
| 架构 | 新增路由挂在现有 `gameserver`，不新建服务 |
| 服务器 | 香港 DMIT，无需备案 |

---

## 数据库 Schema

```sql
-- 用户主表
CREATE TABLE users (
  id           BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  mp_openid    VARCHAR(64)  UNIQUE,                    -- 公众号 openid（备案后 H5 登录用，暂留空）
  open_openid  VARCHAR(64)  UNIQUE,                    -- 开放平台 openid（扫码登录）
  union_id     VARCHAR(64)  UNIQUE,                    -- unionid，跨平台合并同一用户
  nickname     VARCHAR(64)  NOT NULL DEFAULT '',
  avatar_url   VARCHAR(512) NOT NULL DEFAULT '',
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- QR 扫码状态（PC 登录轮询用）
CREATE TABLE auth_states (
  state      VARCHAR(64) NOT NULL PRIMARY KEY,         -- 随机 UUID，前端轮询 key
  jwt        TEXT,                                      -- 扫码成功后填入
  expires_at DATETIME    NOT NULL,                      -- 5 分钟后过期
  INDEX idx_expires_at (expires_at)
);
```

**说明：**
- `mp_openid` 保留字段，备案完成后补 H5 登录不改 schema
- `union_id` 有值时作为跨平台用户合并依据
- `auth_states` 过期记录在 `qr/status` 查询时顺手清理，不需要独立定时任务

---

## 后端 API

### 新增配置字段

```yaml
# etc/gameserver.yaml 新增
WechatOpen:
  AppID: ""       # 开放平台 AppID
  AppSecret: ""   # 开放平台 AppSecret

JWT:
  Secret: ""      # 签名密钥，生产环境从环境变量注入
  TTLDays: 7

DB:
  DSN: ""         # MySQL DSN，格式：user:pass@tcp(host:port)/dbname?parseTime=true
```

### 路由清单

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/auth/wechat/qr/start` | 返回 `{ state, qr_url }`，前端展示二维码 |
| `GET` | `/api/auth/wechat/qr/callback` | 微信扫码回调，签发 JWT 写入 auth_states |
| `GET` | `/api/auth/wechat/qr/status?state=xxx` | 轮询，返回 `{ status: "pending"\|"done"\|"expired", token? }` |
| `GET` | `/api/auth/me` | 需 `Authorization: Bearer <jwt>`，返回当前用户信息 |

> 公众号 H5 路由（`/api/auth/wechat/mp/start`、`/api/auth/wechat/mp/callback`）备案后补充，接口位置已预留。

### 核心业务逻辑（两条路径共用）

```
收到 code（来自微信回调）
  └─→ POST https://api.weixin.qq.com/sns/oauth2/access_token（换 access_token + openid + unionid）
        └─→ GET https://api.weixin.qq.com/sns/userinfo（拉 nickname、headimgurl）
              └─→ upsert users 表（按 union_id 或 open_openid 查找，不存在则插入）
                    └─→ 签发 JWT（sub = user_id，exp = 7天）
```

### `qr/start` 详细流程

1. 生成随机 `state`（UUID），写入 `auth_states`，`expires_at = NOW() + 5min`
2. 构造微信开放平台 QR 授权 URL：
   ```
   https://open.weixin.qq.com/connect/qrconnect
     ?appid=<OPEN_APPID>
     &redirect_uri=<后端 callback URL>
     &response_type=code
     &scope=snsapi_login
     &state=<state>
   ```
3. 返回 `{ state, qr_url }`（前端用 `<img src=qr_url>` 展示，微信会渲染成二维码图片）

### `qr/status` 详细流程

1. 顺手执行 `DELETE FROM auth_states WHERE expires_at < NOW()`
2. 查询 `state` 记录
   - 不存在 → `{ status: "expired" }`
   - `jwt` 为空 → `{ status: "pending" }`
   - `jwt` 有值 → `{ status: "done", token: "<jwt>" }`，**查询后删除该记录**

---

## 前端结构

### 新增文件

```
src/
├── stores/
│   └── auth.ts           # Pinia store：token、user、login/logout
├── composables/
│   └── useWechatQR.ts    # QR 轮询（每 2s 调 qr/status）
├── pages/
│   └── LoginPage.vue     # 登录页，展示 QR 码
├── pages/
│   └── AuthCallbackPage.vue  # /auth/callback 路由，读 URL token 存 store 后跳首页
└── http.ts               # axios 实例，自动带 Authorization header
```

### auth store（Pinia）

```typescript
// 状态
token: string | null       // 存/读 localStorage
user: { id, nickname, avatar_url } | null

// 方法
init()      // 页面加载时从 localStorage 恢复 token，调 /api/auth/me 拉用户信息
logout()    // 清除 token，跳 /login
```

### LoginPage.vue 流程

```
进入 /login
  └─→ 调 /api/auth/wechat/qr/start
        └─→ 展示 <img :src="qr_url">
              └─→ 每 2s 轮询 /api/auth/wechat/qr/status?state=xxx
                    ├─ pending → 继续等待
                    ├─ expired → 提示"二维码已过期"，展示刷新按钮
                    └─ done    → auth.token = token → router.push('/')
```

### AuthCallbackPage.vue（公众号 H5 备案后使用）

```
/auth/callback?token=xxx
  └─→ auth.token = route.query.token
        └─→ router.push('/')
```

### 401 处理

axios 响应拦截器：收到 401 → `auth.logout()`（清 token，跳 /login）。无需 refresh 逻辑。

---

## 错误处理

| 场景 | 处理 |
|------|------|
| 微信 `code` 无效/过期 | 302 回 `/login?error=wechat_code_invalid` |
| 微信 API 超时 | 302 回 `/login?error=wechat_timeout` |
| `state` 不存在或已过期 | `qr/status` 返回 `{ status: "expired" }` |
| JWT 签名错误或过期 | `/api/auth/me` 返回 401 |
| DB 连接失败 | 500 + 日志，前端展示"服务暂时不可用" |
| URL 含 `error` 参数 | LoginPage 读取并展示友好文案 |

---

## 微信开放平台申请清单

完成实现前需要准备：

1. 登录 `open.weixin.qq.com`，创建「网站应用」
2. 填写网站信息、上传 logo，等待审核（1~7 个工作日）
3. 审核通过后获取 `AppID` 和 `AppSecret`
4. 在应用详情填写**授权回调域**：填 `api.hls-club.com`（不带协议和路径）
5. 将 `AppID`、`AppSecret` 写入服务器环境变量

---

## 数据库部署方案

MySQL 运行在 DMIT 服务器的 Docker 容器中，与 `gameserver` 同机，通过 `localhost` 连接。

**docker-compose.yml 新增 service：**

```yaml
mysql:
  image: mysql:8.0
  restart: always
  environment:
    MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
    MYSQL_DATABASE: hls
  volumes:
    - mysql_data:/var/lib/mysql
  ports:
    - "127.0.0.1:3306:3306"   # 只绑定本机，不对外暴露

volumes:
  mysql_data:
```

**备份方案：** 服务器上添加每日 cron job，执行 `mysqldump` 并将备份文件上传至腾讯云 COS。COS 费用极低（约 ¥0.1/GB/月），实现异地容灾。

**选择自建而非腾讯云 MySQL 的原因：**
- 当前用户规模小，无高可用需求
- 自建节省约 ¥1800+/年
- localhost 连接延迟 < 0.1ms，优于任何云数据库
- Docker 管理维护成本极低

---

## 后续扩展（本期不做）

- 公众号 H5 OAuth（域名备案后补）：新增 `/api/auth/wechat/mp/start` + `/api/auth/wechat/mp/callback`，`users.mp_openid` 字段已预留
- Refresh Token：JWT 改短 TTL + refresh 表，前端加拦截器
- 用户身份绑定游戏房间
