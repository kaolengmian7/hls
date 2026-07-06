# Docker Hub 私有镜像发布流程 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `server/docker-compose.yml` 改为使用 Docker Hub 私有镜像，并在 `DEPLOYMENT.md` 补充完整的构建、推送、拉取发布流程文档。

**Architecture:** 修改 `gameserver` 服务配置，去掉 `build:` 块，改用预构建镜像 `kaolengmian77/hls-gameserver:latest`。新增 DEPLOYMENT.md 章节，覆盖 Docker Hub 仓库设置、Mac 本地跨平台构建推送、服务器登录认证、日常发版三个子流程。

**Tech Stack:** Docker Buildx（跨平台构建，随 Docker Desktop 自带），Docker Hub 私有仓库，docker-compose v2

---

### Task 1: 修改 `server/docker-compose.yml`

**Files:**
- Modify: `server/docker-compose.yml:20-24`

- [ ] **Step 1: 替换 `gameserver` 的 `build:` 块为 `image:`**

将 `server/docker-compose.yml` 中 `gameserver` 服务的以下内容：

```yaml
  gameserver:
    build:
      context: .
      dockerfile: Dockerfile
    image: hls-gameserver:local
```

替换为：

```yaml
  gameserver:
    image: kaolengmian77/hls-gameserver:latest
```

- [ ] **Step 2: 验证 YAML 语法**

```bash
cd server && docker compose config --quiet && echo "YAML OK"
```

预期输出：`YAML OK`（无报错）

- [ ] **Step 3: Commit**

```bash
git add server/docker-compose.yml
git commit -m "chore(deploy): use Docker Hub image instead of local build"
```

---

### Task 2: 在 `DEPLOYMENT.md` 增加目录条目

**Files:**
- Modify: `server/DEPLOYMENT.md:4-16`（目录区）

- [ ] **Step 1: 在目录末尾（`常见问题排查` 条目之前）插入新条目**

找到目录中：

```markdown
- [六、验证与下线旧机](#六验证与下线旧机)
- [常见问题排查](#常见问题排查)
```

替换为：

```markdown
- [六、验证与下线旧机](#六验证与下线旧机)
- [七、Docker Hub 私有镜像发布](#七docker-hub-私有镜像发布)
- [常见问题排查](#常见问题排查)
```

- [ ] **Step 2: Commit**

```bash
git add server/DEPLOYMENT.md
git commit -m "docs: add Docker Hub workflow TOC entry"
```

---

### Task 3: 在 `DEPLOYMENT.md` 插入新章节内容

**Files:**
- Modify: `server/DEPLOYMENT.md`（在 `## 常见问题排查` 之前插入新章节）

- [ ] **Step 1: 在 `## 常见问题排查` 标题行之前插入以下内容**

找到：

```markdown
## 常见问题排查
```

在其上方插入（注意上方保留一个空行）：

```markdown
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

```

- [ ] **Step 2: 验证新章节在文件中存在**

```bash
grep -n "七、Docker Hub" server/DEPLOYMENT.md
```

预期输出：匹配到对应行号，如 `308:## 七、Docker Hub 私有镜像发布`

- [ ] **Step 3: Commit**

```bash
git add server/DEPLOYMENT.md
git commit -m "docs: add Docker Hub private image publish workflow to DEPLOYMENT.md"
```

---

## 自检

- [x] spec 要求的所有内容均有对应 task：Docker Hub 设置、本地登录、构建推送、服务器登录、日常发版、回滚
- [x] 无 TBD / TODO / 占位符
- [x] `docker-compose.yml` 修改与文档中描述的 `image:` 字段一致
- [x] 版本号示例前后一致（`v1.0.0` → `v1.0.1`）
- [x] 平台参数（`linux/amd64`）贯穿始终
