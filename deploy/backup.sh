#!/usr/bin/env bash
# 时光集备份（spec §10）：mysqldump 全量滚动 7 天 + uploads 增量打包滚动 7 份
#
# 用法（目标机 cron，每天凌晨 3:17）：
#   crontab -e
#   17 3 * * * /var/www/blog/deploy/backup.sh >> /var/log/blog-backup.log 2>&1
#
# 数据库账号通过环境变量或本文件默认值指定（与 .env 中 DSN 的账号保持一致）：
#   MYSQL_USER（默认 blog）、MYSQL_PASSWORD（必填）、MYSQL_DB（默认 personal_blog）
#
# 建议异地容灾：rsync 备份目录到另一台机器或对象存储（本脚本不管，防误删主备份）。

set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/var/www/blog}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/blog}"
MYSQL_USER="${MYSQL_USER:-blog}"
MYSQL_DB="${MYSQL_DB:-personal_blog}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:?请通过环境变量提供 MYSQL_PASSWORD}"
KEEP_DAYS=7

STAMP="$(date +%Y%m%d-%H%M%S)"
MYSQL_DIR="$BACKUP_DIR/mysql"
UPLOADS_DIR="$BACKUP_DIR/uploads"
mkdir -p "$MYSQL_DIR" "$UPLOADS_DIR"

echo "[$(date '+%F %T')] backup start"

# ---------- 数据库：每日全量，滚动 7 天 ----------
DUMP_FILE="$MYSQL_DIR/$MYSQL_DB-$STAMP.sql.gz"
mysqldump --single-transaction --routines --triggers \
  -h 127.0.0.1 -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DB" | gzip > "$DUMP_FILE"
echo "mysql dumped: $DUMP_FILE ($(du -h "$DUMP_FILE" | cut -f1))"
find "$MYSQL_DIR" -name '*.sql.gz' -mtime +$KEEP_DAYS -delete

# ---------- uploads：GNU tar listed-incremental 每日增量链，链满 7 份整体重置 ----------
# 无快照时首包为全量（level-0），此后每天只打包自上次以来新增/修改的文件；
# 增量包依赖链头全量包，因此满 7 份时删除整条链 + 快照，次日重新开全量。
# 恢复（按文件名时间顺序，全量在最前）：
#   for f in uploads-*.tar.gz; do tar -xzf $f --listed-incremental=/dev/null -C /var/www/blog; done
SNAPSHOT="$UPLOADS_DIR/snapshot.snar"
if (( $(ls -1 "$UPLOADS_DIR"/uploads-*.tar.gz 2>/dev/null | wc -l) >= 7 )); then
  rm -f "$UPLOADS_DIR"/uploads-*.tar.gz "$SNAPSHOT"
fi
TAR_FILE="$UPLOADS_DIR/uploads-$STAMP.tar.gz"
tar --create --gzip --listed-incremental="$SNAPSHOT" --file="$TAR_FILE" -C "$REMOTE_DIR" uploads
echo "uploads packed: $TAR_FILE ($(du -h "$TAR_FILE" | cut -f1))"

echo "[$(date '+%F %T')] backup done"
