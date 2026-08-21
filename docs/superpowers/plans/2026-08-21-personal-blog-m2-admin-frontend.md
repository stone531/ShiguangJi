# 个人博客 M2 后台前端 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建管理后台 SPA：登录、仪表盘、文章管理（md-editor-v3 编辑 + 自动保存兜底）、分类标签、评论管理（M4 数据接入前先占位）、站点设置（含收款码上传）。

**Architecture:** Vue 3 + Vite + TS 单页应用，`/admin` 路由组懒加载独立 chunk；axios 统一拦截（业务码/401 跳登录）；Pinia 管理员状态；Naive UI 暗色主题。与后端同域反代（开发期 Vite proxy `/api` → :8080）。

**Tech Stack:** Vue 3 / Vite / TypeScript / Naive UI / Pinia / Vue Router / axios / md-editor-v3

**Spec:** `docs/superpowers/specs/2026-08-21-personal-blog-design.md`（§7 后台管理设计 / §5.2 admin API / §5.5 评论净化）

**后端现状（M1 已交付）**：登录 `POST /api/admin/login`（返回 HttpOnly cookie）、改密 `POST /api/admin/password`、stats 占位、文章 CRUD（GET/POST `/api/admin/posts`、GET/PUT/DELETE `/api/admin/posts/:id`、PUT `:id/status`）、上传 `POST /api/admin/upload`（返回 {url,thumb_url}）、前台公开 GET `/api/posts`。**评论与站点设置接口后端尚未实现**——本计划前端部分做成真实页面 + mock 降级（接口 404 时用本地假数据渲染，标注 TODO(M4)），保证 M2 可独立验收。

---

## 编码标准（前端，全计划强制）

### 命名规范

| 项 | 规则 | 示例 |
|---|---|---|
| 组件文件 | 多词 PascalCase.vue | `PostEditor.vue`、`CommentManage.vue` |
| 组合式函数 | use 前缀 camelCase.ts | `useAutoSave.ts` |
| 目录 | kebab-case 或语义分组 | `views/admin/`、`api/` |
| 变量/函数 | camelCase；常量 SCREAMING_SNAKE | `const MAX_DRAFT_SIZE = 2_000_000` |
| 类型/接口 | PascalCase，API 类型与后端 JSON 字段一致（snake_case 字段） | `interface Post { id: number; published_at: string \| null }` |
| 事件 | emit kebab-case | `emit('status-change', id)` |
| CSS | Tailwind 优先；scoped style 用 kebab class | `.editor-toolbar` |

### 质量红线（防 bug / 内存泄漏）

1. **所有组件内定时器/事件监听必须在 `onBeforeUnmount` 清理**（自动保存 interval、beforeunload、resize 监听）
2. **axios 实例单例**（`api/http.ts` 导出唯一实例），禁止组件内 `axios.create`
3. **路由懒加载**：admin 全部 `() => import(...)`；登录页与后台布局分离
4. **请求竞态**：列表翻页/筛选用「请求序号守卫」（最新请求才落 state）
5. **错误统一拦截**：`code≠0` 弹 message；401 清状态跳 `/admin/login`；网络错误重试提示
6. **TypeScript strict**：`tsconfig` 开 strict；禁止 `any`（API 响应定义完整 interface）
7. **表单离开守卫**：编辑器有未保存内容时 `onBeforeRouteLeave` 确认

### 目录结构

