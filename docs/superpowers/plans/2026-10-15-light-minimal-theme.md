# 前台清新极简风改版 + 阅读元信息 + 后台浅色 + 发布前预览 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 前台从暗色科技风改为 tao.zz.ac 式清新极简浅色风，新增详情页阅读元信息行（含红底白字徽章）与顶部索引目录，后台统一浅色，编辑器增加发布前全屏预览。

**Architecture:** 核心 trick 是 `@theme` token **同名翻值**（`--color-void` 深底→白底、`--color-ink` 浅字→墨字等），118 处 `text-dim/border-line/bg-void` 工具类自动变浅，组件只删装饰类（glass/spotlight/text-gradient/渐变按钮）与改版式；HomeView/PostDetailView 结构重写；后台切 Naive UI 默认浅色主题 + 蓝色主色。后端零改动。

**Tech Stack:** Vue 3 + Tailwind v4（@theme token）+ 霞鹜文楷 CDN + Shiki(solarized-light) + Naive UI(light) + md-editor-v3。

**Spec:** `docs/superpowers/specs/2026-10-15-light-minimal-theme-design.md`

**验证手段（前端无测试框架）:** 每任务后 `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`（期望：无输出）；最终 `npm run build`。所有命令在 `frontend/` 下执行。

---

### Task 1: theme.css 浅色重写

**Files:**
- Modify: `frontend/src/styles/theme.css`（全文件重写）

- [ ] **Step 1: 重写 theme.css**

```css
/* 前台清新极简浅色主题（tao.zz.ac 风格：白底黑字、霞鹜文楷、阅读为主，零装饰）。
   霞鹜文楷走 CDN 按 unicode-range 分包，页面只加载用到的字形；失败回退系统字体。 */
@import url('https://dl.lehu.in/wk/lxgwwenkai-light.css');
@import url('https://dl.lehu.in/wk/lxgwwenkai-regular.css');
@import url('https://dl.lehu.in/wk/lxgwwenkai-bold.css');
@import '@fontsource/jetbrains-mono/400.css';
@import 'tailwindcss';

@theme {
  /* token 同名翻值：所有 text-ink/text-dim/border-line/bg-void 工具类自动切浅色 */
  --font-sans: 'LXGW WenKai', system-ui, -apple-system, 'PingFang SC', 'HarmonyOS Sans SC', 'Microsoft YaHei', sans-serif;
  --font-display: system-ui, -apple-system, 'PingFang SC', 'HarmonyOS Sans SC', 'Microsoft YaHei', sans-serif;
  --font-mono: 'JetBrains Mono', ui-monospace, SFMono-Regular, monospace;

  --color-void: #ffffff;      /* 页面底色（白） */
  --color-panel: #f6f8fa;     /* 浅灰面板（表头等） */
  --color-ink: #24292f;       /* 正文墨色 */
  --color-dim: #6a737d;       /* 次要文字 */
  --color-line: #e5e7eb;      /* 分隔线 */
  --color-accent: #268bd2;    /* Solarized Blue：链接/强调 */
  --color-accent-2: #2aa198;  /* Solarized Cyan：行内代码 */
}

html {
  color-scheme: light;
}

body {
  background: var(--color-void);
  color: var(--color-ink);
  font-family: var(--font-sans);
}

/* ===== 「创作不易请勿广告」徽章（tao #noad 样式照搬：白字红底加粗） ===== */
.noad-badge {
  color: #fff;
  background: red;
  border-radius: 0.2em;
  font-weight: bold;
  padding: 0.2em;
  opacity: 0;
  animation: noad-fade-in 2s ease-in forwards;
}

@keyframes noad-fade-in {
  to { opacity: 1; }
}

/* ===== Markdown 正文排版（spec §9：18px 文楷 / 行高 1.8 / 42em 行宽） ===== */
.md-body {
  max-width: 42em;
  margin: 0 auto;
  font-size: 1.125rem;
  line-height: 1.8;
  word-break: break-word;
}

.md-body > :first-child { margin-top: 0; }

/* 标题：系统 sans 加粗，锚点跳转黄底闪高亮 3s 渐隐（tao :target 同款） */
.md-body h1, .md-body h2, .md-body h3, .md-body h4 {
  font-family: var(--font-display);
  font-weight: 700;
  line-height: 1.35;
  margin: 1.6em 0 0.7em;
  scroll-margin-top: 24px;
}

.md-body h1 { font-size: 1.65rem; }
.md-body h2 { font-size: 1.35rem; }
.md-body h3 { font-size: 1.15rem; }
.md-body h4 { font-size: 1.02rem; }

.md-body :target {
  background-color: yellow;
  animation: restore-color 3s forwards;
}

@keyframes restore-color {
  to { background-color: inherit; }
}

.md-body p { margin: 1.1em 0; }

.md-body a {
  color: var(--color-accent);
  text-decoration: none;
  border-bottom: 1px dashed rgba(38, 139, 210, 0.45);
}

.md-body a:hover { border-bottom-style: solid; }

/* 图片：居中 + 灰色小字图注（figcaption 由 MarkdownView 渲染注入） */
.md-body img {
  max-width: 100%;
  border-radius: 4px;
  margin: 1.2em auto;
  cursor: zoom-in;
}

.md-body figure { margin: 1.2em 0; text-align: center; }

.md-body figcaption {
  color: var(--color-dim);
  font-size: 0.8rem;
  margin-top: 0.4em;
}

.md-body code {
  font-family: var(--font-mono);
  font-size: 0.88em;
}

/* 行内代码：青色文字，无底色无边框（tao 同款） */
.md-body :not(pre) > code {
  color: var(--color-accent-2);
}

/* 代码块：Solarized Light 米黄底 */
.md-body pre {
  background: #fdf6e3;
  border-radius: 4px;
  overflow-x: auto;
  margin: 1.2em 0;
}

.md-body pre code {
  display: block;
  padding: 1em 1.2em;
  font-size: 0.86rem;
  line-height: 1.7;
}

.md-body blockquote {
  margin: 1.2em 0;
  padding: 0.1em 1.2em;
  border-left: 3px solid var(--color-line);
  color: var(--color-dim);
}

.md-body ul, .md-body ol { padding-left: 1.5em; margin: 0.9em 0; }
.md-body li { margin: 0.35em 0; }
.md-body li::marker { color: var(--color-dim); }

.md-body table {
  width: 100%;
  border-collapse: collapse;
  margin: 1.2em 0;
  font-size: 0.92em;
}

.md-body th, .md-body td {
  border: 1px solid var(--color-line);
  padding: 0.5em 0.8em;
}

.md-body th { background: var(--color-panel); font-weight: 600; }

.md-body hr { border: none; border-top: 1px solid var(--color-line); margin: 2em 0; }

/* ===== View Transitions 页面切换（不支持的浏览器自动降级） ===== */
::view-transition-old(root),
::view-transition-new(root) {
  animation-duration: 0.22s;
}

/* ===== 滚动条（浅色） ===== */
::-webkit-scrollbar { width: 10px; height: 10px; }
::-webkit-scrollbar-thumb { background: rgba(0, 0, 0, 0.15); border-radius: 6px; }
::-webkit-scrollbar-thumb:hover { background: rgba(0, 0, 0, 0.25); }
::-webkit-scrollbar-track { background: transparent; }

/* ===== 动效关照 ===== */
@media (prefers-reduced-motion: reduce) {
  .noad-badge {
    animation: none;
    opacity: 1;
  }
}
```

