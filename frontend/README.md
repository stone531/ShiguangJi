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
├── vite.config.ts          # @ alias + proxy /api,/uploads → :8080
├── tsconfig*.json
└── src/
    ├── main.ts
    ├── App.vue             # NConfigProvider(darkTheme) + Message/Dialog Provider
    ├── api/
    │   ├── http.ts         # axios 单例 + 统一拦截（code≠0 / 401 跳登录）
    │   ├── auth.ts         # login / changePassword
    │   ├── post.ts         # 文章 CRUD / status / uploadImage
    │   ├── comment.ts      # 评论（M4 前 404 → mock 降级）
    │   └── site.ts         # 站点设置（M4 前 404 → 默认值降级）
    ├── stores/admin.ts     # 管理员状态（sessionStorage 恢复）
    ├── router/index.ts     # /admin/login 公开 + /admin/* 守卫 + 懒加载
    ├── layouts/AdminLayout.vue
    ├── composables/useAutoSave.ts  # 30s 自动保存 + localStorage 兜底
    ├── types/api.d.ts      # Response/PageData/Post/CommentItem/SiteSettingVO
    └── views/admin/        # Login/Dashboard/PostList/PostEditor/CommentManage/SiteSettings
```

## M4 接口对接清单（后端就绪后移除 mock）

| 位置 | 当前行为 | M4 动作 |
|---|---|---|
| `src/api/comment.ts` | `listComments/reviewComment/replyComment/deleteComment` 捕获 404 → 本地假数据 / 假成功（`console.warn TODO(M4)`） | 后端实现 `/admin/comments` 系列接口后删除 `isNotFound` 降级分支与 `mockComments`，恢复直接走 `get/put/post/del` |
| `src/api/site.ts` | `getSiteSettings` 404 → spec §4.8 默认值；`saveSiteSettings` 404 → 假成功；`isSiteApiAvailable` 探测接口是否可用 | 后端实现 `GET /admin/site`、`PUT /admin/site` 后删除降级分支与 `isSiteApiAvailable`（同时移除 SiteSettings.vue 的 mock 提示 NAlert 与 `isMock` 逻辑） |
| `src/views/admin/CommentManage.vue` | 页面顶部 NAlert「评论接口后端将在 M4 提供，当前展示示例数据」 | 接口真实化后删除该 NAlert |
| `src/views/admin/PostEditor.vue` | 分类/标签为本地静态占位（提交 `category_id: null`、`tag_ids: []`，标注 TODO(M4)） | 后端实现分类/标签 CRUD 后，NSelect 换成接口数据，启用真实 category_id / tag_ids 提交 |

> 说明：mock 降级是有意设计（保证 M2 可独立验收），非未完成 TODO。搜索 `TODO(M4)` 可定位全部接入点。