```
frontend/
├── index.html
├── vite.config.ts          # @ alias + proxy /api→8080
├── tsconfig.json
├── package.json
└── src/
    ├── main.ts
    ├── App.vue             # NConfigProvider(darkTheme) + NMessageProvider
    ├── api/
    │   ├── http.ts         # axios 单例 + 拦截器 + Response<T> 泛型
    │   ├── auth.ts         # login/changePassword
    │   ├── post.ts         # CRUD/status/upload + Post 接口
    │   ├── comment.ts      # M4 前降级 mock
    │   └── site.ts         # M4 前降级 mock
    ├── stores/
    │   └── admin.ts        # 管理员信息（nickname/avatar/must_change_password）
    ├── router/
    │   └── index.ts        # /admin/login 公开 + /admin/* 守卫
    ├── layouts/
    │   └── AdminLayout.vue # 侧边栏 + 顶栏 + 内容区
    ├── composables/
    │   └── useAutoSave.ts  # 30s 自动保存 + localStorage 兜底
    ├── views/admin/
    │   ├── Login.vue
    │   ├── Dashboard.vue
    │   ├── PostList.vue
    │   ├── PostEditor.vue
    │   ├── CommentManage.vue   # mock
    │   └── SiteSettings.vue    # mock（含收款码上传，上传接口真实）
    └── types/
        └── api.d.ts        # Response/PageData/Post/Comment/SiteSetting
```

---

### Task 1: 工程初始化 + axios 层 + 类型

**Files:** `frontend/package.json`、`vite.config.ts`、`tsconfig.json`、`src/main.ts`、`App.vue`、`api/http.ts`、`types/api.d.ts`

- [ ] **Step 1: 脚手架**

```bash
cd D:/code/test/ai/frontend
npm create vite@latest . -- --template vue-ts
npm i naive-ui pinia vue-router@4 axios md-editor-v3
npm i -D @types/node
```

- [ ] **Step 2: vite.config.ts**

```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/uploads': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
```

- [ ] **Step 3: types/api.d.ts（与后端 JSON 完全对齐）**

```ts
export interface Response<T = unknown> {
  code: number
  msg: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  size: number
}

export interface Post {
  id: number
  title: string
  slug: string
  summary: string
  cover: string
  content: string
  type: 1 | 2 | 3
  video_url: string
  category_id: number | null
  status: 0 | 1 | 2
  is_top: boolean
  is_featured: boolean
  views: number
  like_count: number
  comment_count: number
  published_at: string | null
  created_at: string
  updated_at: string
  category?: { id: number; name: string } | null
  tags?: { id: number; name: string }[]
}

export interface UploadResult {
  url: string
  thumb_url: string
}

export interface AdminInfo {
  nickname: string
  avatar: string
  must_change_password: boolean
}

export interface CommentItem {
  id: number
  post_id: number
  post_title: string
  parent_id: number
  nickname: string
  content: string
  is_admin: boolean
  status: 0 | 1 | 2
  created_at: string
}

export interface SiteSettingVO {
  site_name: string
  site_desc: string
  avatar: string
  wechat_qr: string
  alipay_qr: string
  donate_text: string
  social_links: { name: string; url: string; icon: string }[]
  page_size: number
  comment_enabled: boolean
}
```

- [ ] **Step 4: api/http.ts（单例 + 拦截器）**

```ts
import axios from 'axios'
import type { AxiosResponse } from 'axios'
import type { Response } from '@/types/api'
import { useAdminStore } from '@/stores/admin'
import router from '@/router'

export const http = axios.create({
  baseURL: '/api',
  timeout: 15_000,
  withCredentials: true,
})

http.interceptors.response.use(
  (resp: AxiosResponse<Response>) => {
    const body = resp.data
    if (body.code !== 0) {
      if (body.code === 20001 || body.code === 20002) {
        useAdminStore().clear()
        router.push('/admin/login')
      }
      return Promise.reject(new Error(body.msg))
    }
    return resp
  },
  (error) => {
    if (error.response?.status === 401) {
      useAdminStore().clear()
      router.push('/admin/login')
    }
    return Promise.reject(new Error(error.response?.data?.msg ?? '网络错误，请稍后重试'))
  },
)

/** 统一取 data 的请求方法 */
export async function get<T>(url: string, params?: object): Promise<T> {
  const resp = await http.get<Response<T>>(url, { params })
  return resp.data.data
}

export async function post<T>(url: string, body?: object): Promise<T> {
  const resp = await http.post<Response<T>>(url, body)
  return resp.data.data
}

export async function put<T>(url: string, body?: object): Promise<T> {
  const resp = await http.put<Response<T>>(url, body)
  return resp.data.data
}

export async function del<T>(url: string): Promise<T> {
  const resp = await http.delete<Response<T>>(url)
  return resp.data.data
}
```

