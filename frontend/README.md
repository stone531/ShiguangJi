# 个人博客管理后台（M2）

Vue 3 + Vite + TypeScript + Naive UI（暗色主题）+ Pinia + Vue Router + axios + md-editor-v3。

## 启动步骤

1. 启动后端（仓库根目录，需 Go 环境与已初始化管理员账号）：

   ```bash
   make dev        # 监听 :8080
   # 首次：按后端 README 初始化管理员（initadmin）
   ```

2. 启动前端：

   ```bash
   cd frontend
   npm install
   npm run dev     # 监听 :5173，/api 与 /uploads 已代理到 :8080
   ```

3. 浏览器访问 <http://localhost:5173/admin/login> 登录。

## 构建与检查

```bash
npm run build                              # 产出 dist/
npx vue-tsc --noEmit -p tsconfig.app.json  # strict 类型检查
```

后台各页面均为路由懒加载独立 chunk（`dist/assets/CommentManage-*.js`、`SiteSettings-*.js` 等）。

## 目录结构

```
frontend/
├── index.html
├── vite.config.ts          # @ alias + proxy /api,/uploads → :8080 + Tailwind v4
├── tsconfig*.json
└── src/
    ├── main.ts
    ├── App.vue             # NConfigProvider(darkTheme) + Message/Dialog Provider
    ├── styles/theme.css    # Tailwind v4 + 暗色科技风 token/aurora/噪点/聚光/md-body 排版
    ├── api/
    │   ├── http.ts         # axios 单例 + 统一拦截（code≠0 / 401 跳登录；默认头 X-Requested-With）
    │   ├── auth.ts         # login / changePassword / profile
    │   ├── post.ts         # 文章 CRUD / status / uploadImage
    │   ├── comment.ts      # 评论（前台列表+发表 / 后台审核回复删除）
    │   ├── site.ts         # 站点设置（含敏感词表，公开接口不含该键）
    │   ├── taxonomy.ts     # 后台分类/标签 CRUD + 标签合并
    │   └── public.ts       # 前台公开接口（site/posts/categories/tags/archive）
    ├── stores/admin.ts     # 管理员状态（sessionStorage 恢复）
    ├── stores/site.ts      # 前台站点信息（load once）
    ├── router/index.ts     # / 前台路由组 + /admin/login 公开 + /admin/* 守卫，全部懒加载
    ├── layouts/AdminLayout.vue
    ├── layouts/SiteLayout.vue      # 前台：玻璃导航 + 页脚 + aurora/噪点背景
    ├── composables/useAutoSave.ts  # 30s 自动保存 + localStorage 兜底
    ├── composables/useTypewriter.ts # Hero 打字机
    ├── components/site/    # ArticleCard/MarkdownView/Toc/Lightbox/DonateModal/LikeButton/CommentSection
    ├── components/admin/   # ChangePasswordForm（顶栏弹窗与 Profile 页共用）
    ├── types/api.d.ts      # Response/PageData/Post/PostCard/CommentItem/SiteSettingVO/SitePublicVO...
    ├── views/admin/        # Login/Dashboard/PostList/PostEditor/CommentManage/SiteSettings/Profile
    └── views/site/         # Home/PostDetail/ArticleList/Archive/About/Search/NotFound
```

## 前台展示站（M3）

- 视觉按 spec §6.1.1：Bento Grid 首页、光标聚光卡片、aurora + 噪点背景、打字机 Hero、View Transitions（vue-router 内置，浏览器不支持自动降级）
- Markdown 渲染管线：markdown-it（`html:false`）→ DOMPurify → v-html → Shiki（dark-plus，按语言懒加载）
- 互动（M4）：文章页 LikeButton（指纹点赞切换）+ CommentSection（两级树、博主徽章、蜜罐表单、成功提示待审核）；`comment_enabled=false` 时整个评论区隐藏；赞赏弹窗收款码来自 `/api/site`

## 接口约定（M4 起）

- M2 时代的 404 mock 降级已全部移除（`comment.ts`/`site.ts`/`CommentManage`/`PostEditor` 均已真实化），后端未启动时接口直接报错。
- 除登录外全部 `/api` 写操作要求 `X-Requested-With: XMLHttpRequest` 头（CSRF 纵深防御），已在 `http.ts` axios 默认头统一注入，新增自建的 fetch 调用需自行携带。
