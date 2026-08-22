# personal-blog backend (M1)

Go 1.22+ / Gin / GORM / MySQL 8。M1 范围：登录（JWT 滑动过期 + HttpOnly Cookie）、文章 CRUD、图片上传（魔数校验 + MD5 命名 + 缩略图）。

> **注意：首次运行需要本机可用的 MySQL 8.0+ 服务**（本仓库的自动化测试用 sqlite 内存库，不需要 MySQL）。

## 本地启动

```bash
# 1. 建库（MySQL 8.0+）
mysql -uroot -p -e "CREATE DATABASE IF NOT EXISTS personal_blog CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"

# 2. 配置环境
cp .env.example .env
# 编辑 .env：修改 DSN 中的数据库密码、JWT_SECRET（随机 64 字符）、LIKE_SALT

# 3. 初始化（应用 migrations/*.sql + 创建/重置管理员，幂等）
go run ./cmd/initadmin -username admin -password your-password-123

# 4. 启动服务
make dev
```

## curl 冒烟

```bash
# 登录（成功后 Set-Cookie: blog_token=...; HttpOnly）
curl -i -c cookies.txt -X POST localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password-123"}'

# 无 cookie 访问受保护接口 → 401 {"code":20001,...}
curl -i localhost:8080/api/admin/stats

# 上传图片（需登录 cookie；受保护组写操作要求 XHR 头防 CSRF）
curl -i -b cookies.txt -X POST localhost:8080/api/admin/upload \
  -H "X-Requested-With: XMLHttpRequest" \
  -F "file=@./test.png"

# 发表评论（先审后显；蜜罐 website 字段有值则静默假成功）
curl -i -X POST localhost:8080/api/posts/hello-world/comments \
  -H "Content-Type: application/json" -H "X-Requested-With: XMLHttpRequest" \
  -d '{"nickname":"访客","content":"写得不错"}'

# 点赞切换（指纹 = SHA256(IP+UA+LIKE_SALT)）
curl -i -X POST localhost:8080/api/posts/hello-world/like \
  -H "X-Requested-With: XMLHttpRequest"
```

除登录外的全部 `/api` 写操作（含公开评论/点赞 POST）都要求 `X-Requested-With: XMLHttpRequest` 头，缺失返回 400。

## make 命令

| 命令 | 说明 |
|---|---|
| `make dev` | 启动开发服务器（`go run ./cmd/server`） |
| `make test` | 全量测试（`go test ./... -race -count=1`） |
| `make lint` | `go vet ./...` |
| `make fmt` | gofmt 格式化 |

## 目录结构

```
backend/
├── cmd/
│   ├── server/main.go        # API 服务入口（优雅关停）
│   └── initadmin/main.go     # 迁移 + 管理员初始化脚本
├── internal/
│   ├── config/               # .env 配置加载（进程单例）
│   ├── dao/                  # gorm.Open + 连接池
│   ├── errs/                 # 业务错误码
│   ├── handler/              # HTTP 层（auth/post/comment/like/site/taxonomy + 统一响应）
│   ├── middleware/           # JWT 滑动过期 / IP 限流 / XHR 校验 / 安全头 / recovery
│   ├── model/                # GORM 模型
│   ├── pkg/jwtutil/          # JWT 签发/解析
│   ├── router/               # 路由装配 + 静态文件 + SPA/SEO 注入（spa.go）
│   └── service/              # 业务逻辑（auth/post/comment/like/site/taxonomy/stats/upload）
├── migrations/               # 建表与敏感词初始 SQL（幂等，initadmin 应用）
├── uploads/                  # 上传文件目录
├── Makefile
└── .env.example
```