注意：拦截器里 import store/router 会形成循环依赖风险——store 和 router 模块顶层不得 import http.ts。admin store 保持无 API 依赖（纯状态），满足此约束。

- [ ] **Step 5: 验证** `npm run build` 通过（空路由先占位 main.ts 只挂 App）后进入 Task 2

---

### Task 2: 路由 + 登录页 + 管理员 store

**Files:** `src/router/index.ts`、`src/stores/admin.ts`、`src/views/admin/Login.vue`、`src/api/auth.ts`

- [ ] **Step 1: stores/admin.ts**

```ts
import { defineStore } from 'pinia'
import type { AdminInfo } from '@/types/api'

export const useAdminStore = defineStore('admin', {
  state: () => ({
    info: null as AdminInfo | null,
  }),
  getters: {
    isLoggedIn: (s) => s.info !== null,
  },
  actions: {
    set(info: AdminInfo) {
      this.info = info
    },
    clear() {
      this.info = null
    },
  },
})
```

- [ ] **Step 2: router/index.ts（守卫 + 懒加载）**

```ts
import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/admin/login', component: () => import('@/views/admin/Login.vue') },
    {
      path: '/admin',
      component: () => import('@/layouts/AdminLayout.vue'),
      children: [
        { path: '', redirect: '/admin/dashboard' },
        { path: 'dashboard', component: () => import('@/views/admin/Dashboard.vue') },
        { path: 'posts', component: () => import('@/views/admin/PostList.vue') },
        { path: 'posts/new', component: () => import('@/views/admin/PostEditor.vue') },
        { path: 'posts/:id/edit', component: () => import('@/views/admin/PostEditor.vue') },
        { path: 'comments', component: () => import('@/views/admin/CommentManage.vue') },
        { path: 'settings', component: () => import('@/views/admin/SiteSettings.vue') },
      ],
    },
    { path: '/', redirect: '/admin' }, // M3 前台占位
    { path: '/:pathMatch(.*)*', redirect: '/admin' },
  ],
})

router.beforeEach((to) => {
  const store = useAdminStore()
  if (to.path.startsWith('/admin') && to.path !== '/admin/login' && !store.isLoggedIn) {
    // 无本地状态时尝试一次轻量探测（stats 接口 401 会触发跳登录）
    return { path: '/admin/login' }
  }
})

export default router
```

问题：刷新后 Pinia 状态丢失会误判未登录。方案：登录成功后 `sessionStorage.setItem('admin_info', JSON.stringify(info))`，store 初始化时恢复；真实校验靠接口 401 兜底（拦截器跳登录）。

```ts
// stores/admin.ts 的 state 改为：
state: () => ({
  info: restore(),
}),
// 文件底部：
function restore(): AdminInfo | null {
  try {
    const raw = sessionStorage.getItem('admin_info')
    return raw ? (JSON.parse(raw) as AdminInfo) : null
  } catch {
    return null
  }
}
// set/clear 同步写 sessionStorage
```

- [ ] **Step 3: api/auth.ts + Login.vue**

```ts
// api/auth.ts
import { post } from './http'
import type { AdminInfo } from '@/types/api'

export function login(username: string, password: string) {
  return post<AdminInfo>('/admin/login', { username, password })
}

export function changePassword(oldPassword: string, newPassword: string) {
  return post<null>('/admin/password', {
    old_password: oldPassword,
    new_password: newPassword,
  })
}
```

