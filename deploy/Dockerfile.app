# syntax=docker/dockerfile:1
# 底座镜像：FROM 里引用的 ARG 必须在第一个 FROM 之前声明（全局作用域），
# 否则 buildx 报 "FROM argument BASE_IMAGE is not declared / base name should not be blank"。
ARG BASE_IMAGE=swr.cn-north-4.myhuaweicloud.com/swr-hd-mig/loopforge:clis-hermes
# 第 3 层：应用层（每次发版只重建这一层，~130 MB）
#
#   编译 chatbot 前端 + Go 二进制（构建阶段不进最终镜像），
#   只把 /out 里的产物 COPY 到 BASE_IMAGE 之上。
#
# BASE_IMAGE 应当是「第 2 层」（带全部 Agent CLI 的镜像）。因为这一层的
# 唯一内容就是 COPY 二进制，所以每次发版：本地只重建 ×秒级，推送/拉取
# 只涉及这一层的新增字节 —— CLI 那 7-8 GB 永远命中缓存。
#
# 构建（默认底座 = 当前线上跑的镜像，可用 BASE_IMAGE 覆盖）：
#   docker build -f deploy/Dockerfile.app \
#     --build-arg BASE_IMAGE=swr.cn-north-4.myhuaweicloud.com/swr-hd-mig/loopforge:clis \
#     -t loopforge:app-<sha>-<utc> --platform linux/arm64 .
FROM --platform=$BUILDPLATFORM node:22-bookworm-slim AS chatbot
WORKDIR /src
COPY chatbot/package.json chatbot/package-lock.json chatbot/
RUN --mount=type=cache,target=/root/.npm npm --prefix chatbot ci --include=dev --no-audit --no-fund
COPY chatbot/ chatbot/
COPY web/app/ web/app/
RUN npm --prefix chatbot run build && test -f web/native-chat/index.html

FROM --platform=$BUILDPLATFORM golang:1.26.11-bookworm AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
# Empty lets build-binaries.sh size parallelism from the builder's CPUs and memory.
ARG LOOPFORGE_BUILD_JOBS=
ENV LOOPFORGE_BUILD_JOBS=${LOOPFORGE_BUILD_JOBS}
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
# Ref updates must not invalidate the Go build cache.
COPY --exclude=.git . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build GOOS=${TARGETOS} GOARCH=${TARGETARCH} bash scripts/build-binaries.sh --helpers /out
COPY --from=chatbot /src/web/native-chat/ web/native-chat/
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build GOOS=${TARGETOS} GOARCH=${TARGETARCH} bash scripts/build-binaries.sh --platform /out

# ---- 最终层：只做 COPY ----
# 注意：buildx 要求 FROM 用到的 ARG 必须在使用它的 FROM 之前声明，否则报
# "base name ($BASE_IMAGE) should not be blank"
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
WORKDIR /app
# 二进制先落到临时目录再换，避免覆盖正在运行的同一 inode（bind mount 场景）
COPY --from=build /out/ /app/bin.new/
RUN rm -rf /app/bin.prev && ( [ -d /app/bin ] && mv /app/bin /app/bin.prev || true ) \
 && mv /app/bin.new /app/bin && rm -rf /app/bin.prev \
 && chmod 755 /app/bin/*
COPY LICENSE THIRD_PARTY_NOTICES.md /app/
# An explicit LOOPFORGE_COMMIT wins. Otherwise read HEAD from the build context
# so `docker build` stamps the same corner label as the release workflow.
ARG LOOPFORGE_COMMIT=
RUN --mount=type=bind,source=.,target=/ctx,ro \
    sh /ctx/scripts/image-commit.sh /ctx /app/loopforge-commit
EXPOSE 8080
ENTRYPOINT ["/app/bin/agentflow"]
