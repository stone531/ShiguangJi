# 时光集前台清新极简风改版 + 阅读元信息 + 后台统一浅色 + 发布前预览 设计文档

日期：2026-10-15
状态：已评审（用户逐节确认）

## 1. 背景与目标

线上 http://me.yiqiedu.com/ 主页为暗色科技风（深底 + 极光 + 玻璃卡片 Bento），观感偏黑。用户希望改为 https://tao.zz.ac/ 那种**清新极简**风格：白底黑字、纯文字列表、霞鹜文楷、阅读为主、几乎零装饰。

同时新增/调整：

- 文章详情页标题下加元信息行：日期 + ⏳预计阅读时长（字数）+「创作✍️不易👨‍🌾请勿🚫广告」红底白字徽章（tao `#noad` 样式照搬）。
- 目录（TOC）改为 tao 式：正文顶部纯链接索引 + `:target` 黄色闪高亮 3s 渐隐，删除右侧悬浮 TOC 与移动端抽屉。
- 后台 `/admin` 统一浅色风格（Naive UI 默认浅色主题 + 前台同款蓝色主色）。
- 编辑器「发布」改为先开全屏预览（渲染成与前台详情页一致的最终页面），确认后才真正发布。

## 2. 范围

- 前台 `src/layouts/SiteLayout.vue`、`src/views/site/*`、`src/components/site/*` 全量改版。
- 后台 `src/App.vue`（Naive UI 主题切换）+ 各 admin 页面目检适配。
- `src/views/admin/PostEditor.vue` 增加发布前预览。
- **后端零改动**（阅读时长/字数纯前端计算；内容接口已返回所需字段）。
- **不做**（列入"后面慢慢优化"）：暗色模式/主题切换、GitHub Alerts 提示块、markdown 脚注、md-editor-v3 内置预览样式定制。

## 3. 用户已确认的决策

| 决策点 | 结论 |
|---|---|
| 首页版式 | 完全极简列表（标题 + 日期纯文字时间线），移除 Hero 打字机/Bento/分类 Tab |
| 元信息行位置 | 仅文章详情页标题下；首页列表保持「标题+日期」 |
| 中文字体 | 霞鹜文楷 CDN（tao 同源 dl.lehu.in，unicode-range 分包按需加载，失败回退系统字体） |
| 暗色模式 | 纯浅色，不做切换 |
| 后台范围 | 一并改为浅色，风格统一（配色统一，后台字体仍用系统黑体） |
| 发布交互 | 发布前必须全屏预览确认；存草稿不弹预览 |
| 索引（TOC） | tao 式正文顶部索引 + 锚点黄色高亮渐隐 |
| 红色徽章 | 「创作✍️不易👨‍🌾请勿🚫广告」白字红底加粗圆角徽章，含 2s 淡入动画，照搬 `#noad` |

## 4. 主题 token（重写 `src/styles/theme.css`）

- `color-scheme: light`；底色纯白 `#fff`；正文墨色 `#24292f`；次要文字 `#6a737d`；分隔线 `#e5e7eb`。
- 强调色/链接：Solarized Blue `#268bd2`；行内代码 Solarized Cyan `#2aa198`。
- 字体：
  - 正文：`'LXGW WenKai', system-ui, ...`（CDN `@import` light/regular/bold 三个字重 + mono 变体按需）。
  - 标题：系统 sans-serif 栈（tao 同款做法）。
  - 日期/数字：保留 JetBrains Mono（灰色 mono 日期是列表辨识度核心）。
- 删除效果类：`aurora`、`noise`、`glass`、`spotlight`、`text-gradient`、渐变按钮；全部引用点同步清理。
- 滚动条改浅色；View Transitions 保留；`prefers-reduced-motion` 保留（徽章淡入遵守）。

## 5. 布局（SiteLayout）

- Header：白底，底部 1px 灰色**虚线**（tao 标志性细节）；左 = 小圆头像 + 站名，右 = 导航（首页/归档/关于/搜索，路由不变）；**不做吸顶**。
- 主容器：`max-w-3xl`（~768px）窄栏阅读版式。
- Footer：居中灰色小字 + 社交链接。
- 移除模板中的 `aurora`/`noise` 两行装饰节点。

## 6. 首页（HomeView 重写）

- 顶部：头像 + 站名 + 一行简介，虚线分隔（极简版 Hero；删除打字机与按钮）。
- 主体：纯文字列表，每行「标题（黑字，hover 蓝）+ 右侧灰色 mono 日期」；type≠1（图片集/视频）在日期左侧加灰色小字「图集」/「视频」标记。
- 底部：上一页/下一页文字分页。
- 移除：Bento 六宫格（关于我/精选/标签云/赞赏/最新/统计）、分类 Tab、DonateModal 引用。
- `composables/useTypewriter.ts`、`components/site/ArticleCard.vue` 删除（PostDetailView 的"相关文章"同改纯文字列表）。

## 7. 详情页（PostDetailView）

