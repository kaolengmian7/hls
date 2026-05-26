# 三人对局 · 全栈实现

> 基于设计文档 [`docs/page.MD`](./docs/page.MD) 实现的 3 人房间实时对局应用。
> **前后端分离**架构，服务端权威（server-authoritative）。

## 技术栈

| 层 | 选型 |
| --- | --- |
| 前端 | Vue 3 + Vite + TypeScript + Pinia + Tailwind CSS |
| 后端 | Go + [go-zero](https://github.com/zeromicro/go-zero) (rest/logx/conf) + [gorilla/websocket](https://github.com/gorilla/websocket) |
| 通信 | WebSocket 全双工 + 少量 REST（创建/查询房间） |
| 协议 | JSON Envelope `{ type, payload, reqId?, ts }` |
| 随机源 | 服务端 `crypto/rand`（抽奖唯一可信随机源） |
| 部署 | 前端 Vercel，后端任意 Go 运行时 / K8s |

## 目录结构

```
hls/
├── docs/page.MD              # 设计文档（权威）
├── src/                      # 前端 Vue 3
│   ├── pages/                # HomePage / RoomPage / PlayPage
│   ├── components/           # 通用组件
│   ├── stores/               # player / room（薄客户端）
│   ├── net/                  # ws.ts / client.ts / api.ts / protocol.ts / config.ts
│   ├── game/                 # 仅 UI 用元数据：rules / lottery / equipment
│   └── utils/                # id / storage / avatar
├── server/                   # 后端 Go
│   ├── cmd/gameserver/main.go
│   ├── etc/gameserver.yaml
│   ├── internal/
│   │   ├── config/           # 配置结构
│   │   ├── handler/          # REST + WS 入口
│   │   ├── room/             # Hub / Room Loop / Client
│   │   ├── game/             # 业务规则（纯函数）
│   │   ├── protocol/         # Envelope / Intent / Event
│   │   └── svc/              # 服务上下文
│   └── pkg/                  # errors / code / token 通用包
├── package.json / vite.config.ts ...
└── README.md
```

## 启动

### 1. 启动后端

```bash
cd server
go mod tidy
go run ./cmd/gameserver -f etc/gameserver.yaml
# 监听 0.0.0.0:8088
```

> 主要依赖：
> - `github.com/zeromicro/go-zero` （rest / conf / logx）
> - `github.com/gorilla/websocket`

### 2. 启动前端

```bash
# 在仓库根目录
cp .env.example .env   # 本地开发默认指向 http://localhost:8088
npm install
npm run dev
# 监听 http://localhost:5173
```

打开三个标签页：一个创建房间，两个用同一房间号加入。

## 环境变量（前端）

```env
# .env，本地开发
VITE_API_BASE=http://localhost:8088
# VITE_WS_BASE=ws://localhost:8088   # 留空则自动从 API_BASE 派生

# .env.production，生产构建
VITE_API_BASE=https://api.hls-club.com
# VITE_WS_BASE=wss://api.hls-club.com   # 留空则自动从 API_BASE 派生
```

## 网络拓扑

```
                ┌───────────────────────────────┐
                │  Go gameserver (single proc)  │
                │  Hub + per-Room goroutine     │
                │                               │
                │  ┌─────┐  ┌─────┐  ┌─────┐    │
                │  │ R1  │  │ R2  │  │ R3  │... │
                │  └─────┘  └─────┘  └─────┘    │
                └──────────────┬────────────────┘
                               │ WebSocket (ws/wss)
            ┌──────────────────┼──────────────────┐
            │                  │                  │
        ┌───▼────┐         ┌───▼────┐         ┌───▼────┐
        │Escort A│         │  Host  │         │Escort B│
        └────────┘         └────────┘         └────────┘
```

- 三名玩家**各自**与服务端建立 WebSocket，不存在客户端互联。
- 服务端为每个房间维护一个 goroutine（Room Loop），串行处理所有 Intent；无需在房间内部加锁。
- 业务规则、随机源、状态裁决全部在 Go 服务端，客户端只是薄渲染层。

## 消息协议（节选）

详见 `src/net/protocol.ts` 与 `server/internal/protocol/*.go`。

```
Client → Server (Intent):
  JOIN / LEAVE / DISMISS_ROOM / CHANGE_ROLE / READY / START_GAME /
  BUY_EQUIPMENT / DRAW_LOTTERY / PING

Server → Client (Event):
  JOIN_ACCEPTED / JOIN_REJECTED / STATE_SNAPSHOT /
  PLAYER_JOINED / PLAYER_LEFT / ROLE_CHANGED / READY_CHANGED / HOST_CHANGED /
  GAME_STARTED / COIN_CHANGED / EQUIPMENT_PURCHASED / LOTTERY_DRAWN /
  ACTION_REJECTED / KICK / PONG
```

## 异常与边界

| 场景 | 处理 |
| --- | --- |
| 客户端单次断连 | 指数退避自动重连（最多 5 次）；重连成功后服务端单播 STATE_SNAPSHOT |
| 房主离线超 30s | 服务端把房主权限迁移给最早加入的在线玩家，广播 HOST_CHANGED |
| 所有玩家离线 | 房间空闲 60s 后服务端清理，释放 Code |
| 服务端重启 | 房间销毁；客户端收到 close 后弹窗并回首页 |
| 写阻塞 / 慢客户端 | 写 channel 缓冲满 → 强制关闭该连接，由其重连补全 |
| sessionToken 失效 | 客户端走 JOIN 重新入场（仍以同一 playerId 复用座位） |

## 测试

```bash
# 后端
cd server && go test ./...

# 前端
npm run typecheck
npm run build
```

## 部署

- 前端：`npm run build` → `dist/`，部署到 Vercel/Nginx/CDN；保留 SPA fallback 到 `index.html`。
- 后端：`server/docker-compose.yml` 使用 Caddy 对外暴露 `80/443` 并自动申请 / 续期 HTTPS 证书，Go 服务只在 Docker 网络内监听 `8088`。
- 房间状态目前仅存内存。如需水平扩展或滚动升级，参考设计文档 §7 引入 Redis。

### 后端 HTTPS 部署

1. 在 DNS 中添加 `api.hls-club.com` 的 `A` 记录，指向腾讯云公网 IP。
2. 腾讯云安全组放行 `80`、`443`；`8088` 可不再对公网开放。
3. 在服务器上配置环境变量：

```bash
cd server
cp .env.example .env
```

```env
HLS_API_DOMAIN=api.hls-club.com
HLS_SESSION_SECRET=replace-with-a-long-random-secret
HLS_CORS_ALLOW_ORIGINS=https://hls-club.com,https://你的-vercel域名.vercel.app
```

4. 启动服务：

```bash
docker compose up -d --build
```

5. Vercel 中设置并重新部署：

```env
VITE_API_BASE=https://api.hls-club.com
# VITE_WS_BASE 留空即可自动推导为 wss://api.hls-club.com
```
