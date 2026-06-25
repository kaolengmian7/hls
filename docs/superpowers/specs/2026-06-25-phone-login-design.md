# 手机号验证码登录设计

**日期：** 2026-06-25
**状态：** 待实现

## 背景

原登录入口使用微信扫码（PC 端 QR iframe），依赖微信开放平台资质，对非微信用户不友好。本次改造将前端登录入口替换为手机号 + 短信验证码，同时保留后端微信登录能力（接口不删除）。

---

## 一、整体架构

### 前端改动

| 文件 | 操作 |
|------|------|
| `src/pages/LoginPage.vue` | 全量替换，删除微信 QR iframe，改为两阶段手机号表单 |
| `src/composables/usePhoneLogin.ts` | 新建，封装发送/验证/状态逻辑 |
| `src/shared/components/SliderCaptcha.vue` | 新建，纯前端滑块人机验证组件 |
| `src/composables/useWechatQR.ts` | 保留文件，移除 UI 引用 |
| `src/pages/AuthCallbackPage.vue` | 不改动（微信后端回调仍有效） |

### 后端改动

| 新增 | 说明 |
|------|------|
| `POST /api/auth/sms/send` | 发送验证码；防刷检查在此拦截 |
| `POST /api/auth/sms/verify` | 校验验证码；成功返回 JWT |
| `server/internal/handler/sms.go` | 两个 handler 实现 |
| `server/internal/svc/sms_store.go` | 进程内验证码存储 + 限流计数器 |
| `server/internal/auth/sms_sender.go` | 腾讯云 SMS 封装（接口化） |
| `server/internal/config/config.go` | 新增 `TencentSMS` 配置结构 |
| `server/etc/gameserver.yaml` | 新增 `TencentSMS` 配置块 |

### 数据库

```sql
ALTER TABLE users
  ADD COLUMN phone VARCHAR(20) NULL UNIQUE
  AFTER avatar_url;
```

`UserRepo` 新增 `UpsertByPhone(ctx, phone, nickname, avatarURL)` 方法。

---

## 二、防刷层

所有限流数据存进程内（`sync.Map` + 原子计数），按自然日（`time.Now().Format("20060102")` in UTC+8）重置。服务重启计数归零，属可接受行为。

### `POST /api/auth/sms/send` 拦截顺序

1. **手机号格式校验** — 仅接受 `1[3-9]\d{9}` 大陆号
2. **滑块 token 非空校验** — 字段必须存在（后端不做密码学验证，防刷由限流兜底）
3. **全局每日硬上限** ≥ 200 → HTTP 503，`"global_limit"`
4. **同手机号 60s 冷却** → HTTP 429，`"cooldown"`，返回 `retry_after` 秒数
5. **同手机号今日** ≥ 5 条 → HTTP 429，`"phone_limit"`
6. **同 IP 今日** ≥ 10 条 → HTTP 429，`"ip_limit"`
7. 调腾讯云 SMS 发送；写入 code store（6 位数字，TTL 10 分钟）
8. 全局计数 +1

### `POST /api/auth/sms/verify` 防暴力

- 同手机号验证失败 ≥ 5 次 → 该 code 立即作废
- 验证成功后立即删除 code（一次性使用）

---

## 三、API 契约

### `POST /api/auth/sms/send`

```json
// 请求
{ "phone": "13800138000", "captcha": "any-non-empty-string" }

// 成功 200
{ "ok": true }

// 冷却中 429
{ "error": "cooldown", "retry_after": 42 }

// 其他限流 429
{ "error": "phone_limit" }
{ "error": "ip_limit" }

// 全局封顶 503
{ "error": "global_limit" }
```

### `POST /api/auth/sms/verify`

```json
// 请求
{ "phone": "13800138000", "code": "123456" }

// 成功 200
{ "token": "<jwt>" }

// 验证码错误/已作废 400
{ "error": "invalid_code" }

// 验证码过期 400
{ "error": "code_expired" }
```

---

## 四、新用户处理

首次手机号登录时，`UpsertByPhone` 若创建新用户：

- **昵称：** `"小小花栗鼠"` + 4 位大写字母数字随机串（例：`小小花栗鼠X3K9`）
- **头像：** 空字符串（由前端 Avatar 组件用 ID seed 生成默认头像）
- **phone 列：** 填入手机号
- **微信字段：** 全部 NULL

---

## 五、腾讯云 SMS 配置

`gameserver.yaml` 新增：

```yaml
TencentSMS:
  SecretID: ""
  SecretKey: ""
  SmsSdkAppID: ""
  SignName: ""
  TemplateID: ""   # 模板内容示例：您的验证码为 {1}，{2}分钟内有效，请勿泄露。
```

`sms_sender.go` 定义接口 `SmsSender`，生产实现调腾讯云 SDK，测试可注入 mock。

---

## 六、前端 UI 流程

### 两阶段状态机

**step = `phone`（默认）**
- 滑块验证组件（通过后才允许提交）
- 手机号输入框（11 位数字）
- "发送验证码"按钮

**step = `code`（发送成功后切换）**
- 显示脱敏手机号（`138****8000`）
- 6 格验证码输入框
- "登录"按钮
- 倒计时重发（冷却期间禁用，到期显示"重新发送"）

### `SliderCaptcha.vue` 行为

- 轨道宽 280px，滑块 44px，拖到底（> 90%）视为通过
- 通过后轨道变绿、锁定，`emit('pass')`
- 点"发送验证码"时未通过则组件抖动提示
- 发送成功后自动重置（下次发送前需再次滑动）

### `usePhoneLogin.ts` 暴露接口

```ts
const {
  step,       // 'phone' | 'code'
  phone,      // 手机号 v-model
  code,       // 验证码 v-model
  cooldown,   // 剩余冷却秒数（0 表示可发送）
  sending,    // 发送中 loading
  verifying,  // 登录中 loading
  error,      // 当前错误文案（空字符串表示无错误）
  sendCode,   // 调 /api/auth/sms/send
  verifyCode, // 调 /api/auth/sms/verify → 写 token → router.push('/')
} = usePhoneLogin()
```

### 错误文案映射

| 后端 `error` | 前端显示 |
|---|---|
| `cooldown` | `请 {n} 秒后再试` |
| `phone_limit` | `该号码今日发送已达上限` |
| `ip_limit` | `操作过于频繁，请明日再试` |
| `global_limit` | `服务繁忙，请明日再试` |
| `invalid_code` | `验证码错误，请重新输入` |
| `code_expired` | `验证码已过期，请重新发送` |

---

## 七、保留的微信登录能力

以下后端接口、文件、配置均**不删除**：

- `GET /api/auth/wechat/qr/start`
- `GET /api/auth/wechat/qr/callback`
- `GET /api/auth/wechat/qr/status`
- `server/internal/auth/wechat.go`
- `server/internal/repo/` 中的 `auth_state` 相关逻辑
- `WechatOpen` 配置块
- 前端 `src/composables/useWechatQR.ts`（移除 UI 引用，保留文件）
- 前端 `src/pages/AuthCallbackPage.vue`
