#!/usr/bin/env python3
"""把 app 层的「COPY 到临时目录 + RUN 移动」改成单层 COPY --chmod。

原因：overlay 里 mv 同一批文件会再落一层，实测压缩后多 68 MB —— 每发一次版付两次钱。
"""
import pathlib

p = pathlib.Path(__file__).resolve().parents[1] / "deploy" / "Dockerfile.app"
s = p.read_text(encoding="utf-8")

start = s.index("# 二进制先落到临时目录再换")
end = s.index("COPY LICENSE THIRD_PARTY_NOTICES.md /app/")
old_block = s[start:end]

new_block = """# 直接 COPY 覆盖，权限在同一条指令里给定（--chmod），不再额外产生一层。
# 不要写成「COPY 到临时目录再 RUN 移动」：overlay 里移动同一批文件会再落一层，
# 实测压缩后要多传 68 MB，等于每发一次版付两次钱。
COPY --chmod=755 --from=build /out/ /app/bin/
"""

s = s[:start] + new_block + s[end:]
p.write_text(s, encoding="utf-8")
print("已改为单层 COPY --chmod=755")
print("--- 最终层现在长这样 ---")
print("\n".join(s.splitlines()[-12:]))
