#!/usr/bin/env bash
# 时光集一键安装/启动脚本（CentOS 7，随离线包携带，root 在解包目录执行）
#
# 用法：
#   tar xzf blog-centos7.tar.gz && cd blog && sudo bash setup.sh
#   # 已装过再次执行 = 升级：仅更新程序文件，不覆盖 .env / 不重置数据库
#
# 做什么（全部幂等）：
#   1. 建 blog 系统用户与 /var/www/blog 目录，复制程序文件
#   2. 首次安装生成 .env（随机 JWT_SECRET/LIKE_SALT、随机库密码并建 MySQL 账号）
#   3. 导入 init.sql（建库+建表+初始管理员 admin/admin123456，重复执行无副作用）
#   4. 安装 systemd 服务并启动（enable --now blog）
#   5. 有 nginx 就挂配置（blogctl nginx）；没有则提示后续手装，服务本身不受影响
#   6. 探活 /api/site
#
# 可选环境变量：
#   DB_ROOT_PASS   MySQL root 密码（交互式 mysql 也可，见下）
#   MYSQL_BIN      mysql 客户端路径（默认自动探测 mysql / /usr/local/mysql/bin/mysql）
#   SKIP_NGINX=1   跳过 nginx 步骤
#   SKIP_DB=1      跳过数据库初始化（程序文件已就位、库先前建好时用）

set -euo pipefail

APP_DIR="/var/www/blog"
SRC_DIR="$(cd "$(dirname "$0")" && pwd)"

[ "$(id -u)" = 0 ] || { echo "请用 root 执行：sudo bash $0"; exit 1; }

log()  { echo -e "\033[1;32m==>\033[0m $*"; }
warn() { echo -e "\033[1;33m警告:\033[0m $*" >&2; }

# —— 工具与输入 ---------------------------------------------------------------

# mysql 客户端（EPEL/官方源装在 PATH，源码装常在 /usr/local/mysql/bin）
MYSQL_BIN="${MYSQL_BIN:-}"
if [ -z "$MYSQL_BIN" ]; then
	for c in mysql /usr/local/mysql/bin/mysql; do
		command -v "$c" >/dev/null 2>&1 && { MYSQL_BIN="$c"; break; }
	done
fi

# 需要动数据库且没找到客户端时才报错
need_db=0
[ "${SKIP_DB:-0}" = 1 ] || need_db=1
if [ "$need_db" = 1 ] && [ -z "$MYSQL_BIN" ]; then
	echo "未找到 mysql 客户端（已试 PATH 与 /usr/local/mysql/bin）。" >&2
	echo "安装：yum install -y mysql；或指定路径：MYSQL_BIN=/路径/mysql bash $0；或 SKIP_DB=1 跳过。" >&2
	exit 1
fi

# 跑一条 MySQL 语句：有 DB_ROOT_PASS 走非交互，否则交互输密码
mysql_exec() {
	if [ -n "${DB_ROOT_PASS:-}" ]; then
		"$MYSQL_BIN" -uroot -p"$DB_ROOT_PASS" "$@"
	else
		"$MYSQL_BIN" -uroot -p "$@"
	fi
}

rand() { head -c 48 /dev/urandom | base64 | tr -d '/+=' | cut -c1-"$1"; }

# —— 1. 用户、目录与文件 -------------------------------------------------------

log "安装文件到 $APP_DIR"
id blog >/dev/null 2>&1 || useradd -r -s /sbin/nologin blog

first_install=0
[ -f "$APP_DIR/.env" ] || first_install=1

mkdir -p "$APP_DIR"/{dist,uploads,migrations}
for f in blog-server blog-initadmin blogctl init.sql blog.service nginx-centos7.conf; do
	[ -f "$SRC_DIR/$f" ] && cp -f "$SRC_DIR/$f" "$APP_DIR/$f"
