# DMIT 服务器部署与迁移文档

本文档记录将 HLS 后端从腾讯云迁移到 DMIT 服务器的完整流程，包括 SSH 密钥连接、本地文件上传、Docker 部署，以及通过 DNS 切换完成迁移。

## 目录

- [架构概览](#架构概览)
- [本地开发](#本地开发)
- [服务器信息](#服务器信息)
- [一、SSH 密钥连接](#一ssh-密钥连接)
- [二、服务器环境准备](#二服务器环境准备)
- [三、上传 server 目录](#三上传-server-目录)
- [四、配置环境变量并启动](#四配置环境变量并启动)
- [五、迁移：DNS 切换](#五迁移dns-切换)
- [六、验证与下线旧机](#六验证与下线旧机)
- [七、Docker Hub 私有镜像发布](#七docker-hub-私有镜像发布)
- [常见问题排查](#常见问题排查)

---

## 架构概览

- **前端**：Vercel（`hls-club.com`），通过 `VITE_API_BASE=https://api.hls-club.com` 访问后端。
- **后端**：服务器上用 `docker compose` 运行两个服务：
  - `caddy`：监听 80/443，自动为 `api.hls-club.com` 申请 / 续期 Let's Encrypt 证书，反向代理到内网 `gameserver:8088`。
  - `gameserver`：Go 编写的游戏服务器，仅在 Docker 内网暴露 `8088`。
- **关键点**：所有配置都基于域名，代码层面零改动。迁移本质 = **新机重新部署 + DNS 指向新 IP + 关停旧机**。

> 因为房间状态只存在内存中，迁移（以及任何重启）都会清空所有房间，建议挑无人在线的时段操作。

---

## 本地开发

### 前置条件

- Go 1.25+
- 本地 MySQL（默认 `root@localhost:3306`，无密码）

### 首次设置

```bash
cd server
cp .env.example .env
```

编辑 `server/.env`，填入本地值（与生产不同的部分）：

```env
HLS_MODE=dev
HLS_CORS_ALLOW_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
HLS_DB_DSN=root@tcp(localhost:3306)/hls?parseTime=true
HLS_JWT_SECRET=any-dev-secret
HLS_WECHAT_OPEN_CALLBACK_DOMAIN=localhost:8088
HLS_WECHAT_OPEN_FRONTEND_URL=http://localhost:5173
```

其余字段（TencentSMS、Shop 等）保持 `.env.example` 的默认值即可，留空不影响启动。

### 启动服务器

```bash
cd server && go run ./cmd/gameserver
```

服务器启动时会自动加载当前目录的 `.env` 文件。`server/.env` 已加入 `.gitignore`，不会被提交。

---

## 服务器信息

| 项目 | 值 |
| --- | --- |
| 提供商 | DMIT |
| 公网 IP | `103.135.249.127` |
| SSH 端口 | `22` |
| 登录用户 | `root` |
| 私钥 | `~/.ssh/dmit.pem` |
| 远程部署目录 | `/root/hls/server` |

> DMIT 的 SSH 端口通常是随机高位端口（不一定是 22），以后台 / 开通邮件为准。本次实例为 `22`。

---

## 一、SSH 密钥连接

DMIT 后台下载的密钥包含三个文件，macOS 上只用 `.pem`：

| 文件 | 用途 | macOS 是否需要 |
| --- | --- | --- |
| `.pem` | OpenSSH 私钥（macOS / Linux 用） | ✅ 用这个 |
| `.ppk` | PuTTY 私钥（Windows PuTTY 用） | ❌ 不用 |
| `.pub` | 公钥（服务器端已部署） | ❌ 一般不用 |

> 私钥（`.pem` / `.ppk`）是机密文件，**切勿提交到 git、切勿外发**。

### 1. 放置私钥并设置权限

```bash
mkdir -p ~/.ssh
mv ~/Downloads/dmit.pem ~/.ssh/dmit.pem
chmod 600 ~/.ssh/dmit.pem
```

> `chmod 600` 必须做。权限过宽 SSH 会拒绝使用密钥并报 `UNPROTECTED PRIVATE KEY FILE`。

### 2. 连接

```bash
ssh -i ~/.ssh/dmit.pem -p 22 root@103.135.249.127
```

### 3. 配置别名（推荐，之后只敲 `ssh dmit`）

编辑 `~/.ssh/config`（没有就新建）：

```
Host dmit
    HostName 103.135.249.127
    Port 22
    User root
    IdentityFile ~/.ssh/dmit.pem
```

之后即可：

```bash
ssh dmit
```

---

## 二、服务器环境准备

DMIT 默认以 `root` 登录，**无需 `sudo`**（系统可能也没装 `sudo`），直接执行命令即可。可先确认身份：

```bash
whoami   # 输出 root 即所有命令去掉 sudo 直接跑
```

### 1. 安装 Docker 与 Compose 插件

```bash
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker
docker compose version   # 确认 compose 可用
```

### 2. 安装 rsync（上传需要）

`rsync` 需本地和远端**两端都有**。macOS 自带，DMIT 全新系统默认没有，需在服务器上安装：

```bash
# Debian / Ubuntu
apt update && apt install -y rsync

# CentOS / Rocky / Alma
yum install -y rsync
```

### 3. 防火墙

DMIT 没有类似腾讯云的「安全组」，默认端口在网络层是放开的，通常**无需配置**。可确认是否有活跃防火墙：

```bash
systemctl status firewalld 2>/dev/null | head -3
iptables -L -n 2>/dev/null | head -20
nft list ruleset 2>/dev/null | head -20
```

若都为空 / 未运行，则 80/443 已通，跳过即可。如确需用 `ufw`：

```bash
apt update && apt install -y ufw
ufw allow 22        # 务必先放行 SSH，否则 enable 后会把自己锁在外面
ufw allow 80
ufw allow 443
ufw enable
```

> `8088` **不需要**对公网开放（仅 Docker 内网使用）。

---

## 三、上传 server 目录

仓库内提供了 `server/upload.sh`，基于 `rsync` 增量同步本地 `server/` 到服务器。

### 用法

在仓库根目录运行，用环境变量传入连接信息：

```bash
SSH_HOST=103.135.249.127 SSH_PORT=22 SSH_KEY=~/.ssh/dmit.pem ./server/upload.sh
```

也可直接修改 `upload.sh` 顶部配置区（`SSH_HOST` / `SSH_PORT` / `SSH_KEY` / `REMOTE_DIR`），之后直接 `./server/upload.sh`。

### 脚本行为

- 用 `rsync -avz --delete` 增量同步（只传改动文件，重复上传很快）。
- 默认上传到服务器的 `/root/hls/server`。
- 自动排除：`.git/`、`.env`、`.DS_Store`、`*.log`。
  - **排除 `.env`** 是为了不覆盖服务器上的生产配置，首次部署需在服务器手动创建。
- `--delete` 会删除远端多出的文件，保持两边一致。

---

## 四、配置环境变量并启动

首次部署，在服务器上创建并填写 `.env`：

```bash
cd /root/hls/server
cp .env.example .env
```

按 `.env.example` 的格式填写所有字段。最少需要设置：

```env
HLS_API_DOMAIN=api.hls-club.com
MYSQL_ROOT_PASSWORD=<随机密码>

HLS_MODE=prod
HLS_CORS_ALLOW_ORIGINS=https://hls-club.com,https://www.hls-club.com
HLS_DB_DSN=root:<随机密码>@tcp(mysql:3306)/hls?parseTime=true
HLS_JWT_SECRET=<随机密钥>
HLS_WECHAT_OPEN_APP_ID=<微信 AppID>
HLS_WECHAT_OPEN_APP_SECRET=<微信 AppSecret>
HLS_WECHAT_OPEN_CALLBACK_DOMAIN=api.hls-club.com
HLS_WECHAT_OPEN_FRONTEND_URL=https://hls-club.com
```

其余字段（TencentSMS、Shop 等）参考 `.env.example` 的注释按需填写。

启动（此时**先不要改 DNS**）：

```bash
docker compose up -d --build
docker compose logs -f
```

此时 Caddy 还拿不到证书（域名仍指向旧机），属正常。可用容器内健康检查验证后端：

```bash
curl http://127.0.0.1:8088/api/healthz
```

---

## 五、迁移：DNS 切换

到 DNS 服务商，把 `api.hls-club.com` 的 A 记录从**腾讯云 IP** 改为 **DMIT IP `103.135.249.127`**。

- 建议提前把该记录 **TTL 调小**（如 60s），切换更快生效。
- 生效后，Caddy 会自动为新 IP 申请 / 验证 Let's Encrypt 证书。

验证 DNS：

```bash
dig +short api.hls-club.com    # 应返回 103.135.249.127
```

---

## 六、验证与下线旧机

DNS 生效后看 Caddy 日志，确认证书签发成功：

```bash
docker compose logs caddy | grep -i certificate
curl https://api.hls-club.com/api/healthz   # 返回正常即成功
```

然后在浏览器实际开个房间、多个标签页连 WebSocket 跑一遍，确认 `wss://api.hls-club.com` 正常。

确认 DMIT 稳定运行一段时间（建议观察一天）后，下线腾讯云旧机：

```bash
# 在腾讯云旧机上
cd server && docker compose down
```

再到腾讯云控制台退订 / 销毁实例。

> 前端无需任何改动：`VITE_API_BASE` 仍是 `https://api.hls-club.com`，域名未变，Vercel 不用动。

---

## 七、Docker Hub 私有镜像发布

本章节描述推荐的发布流程：在 **Mac 本地**跨平台构建镜像，推送到 **Docker Hub 私有仓库**，服务器直接拉取预构建镜像，无需在服务器端编译源码。

> `docker-compose.yml` 中的 `gameserver` 已配置为 `image: kaolengmian77/hls-gameserver:latest`，不再使用 `build:`。

### 1. Docker Hub 仓库设置（一次性）

1. 登录 [hub.docker.com](https://hub.docker.com)，创建 repository `kaolengmian77/hls-gameserver`
2. Visibility 设为 **Private**
3. 进入 **Settings → Security → New Access Token**，创建两个 Token：
   - **本地推送用**：权限 `Read, Write, Delete`，名称如 `mac-push`
   - **服务器拉取用**：权限 `Read-only`，名称如 `dmit-pull`
4. 妥善保存两个 Token（只显示一次）

### 2. Mac 本地：登录 Docker Hub（一次性）

```bash
docker login -u kaolengmian77
# 密码处粘贴「本地推送用」Access Token
```

### 3. Mac 本地：构建并推送镜像

在仓库根目录执行（`v1.0.0` 替换为实际版本号）：

```bash
docker buildx build \
  --platform linux/amd64 \
  -t kaolengmian77/hls-gameserver:v1.0.0 \
  -t kaolengmian77/hls-gameserver:latest \
  --push \
  ./server
```

| 参数 | 说明 |
| --- | --- |
| `--platform linux/amd64` | 跨平台构建，生成服务器（amd64）可用的镜像 |
| `-t ...:v1.0.0 -t ...:latest` | 同时打版本 tag 和 latest |
| `--push` | 构建完直接推送，无需单独 `docker push` |
| `./server` | 构建上下文，含 Dockerfile 和 Go 源码 |

> 首次执行时 Docker Desktop 会下载跨平台工具链，速度较慢，后续有缓存会快很多。

### 4. 服务器：登录 Docker Hub（一次性）

```bash
docker login -u kaolengmian77
# 密码处粘贴「服务器拉取用」Access Token
# 凭据保存至 ~/.docker/config.json，后续 pull 自动使用
```

### 5. 日常发版流程

**每次发布新版本：**

```bash
# Mac 端：构建并推送（版本号自增）
docker buildx build \
  --platform linux/amd64 \
  -t kaolengmian77/hls-gameserver:v1.0.1 \
  -t kaolengmian77/hls-gameserver:latest \
  --push \
  ./server

# 服务器端：拉取新镜像并重启
cd /root/hls/server
docker compose pull
docker compose up -d
```

**回滚到指定版本：**

```bash
# 服务器端：修改 docker-compose.yml 中的 tag，然后重启
sed -i 's|hls-gameserver:latest|hls-gameserver:v1.0.0|' docker-compose.yml
docker compose pull
docker compose up -d
```

---

## 常见问题排查

| 现象 | 原因 / 解决 |
| --- | --- |
| `UNPROTECTED PRIVATE KEY FILE` | 私钥权限过宽，执行 `chmod 600 ~/.ssh/dmit.pem`。 |
| `Permission denied (publickey)` | `.pem` 路径不对、用户名错（DMIT 一般是 `root`），或密钥未绑定。 |
| `Connection refused` / 超时 | 端口或 IP 不对，多半漏了 `-p` 或端口号错。 |
| `sudo: command not found` | 已是 root，去掉命令前的 `sudo` 直接跑。 |
| `rsync: command not found`（服务器端） | 服务器没装 rsync，执行 `apt install -y rsync`。 |
| `ufw: command not found` | 系统未启用 ufw，DMIT 默认无防火墙拦截，一般无需处理。 |
| Caddy 拿不到证书 | 99% 是 80/443 未放行，或 DNS 还没生效到 DMIT IP。 |