注意：删掉的类（`aurora`/`noise`/`glass`/`spotlight`/`text-gradient`）仍被组件引用，类型检查不受影响（纯 CSS 类），后续任务逐个清理。

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出（通过）

- [ ] **Step 3: Commit**

```bash
git add frontend/src/styles/theme.css
git commit -m "feat: theme.css 重写为清新浅色主题（文楷 CDN + solarized 配色 + 徽章/锚点高亮）"
```

---

### Task 2: 字数/阅读时长工具 + PostMetaLine 组件

**Files:**
- Create: `frontend/src/utils/reading.ts`
- Create: `frontend/src/components/site/PostMetaLine.vue`

- [ ] **Step 1: 写 utils/reading.ts**

```ts
/** 阅读元信息计算：剥离 markdown 标记后按字符数统计（spec §8） */

/** 字数：去掉代码块/行内代码/图片/链接语法/markdown 符号/空白后的字符数 */
export function countChars(markdown: string): number {
  return markdown
    .replace(/```[\s\S]*?```/g, ' ') // 围栏代码块
    .replace(/`[^`\n]*`/g, ' ') // 行内代码
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ') // 图片
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1') // 链接取文字部分
    .replace(/^\s{0,3}#{1,6}\s+/gm, ' ') // 标题符号
    .replace(/[*_~>|]+/g, ' ') // 强调/引用等标记符号
    .replace(/\s+/g, '')
    .length
}

/** 预计阅读时长（分钟，1 位小数）：按 300 字/分钟 */
export function readingMinutes(chars: number): number {
  return Math.round((chars / 300) * 10) / 10
}

/** 千字显示（1 位小数），如 812 字 → 0.8 */
export function kchars(chars: number): string {
  return (chars / 1000).toFixed(1)
}
```

- [ ] **Step 2: 写 PostMetaLine.vue**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { countChars, kchars, readingMinutes } from '@/utils/reading'

/**
 * 文章元信息行（spec §8）：日期 🕰️ ⏳阅读时长(字数) [创作✍️不易👨‍🌾请勿🚫广告]
 * 前台详情页与后台发布预览共用，保证两边完全一致。
 */
const props = defineProps<{
  date: string | null
  content: string
  type?: number
  categoryName?: string | null
  categoryLink?: string
  views?: number
}>()

const typeLabel: Record<number, string> = { 1: '文章', 2: '图集', 3: '视频' }

const chars = computed(() => countChars(props.content))
const minutes = computed(() => readingMinutes(chars.value))
const kc = computed(() => kchars(chars.value))

function fmtDate(d: string | null): string {
  if (!d) return ''
  const t = new Date(d)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${t.getFullYear()}-${p(t.getMonth() + 1)}-${p(t.getDate())}`
}
</script>

