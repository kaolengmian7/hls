---
name: docker-hub-workflow
description: 使用 Docker Hub 私有仓库：Mac 本地跨平台构建镜像，推送至 Docker Hub，服务器拉取并部署
metadata:
  type: project
---

# Docker Hub 私有镜像发布流程设计

## 背景

当前 `docker-compose.yml` 使用 `build:` 在服务器端从源码构建镜像（`docker compose up --build`）。
新流程改为：**Mac 本地跨平台构建 → 推送 Docker Hub 私有仓库 → 服务器拉取预构建镜像部署**。

服务器架构：`linux/amd64`。Mac 为 Apple Silicon（arm64），需跨平台构建。

## 镜像命名

- Docker Hub 用户：`kaolengmian77`
- 镜像名：`kaolengmian77/hls-gameserver`
- 版本策略：语义化版本 tag（如 `v1.0.0`）+ 同步更新 `latest`

## 整体流程

```
Mac 本地                    Docker Hub (私有)         服务器
  │                               │                     │
  ├─ docker buildx build          │                     │
  │   --platform linux/amd64      │                     │
  │   -t kaolengmian77/           │                     │
  │      hls-gameserver:v1.0.0    │                     │
  │   -t .../hls-gameserver:latest│                     │
  │   --push ./server             │                     │
  │──────────────────────────────>│                     │
  │                               │                     │
  │                               │<── docker login ────┤ (一次性)
  │                               │                     │
  │                               │<── docker compose pull + up -d
```

## 各部分说明

### 一、Docker Hub 仓库设置（一次性）

1. 登录 hub.docker.com，创建 repository `kaolengmian77/hls-gameserver`
2. Visibility 设为 **Private**
3. 进入 Settings → Security → New Access Token，权限选 `Read, Write, Delete`（本地推送需要），保存 Token

> 服务器侧 Token 只需 `Read-only` 权限即可拉取。可单独为服务器创建一个只读 Token。

### 二、Mac 本地：登录 Docker Hub（一次性）

```bash
docker login -u kaolengmian77
# 提示输入密码时，粘贴 Access Token（不是账号密码）
```

### 三、Mac 本地：构建并推送镜像

在仓库根目录执行：

```bash
docker buildx build \
  --platform linux/amd64 \
  -t kaolengmian77/hls-gameserver:v1.0.0 \
  -t kaolengmian77/hls-gameserver:latest \
  --push \
  ./server
```

- `--platform linux/amd64`：跨平台构建，生成服务器可用的 amd64 镜像
- `-t ... :v1.0.0 -t ... :latest`：同时打两个 tag
- `--push`：构建完直接推送，无需本地 `docker push`
- `./server`：构建上下文（包含 Dockerfile 和源码）

### 四、修改 `server/docker-compose.yml`

将 `gameserver` 服务的 `build:` 块替换为 `image:`：

```yaml
# 修改前
gameserver:
  build:
    context: .
    dockerfile: Dockerfile
  image: hls-gameserver:local

# 修改后
gameserver:
  image: kaolengmian77/hls-gameserver:latest
```

其余配置（`container_name`、`env_file`、`healthcheck` 等）不变。

### 五、服务器：一次性配置登录凭据

```bash
docker login -u kaolengmian77
# 输入只读 Access Token 作为密码
# 凭据保存至 ~/.docker/config.json，后续 pull 自动使用
```

### 六、日常发版流程

**Mac 端（每次发新版本）：**

```bash
# 更新版本号，推送到 Docker Hub
docker buildx build \
  --platform linux/amd64 \
  -t kaolengmian77/hls-gameserver:v1.0.1 \
  -t kaolengmian77/hls-gameserver:latest \
  --push \
  ./server
```

**服务器端（拉取并重启）：**

```bash
cd /root/hls/server
docker compose pull
docker compose up -d
```

## 补充到 DEPLOYMENT.md 的位置

在现有 `## 四、配置环境变量并启动` 章节之后，新增 `## 七、使用 Docker Hub 发布镜像（推荐）` 章节，替代 `docker compose up --build` 的旧方式。同时更新 `docker-compose.yml` 中 `gameserver` 的 `image` 字段。

## 注意事项

- `docker buildx` 随 Docker Desktop 自带，无需单独安装
- 首次跨平台构建需要 QEMU 支持，Docker Desktop 已内置，Mac 上直接可用
- `.env` 文件始终保留在服务器，不通过镜像传递（镜像不含任何密钥）
- 回滚：将 `docker-compose.yml` 中的 tag 改为旧版本号后 `docker compose pull && up -d`
