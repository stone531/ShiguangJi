# 个人博客 M4 互动与上线 — 实施计划

> **For agentic workers:** 按任务顺序实施，步骤用 checkbox（`- [ ]`）跟踪。

**Goal:** 评论/点赞闭环（四层防垃圾 + §5.5 内容净化）、后台遗留接口真实化（评论管理/站点设置/分类标签/统计/个人资料）、前台与后台去 mock、SEO meta 注入、安全加固、部署脚本。

**Architecture:** 复用既有分层（handler → service → model）与 IPLimiter（键值化复用做评论双窗口限流）。敏感词用手写 trie（DFA），匹配前归一化对抗变形。SEO 注入：启动缓存 `dist/index.html`，NoRoute 对 `/post/:slug` 注入 meta 后返回。

**Spec:** `§5.1/§5.2 API`、`§5.4 防垃圾四层`、`§5.5 评论净化`、`§4.6/§4.7 评论点赞表`、`§8 安全`、`§8.1 SEO`、`§10 部署`

---

## 与 spec 的偏离（先行声明）

1. **敏感词表**存 `site_settings.sensitive_words`（spec §4.8 原设计），**每次发表评论时现查现建 trie**：个人博客量级下单行 SELECT 成本可忽略，换取「后台改词表即时生效」，不做缓存失效管理。
2. **限流器内存实现**（spec §5.4 同款）：IPLimiter 复用两实例——`ip:postID` 1 条/分钟、`ip` 10 条/天；单机部署语义不变。
3. **Honeypot 字段名**用 `website`（隐藏输入框）。
4. **链接数 >2 自动驳回**并入 40006 违规拒绝。
5. **标签合并**实现为 `POST /admin/tags/:id/merge {target_id}`（移动 post_tags 后删除源标签）。
6. **SameSite=Strict 仅生产**（IsProd），开发保留 Lax 便于跨端调试本地。
7. **部署脚本只交付文件**（nginx.conf/systemd/deploy.sh/backup.sh），不在本环境实际部署验证。

## API 契约（M4 新增）

```
# 评论（公开）
POST /api/posts/:slug/comments      {nickname, email?, content, website(蜜罐)}  → 先审后显；蜜罐静默假成功
GET  /api/posts/:slug/comments?page= → {list:[{id,nickname,content,is_admin,created_at,children:[...]}], total, page, size}  # 仅 status=1 两级树

# 点赞（公开，指纹=SHA256(IP+UA+LIKE_SALT)）
GET  /api/posts/:slug/like          → {liked, like_count}
POST /api/posts/:slug/like          → {liked, like_count}    # 切换

# 评论管理（admin）
GET    /api/admin/comments?status=&page=&size= → {list:[AdminCommentVO(含 email/post_title)], total, page, size}
PUT    /api/admin/comments/:id/review  {status:1|2}
POST   /api/admin/comments/:id/reply   {content}              # is_admin=1 自动通过
DELETE /api/admin/comments/:id                                  # 软删

# 站点/资料/统计（admin）
GET/PUT /api/admin/site             # PUT 含 sensitive_words；公开 /site 永不含该键
GET/PUT /api/admin/profile          {nickname, avatar}
GET    /api/admin/stats             → {post_total, post_published, comment_total, comment_pending, view_total, like_total}

# 分类标签（admin）
CRUD /api/admin/categories|tags     # 分类删除：其下有文章（含回收站）→ 40001 拒绝
POST   /api/admin/tags/:id/merge    {target_id}

# 安全：所有 /api 写操作（含公开评论/点赞 POST）要求头 X-Requested-With: XMLHttpRequest
# SEO：GET /post/:slug（非 /api）→ 注入 <title>/meta description/OG 后返回 index.html
```

错误码新增：`40006 评论包含违规内容，请修改`（敏感词/超链接>2/长度越界均业务参数类，长度用 40001）。

---

### Task 1: 评论域（service + handler + 测试）

- [x] `migrations/002_sensitive_words.sql`：INSERT IGNORE 内置初始词表（少量示例，后台可扩展）
- [x] `service/sensitive.go`：归一化（全角→半角/大写→小写/剔除空白与符号）+ trie 匹配
- [x] `service/comment_service.go`：净化（零宽/控制字符剔除、连续换行≤2、长度 rune 校验）→ 敏感词 → 链接数 → 限流（1/min·post、10/day·IP）→ status=0 落库；公开两级树列表；后台列表（联文章标题）；审核/回复（is_admin 自动通过）/软删；comment_count 重算同步
- [x] handler + 路由：公开组挂 XHR 校验；蜜罐假成功
- [x] 测试：净化边界、变形敏感词（全角/插符号）、限流、树结构、审核计数同步、回复挂到顶级

### Task 2: 点赞域

- [x] `service/like_service.go`：指纹、事务切换（INSERT/DELETE + like_count ±1）、唯一冲突幂等
- [x] handler + 路由；测试：切换往返、并发幂等语义、未发布 404

### Task 3: 后台接口补全

- [x] SiteService.AdminGet/Save（含 sensitive_words，tx upsert）
- [x] AuthService.GetProfile/UpdateProfile
- [x] StatsService（六指标，待审数供仪表盘提醒）
- [x] TaxonomyService 增删改 + 分类删除守护 + 标签合并
- [x] handler/路由/测试

### Task 4: 安全加固 + SEO 注入