<template>
  <div class="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 text-xs text-dim">
    <span v-if="type">{{ typeLabel[type] }}</span>
    <RouterLink v-if="categoryName && categoryLink" :to="categoryLink" class="text-accent hover:underline">
      {{ categoryName }}
    </RouterLink>
    <span v-else-if="categoryName">{{ categoryName }}</span>
    <span class="font-mono">{{ fmtDate(date) }}</span>
    <span aria-hidden="true">🕰️</span>
    <span>⏳{{ minutes }}分钟({{ kc }}千字)</span>
    <span v-if="views !== undefined">浏览 {{ views }}</span>
    <span class="noad-badge">创作✍️不易👨‍🌾请勿🚫广告</span>
  </div>
</template>
```

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出

- [ ] **Step 4: Commit**

```bash
git add frontend/src/utils/reading.ts frontend/src/components/site/PostMetaLine.vue
git commit -m "feat: 阅读元信息行组件（字数/时长计算 + 红底白字创作不易徽章）"
```

---

### Task 3: SiteLayout 浅色化

**Files:**
- Modify: `frontend/src/layouts/SiteLayout.vue`

- [ ] **Step 1: 重写模板与样式**

`<script setup>` 部分不变。template 整体替换：

```vue
<template>
  <header class="border-b border-dashed border-dim/50">
    <div class="mx-auto flex max-w-3xl items-center gap-3 px-4 py-5">
      <img
        v-if="site.info?.avatar"
        :src="site.info.avatar"
        alt="博主头像"
        class="h-6 w-6 rounded-full object-cover"
      />
      <RouterLink to="/" class="shrink-0 text-lg font-semibold tracking-wide">{{ site.siteName }}</RouterLink>
      <nav class="ml-auto flex items-center gap-5 text-sm">
        <RouterLink to="/" class="nav-link" :class="{ 'nav-link--active': route.path === '/' }">首页</RouterLink>
        <RouterLink to="/archive" class="nav-link" :class="{ 'nav-link--active': route.path.startsWith('/archive') }">归档</RouterLink>
        <RouterLink to="/about" class="nav-link" :class="{ 'nav-link--active': route.path.startsWith('/about') }">关于</RouterLink>
        <RouterLink to="/search" class="nav-link" :class="{ 'nav-link--active': route.path.startsWith('/search') }" aria-label="搜索">搜索</RouterLink>
      </nav>
    </div>
  </header>

  <main class="mx-auto min-h-[70vh] max-w-3xl px-4 pb-24">
    <RouterView />
  </main>

  <footer class="border-t border-line">
    <div class="mx-auto flex max-w-3xl flex-col items-center gap-3 px-4 py-8 text-xs text-dim sm:flex-row">
      <span>© {{ new Date().getFullYear() }} {{ site.siteName }} · {{ site.siteDesc }}</span>
      <nav v-if="site.info?.social_links?.length" class="flex gap-4 sm:ml-auto">
        <a
          v-for="l in site.info.social_links"
          :key="l.name"
          :href="l.url"
          target="_blank"
          rel="noopener noreferrer"
          class="transition-colors hover:text-ink"
        >{{ l.name }}</a>
      </nav>
    </div>
  </footer>
</template>
```

`<style scoped>` 的 `.nav-link--active` 颜色改为墨色：

```css
.nav-link {
  color: var(--color-dim);
  transition: color 0.2s;
}

.nav-link:hover,
.nav-link--active {
  color: var(--color-ink);
}
```

要点：删除 `<div class="aurora">`/`<div class="noise">` 两行装饰；header 去掉 `glass sticky top-0 z-40`（不吸顶）；站名去掉 `font-display text-gradient`；容器 `max-w-6xl` → `max-w-3xl`。

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add frontend/src/layouts/SiteLayout.vue
git commit -m "feat: SiteLayout 清新浅色化（虚线头部 + 窄栏 3xl + 去装饰）"
```

---

### Task 4: PostList 共用列表组件 + HomeView 重写

**Files:**
- Create: `frontend/src/components/site/PostList.vue`
- Modify: `frontend/src/views/site/HomeView.vue`（全文件重写）

- [ ] **Step 1: 写 PostList.vue（首页/分类标签/搜索/相关文章共用纯文字列表）**

```vue
<script setup lang="ts">
import type { PostCard } from '@/types/api'

/** 纯文字文章列表（tao 式）：标题 + 灰色 mono 日期；图集/视频加灰色小标记 */
defineProps<{ posts: PostCard[] }>()

const typeTag: Record<number, string> = { 2: '图集', 3: '视频' }

function fmtDate(d: string | null) {
  return d ? new Date(d).toLocaleDateString('zh-CN') : ''
}
</script>

<template>
  <ul>
    <li v-for="p in posts" :key="p.id">
      <RouterLink
        :to="`/post/${p.slug}`"
        class="group flex items-baseline gap-3 border-b border-line py-3"
      >
        <span class="truncate transition-colors group-hover:text-accent">{{ p.title }}</span>
        <span v-if="typeTag[p.type]" class="shrink-0 text-xs text-dim">{{ typeTag[p.type] }}</span>
        <span class="ml-auto shrink-0 font-mono text-xs text-dim">{{ fmtDate(p.published_at) }}</span>
      </RouterLink>
    </li>
  </ul>
</template>
```

- [ ] **Step 2: 重写 HomeView.vue**

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listPublicPosts } from '@/api/public'
import { useSiteStore } from '@/stores/site'
import PostList from '@/components/site/PostList.vue'
import type { PostCard } from '@/types/api'

const site = useSiteStore()

