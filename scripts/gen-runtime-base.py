#!/usr/bin/env python3
"""从仓库根 Dockerfile 的 runtime 阶段抽出第 1 层 runtime-base。

为什么用脚本而不是手写：
    直接手抄 Dockerfile 会经过"读取→显示→重写"，而显示环节会把看起来像密钥的
    长串（例如 python-build-standalone 的版本号/SHA）打码。抄一次就会把真值换成
    星号，构建时 curl 404 / sha256 校验失败。所以这里整段从磁盘原样搬。
"""
import pathlib
import re

root = pathlib.Path(__file__).resolve().parents[1]  # scripts/ → 仓库根
src = (root / "Dockerfile").read_text(encoding="utf-8")

# runtime 阶段：从 FROM node:22-bookworm 到文件结尾
m = re.search(r"^FROM node:22-bookworm$", src, re.M)
assert m, "Dockerfile 里没找到 runtime 阶段的 FROM"
stage = src[m.start():]

kept = []
for line in stage.splitlines():
    # 这些属于应用层（Dockerfile.app），不放底座
    if line.startswith("COPY --from=build"):
        continue
    if line.startswith("COPY LICENSE"):
        continue
    if line.strip().startswith("ENTRYPOINT"):
        continue
    if line.strip().startswith("ARG LOOPFORGE_COMMIT"):
        continue
    # commit 戳属于应用层；注意要连 RUN 行本身一起跳过，否则会留下悬空的
    # `RUN --mount=...` 与下一行 EXPOSE 粘连，构建报 "EXPOSE: not found"（exit 127）
    if line.strip().startswith("RUN --mount=type=bind"):
        continue
    if line.strip().startswith("sh /ctx/scripts/image-commit.sh"):
        continue
    if line.strip().startswith("--mount=type=bind,source=.,target=/ctx,ro"):
        continue
    kept.append(line)

body = "\n".join(kept).strip() + "\n"

# 构建时走镜像源（CI 在境外可覆盖）
body = body.replace(
    "FROM node:22-bookworm\n",
    "FROM node:22-bookworm\n"
    "# 国内构建走镜像源；CI 在境外可 --build-arg NPM_REGISTRY=https://registry.npmjs.org 覆盖\n"
    'ARG NPM_REGISTRY=https://registry.npmmirror.com\n',
    1,
)
assert "npm install -g" in body
body = body.replace("npm install -g ", 'npm install -g --registry "${NPM_REGISTRY}" ', 1)

header = """# syntax=docker/dockerfile:1
# 第 1 层：运行环境底座（很少变）
#
#   node:22 + apt 工具 + 基础 CLI（codex / codebuddy / dsh）+ 可搬迁 Python + docker CLI
#
# 这一层刻意不含平台二进制与业务代码：
#   - 二进制在第 3 层（Dockerfile.app）里 COPY 进来，改业务代码不会让它失效
#   - 只有「换 Node / 换 apt 包 / 换基础 CLI 版本 / 换 Python」才需要重建
#
# 由 scripts/gen-runtime-base.py 从根 Dockerfile 原样生成，改根文件后重跑即可。
# 构建：docker build -f deploy/Dockerfile.runtime-base -t loopforge:runtime-<date> .
"""
out = root / "deploy" / "Dockerfile.runtime-base"
out.write_text(header + body, encoding="utf-8")
print(f"已生成 {out.relative_to(root)}（{len(body.splitlines())} 行底座内容）")
