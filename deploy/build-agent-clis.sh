#!/usr/bin/env bash
# 构建「平台 + Agent CLI」能力镜像。默认只构建、不推送；加 --push 才推。
#
#   BASE_IMAGE=<平台镜像> WITH_HERMES=1 ./deploy/build-agent-clis.sh [--push]
#
# 产物标签：clis-<commit>-<UTC>，另打别名 <REG>/<NS>/loopforge:clis
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

REG="${REG:-swr.cn-north-4.myhuaweicloud.com}"
NS="${NS:-swr-hd-mig}"
BASE_IMAGE="${BASE_IMAGE:-$REG/$NS/loopforge:190d948-202611161657UTC}"
WITH_HERMES="${WITH_HERMES:-0}"
SHA=$(git rev-parse --short HEAD)
SUFFIX=""
[[ "$WITH_HERMES" == "1" ]] && SUFFIX="-hermes"
TAG="clis${SUFFIX}-${SHA}-$(date -u +%Y%m%d%H%M)UTC"
LOCAL="loopforge:${TAG}"
REMOTE="${REG}/${NS}/loopforge:${TAG}"
ALIAS="${REG}/${NS}/loopforge:clis"

echo "base      : $BASE_IMAGE"
echo "hermes    : $WITH_HERMES (1=预装)"
echo "local tag : $LOCAL"

docker build -f deploy/Dockerfile.agent-clis \
  --build-arg BASE="$BASE_IMAGE" \
  --build-arg WITH_HERMES="$WITH_HERMES" \
  -t "$LOCAL" .

echo "--- 自检（镜像内 CLI 版本）---"
docker run --rm --entrypoint sh "$LOCAL" -c '
  for b in opencode codex dsh; do printf "%s: " "$b"; command -v "$b" >/dev/null && "$b" --version 2>&1 | head -1 || echo MISSING; done
  printf "hermes: "; if [ -x "${AGENTFLOW_HERMES_BIN:-/nonexistent}" ]; then HERMES_HOME="${AGENTFLOW_HERMES_HOME}" "$AGENTFLOW_HERMES_BIN" --version 2>&1 | head -1; else echo "not baked"; fi
  echo "env: AGENTFLOW_OPENCODE_BIN=$AGENTFLOW_OPENCODE_BIN"
' || echo "自检失败（镜像仍已构建，请人工核查）"

if [[ "${1:-}" == "--push" ]]; then
  docker tag "$LOCAL" "$REMOTE"
  docker tag "$LOCAL" "$ALIAS"
  docker push "$REMOTE"
  docker push "$ALIAS"
  echo "pushed: $REMOTE"
  echo "alias : $ALIAS"
else
  echo "未推送（加 --push 才推）。手动推送命令："
  echo "  docker tag $LOCAL $REMOTE && docker push $REMOTE"
  echo "  docker tag $LOCAL $ALIAS && docker push $ALIAS"
fi