#!/usr/bin/env bash
# 局域网直传部署：不经过镜像仓库，把本机构建好的镜像原样送到目标机器并重启平台。
#
#   SRC=loopforge:clis-hermes-a044d95-202611162334UTC HOST=root@192.168.0.42 \
#     ./deploy/lan-transfer.sh
#
# 变量：SRC（必填，本地镜像）、HOST（默认 root@192.168.0.42）、
#       APP_DIR（远端目录，默认 /opt/LoopForge）、RESTART=1（默认 1，改 .env 并重启）、
#       COMPRESS=1（可选，gzip -1；局域网通常不需要）
# SSH：走 SSH_ASKPASS/密钥均可（脚本不读明文密码）
set -euo pipefail
cd "$(git rev-parse --show-toplevel 2>/dev/null || echo .)"

SRC="${SRC:?需要 SRC=<本地镜像名>}"
HOST="${HOST:-root@192.168.0.42}"
APP_DIR="${APP_DIR:-/opt/LoopForge}"
RESTART="${RESTART:-1}"
COMPRESS="${COMPRESS:-0}"

SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=10)
if [[ -n "${SSH_ASKPASS:-}" ]]; then
  SSH_OPTS+=(-o PreferredAuthentications=password -o PubkeyAuthentication=no -o NumberOfPasswordPrompts=1)
fi
rssh() { ssh "${SSH_OPTS[@]}" "$HOST" "$@" </dev/null; }
# 传输时必须保留本地 stdin（docker save 的流要灌进去），不能重定向 /dev/null
rssh_pipe() { ssh "${SSH_OPTS[@]}" "$HOST" "$@"; }

docker image inspect "$SRC" >/dev/null 2>&1 || { echo "本地没有镜像 $SRC"; exit 1; }
BYTES=$(docker image inspect "$SRC" --format '{{.Size}}')
ARCH=$(docker image inspect "$SRC" --format '{{.Os}}/{{.Architecture}}')
echo "镜像   : $SRC ($ARCH, $(( BYTES / 1024 / 1024 )) MB)"
echo "目标   : $HOST:$APP_DIR"

START=$(date +%s)
if [[ "$COMPRESS" == "1" ]]; then
  docker save "$SRC" | gzip -1 | rssh_pipe "gunzip | docker load" | tail -2
else
  docker save "$SRC" | rssh_pipe "docker load" | tail -2
fi
SECS=$(( $(date +%s) - START ))
(( SECS > 0 )) || SECS=1
echo "传输+导入: ${SECS}s（$(( BYTES / 1024 / 1024 / SECS )) MB/s 平均）"

rssh "docker image inspect '$SRC' >/dev/null 2>&1 && echo '远端镜像已就位'"

if [[ "$RESTART" == "1" ]]; then
  echo "--- 更新 .env 并重启（LOOPFORGE_IMAGE 指向本传镜像，不带 registry 前缀 → compose 不会去 pull）---"
  rssh "cd '$APP_DIR' && cp -a .env .env.bak-\$(date +%Y%m%d-%H%M) && sed -i \"s|^LOOPFORGE_IMAGE=.*|LOOPFORGE_IMAGE='$SRC'|\" .env && grep -E '^LOOPFORGE_(IMAGE|BIND)' .env && docker compose --profile platform up -d 2>&1 | tail -3"
  sleep 12
  rssh "docker ps --format '{{.Names}} {{.Status}} {{.Image}}'; echo '--- health ---'; curl -sS -m 8 http://127.0.0.1:8090/api/health; echo; echo '--- CLI ---'; docker exec loopforge-platform-1 sh -c 'for b in opencode codex dsh; do printf \"%s: \" \"\$b\"; \"\$b\" --version 2>/dev/null | tail -1; done; printf \"hermes: \"; HERMES_HOME=/opt/hermes-home/.hermes /usr/local/bin/hermes --version 2>&1 | head -1'"
fi