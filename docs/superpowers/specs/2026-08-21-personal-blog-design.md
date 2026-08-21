# 个人生活/技术分享博客网站 — 需求设计文档

- **日期**：2026-08-21
- **状态**：已评审通过（设计四节均经用户逐节确认）
- **项目代号**：personal-blog
- **参考站点**：https://aiislove.tech/（视觉气质参考，功能为博客而非产品矩阵）

---

## 1. 项目背景与目标

构建一个**个人内容展示网站**，发布生活与技术分享内容，形态包括：文章（技术博客）、图片集、视频（嵌入第三方平台）。网站分两部分：

- **前台**：面向所有访客，浏览、评论、点赞，无需注册
- **后台**：管理员单账号登录，发布与管理内容

### 1.1 核心需求（用户已确认的决策）

| 决策点 | 结论 |
|---|---|
| 技术栈 | 前端 Vue 3 + 后端 Golang + MySQL |
| 视频方案 | 嵌入第三方平台（B站/YouTube），不自托管视频文件 |
| 访客互动 | 匿名评论 + 点赞，无用户注册系统 |
| 部署 | 单台云服务器（Linux），Nginx + Go 二进制 + MySQL |
| 编辑器 | Markdown 编辑器（后台写作） |
| 内容组织 | 分类 + 标签 + 推荐位/焦点图 + 时间线归档页 |
| 视觉风格 | 暗色科技风（参考 aiislove.tech） |
| 图片存储 | 服务器本地磁盘，Nginx 静态直出 |
| 前后端组织 | 前后端分离 + Nginx 同域反代（无跨域），**方案 A：单 Vue SPA 含前台与后台** |
| SEO 策略 | SPA + 预渲染首页/关于页 + **Go 服务端 meta 注入**（见 §8.1）；文章详情靠搜索引擎对 SPA 的动态渲染收录，接受中等 SEO 效果 |
| 图片管线 | 上传时服务端生成 WebP 缩略图（封面 ~800px 宽），列表页用缩略图、点击看原图 |
| 打赏捐赠 | 后台上传微信/支付宝收款码，前台文章页与关于页展示「爱心捐赠」入口（弹窗展示二维码） |

### 1.2 不做的事（YAGNI）

- 不做用户注册/登录系统（访客匿名互动）
- 不做多管理员/角色权限（单账号）
- 不做自托管视频、转码、进度记忆
- 不做主题系统/插件机制（区别于 Halo 这类 CMS）
- 不做 RSS 之外的推送类功能（RSS 可作为后续增强）
- 首版不做 Redis 缓存/CDN（流量余量充足，按需再加）

---

## 2. 技术选型（含调研依据）

> 调研时间 2026-08-21，数据来源：npm registry API、GitHub releases 页面实时抓取；部分社区趋势结论基于训练知识（子代理联网受限），已标注。

| 层 | 选型 | 理由 |
|---|---|---|
| 前端框架 | Vue 3 + Vite + TypeScript + Vue Router + Pinia | 用户指定 Vue；生态标准组合 |
| 前端组织 | **方案 A：单 SPA**，`/admin` 路由组懒加载独立 chunk | 一个工程、一次构建、一份部署；SEO 用预渲染弥补 |
| UI 组件库 | **Naive UI** | Vue3 原生、暗色主题为一等公民（JS 主题对象精细调色）、TS 友好；npm 周下载 ~16.7 万 |
| 样式 | Tailwind CSS | 玻璃拟态/渐变/网格纹理等暗色科技风自定义自由度最高 |
| 动效 | @vueuse/motion（入场/hover）+ GSAP 按需（首屏 Hero） | GSAP 2025-04 起全插件免费（Webflow 收购）；Three.js 明确不用 |
| Markdown 编辑 | **md-editor-v3**（v6.x） | Vue3 原生、维护最积极（调研时 2 天前发版）、内置暗色主题/图片粘贴上传/中文；Vditor 7 个月未更新被排除 |
| Markdown 渲染 | markdown-it（`html:false`）+ Shiki + DOMPurify | markdown-it 默认安全 + 插件生态最全（周下载 2350 万）；Shiki 用 VS Code 同源语法（VS Code 暗色主题零成本）；DOMPurify 为 XSS 净化事实标准（周下载 5100 万） |
| 图片处理 | `disintegration/imaging`（Go） | 上传时生成 WebP 缩略图，控制列表页流量（避免 5MB 原图打爆单机带宽） |
| 后端框架 | **Gin** | 生态/中间件/中文资料最全，性能对博客绰绰有余 |
| ORM | **GORM**（v1.31.x） | 关联模型（文章↔标签↔评论）舒适区，AutoMigrate 起步；sqlc 备选未采用（开发速度优先） |
| 认证 | golang-jwt/v5 + bcrypt | 成熟组合，参考 go-gin-example 的 JWT 错误码设计 |
| 数据库 | MySQL 8 | 用户指定 |
| Web 服务器 | Nginx（静态 + 反代 + 安全头） | 单机部署标准做法 |

