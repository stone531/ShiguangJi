#!/usr/bin/env bash
# 时光集离线部署包打包脚本（CentOS 7）：本机构建 → 组装单个 tar.gz 手动上传
#
# 用法：
#   bash deploy/package.sh                    # linux/amd64（阿里云 x86_64 实例）
#   GOARCH=arm64 bash deploy/package.sh       # arm 实例
#
# 产物：build/blog-centos7.tar.gz
# 服务器步骤见 deploy/README-部署-centos7.md（包内同名携带）

set -euo pipefail

GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUILD_DIR="$ROOT/build"
STAGE="$BUILD_DIR/stage/blog"

echo "==> [1/4] 构建前端"
(cd "$ROOT/frontend" && npm ci --no-fund --no-audit && npm run build)

echo "==> [2/4] 交叉编译后端 ($GOOS/$GOARCH 静态链接)"
rm -rf "$BUILD_DIR/stage"
mkdir -p "$STAGE"
(cd "$ROOT/backend" && \
  CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
  go build -trimpath -ldflags "-s -w" -o "$STAGE/blog-server" ./cmd/server && \
  CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
  go build -trimpath -ldflags "-s -w" -o "$STAGE/blog-initadmin" ./cmd/initadmin)

echo "==> [3/4] 组装"
cp -R "$ROOT/frontend/dist" "$STAGE/dist"
cp -R "$ROOT/backend/migrations" "$STAGE/migrations"
cp "$ROOT/deploy/init.sql" \
   "$ROOT/deploy/blog.service" \
   "$ROOT/deploy/blogctl" \
   "$ROOT/deploy/nginx-centos7.conf" \
   "$ROOT/deploy/README-部署-centos7.md" \
   "$ROOT/backend/.env.example" "$STAGE/"
chmod +x "$STAGE/blogctl"
echo "==> [4/4] 压缩"
rm -f "$BUILD_DIR/blog-centos7.tar.gz"
tar -czf "$BUILD_DIR/blog-centos7.tar.gz" -C "$BUILD_DIR/stage" blog
rm -rf "$BUILD_DIR/stage"

echo "完成: $BUILD_DIR/blog-centos7.tar.gz ($(du -h "$BUILD_DIR/blog-centos7.tar.gz" | cut -f1))"
echo "下一步：scp build/blog-centos7.tar.gz root@服务器:/tmp/  然后按包内 README 操作"
