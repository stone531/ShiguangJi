# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概览

时光集（ShiguangJi）：个人博客，单 Vue SPA（前台 + `/admin` 后台）+ Go API + MySQL 5.6.5+（排序规则用 utf8mb4_unicode_ci，勿用 8.0 专用的 utf8mb4_0900_ai_ci；utf8mb4 索引列不超 191 字符），部署形态为 Nginx 同域反代 + systemd 单二进制。当前进度：M1 后端、M2 后台前端、M3 前台展示站、M4 互动与上线（评论点赞/敏感词/SEO 注入/安全加固/部署文件）全部完成。

仓库内注释与文档均使用中文，新代码保持一致。总设计与各里程碑计划在 `docs/superpowers/`（specs/ 为已评审的总设计，plans/ 为里程碑实施计划）。生产部署文件在 `deploy/`（nginx.conf/blog.service/deploy.sh/backup.sh；CentOS 7 目标机用 `bash deploy/package.sh` 产出离线包 build/blog-centos7.tar.gz，nginx 用 el7 适配版 nginx-centos7.conf——旧 http2 语法、无 TLS1.3，服务器步骤见 deploy/README-部署-centos7.md；端口统一在 .env 配 ADDR(后端)/FRONT_PORT(前端)，服务器端控制脚本 deploy/blogctl 按其生成 nginx 配置；均只交付未实测）。

## 常用命令

### 后端（`cd backend/`，Go module 名 `shiguangji`）

```bash
make dev                                    # 启动 API（go run ./cmd/server，监听 :8080）
make test                                   # 全量测试（go test ./... -race -count=1）
go test ./internal/service -run '^TestCreatePostAutoSummary$' -race -count=1   # 单个测试
make lint                                   # go vet ./...
make fmt                                    # gofmt
```

- 测试用 sqlite 内存库（glebarez/sqlite + AutoMigrate），不需要本地 MySQL；只有 `make dev` 需要。
- 首次启动：建库建表推荐直接导入 `deploy/init.sql`（建库+8 表+初始数据+初始管理员 admin/admin123456 首登强制改密，幂等），或 `go run ./cmd/initadmin -username admin -password <8+位>`（幂等：应用 `migrations/*.sql` + 创建/重置管理员，server 启动不做迁移；init.sql 内容须与 migrations/ 同步维护）→ `cp .env.example .env`（`JWT_SECRET` 必填，缺失时进程启动即 log.Fatal）。

### 前端（`cd frontend/`）

```bash
npm run dev                                 # :5173，/api 与 /uploads 已代理到 :8080（需先启动后端）
npm run build                               # vue-tsc -b && vite build
npx vue-tsc --noEmit -p tsconfig.app.json   # 仅类型检查（strict）
```

无测试框架与 lint 配置；类型检查即主要验证手段。

## 架构

### 后端三层：router → handler → service → model

- 统一响应 `{code, msg, data}`，code 0 为成功；一律经 `internal/handler/response.go` 的 `OK`/`Fail`。业务错误用 `internal/errs` 的 `BizError` 携带业务码（40001 参数 / 40004 不存在 / 20001、20002 token / 42901 限流），`Fail` 映射：42901→HTTP 429，其余业务码→400，未知错误→500/50000。
- 认证：JWT 存 HttpOnly cookie `blog_token`（Path=/api）。`middleware/jwt.go` 做**滑动续期**——剩余寿命不足一半时重签写回，避免长文写作掉线；登录接口按 IP 限流 5 次/分钟。
- 路由分组（`internal/router/router.go`）：`/api/admin/login` 公开；`/api/admin/*` 受 JWT 保护，路径参数用 `:id`；`/api/posts`（列表支持 type/category/tag/keyword/featured 筛选，返回轻量 PostCard VO）、`/api/posts/:slug`（响应为 `{post, prev, next}`）、`/api/site`、`/api/categories`、`/api/tags`、`/api/archive` 前台公开，用 `:slug`——两组分开正是为了避开 Gin 路由冲突。handler 测试复用 `RegisterAuthRoutes`（不挂 RequestID/Recover）。
- 上传：魔数白名单 + 内容嗅探校验、MD5 命名、同步生成缩略图（disintegration/imaging），5MB 上限，落盘 `uploads/`。开发模式由 Go 静态服务（目录列举已遮蔽为 404），生产由 Nginx 直出。
- 配置：`internal/config` 进程单例，从 `.env`/环境变量加载；数据库优先用 `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME` 离散变量（内部组装 DSN），未设 `DB_USER` 时回落整串 `DSN`。

### 前端

- 单 SPA 双路由区：`/`（前台 SiteLayout，暗色科技风）与 `/admin/*`（路由守卫查 Pinia admin store），全部懒加载独立 chunk。路径别名 `@` → `src/`。样式：Tailwind v4 + `src/styles/theme.css`（token/aurora/噪点/聚光/`.md-body` 排版）。
- Markdown 渲染管线（`components/site/MarkdownView.vue`）：markdown-it（`html:false`）→ DOMPurify → v-html → Shiki 异步高亮。改渲染逻辑时保持该顺序，勿绕过净化。
- `src/api/http.ts` axios 单例：`code≠0` 统一 reject；20001/20002 或 HTTP 401 → 清空 admin store 并跳 `/admin/login`；默认头带 `X-Requested-With: XMLHttpRequest`（后端对除登录外的全部 `/api` 写操作强制校验该头，CSRF 纵深防御，前端绕过 `http.ts` 的自建请求需自行携带）。新接口一律走该文件的 `get/post/put/del` 封装。
- 评论（先审后显，蜜罐字段 `website` 假成功）与点赞（IP+UA+LIKE_SALT 指纹）为公开写接口；敏感词 trie 现查现建（`service/sensitive.go`，词表存 `site_settings.sensitive_words`，仅 `/api/admin/site` 可读写该键，公开 `/api/site` 永不含它）。SEO meta 注入与 SPA fallback 在 `internal/router/spa.go`（启动缓存 `dist/index.html`）。
