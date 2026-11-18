#!/usr/bin/env bash
# 三层镜像构建：runtime-base → clis → app
#
#   ./deploy/build-images.sh [--stage runtime|clis|app|all] [--push] [--hermes]
#                            [--platform linux/arm64] [--reuse-runtime] [--reuse-clis]
#
# 为什么要分层（实测动机）：
#   把「运行底座」「Agent CLI」「平台二进制」塞在一个镜像里时，改一行 Go 代码会让
#   CLI 层（7~8 GB）的父层失效，于是每次都重装 npm、重推镜像。分层之后：
#     runtime-base  只在换 Node / apt 包 / 基础 CLI / Python 时重建
#     clis          只在装/升 Agent CLI 时重建
#     app           每次发版都重建，但只有 ~130 MB，推送与拉取都只涉及这一层
#
# 产物标签：
#   <REG>/<NS>/loopforge:runtime-<date>             + :runtime
#   <REG>/<NS>/loopforge:clis[-hermes]-<sha>-<utc>  + :clis[-hermes]
#   <REG>/<NS>/loopforge:app-<sha>-<utc>            + :app
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

REG="${REG:-swr.cn-north-4.myhuaweicloud.com}"
NS="${NS:-swr-hd-mig}"
REPO="${REPO:-loopforge}"
PLATFORM="${PLATFORM:-linux/arm64}"
STAGE=all
PUSH=0
HERMES=0
REUSE_RUNTIME=0
REUSE_CLIS=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --stage) STAGE="$2"; shift 2 ;;
    --push) PUSH=1; shift ;;
    --hermes) HERMES=1; shift ;;
    --platform) PLATFORM="$2"; shift 2 ;;
    --reuse-runtime) REUSE_RUNTIME=1; shift ;;
    --reuse-clis) REUSE_CLIS=1; shift ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

SHA=$(git rev-parse --short HEAD)
UTC=$(date -u +%Y%m%d%H%M)UTC
DATE=$(date -u +%Y%m%d)
SUFFIX=""; [[ $HERMES == 1 ]] && SUFFIX="-hermes"

RUNTIME_LOCAL="loopforge:runtime-$DATE"
RUNTIME_REMOTE="$REG/$NS/$REPO:runtime-$DATE"
CLIS_LOCAL="loopforge:clis$SUFFIX-$SHA-$UTC"
CLIS_REMOTE="$REG/$NS/$REPO:clis$SUFFIX-$SHA-$UTC"
APP_LOCAL="loopforge:app-$SHA-$UTC"
APP_REMOTE="$REG/$NS/$REPO:app-$SHA-$UTC"

say() { printf '\n=== %s ===\n' "$*"; }
exists() { docker image inspect "$1" >/dev/null 2>&1; }

push_pair() { # $1 local  $2 remote  $3 alias
  docker tag "$1" "$2"
  docker push "$2"
  docker tag "$1" "$3"
  docker push "$3"
  echo "pushed: $2  (alias $3)"
}

build_runtime() {
  say "1/3 runtime-base → $RUNTIME_LOCAL"
  if [[ $REUSE_RUNTIME == 1 ]] && exists "$RUNTIME_LOCAL"; then
    echo "复用已存在的 $RUNTIME_LOCAL"
  else
    docker build -f deploy/Dockerfile.runtime-base --platform "$PLATFORM" -t "$RUNTIME_LOCAL" .
  fi
  [[ $PUSH == 1 ]] && push_pair "$RUNTIME_LOCAL" "$RUNTIME_REMOTE" "$REG/$NS/$REPO:runtime"
  return 0
}

build_clis() {
  say "2/3 clis → $CLIS_LOCAL（hermes=$HERMES）"
  if [[ $REUSE_CLIS == 1 ]] && exists "$CLIS_LOCAL"; then
    echo "复用已存在的 $CLIS_LOCAL"
  else
    docker build -f deploy/Dockerfile.agent-clis --platform "$PLATFORM" \
      --build-arg BASE="$RUNTIME_LOCAL" \
      --build-arg WITH_HERMES="$HERMES" \
      -t "$CLIS_LOCAL" .
  fi
  [[ $PUSH == 1 ]] && push_pair "$CLIS_LOCAL" "$CLIS_REMOTE" "$REG/$NS/$REPO:clis$SUFFIX"
  return 0
}

build_app() {
  say "3/3 app → $APP_LOCAL（底座 $CLIS_LOCAL）"
  docker build -f deploy/Dockerfile.app --platform "$PLATFORM" \
    --build-arg BASE_IMAGE="$CLIS_LOCAL" \
    -t "$APP_LOCAL" .
  docker run --rm --entrypoint sh "$APP_LOCAL" -c '
    printf "commit: "; cat /app/loopforge-commit 2>/dev/null || echo MISSING
    printf "agentflow: "; /app/bin/agentflow --version 2>&1 | head -1 || echo MISSING
    for b in codex opencode dsh; do printf "%s: " "$b"; command -v "$b" >/dev/null && "$b" --version 2>&1 | head -1 || echo MISSING; done'
  [[ $PUSH == 1 ]] && push_pair "$APP_LOCAL" "$APP_REMOTE" "$REG/$NS/$REPO:app"
  return 0
}

case "$STAGE" in
  runtime) build_runtime ;;
  clis)    build_clis ;;
  app)     build_app ;;
  all)     build_runtime; build_clis; build_app ;;
  *) echo "--stage must be runtime|clis|app|all" >&2; exit 2 ;;
esac

say "完成"
printf 'runtime : %s\nclis    : %s\napp     : %s\n' "$RUNTIME_LOCAL" "$CLIS_LOCAL" "$APP_LOCAL"
[[ $PUSH == 1 ]] && echo "（已推送至 $REG/$NS/$REPO）" || echo "（未推送，加 --push 才推）"