- [x] `middleware/xhr.go`：写方法校验 X-Requested-With；挂 admin 保护组 + 公开评论/点赞 POST
- [x] SameSite：prod=Strict（login + 滑动续期两处）
- [x] `router/static.go` 扩展：启动缓存 dist/index.html（cfg.WebDist）；NoRoute——`/api/*` JSON 404；`/post/:slug` 查已发布文章注入 title/description/og:image；其余路径回 index.html（SPA fallback）；dist 不存在时维持 JSON 404（开发态）
- [x] 测试：注入正确性、未发布/不存在回退原样

### Task 5: 前端互动

- [x] `http.ts` 默认头 X-Requested-With
- [x] `components/site/LikeButton.vue`：挂载查询、点击切换、动效
- [x] `components/site/CommentSection.vue`：树列表（博主徽章）、表单（昵称/邮箱选填/内容 500 字/蜜罐隐藏域）、成功提示「待审核」；comment_enabled=false 隐藏；接入 PostDetailView 替换占位

### Task 6: 后台真实化

- [x] comment.ts/site.ts 去 mock 与 isNotFound 分支；CommentManage 去 NAlert；SiteSettings 去 isMock + 增敏感词编辑
- [x] Dashboard 六指标真实化 + 待审评论 NAlert 跳转
- [x] PostEditor：分类/标签 NSelect 接接口，标签支持即时新建（POST /admin/tags）
- [x] 新增 Profile 页（昵称/头像上传/改密入口）+ 路由 + 菜单
- [x] 类型补齐：AdminCommentVO/SiteAdminVO/StatsVO/CategoryFull/TagFull

### Task 7: 部署文件（只交付不执行）

- [x] `deploy/nginx.conf`：80→443 跳转、SPA 静态、/uploads 禁执行、/api 与 /post/ 反代 :8080、安全头、gzip
- [x] `deploy/blog.service`：systemd 单元（EnvironmentFile）
- [x] `deploy/deploy.sh`：前端 build → go 交叉编译 linux/amd64 → scp（binary/dist/migrations）→ systemctl restart
- [x] `deploy/backup.sh`：mysqldump 滚动 7 天 + uploads 增量打包

### Task 8: 验证与收尾

- [x] `make test`/`make lint`/`make fmt` 全绿；`vue-tsc` 0 error；`npm run build`
- [x] 后端 README curl 冒烟补 X-Requested-With；根 README 里程碑勾 M4；CLAUDE.md 更新；本计划 Self-Review

---

## Self-Review 记录

**完成情况（2026-08-22）**：8 个任务全部完成。验证结果：`make test`（-race）全绿、`make lint`/`make fmt` 通过、`vue-tsc` 0 error、`npm run build` 成功。

**Code Review 修复（同日）**：① main.go 补 `SetTrustedProxies(["127.0.0.1","::1"])`（默认信任全代理时 XFF 可伪造，限流/点赞指纹失效）；② 净化与敏感词归一化剔除 Unicode Cf 格式字符（U+2060/U+00AD 等插字绕过已实证并补测试）；③ LikeService.Toggle 并发幂等（插入冲突视为已赞、删除零行跳过减计数，测试加 count==rows 不变量）；④ nginx.conf 修复 add_header 继承陷阱（三个带 Cache-Control 的 location 重复安全头，/assets/ 去掉 expires 避免双头）；⑤ 补 HTTP 层测试：`middleware/xhr_test.go`（写方法缺头 400/安全方法免头）+ `handler/interaction_http_test.go`（评论 XHR 400 不落库、蜜罐假成功零落库、同 IP 第二条 → HTTP 429/42901、点赞缺头 400 与切换往返、admin 评论 401/审核后公开可见；铸 token 免登录防限流配额互扰）。测试还揪出一个契约 bug：列表接口 `size` 绑定 `min=1` 导致缺省 400（handler 默认值成死代码），已改 `omitempty,min=1,max=50`（comment/post 四个 query 结构体）。

**实现要点与计划外决策**：

1. 评论/点赞/后台补全接口（Task 1–3）按契约实现；测试覆盖净化边界、变形敏感词（全角/插符号/零宽字符）、双窗口限流、两级树、审核计数同步、点赞切换往返与未发布 404。
2. PostEditor 标签即时新建（Task 6）未用 naive-ui 的 `@create`：实测 2.45.2 中 `onCreate` 在**键入时逐键同步触发**且返回值即临时选项 value，无法承载异步 POST。改为受控 `:value` + `@update:value` 拦截字符串临时项 → 异步 `POST /admin/tags` 后补选真实 id；输入重名标签时前端按 label 去重直接选中（`tags.name` 唯一索引，直接 POST 会 40001）。
3. 修改密码表单抽取为 `components/admin/ChangePasswordForm.vue`，AdminLayout 弹窗与新增 Profile 页共用，避免复制校验逻辑；Profile 保存后同步 Pinia admin store 使顶栏昵称/头像即时更新。
4. `deploy.sh` 不自动执行 initadmin：实测其幂等语义为「应用迁移 + **重置管理员密码**」，放进部署链路会周期性锁死账号；脚本只上传二进制并在含新增迁移时打印人工执行提示。`backup.sh` uploads 用 GNU tar listed-incremental 每日增量链（满 7 份整链重置重新全量，避免删链头留增量导致不可恢复）。
5. 计划声明中的偏离全部按声明实现（敏感词现查现建 trie、限流复用 IPLimiter、蜜罐字段名 website、链接>2 并入 40006、SameSite 仅生产 Strict、部署只交付文件未实测）。

**遗留**：部署文件未经真实服务器验证（计划内声明）；nginx CSP 头以注释提供（md-editor-v3/Shiki 需要 inline style，放开与否留给站长决策）。