/* ===== 分类 Tab 移除后的纯列表 + 分页（请求序号守卫防翻页竞态） ===== */
const page = ref(1)
const feed = ref<PostCard[]>([])
const total = ref(0)
const loadingFeed = ref(false)
const feedError = ref(false)
let seq = 0

const pageSize = computed(() => site.info?.page_size ?? 10)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

async function loadFeed() {
  const current = ++seq
  loadingFeed.value = true
  feedError.value = false
  try {
    const data = await listPublicPosts({ page: page.value, size: pageSize.value })
    if (current !== seq) return // 过期响应丢弃
    feed.value = data.list
    total.value = data.total
  } catch {
    if (current === seq) feedError.value = true
  } finally {
    if (current === seq) loadingFeed.value = false
  }
}

function turnPage(p: number) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  loadFeed()
}

onMounted(() => {
  site.ensure()
  loadFeed()
})
</script>

<template>
  <!-- 极简头部：头像 + 站名 + 一行简介 -->
  <header class="flex flex-col items-center py-14 text-center">
    <img
      v-if="site.info?.avatar"
      :src="site.info.avatar"
      alt="博主头像"
      class="h-16 w-16 rounded-full object-cover"
    />
    <h1 class="mt-4 text-2xl font-bold tracking-wide">{{ site.siteName }}</h1>
    <p class="mt-2 text-sm text-dim">{{ site.siteDesc }}</p>
  </header>

  <!-- 文章列表 -->
  <div v-if="feedError" class="py-10 text-center text-sm text-dim">
    加载失败，<button class="text-accent hover:underline" @click="loadFeed">重试</button>
  </div>
  <div v-else-if="loadingFeed && !feed.length" class="py-10 text-center text-sm text-dim">加载中…</div>
  <div v-else-if="!feed.length" class="py-10 text-center text-sm text-dim">这里还没有内容</div>
  <PostList v-else :posts="feed" />

  <!-- 分页 -->
  <nav v-if="totalPages > 1" class="mt-8 flex items-center justify-center gap-4 text-sm">
    <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page <= 1" @click="turnPage(page - 1)">← 上一页</button>
    <span class="font-mono text-dim">{{ page }} / {{ totalPages }}</span>
    <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page >= totalPages" @click="turnPage(page + 1)">下一页 →</button>
  </nav>
</template>
```

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出（注意 `useTypewriter`/`DonateModal`/`ArticleCard` 的引用已随重写移除，类型不报错才对）

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/site/PostList.vue frontend/src/views/site/HomeView.vue
git commit -m "feat: 首页重写为纯文字极简列表 + PostList 共用组件"
```

---

### Task 5: MarkdownView 浅色化（solarized-light + 图片图注）

**Files:**
- Modify: `frontend/src/components/site/MarkdownView.vue`

- [ ] **Step 1: 加 figure/figcaption 渲染规则**

在 `md.renderer.rules.heading_open = ...` 之后追加（同层）：

```ts
// 有 alt 的图片包 figure + figcaption（alt 即图注，spec §9/§4b）
const defaultImage = md.renderer.rules.image!
md.renderer.rules.image = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const alt = token.content || ''
  const img = defaultImage(tokens, idx, options, env, self)
  if (!alt) return img
  return `<figure>${img}<figcaption>${md.utils.escapeHtml(alt)}</figcaption></figure>`
}
```

注意：DOMPurify 默认允许 `figure`/`figcaption` 标签，无需额外配置。

- [ ] **Step 2: Shiki 主题切换**

把 `MarkdownView.vue` 中：

```ts
const html = await codeToHtml(code, { lang, theme: 'dark-plus' })
```

改为：

```ts
const html = await codeToHtml(code, { lang, theme: 'solarized-light' })
```