- 头部居中：元信息行（见 §8）+ 大标题 + 标签行小字。
- 正文容器去掉玻璃卡片边框背景，白底直接排版；视频 16:9 头部/图集封面保留，圆角收小 4px。
- 目录：正文前「索引」块（由 MarkdownView 抛出的 toc 项过滤 h2 渲染链接列表）；点击跳转目标 h2 黄底高亮 3s 渐隐（`:target` + keyframes）；删除悬浮 TOC 侧栏、移动端抽屉、悬浮按钮与 `components/site/Toc.vue`。
- 相关文章：纯文字列表（标题 + 日期）。
- 点赞/赞赏区：浅色化（白底细边框卡片或无框居中），LikeButton/DonateModal 浅色化。
- 上下篇：文字链接式「← 上一篇标题」/「下一篇标题 →」。
- CommentSection 浅色化，逻辑不动。

## 8. 元信息行（新共用组件 `components/site/PostMetaLine.vue`）

格式（灰色小字一行）：

```
{yyyy-MM-dd} 🕰️ ⏳{N.N}分钟({K.K}千字) [创作✍️不易👨‍🌾请勿🚫广告]
```

- 日期取 `published_at`；🕰️ 为时光集版 tao 的 🕸️ 站标。
- 字数：`content` 去除 markdown 标记/代码块/HTML 标签/空白后的字符数；时长 = 字数 ÷ 300 字/分钟，1 位小数。
- 徽章：白字、红底（tao 原版正红）、`font-weight: bold`、`border-radius: 0.2em`、`padding: 0.2em`；加载后 2s `ease-in` 淡入，之后常驻；淡入遵守 `prefers-reduced-motion`。
- 类型（文章/图集/视频）、分类链接、浏览量合并进本行；标签单独一行放标题下。
- 前台详情页与后台发布预览**共用**此组件与字数/时长计算函数（计算函数放 `src/utils/reading.ts`，组件 `PostMetaLine.vue` 内部调用）。

## 9. 文章排版规格（`.md-body` 修订）

| 项 | 新版 |
|---|---|
| 正文字号 | 1.125rem (18px)，移动端不缩小 |
| 正文字体 | 霞鹜文楷（light/regular 分粗细） |
| 行高 | 1.8 |
| 段落间距 | 1.1em |
| 行宽 | 42em（相对字号自适应） |
| 标题 | 系统 sans 加粗，h2 去掉底部横线 |
| 引用块 | 灰色实线左边框 + 灰字，无底色 |
| 表格 | 白底 + `#e5e7eb` 边框，表头 `#f6f8fa` 浅灰底 |
| 图片 | 居中、圆角 4px、灰色小字图注（figcaption 取 alt 文本） |
| 代码块 | `#fdf6e3` 米黄底 + solarized-light，4px 圆角 |
| 行内代码 | 青色 `#2aa198` 文字，无底色无边框 |
| 链接 | `#268bd2`，hover 实线下划 |
| 加粗 | 文楷 700 字重，墨色 |
| 斜体/删除线 | 浏览器默认（删除线灰字） |
| 列表 | 1.5em 缩进，marker 灰色 |
| 分隔线 | 浅灰细线 |

列表/归档/关于等正文外页面统一同套参数（字号维持 1rem）。旧文章暗底截图不做滤镜处理，自然过渡。

## 10. 其余前台页面

- 归档页：年份分组纯文字列表，浅色化。
- 搜索页：浅色化，结果同首页列表样式。
- 分类/标签列表页（ArticleListView）：复用首页列表样式，顶部显示分类名/标签名。
- 关于页：`.md-body` 排版。
- 404：浅色化。
- Lightbox：弹层改白底浅色。

## 11. 后台浅色统一

- `App.vue`：`darkTheme` → 移除（用默认浅色）；`themeOverrides.primaryColor = #268bd2`（含 hover/pressed/磁贴色，用 Naive UI 的 `primaryColorHover` 等字段）。
- 后台界面字体保持系统黑体（工作界面可读性优先）。
- 各 admin 页面为纯 Naive UI 组件，切主题后自动适配；目检少量自定义 CSS（PostEditor 的 `save-hint`/`cover-preview` 等灰字颜色）。
- 全局 body 白底，后台不再需要深色兜底。

## 12. 发布前预览（PostEditor）

- 「发布」按钮 → 打开全屏预览弹层（NModal 或全屏 Teleport）。
- 预览内容 = 前台详情页最终形态：居中标题 + 元信息行（PostMetaLine，实时计算字数/时长）+ 标签 + 视频 16:9/图集封面 + 正文（复用 `MarkdownView`，solarized-light）。
- 底部按钮：「确认发布」（执行现有 `save(1)` 逻辑后关闭）、「返回修改」。
- 「存草稿」不弹预览，行为不变。
- md-editor-v3 自带分屏预览保留不动。

## 13. Markdown 渲染管线约束（不变）

markdown-it（`html:false`）→ DOMPurify → v-html → Shiki 异步高亮（主题由 `dark-plus` 改 `solarized-light`，`MarkdownView.vue` 一处）。管线顺序不得绕过净化。

## 14. 验证

- `npx vue-tsc --noEmit -p tsconfig.app.json` 类型检查通过。
- `npm run build` 通过。
- 本地 `npm run dev` + 后端 `make dev` 目检：首页/详情页（元信息行、徽章淡入、索引跳转高亮、代码块/表格/图片排版）/归档/搜索/分类/标签/关于/404/移动端宽度；后台登录/仪表盘/文章管理/编辑器预览发布/评论/设置全流程浅色正常。
- 验证发布预览与线上详情页元信息一致（同组件同函数）。