Login.vue 要点（Naive UI 表单）：NCard 居中 + NForm（username required 3-50 / password required 8-72）→ 调 login → store.set + sessionStorage → `must_change_password` 为 true 时弹 NModal 强制改密（调 changePassword）→ 跳 /admin/dashboard。错误用 useMessage 弹出。**登录按钮 loading 防重复提交**。

- [ ] **Step 4: 手动验证**：后端 `make dev` + 前端 `npm run dev`，浏览器登录成功跳转、错误密码弹错、直接访问 /admin/dashboard 未登录跳登录页

---

### Task 3: AdminLayout + 仪表盘

**Files:** `src/layouts/AdminLayout.vue`、`src/views/admin/Dashboard.vue`、`src/App.vue`

- [ ] **Step 1: App.vue**

```vue
<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NDialogProvider, darkTheme, zhCN, dateZhCN } from 'naive-ui'
</script>

<template>
  <n-config-provider :theme="darkTheme" :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
```

- [ ] **Step 2: AdminLayout.vue**

NLayout has-sider：侧边栏 NMenu（仪表盘/文章管理/评论管理/站点设置，`router-link` 模式）+ 顶栏（nickname 头像下拉：修改密码弹窗 / 退出登录）。退出 = store.clear() + 跳登录。**修改密码弹窗**复用 Login 的 changePassword 逻辑（旧密码 + 新密码 ×2 确认）。

- [ ] **Step 3: Dashboard.vue**

调 `get('/admin/stats')`（M1 返回 `{todo}`）——渲染四个统计卡（文章数/评论数/点赞/浏览），M2 阶段后端未实现真实数据，前端对 `data.todo` 存在时显示占位数字 0 + 页面顶部 NAlert「统计接口将在 M4 提供」。**竞态守卫示例**：

```ts
let seq = 0
async function load() {
  const current = ++seq
  const data = await get<StatsVO>('/admin/stats')
  if (current !== seq) return // 过期响应丢弃
  stats.value = data
}
```

---

### Task 4: 文章列表 + 编辑器（核心）

**Files:** `src/api/post.ts`、`src/views/admin/PostList.vue`、`src/views/admin/PostEditor.vue`、`src/composables/useAutoSave.ts`

- [ ] **Step 1: api/post.ts**

```ts
import { get, post, put, del } from './http'
import type { PageData, Post, UploadResult } from '@/types/api'

export interface PostListQuery {
  page?: number
  size?: number
  status?: number
  keyword?: string
}

export function listPosts(q: PostListQuery) {
  return get<PageData<Post>>('/admin/posts', q)
}

export function getPost(id: number) {
  return get<Post>(`/admin/posts/${id}`)
}

export interface PostPayload {
  title: string
  slug: string
  summary: string
  cover: string
  content: string
  type: 1 | 2 | 3
  video_url: string
  category_id: number | null
  status: 0 | 1 | 2
  is_top: boolean
  is_featured: boolean
  tag_ids: number[]
}

export function createPost(p: PostPayload) {
  return post<Post>('/admin/posts', p)
}

export function updatePost(id: number, p: PostPayload) {
  return put<null>(`/admin/posts/${id}`, p)
}

export function setPostStatus(id: number, status: 0 | 1 | 2) {
  return put<null>(`/admin/posts/${id}/status`, { status })
}

export function deletePost(id: number) {
  return del<null>(`/admin/posts/${id}`)
}

export function uploadImage(file: File) {
  const form = new FormData()
  form.append('file', file)
  return post<UploadResult>('/admin/upload', form)
}
```

- [ ] **Step 2: PostList.vue**

NDataTable（远程分页）：列 = 标题（点击进编辑）/ 类型 tag（文章/图片集/视频）/ 状态 tag（草稿灰·已发布绿·回收站红）/ 浏览/评论/赞 / 更新时间 / 操作（编辑·发布↔回收站切换·删除需 NPopconfirm）。顶部 NSelect 状态筛选 + NInput 关键词搜索（防抖 300ms）+「新建文章」按钮。**分页翻页请求竞态守卫**（同 Task 3 seq 模式）。