同步更新文件头注释：`Shiki 异步高亮（dark-plus）` → `Shiki 异步高亮（solarized-light）`。

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/site/MarkdownView.vue
git commit -m "feat: MarkdownView 切 solarized-light 高亮 + 图片 figure 图注"
```

---

### Task 6: PostDetailView 改版

**Files:**
- Modify: `frontend/src/views/site/PostDetailView.vue`

- [ ] **Step 1: 改 script**

删除 `import Toc from '@/components/site/Toc.vue'`、`import ArticleCard from '@/components/site/ArticleCard.vue'`，改为：

```ts
import PostList from '@/components/site/PostList.vue'
import PostMetaLine from '@/components/site/PostMetaLine.vue'
```

删除 `tocDrawerOpen` ref 与 `typeLabel` 常量（后者移入 PostMetaLine）。

新增 h2 索引 computed（放在 `toc` ref 之后）：

```ts
/** 顶部索引：仅 h2（spec §10，tao 式正文目录） */
const tocH2 = computed(() => toc.value.filter((t) => t.level === 2))
```

`related` 不变（`loadRelated` 逻辑不动）。

- [ ] **Step 2: 重写 template**

`<article v-else-if="post" class="mt-10">` 整块替换：

```vue
<article v-else-if="post" class="mt-10">
  <!-- 头部：元信息行 + 标题 + 标签 -->
  <header class="text-center">
    <PostMetaLine
      :date="post.published_at"
      :content="post.content"
      :type="post.type"
      :category-name="post.category?.name ?? null"
      :category-link="post.category ? `/category/${post.category.name}` : undefined"
      :views="post.views"
    />
    <h1 class="mx-auto mt-4 max-w-3xl text-3xl font-bold leading-snug">{{ post.title }}</h1>
    <div v-if="post.tags?.length" class="mt-4 flex flex-wrap justify-center gap-2 text-xs text-dim">
      <RouterLink v-for="t in post.tags" :key="t.id" :to="`/tag/${t.name}`" class="hover:text-accent">
        # {{ t.name }}
      </RouterLink>
    </div>
  </header>

  <!-- 顶部索引（h2 目录，tao 式） -->
  <nav v-if="tocH2.length" class="mx-auto mt-8 max-w-[42em] text-sm">
    <p class="mb-2 text-xs font-semibold text-dim">索引</p>
    <ol class="space-y-1">
      <li v-for="h in tocH2" :key="h.id">
        <a :href="`#${h.id}`" class="text-accent hover:underline">{{ h.text }}</a>
      </li>
    </ol>
  </nav>

  <!-- 正文（白底直接排版，视频置顶） -->
  <div class="mt-8">
    <div v-if="post.type === 3 && post.video_url" class="mx-auto mb-8 aspect-video max-w-[42em] overflow-hidden rounded" style="border-radius: 4px">
      <iframe :src="post.video_url" class="h-full w-full" allowfullscreen title="视频" />
    </div>

    <p v-if="post.summary" class="mx-auto mb-8 max-w-[42em] border-l-2 border-line pl-4 text-dim">{{ post.summary }}</p>

    <MarkdownView :source="post.content" @toc="(items: TocItem[]) => (toc = items)" />
  </div>

  <!-- 上/下一篇（文字链接式） -->
  <nav class="mx-auto mt-12 flex max-w-[42em] justify-between gap-4 text-sm">
    <RouterLink v-if="prev" :to="`/post/${prev.slug}`" class="truncate text-dim hover:text-accent">← {{ prev.title }}</RouterLink>
    <span v-else />
    <RouterLink v-if="next" :to="`/post/${next.slug}`" class="truncate text-dim hover:text-accent">{{ next.title }} →</RouterLink>
  </nav>

  <!-- 相关文章（纯文字列表） -->
  <section v-if="related.length" class="mx-auto mt-12 max-w-[42em]">
    <h2 class="mb-3 text-sm font-semibold text-dim">相关文章</h2>
    <PostList :posts="related" />
  </section>

  <!-- 点赞 + 赞赏 -->
  <section class="mt-12 border-t border-line pt-10 text-center">
    <div class="flex justify-center">
      <LikeButton :slug="post.slug" />
    </div>
    <p class="mt-8 text-3xl">☕</p>
    <p class="mt-3 text-dim">{{ site.info?.donate_text ?? '写得不错，请我喝杯咖啡' }}</p>
    <button
      class="mt-5 rounded-full bg-accent px-6 py-2 text-sm font-medium text-white transition-opacity hover:opacity-85"
      @click="donateOpen = true"
    >赞赏支持</button>
  </section>

  <!-- 评论（comment_enabled=false 整区隐藏） -->
  <CommentSection v-if="site.info?.comment_enabled !== false" :key="post.slug" :slug="post.slug" />
</article>
```

同时删除：旧 `<header>` 元信息块、TOC 桌面侧栏与移动端抽屉两段、`<style scoped>` 里 fade 动画（抽屉已删）。

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/site/PostDetailView.vue
git commit -m "feat: 详情页改版（元信息行 + 顶部索引 + 锚点高亮 + 单栏排版）"
```

---

### Task 7: 列表/归档/搜索/关于/404 浅色化

**Files:**
- Modify: `frontend/src/views/site/ArticleListView.vue`
- Modify: `frontend/src/views/site/ArchiveView.vue`
- Modify: `frontend/src/views/site/SearchView.vue`
- Modify: `frontend/src/views/site/AboutView.vue`
- Modify: `frontend/src/views/site/NotFoundView.vue`

- [ ] **Step 1: ArticleListView.vue**

script：`import ArticleCard` 改为 `import PostList from '@/components/site/PostList.vue'`。template 替换：

```vue
<template>
  <header class="mt-12 text-center">
    <p class="text-xs text-dim">{{ mode === 'category' ? '分类' : '标签' }}</p>
    <h1 class="mt-2 text-2xl font-bold">{{ name }}</h1>
    <p class="mt-2 text-xs text-dim">共 {{ total }} 篇</p>
  </header>

  <div v-if="failed" class="mt-8 py-10 text-center text-sm text-dim">
    加载失败，<button class="text-accent hover:underline" @click="load">重试</button>
  </div>
  <div v-else-if="loading && !list.length" class="mt-8 py-10 text-center text-sm text-dim">加载中…</div>
  <div v-else-if="!list.length" class="mt-8 py-10 text-center text-sm text-dim">
    该{{ mode === 'category' ? '分类' : '标签' }}下暂无内容
    <RouterLink to="/" class="ml-2 text-accent hover:underline">回首页 →</RouterLink>
  </div>
  <div v-else class="mt-8">
    <PostList :posts="list" />
  </div>

  <nav v-if="totalPages > 1" class="mt-8 flex items-center justify-center gap-4 text-sm">
    <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page <= 1" @click="turnPage(page - 1)">← 上一页</button>
    <span class="font-mono text-dim">{{ page }} / {{ totalPages }}</span>
    <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page >= totalPages" @click="turnPage(page + 1)">下一页 →</button>
  </nav>
</template>
```