done
[ -f "$SRC_DIR/README-部署-centos7.md" ] && cp -f "$SRC_DIR/README-部署-centos7.md" "$APP_DIR/"
rm -rf "$APP_DIR/dist"/* && cp -R "$SRC_DIR/dist/." "$APP_DIR/dist/"
rm -rf "$APP_DIR/migrations"/* && cp -R "$SRC_DIR/migrations/." "$APP_DIR/migrations/"
chmod +x "$APP_DIR"/blog-server "$APP_DIR"/blog-initadmin "$APP_DIR"/blogctl

# —— 2. .env（仅首次生成；升级绝不覆盖） --------------------------------------

if [ "$first_install" = 1 ]; then
	log "生成 .env（随机密钥）"
	db_pass="$(rand 24)"
	jwt="$(rand 64)"; jwt="${jwt}$(rand 8)"   # 共 72 字符
	like="$(rand 32)"
	sed -e "s|^ADDR=.*|ADDR=127.0.0.1:8080|" \
		-e "s|^APP_ENV=.*|APP_ENV=prod|" \
		-e "s|^DB_PASSWORD=.*|DB_PASSWORD=$db_pass|" \
		-e "s|^JWT_SECRET=.*|JWT_SECRET=$jwt|" \
		-e "s|^LIKE_SALT=.*|LIKE_SALT=$like|" \
		-e "s|^UPLOAD_DIR=.*|UPLOAD_DIR=$APP_DIR/uploads|" \
		-e "s|^# DSN=.*|# DSN=|" \
		"$SRC_DIR/.env.example" > "$APP_DIR/.env"
	# nginx/blogctl 用；WEB_DIST 让 Go 在无 nginx 时也能直出前端（:8080 直接可访问）
	echo "WEB_DIST=$APP_DIR/dist" >> "$APP_DIR/.env"
else
	log "检测到已有 .env，保留不动（升级模式）"
	db_pass="$(awk -F= '$1=="DB_PASSWORD"{print substr($0,length($1)+2)}' "$APP_DIR/.env" | tail -1)"
fi
chmod 600 "$APP_DIR/.env"

# —— 3. 数据库（首次建账号；init.sql 幂等可重复导入） --------------------------

if [ "$need_db" = 1 ]; then
	if [ "$first_install" = 1 ]; then
		log "初始化数据库（导入 init.sql + 创建账号 blog）"
		mysql_exec < "$APP_DIR/init.sql"
		mysql_exec -e "CREATE USER IF NOT EXISTS 'blog'@'127.0.0.1' IDENTIFIED BY '$db_pass';
ALTER USER 'blog'@'127.0.0.1' IDENTIFIED BY '$db_pass';
GRANT ALL PRIVILEGES ON personal_blog.* TO 'blog'@'127.0.0.1'; FLUSH PRIVILEGES;"
		# MySQL 5.6 无 ALTER USER IF EXISTS，重复执行报错可忽略 → 上面 CREATE IF NOT EXISTS 已兜底
	else
		log "校验数据库连通（不改动数据）"
		mysql_exec -e "SELECT 1 FROM personal_blog.admins LIMIT 1" >/dev/null \
			|| { echo "库表校验失败：请确认 MySQL 已启动且先前初始化过；或去掉 SKIP_DB=1 重新导入。" >&2; exit 1; }
	fi
fi

# —— 4. systemd ---------------------------------------------------------------

log "安装并启动 systemd 服务"
# CentOS 7 常见服务名是 mysqld，按实际存在者改单元依赖，避免启动排队卡死
if systemctl list-unit-files 2>/dev/null | grep -q '^mysqld\.service'; then
	sed -i 's/mysql\.service/mysqld.service/g' "$APP_DIR/blog.service"
fi
cp -f "$APP_DIR/blog.service" /etc/systemd/system/blog.service
chown -R blog:blog "$APP_DIR"
systemctl daemon-reload
systemctl enable blog >/dev/null 2>&1 || true
systemctl restart blog

# —— 5. nginx（装了才配，没装不阻塞） ------------------------------------------

if [ "${SKIP_NGINX:-0}" != 1 ]; then
	if command -v nginx >/dev/null 2>&1; then
		log "配置 nginx（按 .env 端口生成并热加载）"
		systemctl enable nginx >/dev/null 2>&1 || true
		(cd "$APP_DIR" && ./blogctl nginx)
	else
		warn "未检测到 nginx：后端已就绪，前端暂由 Go 在 :8080 直出。"
		echo "    后续安装 nginx 后执行：cd $APP_DIR && ./blogctl nginx"
	fi
fi

# —— 6. 探活 -------------------------------------------------------------------

log "探活"
sleep 2
if curl -fsS http://127.0.0.1:8080/api/site >/dev/null; then
	echo "✅ 后端 OK：http://127.0.0.1:8080/api/site"
else
	echo "❌ 探活失败，查看日志：journalctl -u blog -n 50 --no-pager" >&2
	exit 1
fi

front_port="$(awk -F= '$1=="FRONT_PORT"{print substr($0,length($1)+2)}' "$APP_DIR/.env" | tail -1)"
if command -v nginx >/dev/null 2>&1 && [ "${SKIP_NGINX:-0}" != 1 ]; then
	curl -fsS "http://127.0.0.1:${front_port:-80}/api/site" >/dev/null \
		&& echo "✅ nginx OK：http://127.0.0.1:${front_port:-80}/api/site" \
		|| { echo "❌ nginx 探活失败：nginx -t && systemctl status nginx" >&2; exit 1; }
fi

echo
echo "部署完成 🎉  访问 http://服务器IP${front_port:+:$front_port}/ （默认 80 端口即 http://服务器IP/ ）"
echo "后台 http://服务器IP/admin ｜ 初始账号 admin / admin123456（首次登录强制改密）"
echo "密钥与端口都在 $APP_DIR/.env；日常管理：cd $APP_DIR && ./blogctl {start|stop|restart|status|nginx|logs}"
if [ "$first_install" = 1 ]; then
	echo "随机生成的 DB_PASSWORD/JWT_SECRET/LIKE_SALT 已写入 .env，无需手改即可用。"
fi