- [ ] **Step 3: useAutoSave.ts（防丢稿核心）**

```ts
import { onBeforeUnmount, ref } from 'vue'

export interface DraftPayload {
  title: string
  content: string
  savedAt: number
}

/** 自动保存：定时暂存 localStorage；提交失败降级；恢复时提示 */
export function useAutoSave(key: string, intervalMs = 30_000) {
  const storageKey = `draft:${key}`
  const lastSavedAt = ref<number | null>(null)
  let timer: number | null = null

  function readDraft(): DraftPayload | null {
    try {
      const raw = localStorage.getItem(storageKey)
      return raw ? (JSON.parse(raw) as DraftPayload) : null
    } catch {
      return null
    }
  }

  function writeDraft(title: string, content: string) {
    try {
      const payload: DraftPayload = { title, content, savedAt: Date.now() }
      localStorage.setItem(storageKey, JSON.stringify(payload))
      lastSavedAt.value = payload.savedAt
    } catch {
      /* localStorage 满等情况忽略 */
    }
  }

  function clearDraft() {
    localStorage.removeItem(storageKey)
    lastSavedAt.value = null
  }

  function start(getCurrent: () => { title: string; content: string }) {
    stop()
    timer = window.setInterval(() => {
      const cur = getCurrent()
      if (cur.title || cur.content) writeDraft(cur.title, cur.content)
    }, intervalMs)
  }

  function stop() {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  onBeforeUnmount(stop) // 红线：定时器必须清理

  return { readDraft, writeDraft, clearDraft, start, stop, lastSavedAt }
}
```

- [ ] **Step 4: PostEditor.vue（md-editor-v3 集成）**

要点：
1. **元信息表单**（左/上）：标题、slug（失焦校验 `^[a-z0-9-]+$`）、摘要（可空）、类型 NRadioGroup（文章/图片集/视频；选视频出现 video_url 输入，前端先校验域名白名单 player.bilibili.com|www.youtube.com|www.youtube-nocookie.com|player.youku.com）、分类 NSelect（M2 分类接口未实现——本地静态选项占位 TODO(M4)，提交时 category_id 传 null）、标签（同占位传 []）、置顶/推荐 NSwitch、封面上传（调 uploadImage 回填 cover）
2. **编辑器主体**：`<MdEditor v-model="form.content" :theme="'dark'" @on-upload-img="handleUpload" />`；`handleUpload = async (files, callback) => { ... files.forEach 上传后 callback(urls) }`（md-editor-v3 的 onUploadImg API，插入 Markdown 图片语法，用 thumb_url）
3. **自动保存**：挂载时 `readDraft()`，若存在且与服务器内容不同 → NModal 询问「检测到 X 时间前的未保存草稿，恢复/丢弃」；确认后 `start(() => ({ title: form.title, content: form.content }))`
4. **保存动作**：保存草稿（status=0）/ 发布（status=1）按钮；提交失败（含 401）→ `writeDraft` 立即兜底 + message 提示「已本地暂存」；成功 → `clearDraft()`，编辑态留在页面，新建态跳转 `:id/edit`
5. **离开守卫**：`onBeforeRouteLeave` 有未保存变更（内容与上次保存 diff）时 NDialog 确认
6. **编辑态加载**：`getPost(id)` 回填全部字段

- [ ] **Step 5: 手动验证清单**

- 新建 → 输入内容 → 30s 观察 localStorage 出现 draft:post:new
- 刷新 → 弹恢复提示 → 恢复成功
- 发布成功 → localStorage 草稿清除 → 列表出现（后端 make dev + initadmin 建号）
- 粘贴截图 → 自动上传成 Markdown 图片
- 401 场景：停后端再保存 → message 提示已本地暂存