- [ ] **Step 2: ArchiveView.vue**

template 三处 `glass ... rounded-2xl` 提示块改为 `py-10 text-center`（同上风格，去 glass）；h1 去 `text-gradient`；分组标题 `bg-void/90` 保持（void 已是白）；`hover:bg-white/5` → `hover:bg-black/[0.03]`。`<style scoped>` 时间线节点渐变改纯色：

```css
.archive-timeline > div::before {
  content: '';
  position: absolute;
  left: -4px;
  top: 8px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-accent);
}
```

- [ ] **Step 3: SearchView.vue**

script：`import ArticleCard` 改 `import PostList from '@/components/site/PostList.vue'`。template：

```vue
<template>
  <header class="mx-auto mt-14 max-w-xl text-center">
    <h1 class="text-2xl font-bold">搜索</h1>
    <div class="mt-6 flex items-center gap-2 rounded-full border border-line px-5 py-3">
      <span class="text-dim">🔍</span>
      <input
        v-model="q"
        type="search"
        placeholder="标题 / 摘要 / 标签…"
        class="w-full bg-transparent text-sm outline-none placeholder:text-dim/60"
        @input="onInput"
      />
    </div>
  </header>

  <div class="mt-10">
    <p v-if="searched && q" class="mb-5 text-center text-xs text-dim">找到 {{ total }} 篇相关内容</p>
    <div v-if="loading" class="py-10 text-center text-sm text-dim">搜索中…</div>
    <div v-else-if="searched && !list.length" class="py-10 text-center text-sm text-dim">
      没有找到与「{{ q }}」相关的内容
    </div>
    <PostList v-else-if="list.length" :posts="list" />
    <p v-else class="py-10 text-center text-sm text-dim">输入关键词，回车或稍候开始搜索</p>
  </div>
</template>
```

- [ ] **Step 4: AboutView.vue**

```vue
<template>
  <section class="mx-auto mt-16 max-w-2xl text-center">
    <img
      v-if="site.info?.avatar"
      :src="site.info.avatar"
      alt="头像"
      class="mx-auto h-24 w-24 rounded-full object-cover"
    />
    <div v-else class="mx-auto flex h-24 w-24 items-center justify-center rounded-full border border-line text-4xl">👤</div>

    <h1 class="mt-6 text-2xl font-bold">{{ site.siteName }}</h1>
    <p class="mt-4 leading-loose text-dim">{{ site.siteDesc }}</p>

    <nav v-if="site.info?.social_links?.length" class="mt-8 flex flex-wrap justify-center gap-3">
      <a
        v-for="l in site.info.social_links"
        :key="l.name"
        :href="l.url"
        target="_blank"
        rel="noopener noreferrer"
        class="rounded-full border border-line px-5 py-2 text-sm transition-colors hover:border-accent hover:text-accent"
      >{{ l.name }}</a>
    </nav>
  </section>

  <!-- 固定赞赏区 -->
  <section class="mx-auto mt-16 max-w-2xl border-t border-line pt-10 text-center">
    <p class="text-4xl">☕</p>
    <h2 class="mt-4 font-semibold">爱心捐赠</h2>
    <p class="mt-2 text-sm text-dim">{{ site.info?.donate_text ?? '写得不错，请我喝杯咖啡' }}</p>
    <button
      class="mt-6 rounded-full bg-accent px-6 py-2 text-sm font-medium text-white transition-opacity hover:opacity-85"
      @click="donateOpen = true"
    >打开收款码</button>
  </section>

  <DonateModal v-model="donateOpen" />
</template>
```

- [ ] **Step 5: NotFoundView.vue**

```vue
<template>
  <main class="flex min-h-screen flex-col items-center justify-center bg-void px-4 text-center">
    <p class="text-8xl font-bold">404</p>
    <p class="mt-6 text-dim">这个页面去银河系旅行了</p>
    <div class="mt-8 flex gap-4">
      <RouterLink
        to="/"
        class="rounded-full bg-accent px-6 py-2 text-sm font-medium text-white transition-opacity hover:opacity-85"
      >回首页</RouterLink>
      <RouterLink to="/archive" class="rounded-full border border-line px-6 py-2 text-sm transition-colors hover:text-accent">看归档</RouterLink>
    </div>
  </main>
</template>
```

（`aurora`/`noise` 装饰行删除；script 不变。）

- [ ] **Step 6: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/site/
git commit -m "feat: 列表/归档/搜索/关于/404 页面浅色极简化"
```

---

### Task 8: 评论/赞赏组件浅色化

**Files:**
- Modify: `frontend/src/components/site/CommentSection.vue`
- Modify: `frontend/src/components/site/DonateModal.vue`

（LikeButton 全部用 token 类，翻值后自动适配，无需改动；Lightbox 深色遮罩为看图标准交互，保持不变。）

- [ ] **Step 1: CommentSection.vue 样式替换**

区块容器：`<section class="glass mt-8 rounded-2xl p-6 sm:p-8">` → `<section class="mt-12 border-t border-line pt-8">`；标题 `font-display ... uppercase tracking-widest` 简化为 `mb-6 text-sm font-semibold text-dim`。

两个输入框与 textarea 的 `bg-white/5` → `bg-white`，其余（border-line/focus:border-accent/50）不变。

提交按钮：

```vue
<button
  type="submit"
  class="rounded-full bg-accent px-6 py-2 text-sm font-medium text-white transition-opacity hover:opacity-85 disabled:opacity-50"
  :disabled="submitting"
