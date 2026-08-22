#!/usr/bin/env bash
# 时光集一键部署：本机构建 → scp 产物 → 重启服务（spec §10）
#
# 用法：
#   SERVER=root@example.com ./deploy/deploy.sh          # 常规更新
#
# 注意：initadmin 会同时「应用迁移 + 重置管理员密码」，本脚本绝不自动执行；
#   若本次发布包含新增 migrations/*.sql，脚本结束时会打印手动执行提示。
#
# 首次部署（本脚本不覆盖，见根 README「部署」）：
#   建库建用户 → 装 nginx/systemd 单元 → 写 /var/www/blog/.env（ADDR=127.0.0.1:8080、
#   APP_ENV=prod、JWT_SECRET、LIKE_SALT、DB_USER/DB_PASSWORD 或 DSN）→ 初始化数据库
#   （推荐直接导入 deploy/init.sql：建表+初始数据+初始管理员 admin/admin123456；
#    或手动执行一次 blog-initadmin 建管理员，注意其会重置已有管理员密码）
#
# 目标机约定 linux/amd64；如为 arm 服务器改 GOARCH=arm64

set -euo pipefail

SERVER="${SERVER:?请通过环境变量指定目标机，如 SERVER=user@host}"
REMOTE_DIR="${REMOTE_DIR:-/var/www/blog}"
GOARCH="${GOARCH:-amd64}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUILD_DIR="$ROOT/build"

echo "==> [1/4] 构建前端"
(cd "$ROOT/frontend" && npm ci --no-fund --no-audit && npm run build)

echo "==> [2/4] 交叉编译后端 (linux/$GOARCH)"
mkdir -p "$BUILD_DIR"
(cd "$ROOT/backend" && \
  CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
  go build -trimpath -ldflags "-s -w" -o "$BUILD_DIR/blog-server" ./cmd/server)
(cd "$ROOT/backend" && \
  CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
  go build -trimpath -ldflags "-s -w" -o "$BUILD_DIR/blog-initadmin" ./cmd/initadmin)

echo "==> [3/4] 上传产物"
ssh "$SERVER" "mkdir -p $REMOTE_DIR/dist $REMOTE_DIR/uploads $REMOTE_DIR/migrations"
rsync -az --delete "$ROOT/frontend/dist/" "$SERVER:$REMOTE_DIR/dist/"
rsync -az "$BUILD_DIR/blog-server" "$BUILD_DIR/blog-initadmin" "$SERVER:$REMOTE_DIR/"
rsync -az "$ROOT/backend/migrations/" "$SERVER:$REMOTE_DIR/migrations/"
# 服务进程以 blog 用户运行，统一属主（.env 除外：不覆盖远端已有密钥文件）
ssh "$SERVER" "chown -R blog:blog $REMOTE_DIR/dist $REMOTE_DIR/uploads $REMOTE_DIR/blog-server $REMOTE_DIR/blog-initadmin $REMOTE_DIR/migrations || true"

echo "==> [4/4] 重启并探活"
ssh "$SERVER" "systemctl restart blog"
sleep 2
ssh "$SERVER" "curl -fsS http://127.0.0.1:8080/api/site >/dev/null && echo 'deploy OK: /api/site 探活通过'"

echo "部署完成。前端产物与二进制已更新，migrations 已同步。"
echo "⚠️  若本次包含新增 migrations/*.sql：initadmin 同时会重置管理员密码，请人工确认后执行："
echo "      ssh $SERVER 'cd $REMOTE_DIR && sudo -u blog ./blog-initadmin -username admin -password <新8位+密码>'"
