#!/usr/bin/env bash
#
# 将本地 server/ 目录同步到远程服务器。
#
# 用法:
#   1. 按需修改下面的配置，或通过环境变量覆盖：
#        SSH_HOST=1.2.3.4 SSH_PORT=22622 SSH_KEY=~/.ssh/dmit.pem ./upload.sh
#   2. 直接运行：
#        ./upload.sh
#
# 依赖：本地需安装 rsync（macOS 自带）。

set -euo pipefail

# ---- 配置（可用环境变量覆盖）-------------------------------------------------
SSH_HOST="${SSH_HOST:-CHANGE_ME_IP}"          # 服务器公网 IP
SSH_PORT="${SSH_PORT:-22}"                     # DMIT 给的 SSH 端口（注意通常不是 22）
SSH_USER="${SSH_USER:-root}"                   # 登录用户
SSH_KEY="${SSH_KEY:-$HOME/.ssh/dmit.pem}"      # 私钥路径（.pem）
REMOTE_DIR="${REMOTE_DIR:-/root/hls/server}"   # 服务器上的目标目录
# -----------------------------------------------------------------------------

# 本脚本所在目录即为要上传的 server/ 目录
LOCAL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ "$SSH_HOST" == "CHANGE_ME_IP" ]]; then
	echo "❌ 请先设置 SSH_HOST（脚本顶部配置，或用环境变量）。" >&2
	echo "   例：SSH_HOST=1.2.3.4 SSH_PORT=22622 SSH_KEY=~/.ssh/dmit.pem ./upload.sh" >&2
	exit 1
fi

if [[ ! -f "$SSH_KEY" ]]; then
	echo "❌ 找不到私钥文件：$SSH_KEY" >&2
	exit 1
fi

echo "▶ 本地目录 : $LOCAL_DIR"
echo "▶ 远程目标 : ${SSH_USER}@${SSH_HOST}:${REMOTE_DIR} (port ${SSH_PORT})"
echo

# 确保远程目录存在
ssh -i "$SSH_KEY" -p "$SSH_PORT" "${SSH_USER}@${SSH_HOST}" "mkdir -p '${REMOTE_DIR}'"

# rsync 同步
#   -a 归档（保留权限/软链等）  -v 详细  -z 压缩传输  --delete 删除远端多余文件
#   --exclude 排除不应上传的内容
rsync -avz --delete \
	--exclude='.git/' \
	--exclude='.env' \
	--exclude='.DS_Store' \
	--exclude='*.log' \
	-e "ssh -i '$SSH_KEY' -p $SSH_PORT" \
	"$LOCAL_DIR/" \
	"${SSH_USER}@${SSH_HOST}:${REMOTE_DIR}/"

echo
echo "✅ 同步完成。接下来可在服务器上："
echo "   cd ${REMOTE_DIR}"
echo "   cp .env.example .env   # 首次部署，按需填写"
echo "   docker compose up -d --build"