>{{ submitting ? '提交中…' : '发表评论' }}</button>
```

（script 逻辑零改动。）

- [ ] **Step 2: DonateModal.vue 样式替换**

遮罩 `bg-black/70` → `bg-black/40`；卡片 `glass w-full max-w-sm rounded-2xl p-6` → `w-full max-w-sm rounded-lg border border-line bg-white p-6 shadow-lg`。其余 token 类不动。

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/site/CommentSection.vue frontend/src/components/site/DonateModal.vue
git commit -m "feat: 评论/赞赏组件浅色化"
```

---

### Task 9: 删除废弃文件

**Files:**
- Delete: `frontend/src/components/site/ArticleCard.vue`
- Delete: `frontend/src/components/site/Toc.vue`
- Delete: `frontend/src/composables/useTypewriter.ts`

- [ ] **Step 1: 确认无引用后删除**

Run: `cd frontend && grep -rn "ArticleCard\|Toc.vue\|useTypewriter" src/ --include="*.vue" --include="*.ts"`
Expected: 无输出（有输出则说明 Task 4/6/7 有遗漏，先补删引用）

```bash
git rm frontend/src/components/site/ArticleCard.vue frontend/src/components/site/Toc.vue frontend/src/composables/useTypewriter.ts
```

注意：`PostDetailView.vue` 里 `import type { TocItem } from '@/components/site/MarkdownView.vue'` 仍有效（TocItem 定义在 MarkdownView，不是 Toc.vue），不受影响。

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git commit -m "chore: 删除废弃的 ArticleCard/Toc/useTypewriter"
```

---

### Task 10: 后台浅色 + 发布前预览

**Files:**
- Modify: `frontend/src/App.vue`
- Create: `frontend/src/components/admin/PostPreviewModal.vue`
- Modify: `frontend/src/views/admin/PostEditor.vue`

- [ ] **Step 1: App.vue 切浅色主题**

```vue
<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NDialogProvider, zhCN, dateZhCN } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'

/* 后台浅色主题：主色与前台强调色同源（Solarized Blue #268bd2） */
const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#268bd2',
    primaryColorHover: '#4da3e0',
    primaryColorPressed: '#1e6fa8',
    primaryColorSuppl: '#268bd2',
  },
}
</script>

<template>
  <n-config-provider :theme-overrides="themeOverrides" :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
```

- [ ] **Step 2: 写 PostPreviewModal.vue**

```vue
<script setup lang="ts">
import { NButton, NModal } from 'naive-ui'
import MarkdownView from '@/components/site/MarkdownView.vue'
import PostMetaLine from '@/components/site/PostMetaLine.vue'

/**
 * 发布前全屏预览（spec §12）：复刻前台详情页最终形态——
 * 标题 + 元信息行（实时字数/时长）+ 标签 + 视频/正文，所见即所得。
 */
defineProps<{
  show: boolean
  title: string
  content: string
  type: 1 | 2 | 3
  videoUrl: string
  categoryName: string | null
  tags: string[]
}>()

const emit = defineEmits<{ (e: 'update:show', v: boolean): void; (e: 'confirm'): void }>()

function close() {
  emit('update:show', false)
}
</script>

<template>
  <NModal :show="show" @update:show="(v: boolean) => emit('update:show', v)">
    <div class="fixed inset-0 z-50 overflow-y-auto bg-white">
      <div class="mx-auto max-w-3xl px-4 pb-16 pt-10">
        <header class="text-center">
          <PostMetaLine
            :date="new Date().toISOString()"
            :content="content"
            :type="type"
            :category-name="categoryName"
          />
          <h1 class="mx-auto mt-4 max-w-3xl text-3xl font-bold leading-snug">{{ title || '（无标题）' }}</h1>
          <div v-if="tags.length" class="mt-4 flex flex-wrap justify-center gap-2 text-xs text-dim">
            <span v-for="t in tags" :key="t"># {{ t }}</span>
          </div>
        </header>

        <div class="mt-8">
          <div v-if="type === 3 && videoUrl" class="mx-auto mb-8 aspect-video max-w-[42em] overflow-hidden" style="border-radius: 4px">
            <iframe :src="videoUrl" class="h-full w-full" allowfullscreen title="视频预览" />
          </div>
          <MarkdownView :source="content" />
        </div>

        <div class="mt-12 flex justify-center gap-4 border-t border-line pt-8">
          <NButton size="large" @click="close">返回修改</NButton>
          <NButton size="large" type="primary" @click="emit('confirm')">确认发布</NButton>
        </div>
      </div>
    </div>
  </NModal>
</template>
```

- [ ] **Step 3: PostEditor.vue 接预览**

script 增加：

```ts
import PostPreviewModal from '@/components/admin/PostPreviewModal.vue'

/* ---------- 发布前预览（spec §12） ---------- */
const previewOpen = ref(false)

/** 预览用的标签名（selectedTagIds → tagOptions 反查） */
const previewTags = computed(() =>
  selectedTagIds.value
    .map((id) => tagOptions.value.find((o) => o.value === id)?.label)
    .filter((n): n is string => Boolean(n)),
)