**可借鉴开源项目**：
- [go-gin-example](https://github.com/eddycjy/go-gin-example)（已实时验证）：三层目录结构（routers→service→models）、JWT 错误码细分、图片上传白名单 + MD5 命名、软删除 Callback、统一响应格式
- [Halo](https://github.com/halo-dev/halo)（部分实时验证）：后台交互与内容管理功能设计标杆（不借鉴其插件/主题架构）

---

## 3. 总体架构

```
┌─────────────────────────────────────────────────┐
│                   用户浏览器                      │
│   前台访客（浏览/评论/点赞）  管理员（/admin）      │
└──────────────────┬──────────────────────────────┘
                   │ HTTPS
┌──────────────────▼──────────────────────────────┐
│              Nginx（80/443，单台云服务器）          │
│  ├─ /            → Vue SPA 静态文件（dist/）      │
│  ├─ /uploads/    → 本地图片静态服务（直出）         │
│  └─ /api/*       → 反向代理 → Go 服务 :8080        │
└──────────────────┬──────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────┐
│         Go 应用（Gin，systemd 守护，单二进制）      │
│  middleware: JWT/限流/日志/recover                │
│  router → handler → service → model 三层          │
└──────────────────┬──────────────────────────────┘
                   │ GORM
┌──────────────────▼──────────────────────────────┐
│              MySQL 8（本机 3306，仅监听 127.0.0.1）│
└─────────────────────────────────────────────────┘
```

**关键决策说明**：
- 单 SPA 内两个路由区：`/`（前台，无需登录）与 `/admin/*`（后台，路由守卫 + 懒加载 chunk），构建产物一份
- Admin 代码位于同一前端工程 `src/views/admin/`，Vite 自动拆独立 chunk，访客不加载；即使手动获取 admin chunk 也仅含 UI 代码，数据安全由后端 JWT 中间件保障
- 图片直出：Nginx 直接服务 `/uploads/`，上传走 `/api/admin/upload`
- 视频零成本：数据库仅存第三方嵌入 URL（`video_url` 字段）

## 3.1 仓库结构（monorepo）

```
personal-blog/
├── frontend/                  # Vue 3 + Vite + TS
│   ├── src/
│   │   ├── views/
│   │   │   ├── site/          # 前台：Home/ArticleDetail/Archive/About/Search
│   │   │   └── admin/         # 后台：Dashboard/PostEditor/PostList/CommentManage/
│   │   │                     #       CategoryTag/SiteSettings/Profile
│   │   ├── components/        # ArticleCard/MarkdownView/Lightbox/TOC...
│   │   ├── api/               # axios 封装 + 接口模块
│   │   ├── stores/            # Pinia（站点配置/管理员状态）
│   │   ├── router/            # / 与 /admin 分组，懒加载 + 守卫
│   │   ├── layouts/           # SiteLayout / AdminLayout
│   │   └── styles/            # Tailwind + 暗色主题定制
│   └── vite.config.ts
├── backend/                   # Go + Gin + GORM
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── router/
│   │   ├── handler/
│   │   ├── service/
│   │   ├── model/
│   │   ├── middleware/        # JWT/限流/日志/recover
│   │   └── config/
│   ├── uploads/               # 本地上传目录（gitignore）
│   └── go.mod
├── deploy/
│   ├── nginx.conf
│   └── blog.service           # systemd 单元
└── docs/                      # 本文档
```

---

## 4. 数据库表字典（8 张表）

> 通用约定：引擎 InnoDB，字符集 `utf8mb4`，排序规则 `utf8mb4_0900_ai_ci`；所有表主键 `id BIGINT UNSIGNED AUTO_INCREMENT`；时间字段 `DATETIME`，存本地时区（Asia/Shanghai）；GORM 模型统一嵌入 `CreatedAt/UpdatedAt`，软删除表额外嵌入 `DeletedAt`（GORM 软删除索引自动创建）。

### 4.1 admins — 管理员（单账号预置）

| 字段 | 类型 | 空 | 默认 | 说明 |
|---|---|---|---|---|
| id | BIGINT UNSIGNED PK AI | 否 | — | 主键 |
| username | VARCHAR(50) | 否 | — | 登录名，UNIQUE |
| password_hash | VARCHAR(100) | 否 | — | bcrypt(cost=10) |
| nickname | VARCHAR(50) | 否 | '' | 前台展示名（评论回复的博主名） |
| avatar | VARCHAR(255) | 是 | NULL | 头像 URL（/uploads/xxx） |
| created_at | DATETIME | 否 | CURRENT_TIMESTAMP | |
| updated_at | DATETIME | 否 | CURRENT_TIMESTAMP ON UPDATE | |

索引：`UNIQUE(username)`
初始化：部署时通过初始化脚本写入一条管理员（首次登录强制改密码，M1 实现）。

### 4.2 posts — 内容表（文章/图片集/视频统一）

| 字段 | 类型 | 空 | 默认 | 说明 |
|---|---|---|---|---|
| id | BIGINT UNSIGNED PK AI | 否 | — | 主键 |
| title | VARCHAR(200) | 否 | — | 标题 |
| slug | VARCHAR(200) | 否 | — | URL 标识（`/post/:slug`），规则 `^[a-z0-9-]+$`，UNIQUE（含软删除行也占用，防撞车） |
| summary | VARCHAR(500) | 是 | NULL | 摘要；空则服务端截取正文前 120 字符（Markdown 语法剔除后） |
| cover | VARCHAR(255) | 是 | NULL | 封面图 URL（缩略图地址） |
| content | LONGTEXT | 否 | — | Markdown 原文 |
| content_html | LONGTEXT | 是 | NULL | 预留渲染缓存（渲染在前端，暂不使用） |
| type | TINYINT UNSIGNED | 否 | 1 | 1=文章 2=图片集 3=视频 |
| video_url | VARCHAR(500) | 是 | NULL | 第三方嵌入 iframe src（仅 type=3 必填，服务端校验域名白名单 player.bilibili.com / www.youtube.com / player.youku.com） |
| category_id | BIGINT UNSIGNED | 是 | NULL | FK → categories.id，ON DELETE SET NULL |
| status | TINYINT UNSIGNED | 否 | 0 | 0=草稿 1=已发布 2=回收站 |
| is_top | TINYINT(1) | 否 | 0 | 置顶（列表排序权重） |
| is_featured | TINYINT(1) | 否 | 0 | 首页 Bento 推荐位 |
| views | BIGINT UNSIGNED | 否 | 0 | 浏览量（详情接口原子 +1） |
| like_count | BIGINT UNSIGNED | 否 | 0 | 冗余点赞数（事务内原子 ±1） |
| comment_count | BIGINT UNSIGNED | 否 | 0 | 冗余评论数（仅 status=1 的评论计数，审核通过/删除时同步） |
| published_at | DATETIME | 是 | NULL | 发布时间（首次置为 1 时写入，支持手动回填） |
| created_at / updated_at / deleted_at | DATETIME | 否/否/是 | — | GORM 软删除三件套 |

索引：
- `UNIQUE(slug)`
- `idx_status_published` (status, published_at) — 前台列表主查询：`WHERE status=1 AND deleted_at IS NULL ORDER BY published_at DESC`
- `idx_category_status` (category_id, status)
- `idx_is_featured` (is_featured) — 首页推荐位

枚举字典：

| 字段 | 值 | 含义 |
|---|---|---|
| type | 1 / 2 / 3 | 文章 / 图片集 / 视频 |
| status | 0 / 1 / 2 | 草稿 / 已发布 / 回收站 |

### 4.3 categories — 分类

| 字段 | 类型 | 空 | 默认 | 说明 |
|---|---|---|---|---|
| id | BIGINT UNSIGNED PK AI | 否 | — | |
| name | VARCHAR(50) | 否 | — | 分类名，UNIQUE（如"技术/生活/随笔"） |
| sort | INT | 否 | 0 | 排序（后台拖拽，小值在前） |
| created_at / updated_at | DATETIME | 否 | — | |

索引：`UNIQUE(name)`
约束：删除分类时，若其下存在文章（含回收站），拒绝删除并提示（service 层校验）。

### 4.4 tags — 标签

| 字段 | 类型 | 空 | 默认 | 说明 |
|---|---|---|---|---|
| id | BIGINT UNSIGNED PK AI | 否 | — | |
| name | VARCHAR(50) | 否 | — | 标签名，UNIQUE |
| created_at | updated_at | DATETIME | 否 | — | |

索引：`UNIQUE(name)`

### 4.5 post_tags — 文章标签关联（多对多）

| 字段 | 类型 | 空 | 说明 |
|---|---|---|---|
| post_id | BIGINT UNSIGNED | 否 | FK → posts.id，ON DELETE CASCADE |
| tag_id | BIGINT UNSIGNED | 否 | FK → tags.id，ON DELETE CASCADE |
| PRIMARY KEY (post_id, tag_id) | | | 联合主键，天然去重 |

索引：`idx_tag` (tag_id) — 反查"标签下的文章"
行为：文章保存时全量替换关联（事务内 delete+insert）；标签删除时级联清理；标签重命名为 UPDATE 不影响关联。

### 4.6 comments — 评论（匿名 + 博主回复，两级结构）

| 字段 | 类型 | 空 | 默认 | 说明 |
|---|---|---|---|---|
| id | BIGINT UNSIGNED PK AI | 否 | — | |
| post_id | BIGINT UNSIGNED | 否 | — | FK → posts.id，ON DELETE CASCADE |
| parent_id | BIGINT UNSIGNED | 否 | 0 | 0=顶级评论；否则指向**同文章的顶级评论 id**（service 层校验，禁止二级及以下嵌套） |
| nickname | VARCHAR(50) | 否 | — | 访客昵称（自填；博主回复存管理员 nickname） |
| email | VARCHAR(100) | 是 | NULL | 选填，仅后台可见不公开 |
| content | TEXT | 否 | — | 评论内容（纯文本，输出时转义，不渲染 Markdown） |
| is_admin | TINYINT(1) | 否 | 0 | 博主回复标识（前台高亮+徽章） |
| status | TINYINT UNSIGNED | 否 | 0 | 0=待审核 1=已通过 2=已驳回 |
| ip | VARCHAR(45) | 否 | — | IPv6 最长 45 字符；限流与反垃圾用 |
| user_agent | VARCHAR(255) | 是 | NULL | 反垃圾辅助 |
| created_at / deleted_at | DATETIME | 否/是 | — | 软删除（后台"删除"=软删，保留审计） |

索引：
- `idx_post_status_created` (post_id, status, created_at) — 前台拉取某文章已通过评论
- `idx_status_created` (status, created_at) — 后台待审核队列
- `idx_ip_created` (ip, created_at) — 评论限流查询

枚举字典：

| 字段 | 值 | 含义 |
|---|---|---|
| status | 0 / 1 / 2 | 待审核 / 已通过 / 已驳回 |

### 4.7 likes — 点赞（指纹去重）

| 字段 | 类型 | 空 | 默认 | 说明 |
|---|---|---|---|---|
| id | BIGINT UNSIGNED PK AI | 否 | — | |
| post_id | BIGINT UNSIGNED | 否 | — | FK → posts.id，ON DELETE CASCADE |
| fingerprint | CHAR(64) | 否 | — | SHA-256( IP + UA + 盐 )，盐值存服务端配置 |
| created_at | DATETIME | 否 | — | |

索引：`UNIQUE(post_id, fingerprint)`
行为：点赞=INSERT + `like_count+1`（事务）；取消=DELETE + `like_count-1`；唯一键冲突视为已点赞（幂等）。

### 4.8 site_settings — 站点设置（键值对）

| 字段 | 类型 | 空 | 说明 |
|---|---|---|---|
| `key` | VARCHAR(50) PK | 否 | 配置键（保留字 SQL 转义为 \`key\`，GORM 列名 setting_key） |
| value | TEXT | 是 | 配置值 |

预置键字典（初始化脚本写入默认值）：

| key | 类型 | 默认值 | 说明 |
|---|---|---|---|
| site_name | string | 我的数字花园 | 站点名 |
| site_desc | string | 记录技术与生活 | 站点简介（SEO description 兜底） |
| avatar | url | '' | 站长头像 |
| wechat_qr | url | '' | 微信收款码 |
| alipay_qr | url | '' | 支付宝收款码 |
| donate_text | string | 写得不错，请我喝杯咖啡 ☕ | 赞赏文案 |
| social_links | json | [] | `[{"name":"GitHub","url":"...","icon":"github"}]` |
| page_size | int | 10 | 前台列表每页条数（上限 50） |
| comment_enabled | bool | true | 评论总开关（false 时前台隐藏评论区） |
| sensitive_words | text（换行分隔） | （内置初始词表） | 敏感词表，命中即拒（spec §5.5） |

### 4.9 ER 关系图（文字版）

```
admins （独立，单行）
categories 1 ──── * posts （category_id，SET NULL）
posts * ──── * tags        （post_tags，CASCADE）
posts 1 ──── * comments    （post_id，CASCADE；comments.parent_id 自关联→顶级评论）
posts 1 ──── * likes       （post_id，CASCADE）
site_settings （独立 KV）
```

**设计要点**：
- 图片集复用 posts（type=2，正文即 Markdown 图片语法），视频同理（type=3 + video_url），不建 gallery 表
- 冗余计数（like_count/comment_count）避免列表 N+1，全部在事务内原子更新
- 评论两级结构（parent_id 仅指向顶级），避免无限嵌套
- 前台接口只暴露 status=1 且 deleted_at IS NULL 的内容
- 所有 FK 指向内容表的均为 CASCADE（文章删除连带清理），指向维表的为 SET NULL（分类删除不删文章）

---

## 5. API 设计

### 5.1 前台公开接口 `/api`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/posts?type=&category=&tag=&page=&size=` | 列表（分页+筛选，仅已发布） |
| GET | `/posts/:slug` | 详情（views 原子 +1） |
| GET | `/posts/:slug/like` | 查询当前指纹是否已赞 |
| POST | `/posts/:slug/like` | 点赞/取消（指纹去重） |
| GET | `/posts/:slug/comments?page=` | 评论列表（仅 status=1） |
| POST | `/posts/:slug/comments` | 发表评论（进入待审核） |
| GET | `/categories` · `/tags` | 分类/标签及文章数 |
| GET | `/archive` | 按年月分组时间线数据 |
| GET | `/site` | 站点信息（含收款码图片地址，后台可配） |

### 5.2 管理接口 `/api/admin`（JWT 保护）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/login` | 登录（IP 限流 5 次/分钟，锁 15 分钟） |
| GET | `/stats` | 仪表盘统计 |
| CRUD | `/posts` `/posts/:id` | 文章管理（含草稿/回收站/置顶/推荐） |
| PUT | `/posts/:id/status` | 快捷切换状态 |
| POST | `/upload` | 图片上传（白名单 jpg/png/webp/gif，≤5MB，MD5 命名 + 魔数校验）；服务端同步生成 ~800px 宽 WebP 缩略图，返回 `{url, thumb_url}`，正文 Markdown 中图片语法写入 thumb、lightbox 加载原图 |
| CRUD | `/categories` `/tags` | 分类/标签管理 |
| GET | `/comments?status=` | 评论按状态筛选 |
| PUT | `/comments/:id/review` | 通过/驳回 |
| POST | `/comments/:id/reply` | 管理员回复（is_admin=1，自动通过） |
| GET·PUT | `/profile` · `/site` | 个人资料 / 站点设置 |

### 5.3 统一约定

```jsonc
// 响应：{ "code": 0, "msg": "ok", "data": { ... } }
// 分页：data: { "list": [...], "total": 128, "page": 1, "size": 10 }
// 错误码：20001 token校验失败 / 20002 token过期 / 40001 参数错误
//         40004 资源不存在 / 40005 无权限
```

### 5.4 防垃圾评论（四层）

1. **IP 限流**：同 IP 每文章 1 分钟 1 条、每天 10 条（Go 内存限流器）
2. **先审后显**：评论 status=0 进审核队列，后台一键通过/驳回
3. **内容黑名单**：关键词过滤 + 链接数 >2 自动驳回
4. **Honeypot**：隐藏字段被填写直接丢弃

### 5.5 评论内容净化（用户补充需求 2026-08-21）

评论输入按以下规则处理（提交时服务端执行，前端同步预校验提示）：

1. **纯文本 + 白名单表情**：评论不支持 HTML/Markdown 渲染（存储原文，输出转义）；支持的表情仅限 Unicode emoji（系统输入法/复制粘贴的 😊🎉 等，天然安全），**不支持自定义表情包图片**（防止外链图片追踪与恶意内容）
2. **粘贴格式清洗**：
   - 去除零宽字符（​‌‍﻿）与控制字符（除换行）
   - 连续空白/换行压缩（最多保留 2 个连续换行）
   - HTML 实体形式（如 `&lt;script&gt;`）按字面处理，输出时照常转义，不二次解码
3. **敏感词过滤**（骂人/侮辱性词汇）：
   - 维护敏感词表（后台可扩展：site_settings 新增 `sensitive_words` 键，换行分隔）
   - 匹配算法：DFA/AC 自动机（sungelei/fsm 或手写 trie，避免正则回溯）
   - 处理策略：命中 → 直接拒绝提交，返回 40006「评论包含违规内容，请修改」
   - 变形对抗：匹配前做归一化（全角→半角、大写→小写、去除中间插空格/符号如「傻 X」「傻.逼」）
4. **长度限制**：1-500 字符（rune 计），昵称 1-20 字符
5. **输出侧兜底**：即使过滤被绕过，前端渲染评论一律 `{{ }}` 文本插值（Vue 默认转义），永不 v-html

---

## 6. 前台页面设计（暗色科技风）

### 6.1 页面清单

| 路由 | 页面 | 说明 |
|---|---|---|
| `/` | 首页 | Hero（打字机动效 + Aurora 背景）+ **Bento Grid** 首屏 + 分类 Tab 文章流 |
| `/post/:slug` | 详情 | Markdown 渲染 + TOC + 评论点赞 + 赞赏 |
| `/category/:name` `/tag/:name` | 列表 | 分类/标签筛选 |
| `/archive` | 归档 | 按年月分组时间线 |
| `/about` | 关于 | 个人介绍 + 社交链接 + 赞赏区 |
| `/search?q=` | 搜索 | 标题/摘要/标签模糊匹配 |

### 6.1.1 视觉设计规范（暗色科技风 · 2026 现代版）

**设计原则**：轻量 CSS 为主、零重型 3D/全屏粒子（避免廉价感与移动端性能损耗）。

1. **Bento Grid 首屏**（替换传统三栏卡片）：不等宽便当盒网格，信息密度高且自带设计感（linear.dev / apple.com 风格）：
   ```
   ┌────────────┬──────────┬────────┐
   │  关于我(大)  │ 精选文章   │ 统计    │
   │  头像+一句话  │ (宽卡片)  │ 文章/赞  │
   │            ├──────────┼────────┤
   │  最新文章    │  标签云    │ 赞赏☕  │
   └────────────┴──────────┴────────┘
   ```
2. **光标聚光卡片**（spotlight hover）：鼠标移过卡片时径向光跟随光标照亮边框——暗色主题下效果拔群，`radial-gradient` + CSS 变量跟踪 `mousemove`，约 20 行 CSS
3. **View Transitions 页面切换**：Vue 3.4+ 原生 View Transitions API，文章卡片点击平滑放大过渡成详情页（App 般体验，Chrome 已支持，不支持的浏览器优雅降级为普通切换）
4. **Aurora 背景 + 噪点纹理**：暗色底上叠加缓慢漂移的极光色斑（CSS 动画）+ SVG `feTurbulence` 噪点，替代纯渐变的廉价感，零 JS 成本
5. **中文字体策略（硬约束）**：**禁止全量引入中文 webfont**（5-10MB）。英文/数字/代码用有性格的字体（标题点缀 `Space Grotesk`，代码 `JetBrains Mono`），中文走系统字体栈（PingFang SC / 微软雅黑 / HarmonyOS Sans）；如需标题特殊字形，仅对少量关键字做子集化（fonttools）

### 6.2 详情页关键设计

- 正文最大宽度 ~720px；Shiki 代码高亮（VS Code 暗色主题）；图片 lightbox
- 桌面端右侧悬浮 TOC 滚动高亮，移动端抽屉
- 三种内容形态：文章（Markdown）/ 图片集（瀑布流+lightbox）/ 视频（顶部 16:9 iframe + 文字说明）
- 交互：点赞动效 + 指纹防重、上/下一篇、相关文章（共同标签）
- **赞赏（爱心捐赠）**：文章底部赞赏区，点击「请我喝杯咖啡 ☕」弹出模态框，Tab 切换微信/支付宝收款码（图片长按识别）；关于页同样展示固定赞赏区

## 7. 后台管理设计

```
📊 仪表盘      统计卡 + 待审评论提醒
📝 文章管理    列表筛选 · 编辑器（md-editor-v3 分屏）
   └ 编辑器    元信息（slug 自动拼音生成可改/摘要/封面上传）
               分类选择 + 标签多选（可即时新建）
               类型切换（文章/图片集/视频-粘贴嵌入代码）
               保存草稿/预览/发布/定时发布
               编辑器内粘贴截图直接上传；30 秒自动保存草稿
               自动保存失败（401/网络）时降级写入 localStorage 兜底，恢复后提示恢复
🗂 分类标签    分类拖拽排序 + 标签合并/重命名
💬 评论管理    状态 Tab · 通过/驳回/回复/删除
⚙️ 站点设置    站点名/简介/头像/社交链接/每页条数/评论开关/
               赞赏设置（微信/支付宝收款码上传 + 赞赏文案，如"写得不错，请我喝杯咖啡"）
👤 个人资料    昵称/头像/改密码（旧密码验证）
```

- 摘要留空自动截取正文前 120 字；slug 保存时校验唯一
- 草稿/回收站内容前台 404

---

## 8. 安全设计

| 威胁 | 防御 |
|---|---|
| XSS | markdown-it `html:false` + v-html 前 DOMPurify + 评论纯文本转义 |
| JWT 盗用 | 2h 有效期 + **HttpOnly Cookie**（非 localStorage）+ SameSite=Strict；**滑动过期**（管理端每次请求刷新 Cookie 有效期，避免长文写作中 token 过期导致自动保存静默失败）| 
| 密码 | bcrypt(cost=10) + 登录限流 |
| 滥用 | 全站 IP 限流（内存令牌桶 60 req/min/IP） |
| 恶意上传 | 扩展白名单 + **文件头魔数校验** + MD5 重命名 + Nginx 上传目录禁执行 |
| SQL 注入 | GORM 参数化查询全覆盖 |
| CSRF | 同域 + SameSite=Strict + 写操作校验 X-Requested-With |
| 点击劫持 | X-Frame-Options: DENY / nosniff 等安全头 |

### 8.1 SEO：服务端 meta 注入（应对百度不执行 JS）

百度爬虫不渲染 JavaScript，SPA 的文章详情页在其眼中是空壳。对策：**Go 在返回 `index.html` 时按路径动态注入 meta**——

- 前端静态文件仍由 Nginx 托管，但对 `/post/:slug` 路径的请求改由 Go 处理（Nginx `location /post/` 反代到 Go）
- Go 读取 `dist/index.html` 模板（启动时缓存到内存），查询该 slug 文章的标题/摘要/封面，替换 `<title>`、`<meta description>`、OG 标签后返回
- 文章不存在或未发布时直接返回原样 index.html（前端路由正常 404）
- 首页/关于页保持预渲染静态文件

效果：百度/微信分享卡片/短信预览均能拿到正确标题摘要；Google 不受影响。成本约半天，无 SSR 复杂度。
| 目录穿越 | 文件名服务端生成，不用用户原始名 |

## 9. 错误处理与日志

- **后端分层**：handler 参数校验（400+字段错误）→ service 业务码 → middleware panic recover（500+requestID）
- **前端**：axios 拦截器统一弹错；401 跳登录；接口失败显示空状态+重试，不白屏
- **日志**：slog 结构化 JSON（requestID/method/path/status/耗时/IP），按天滚动保留 30 天

## 10. 部署

```
deploy.sh：前端 build → go 交叉编译 → scp 上传 → systemctl restart blog
服务器：Nginx(80/443, certbot) + blog.service(systemd) + MySQL(127.0.0.1)
备份：每日 mysqldump → /data/backup 保留 7 天 + uploads 同步备份
```

性能余量：单机 Go + MySQL 支撑日均 1 万+ PV；首版不加缓存/CDN。

## 11. 测试策略

| 层 | 工具 | 重点 |
|---|---|---|
| 后端单测 | Go testing + testify | 评论审核流转、点赞去重、slug 生成/唯一、限流器 |
| 后端集成 | testcontainers-go（真实 MySQL） | GORM 查询、分页筛选、软删除 |
| API 测试 | httptest + Gin | 登录/JWT/CRUD 全流程、权限边界（无 token → 401） |
| 前端 | Vitest（Playwright 可选） | Markdown 渲染净化（XSS 用例）、表单校验、路由守卫 |
| E2E 主链路 | Playwright | 发布文章→前台可见→发评论→后台审核通过 |

## 12. 里程碑（建议）

1. **M1 后端骨架**：项目结构 + 数据库迁移 + 登录/JWT（滑动过期）+ 文章 CRUD + 上传（含 WebP 缩略图管线）
2. **M2 后台前端**：登录 + 仪表盘 + 文章编辑器（图片粘贴上传、自动保存 + localStorage 兜底）+ 评论管理 + 站点设置（含收款码）
3. **M3 前台**：首页（Bento Grid + Aurora + 聚光卡片）/详情（Markdown 渲染+TOC+View Transitions）/列表/归档/关于/搜索/赞赏
4. **M4 互动与上线**：评论/点赞 + 防垃圾 + SEO meta 注入 + 安全加固 + 部署脚本 + 备份
