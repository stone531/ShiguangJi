# 时光集 ShiguangJi

个人生活与技术分享博客网站。文章 / 图片集 / 视频（第三方嵌入）三种内容形态，前台匿名评论与点赞，后台管理员单账号发布内容。

## 技术栈

- **前端**：Vue 3 + Vite + TypeScript + Naive UI（暗色科技风）
- **后端**：Go (Gin) + GORM + MySQL 8
- **部署**：Nginx 同域反代 + systemd 单二进制（单台云服务器）

## 目录结构

```
├── backend/    # Go 后端（三层架构：router → handler → service → model）
├── frontend/   # Vue 3 单 SPA（前台展示站 + /admin 后台）
├── deploy/     # 生产部署文件（nginx.conf / blog.service / deploy.sh / backup.sh）
└── docs/       # 设计文档（spec）与实施计划
```

## 快速开始

### 后端（需 MySQL 8.0+）

```bash
cd backend
cp .env.example .env      # 修改 DSN 密码与 JWT_SECRET
go run ./cmd/initadmin -username admin -password yourpass123   # 建表+初始化管理员
make dev                  # 监听 :8080
```

### 前端

```bash
cd frontend
npm install
npm run dev               # http://localhost:5173 → /admin/login
```

## 文档

- 设计文档：`docs/superpowers/specs/2026-08-21-personal-blog-design.md`
- M1 后端计划：`docs/superpowers/plans/2026-08-21-personal-blog-m1-backend.md`
- M2 后台前端计划：`docs/superpowers/plans/2026-08-21-personal-blog-m2-admin-frontend.md`
- M3 前台展示站计划：`docs/superpowers/plans/2026-08-22-personal-blog-m3-frontend.md`
- M4 互动与上线计划：`docs/superpowers/plans/2026-08-22-personal-blog-m4-interaction.md`

## 部署

生产部署文件在 `deploy/`（用法见各文件头部注释）：`nginx.conf`（同域反代 + SPA + uploads 禁执行）、`blog.service`（systemd）、`deploy.sh`（本机构建 → scp → 重启）、`backup.sh`（mysqldump 滚动 7 天 + uploads 增量链，配 cron）。首次部署见 `deploy/deploy.sh` 头部说明。

## 里程碑

- [x] M1 后端骨架（登录/JWT/文章 CRUD/上传缩略图）
- [x] M2 后台前端（编辑器/自动保存/评论与设置占位）
- [x] M3 前台展示站（Bento 首页/文章渲染/归档搜索/赞赏 + 公开接口 site/categories/tags/archive）
- [x] M4 互动与上线（评论点赞接口/敏感词过滤/后台接口补全/SEO meta 注入/安全加固/部署文件）
