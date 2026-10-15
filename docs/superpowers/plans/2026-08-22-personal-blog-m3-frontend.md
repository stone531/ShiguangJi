# 个人博客 M3 前台展示站 — 实施计划

> **For agentic workers:** 按任务顺序实施，步骤用 checkbox（`- [ ]`）跟踪。

**Goal:** 构建前台展示站：首页（Hero 打字机 + Bento Grid + 分类 Tab 文章流）、文章详情（Markdown 渲染 + TOC + 赞赏）、分类/标签列表、归档、关于、搜索，以及前台所需的 4 个公开只读接口。

**Architecture:** 复用既有单 SPA：`/` 前台路由组（SiteLayout + 懒加载）与 `/admin` 并存。视觉按 spec §6.1.1「轻量 CSS 为主」：aurora/噪点/聚光卡片/打字机全部 CSS+原生 JS 实现，View Transitions 用 vue-router 内置支持（不支持则优雅降级）。

**Tech Stack:** 现有栈 + markdown-it + DOMPurify + Shiki（dark-plus）+ @fontsource（Space Grotesk / JetBrains Mono，本地打包，不引中文 webfont）

**Spec:** `docs/superpowers/specs/2026-08-21-personal-blog-design.md`（§5.1 公开 API / §6 前台页面 / §6.1.1 视觉规范）

**后端现状（M1/M2 已交付）：** `GET /api/posts`（仅 category_id 筛选）、`GET /api/posts/:slug`（返回整篇 model）。缺 `/site` `/categories` `/tags` `/archive`，列表不支持 type/tag/keyword/featured 筛选。`site_settings` 默认值已由 migration 001 预置。

---

## 与 spec 的偏离（先行声明）

1. **不引入 GSAP / @vueuse/motion**：spec §2 列为「按需」，§6.1.1 明确「轻量 CSS 为主、零重型 3D/粒子」。打字机/aurora/聚光均 ≤30 行 CSS/TS 可达，YAGNI。
2. **样式优先 Tailwind v4（spec §2 选型）**；若与 Vite 8 生态不兼容则降级为「全局 CSS 变量 + scoped 样式」，视觉规格不变（本计划按后者书写亦可执行）。
3. **详情响应扩展**：`GET /posts/:slug` 从直接返回 post 改为 `{post, prev, next}`（实现 §6.2 上下篇导航；无既有消费方，安全）。
4. **列表响应改轻量 VO**（不含 content/content_html），卡片字段对齐 §6.2。
5. **评论/点赞**为 M4：详情页渲染占位区块（读 `comment_enabled` 预留）。
6. **Lightbox 展示原图**：编辑器插入的是 thumb_url，原图 URL 未随内容存储，v1 lightbox 即所点图 src（记录为已知限制）。

## API 契约（M3 新增/变更）

```
GET /api/site        → {site_name, site_desc, avatar, wechat_qr, alipay_qr, donate_text,
                        social_links:[{name,url,icon}], page_size, comment_enabled,
                        post_count, view_count, like_count}        # 不含 sensitive_words
GET /api/categories  → {list:[{id,name,sort,post_count}]}          # 仅计已发布
GET /api/tags        → {list:[{id,name,post_count}]}               # post_count 降序
GET /api/archive     → {list:[{year, month, posts:[{id,title,slug,published_at}]}]}

GET /api/posts?type=&category=<name>&tag=<name>&keyword=&featured=&page=&size=
                      → {list:[PostCard...], total, page, size}    # PostCard 无 content
GET /api/posts/:slug → {post, prev:{slug,title}|null, next:{slug,title}|null}
```

---

### Task 1: 后端公开接口

**Files:** `internal/service/site_service.go`、`taxonomy_service.go`、`post_service.go`（扩展）、`internal/handler/site_handler.go`、`taxonomy_handler.go`、`post_handler.go`（扩展）、`router.go`、`*_test.go`

- [x] SiteService.GetPublic：读 KV → 默认值兜底（§4.8）→ social_links JSON 解析 → 附统计（post_count/view_count/like_count，仅已发布）
- [x] TaxonomyService：分类/标签按已发布文章计数（LEFT JOIN posts status=1 / JOIN post_tags）
- [x] PostService.Archive：已发布按 published_at 倒序 → Go 内按年月分组
- [x] ListPublic 扩展：type/category(name)/tag(name)/keyword(标题+摘要+标签名 LIKE，通配符转义)/featured
- [x] GetBySlugPublic 返回 (post, prev, next)：prev=发布更早相邻、next=更新相邻
- [x] handler：列表映射轻量 PostCard VO；路由注册到既有 /api 组（SecurityHeaders 内）
- [x] 测试（sqlite）：默认值兜底、计数仅含已发布、年月分组、keyword/featured/tag 筛选、prev/next 相邻性

### Task 2: 前端基建

**Files:** `package.json`、`vite.config.ts`（如装 Tailwind）、`src/styles/theme.css`、`src/stores/site.ts`、`src/api/public.ts`、`src/types/api.d.ts`（扩展）、`src/router/index.ts`、`index.html`、`public/favicon.svg`

