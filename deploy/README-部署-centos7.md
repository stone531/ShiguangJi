# 时光集 · CentOS 7 部署指南（离线包）

本包面向 **CentOS 7（x86_64）** 服务器，包含前后端全部产物。按顺序复制粘贴执行即可。

```
包内容：
├── blog-server           # Go 后端（linux/amd64 静态编译，无系统依赖）
├── blog-initadmin        # 初始化工具（建表+管理员，可选用 init.sql 代替）
├── setup.sh              # ⭐ 一键安装/启动脚本（root 执行，幂等）
├── dist/                 # 前端构建产物（nginx 托管）
├── migrations/           # SQL 迁移（initadmin 用）
├── init.sql              # 一键初始化 SQL（建库+建表+初始管理员 admin/admin123456）
├── blog.service          # systemd 服务单元
├── blogctl               # 服务端控制脚本（start/stop/restart/status/nginx/logs）
├── nginx-centos7.conf    # nginx 配置（el7 适配版，默认 HTTP 直跑）
├── .env.example          # 环境变量模板
└── README-部署-centos7.md
```

前提：服务器已装 MySQL 5.6+ 并在本机（127.0.0.1:3306）运行；需要 root 权限。

---

## 0. 一键安装（推荐）

`setup.sh` 会完成下面第 1–7 步的全部内容：建用户/目录、生成 `.env`（自动随机
JWT_SECRET/LIKE_SALT/库密码并创建 MySQL 账号）、导入 init.sql、装 systemd 并启动、
有 nginx 就挂配置、探活。重复执行 = 升级（只更新程序文件，不覆盖 `.env`、不动数据库）。

```bash
# 本机执行：scp build/blog-centos7.tar.gz root@服务器IP:/tmp/
tar xzf /tmp/blog-centos7.tar.gz -C /tmp/
cd /tmp/blog
sudo bash setup.sh                 # 交互输入一次 MySQL root 密码
# 或非交互：sudo DB_ROOT_PASS='root密码' bash setup.sh
```

高级用法：`SKIP_DB=1` 跳过数据库步骤、`SKIP_NGINX=1` 跳过 nginx、
`MYSQL_BIN=/路径/mysql` 指定客户端。装完即用：

- 博客 `http://服务器IP/`，后台 `http://服务器IP/admin`（admin / admin123456，首登强制改密）
- 未装 nginx 也不影响访问：Go 会在 `:8080` 直出前端（`http://服务器IP:8080/`）

以下手工步骤仅当需要精细控制、或排查 setup.sh 问题时参考。

---

## 1. 上传解包、建用户与目录（服务器执行）

```bash
# 本机执行：scp build/blog-centos7.tar.gz root@服务器IP:/tmp/
tar xzf /tmp/blog-centos7.tar.gz -C /tmp/

useradd -r -s /sbin/nologin blog 2>/dev/null || true
mkdir -p /var/www/blog/{dist,uploads}
cp /tmp/blog/{blog-server,blog-initadmin,blogctl,.env.example,init.sql,blog.service,nginx-centos7.conf} /var/www/blog/
cp -r /tmp/blog/dist/* /var/www/blog/dist/
cp -r /tmp/blog/migrations /var/www/blog/migrations
chown -R blog:blog /var/www/blog
chmod +x /var/www/blog/blog-server /var/www/blog/blog-initadmin /var/www/blog/blogctl
```

## 2. 配置环境变量

```bash
cd /var/www/blog
cp .env.example .env && chmod 600 .env
vi .env    # 按下面说明改
```

`.env` 关键项（MySQL 与本程序同机）：

```ini
ADDR=127.0.0.1:8080
APP_ENV=prod
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=blog
DB_PASSWORD=数据库账号密码
DB_NAME=personal_blog
JWT_SECRET=换成随机64字符        # 生成：head -c 64 /dev/urandom | base64
LIKE_SALT=换一段随机串
UPLOAD_DIR=/var/www/blog/uploads
WEB_DIST=/var/www/blog/dist
FRONT_PORT=80             # 前端(nginx)对外端口，blogctl 会读
```

## 3. 初始化数据库（已初始化过可跳过，全部幂等）

```bash
mysql -uroot -p < /var/www/blog/init.sql
mysql -uroot -p -e "CREATE USER 'blog'@'127.0.0.1' IDENTIFIED BY '与.env里DB_PASSWORD一致';
GRANT ALL PRIVILEGES ON personal_blog.* TO 'blog'@'127.0.0.1'; FLUSH PRIVILEGES;"
```

## 4. systemd 服务