---

### Task 5: 评论管理 + 站点设置（mock 降级页）

**Files:** `src/api/comment.ts`、`src/api/site.ts`、`src/views/admin/CommentManage.vue`、`src/views/admin/SiteSettings.vue`

- [ ] **Step 1: api 降层（真实接口 + mock 兜底）**

```ts
// api/comment.ts
import { get, put, post, del } from './http'
import type { PageData, CommentItem } from '@/types/api'

export interface CommentQuery { status?: number; page?: number; size?: number }

export async function listComments(q: CommentQuery): Promise<PageData<CommentItem>> {
  try {
    return await get<PageData<CommentItem>>('/admin/comments', q)
  } catch (e) {
    if (isNotFound(e)) return mockComments(q)
    throw e
  }
}
// isNotFound 判 axios isAxiosError && status===404；mockComments 返回本地假数据（3 条不同状态，含 emoji 内容展示净化效果）
// reviewComment/replyComment/deleteComment 同模式；M4 后端就绪后删 mock 分支
```

site.ts 同模式（`get('/site')` 404 → 返回 spec §4.8 默认值对象；`put('/admin/site')` 404 → 假成功 + console.warn TODO(M4)）。

- [ ] **Step 2: CommentManage.vue**

NTabs（待审核/已通过/已驳回）+ NDataTable：昵称/内容（截断 60 字，点开 NDrawer 看全文+原文 post）/时间/操作（通过/驳回/回复弹 NModal 文本框 500 字限制/删除 NPopconfirm）。页面顶部 NAlert：「评论接口后端将在 M4 提供，当前展示示例数据」。

- [ ] **Step 3: SiteSettings.vue**

NForm：站点名/简介/头像上传/每页条数 NInputNumber(1-50)/评论开关 NSwitch/赞赏文案/**微信+支付宝收款码上传**（调真实 `/admin/upload`，回显图片预览）/社交链接 NDynamicInput（name/url/icon 三字段行）。保存按钮调 site API（mock 降级时提示「已保存（M4 接入后端）」）。

- [ ] **Step 4: 手动验证**：两页渲染正常、mock 数据可见、收款码真实上传成功（后端开着）

---

### Task 6: 构建验证 + 收尾

**Files:** `frontend/package.json` scripts、README

- [ ] **Step 1: 类型与构建**

```bash
cd frontend
npx vue-tsc --noEmit   # strict 类型检查 0 error
npm run build          # dist 产出，检查 admin chunk 独立分离（dist/assets 看 AdminLayout-*.js）
```

- [ ] **Step 2: README.md**：启动（后端 make dev + 前端 npm run dev）/目录结构/与 M4 的接口对接清单（comments/site/settings 真实化）

- [ ] **Step 3: 全量手动回归**（后端 + 前端联跑）：登录→改密→仪表盘占位→新建文章（粘贴图）→发布→列表可见→切回收站→恢复→编辑器草稿恢复→站点设置上传收款码

---

## Self-Review 记录

- **Spec 覆盖**：§7 后台六模块（仪表盘/文章/分类标签/评论/站点/资料）——「分类标签管理页」因后端分类标签接口未实现（M1 只做了模型），M2 中以编辑器内静态占位交付，独立管理页推迟至 M4（与评论/设置接口一并真实化）。此偏离已记录。
- **占位符扫描**：Dashboard/CommentManage/SiteSettings 三处 mock 是**有意的降级设计**（后端 M4 接入），均有 NAlert 明示 + isNotFound 降级函数，非未完成 TODO。
- **类型一致性**：Post interface 与 M1 post_handler JSON 响应字段逐一核对（snake_case 保持）。
- **内存红线落实点**：useAutoSave onBeforeUnmount、http 单例、seq 竞态守卫、beforeunload 无（用 onBeforeRouteLeave 替代 SPA 内场景）。