const previewCategory = computed(
  () => categoryOptions.value.find((o) => o.value === form.category_id)?.label ?? null,
)

/** 预览前走一遍与 save 相同的校验，拦截明显错误 */
function openPreview() {
  if (!form.title.trim()) {
    message.error('标题不能为空')
    return
  }
  validateSlug()
  validateVideoUrl()
  if (slugError.value || videoUrlError.value) {
    message.error('请修正表单错误后再发布')
    return
  }
  previewOpen.value = true
}
```

「发布」按钮 `@click="save(1)"` 改为 `@click="openPreview"`；模板末尾 `</div>` 前加：

```vue
<PostPreviewModal
  v-model:show="previewOpen"
  :title="form.title"
  :content="form.content"
  :type="form.type"
  :video-url="form.video_url"
  :category-name="previewCategory"
  :tags="previewTags"
  @confirm="save(1); previewOpen = false"
/>
```

另两处浅色适配：`<MdEditor ... theme="dark"` → `theme="light"`；`<style scoped>` 的 `.meta-grid` 边框 `rgba(255, 255, 255, 0.09)` → `rgba(0, 0, 0, 0.09)`。

- [ ] **Step 4: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json`
Expected: 无输出

- [ ] **Step 5: Commit**

```bash
git add frontend/src/App.vue frontend/src/components/admin/PostPreviewModal.vue frontend/src/views/admin/PostEditor.vue
git commit -m "feat: 后台浅色统一 + 发布前全屏预览确认"
```

---

### Task 11: 全量验证 + 构建残留扫描 + 文档更新

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: 类型检查 + 构建**

Run:
```bash
cd frontend && npx vue-tsc --noEmit -p tsconfig.app.json
npm run build
```
Expected: 类型检查无输出；build 成功产出 `dist/`。

- [ ] **Step 2: 残留扫描**

Run: `cd frontend && grep -rn "glass\|spotlight\|text-gradient\|aurora\|noise" src/ --include="*.vue" --include="*.css"`
Expected: 无输出（若命中 `.md-body` 无关单词误报可忽略，人工判断）

Run: `cd frontend && grep -rn "bg-white/5\|from-accent\|dark-plus" src/`
Expected: 无输出

- [ ] **Step 3: 本地目检（需后端）**

```bash
cd backend && make dev      # 终端 1
cd frontend && npm run dev  # 终端 2，开 http://localhost:5173
```

检查清单：
- 首页：白底、文楷、纯文字列表 + 灰色 mono 日期、分页可用
- 详情页：元信息行（日期/🕰️/⏳时长(千字)/红底白字徽章 2s 淡入）、顶部索引点击后目标 h2 黄底 3s 渐隐、代码块米黄底、行内代码青色、图片居中带图注、表格浅灰表头、点赞/评论/赞赏正常
- 归档/搜索/分类/标签/关于/404：全部白底浅色
- 后台 `/admin/login` 起：Naive UI 浅色 + 蓝色主色；编辑器浅色；「发布」弹全屏预览（元信息实时计算），「确认发布」成功、「存草稿」不弹预览
- 窄屏（≤600px）：头部/列表/正文不破版

- [ ] **Step 4: 更新 CLAUDE.md**

把「单 SPA 双路由区：`/`（前台 SiteLayout，暗色科技风）…样式：Tailwind v4 + `src/styles/theme.css`（token/aurora/噪点/聚光/`.md-body` 排版）」更新为浅色描述：

```
- 单 SPA 双路由区：`/`（前台 SiteLayout，清新极简浅色风：白底、霞鹜文楷 CDN、窄栏 3xl、纯文字列表）与 `/admin/*`（路由守卫查 Pinia admin store，Naive UI 浅色主题、主色 #268bd2），全部懒加载独立 chunk。路径别名 `@` → `src/`。样式：Tailwind v4 + `src/styles/theme.css`（@theme token 同名翻值即全站换肤；`.md-body` 排版、`.noad-badge` 红徽章、`:target` 锚点高亮）。
```

「UI 组件双轨」一行的后台描述保持；「Markdown 渲染管线」中 `Shiki 异步高亮（dark-plus）` 无需改（CLAUDE.md 未写死主题）；「UI 组件双轨：后台 `/admin` 用 Naive UI + md-editor-v3 编辑器；前台展示站不用组件库」保持有效。

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: CLAUDE.md 前台风格描述更新为清新浅色"
```

---

## Self-Review 记录

- **Spec 覆盖**：§4 token→Task 1；§5 布局→Task 3；§6 首页→Task 4；§7 详情页→Task 6/8；§8 元信息行→Task 2；§9 排版→Task 1/5；§10 其余页面→Task 7；§11 后台→Task 10；§12 预览→Task 10；§13 管线→Task 5（顺序未动）；§14 验证→各任务 + Task 11。无缺口。
- **占位符**：无 TBD/TODO；所有代码块完整。
- **类型一致性**：`PostMetaLine` 的 props（date/content/type/categoryName/categoryLink/views）在 Task 6（详情页传全量）与 Task 10（预览传子集，categoryLink 不传则渲染纯文本）用法一致；`PostList` 在 Task 4/6/7 引用一致；`TocItem` 仍从 MarkdownView 导出。