```bash
# 若 MySQL 服务名是 mysqld 而非 mysql，先改依赖名：
# sed -i 's/mysql.service/mysqld.service/' /var/www/blog/blog.service
cp /var/www/blog/blog.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now blog
systemctl status blog          # 应看到 "mysql connected" 日志：journalctl -u blog -n 20
```

## 5. nginx

**未安装 nginx 的话**（CentOS 7 已 EOL，官方源失效，推荐用 nginx.org 官方源，一步到位且版本新）：

```bash
cat > /etc/yum.repos.d/nginx.repo <<'EOF'
[nginx-stable]
name=nginx stable repo
baseurl=http://nginx.org/packages/centos/7/$basearch/
gpgcheck=1
enabled=1
gpgkey=https://nginx.org/keys/nginx_signing.key
module_hotfixes=true
EOF
yum install -y nginx
```

（若 yum 连 nginx.repo 都拉不动，先把系统源切到阿里云归档镜像：
`sed -e 's|^mirrorlist=|#mirrorlist=|g' -e 's|^#baseurl=http://mirror.centos.org/centos/\$releasever|baseurl=https://mirrors.aliyun.com/centos-vault/7.6.1810|g' -i.bak /etc/yum.repos.d/CentOS-*.repo && yum clean all && yum makecache`）

**挂配置**（默认 HTTP 直跑，先用 IP 就能访问）：

```bash
systemctl enable --now nginx
cd /var/www/blog && ./blogctl nginx   # 按 .env 的 FRONT_PORT/ADDR 生成 /etc/nginx/conf.d/blog.conf 并热加载
```

有域名：生成后把 `/etc/nginx/conf.d/blog.conf` 里的 `server_name _` 改成域名再 `systemctl reload nginx`
（也可手动 `cp nginx-centos7.conf /etc/nginx/conf.d/blog.conf`，但端口要自己保持与 .env 一致）。

## 6. 防火墙放行

```bash
firewall-cmd --permanent --add-service=http --add-service=https
firewall-cmd --reload
# 阿里云还需在控制台安全组放行 80/443
```

## 7. 验证

```bash
curl -fsS http://127.0.0.1:8080/api/site   # Go 直连探活
curl -fsS http://127.0.0.1/api/site        # 经 nginx 探活
```

浏览器访问 `http://服务器IP/` 看到博客首页；`http://服务器IP/admin` 用
**admin / admin123456** 登录（首次登录强制改密，请立即改成强密码）。

## 8. 改端口（可选）

端口统一在 `.env` 配，改完执行对应命令：

| 改什么 | 改哪 | 生效命令 |
|---|---|---|
| 后端端口 | `.env` 的 `ADDR`（如 `127.0.0.1:9090`） | `./blogctl restart` |
| 前端(nginx)端口 | `.env` 的 `FRONT_PORT`（如 `8081`） | `./blogctl nginx` |

`./blogctl start|stop|restart|status|logs` 等同于 systemctl/journalctl 对 blog 服务的包装。

## 9. 可选：HTTPS

有域名后用 acme.sh 免费签证书（纯 shell，不挑系统）：

```bash
curl https://get.acme.sh | sh -s email=my@example.com
~/.acme.sh/acme.sh --issue -d your-domain.com -w /var/www/certbot
mkdir -p /etc/nginx/certs
~/.acme.sh/acme.sh --install-cert -d your-domain.com \
  --key-file /etc/nginx/certs/your-domain.com.key \
  --fullchain-file /etc/nginx/certs/fullchain.cer \
  --reloadcmd "systemctl reload nginx"
```

然后按 `nginx-centos7.conf` 尾部注释启用 443 块。

## 10. 日常更新

开发机重新打包并上传，重跑 setup.sh 即可（升级模式：不覆盖 `.env`、不动数据库）：

```bash
# 开发机：bash deploy/package.sh && scp build/blog-centos7.tar.gz root@服务器:/tmp/
# 服务器：
tar xzf /tmp/blog-centos7.tar.gz -C /tmp/ && cd /tmp/blog && sudo bash setup.sh
```

等价的手工方式（重复第 1 步的 cp 部分后 `systemctl restart blog`）。

（改了数据库结构才需要重导 init.sql 或跑 `sudo -u blog ./blog-initadmin`，注意后者会重置管理员密码。）

## 11. 可选：每日备份

把仓库里的 `deploy/backup.sh` 传到服务器，加 cron（每天 3:17）：

```
17 3 * * * MYSQL_PASSWORD=密码 /var/www/blog/backup.sh >> /var/log/blog-backup.log 2>&1
```

---

**风险提示**：CentOS 7（2024-06 EOL）与 MySQL 5.6（2021-02 EOL）均已停止安全维护，
个人博客短期可用，建议后续规划系统与数据库升级（如 Anolis/OpenCloud + MySQL 8）。