- [x] 依赖：markdown-it @types/markdown-it dompurify shiki @fontsource-variable/space-grotesk @fontsource/jetbrains-mono（Tailwind 视安装结果）
- [x] theme.css：色板 token（bg #0b0b10 系/主色 cyan-violet 渐变）、aurora keyframes、feTurbulence 噪点、glass 卡片、spotlight 变量、`.md-body` 排版（720px）
- [x] site store：load once + ensure；types：SitePublicVO/PostCard/CategoryItem/TagItem/ArchiveGroup
- [x] 路由：`/` 挂 SiteLayout（children 懒加载 + viewTransition:'default'），catch-all 改前台 404；/admin 组不动
- [x] index.html：lang=zh-CN、标题=时光集、favicon

### Task 3: 首页

**Files:** `src/layouts/SiteLayout.vue`、`src/views/site/HomeView.vue`、`src/components/site/ArticleCard.vue`、`src/composables/useTypewriter.ts`

- [x] SiteLayout：吸顶玻璃导航（站名/首页/归档/关于/搜索入口）+ 页脚（© + social）+ 全局 aurora/噪点背景
- [x] Hero：avatar + 站名渐变字 + 打字机（site_desc 起头的短语轮播）+ CTA（开始阅读/了解我）
- [x] Bento Grid（§6.1.1 版式）：关于我(大)/精选文章/统计/最新文章/标签云/赞赏（点击开 DonateModal）
- [x] 分类 Tab（全部+分类）+ 文章流（分页），ArticleCard 聚光悬停 + 封面 thumb + 元信息

### Task 4: 详情页

**Files:** `src/views/site/PostDetailView.vue`、`src/components/site/MarkdownView.vue`、`Toc.vue`、`Lightbox.vue`、`DonateModal.vue`

- [x] MarkdownView：markdown-it(html:false, linkify) → DOMPurify → v-html；标题注入 id；异步 Shiki（dark-plus，动态 import）高亮 pre>code；图片点击开 Lightbox + loading=lazy
- [x] TOC：h2/h3 树、IntersectionObserver 高亮、桌面右侧悬浮/移动抽屉
- [x] 三形态：type=3 顶部 16:9 iframe（域名白名单已由后端保证）；type=1/2 同为 Markdown 渲染
- [x] 上/下一篇（详情响应 prev/next）、相关文章（首标签查列表，排除自身）、赞赏区（DonateModal 微信/支付宝 Tab）、评论占位（M4）

### Task 5: 列表/归档/关于/搜索/404

**Files:** `src/views/site/ArticleListView.vue`、`ArchiveView.vue`、`AboutView.vue`、`SearchView.vue`、`NotFoundView.vue`

- [x] ArticleListView：/category/:name 与 /tag/:name 复用（props 区分），卡片网格 + 分页
- [x] ArchiveView：年份分组时间线（左竖线 + 节点）
- [x] AboutView：头像/介绍/社交链接/固定赞赏区
- [x] SearchView：?q= 同步、防抖 300ms、空态提示
- [x] NotFoundView：404 + 返回首页

### Task 6: 验证与收尾

- [x] `cd backend && make test` / `make lint`
- [x] `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json` 0 error；`npm run build`
- [x] 手动冒烟：首页 Bento/Tab 翻页、详情渲染/TOC/代码高亮/Lightbox、归档、搜索、/admin 回归不受影响
- [x] 更新根 README 里程碑与文档链接、frontend/README（前台结构 + site.ts 读路径已真实化说明）、CLAUDE.md 进度行

---

## Self-Review 记录

- **Spec 覆盖**：§5.1 公开接口新增 `/site` `/categories` `/tags` `/archive` + 列表五条件筛选；§6 六页 + 404 全部交付；§6.1.1 Bento/聚光/aurora/噪点/打字机/View Transitions 全落地（纯 CSS/原生实现，未引动效库）。
- **偏离执行情况**：Tailwind v4 与 Vite 8 兼容安装成功，未触发降级路径；其余偏离按声明执行。
- **实施中发现并处理**：
  - views 计数返回旧值（库内已 +1）→ 返回对象同步自增，带测试；
  - GORM `Scan` 到含切片字段的结构体会被误判关联 → 统计扫进独立结构体；
  - 归档年月分组中 `time.Month` 显式转 int；
  - M2 的 `site.ts` mock 走 `/admin/site`（M4 范围），与本次公开 `/site` 无冲突，未改动。
- **已知限制（M4 或后续）**：Lightbox 展示所点图 src（原图 URL 未随内容存储）；Shiki 语法 chunk 按语言懒加载，单语言最大 ~800KB 仅在文章含该语言时拉取，如需收敛可换 `shiki/bundle/web`；评论/点赞为占位。
- **验证**：`go test ./... -race` 全绿（新增 5 个公开接口测试）；`vue-tsc --noEmit` 0 error；`npm run build` 产出正常、admin chunk 依旧独立。
