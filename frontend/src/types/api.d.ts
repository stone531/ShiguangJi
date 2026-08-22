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

/** 后台评论列表项（含 email/ip/所属文章，供审核参考） */
export interface AdminCommentVO {
  id: number
  post_id: number
  post_title: string
  post_slug: string
  parent_id: number
  nickname: string
  email: string
  content: string
  is_admin: boolean
  status: 0 | 1 | 2
  ip: string
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

/** 后台站点设置：比公开版多敏感词表 */
export interface SiteAdminVO extends SiteSettingVO {
  sensitive_words: string
}

/** 仪表盘六指标 */
export interface StatsVO {
  post_total: number
  post_published: number
  comment_total: number
  comment_pending: number
  view_total: number
  like_total: number
}

export interface AdminProfileVO {
  username: string
  nickname: string
  avatar: string
}

/** 后台全量分类（编辑器选项） */
export interface CategoryFull {
  id: number
  name: string
  sort: number
}

/** 后台全量标签（编辑器选项） */
export interface TagFull {
  id: number
  name: string
}

/* ===== M3 前台 ===== */

export interface SitePublicVO extends SiteSettingVO {
  post_count: number
  view_count: number
  like_count: number
}

/** 前台列表卡片（后端已剥离 content） */
export interface PostCard {
  id: number
  title: string
  slug: string
  summary: string
  cover: string
  type: 1 | 2 | 3
  is_top: boolean
  is_featured: boolean
  views: number
  like_count: number
  comment_count: number
  published_at: string | null
  category?: { id: number; name: string } | null
  tags: { id: number; name: string }[]
}

export interface PostDetailData {
  post: Post
  prev: { slug: string; title: string } | null
  next: { slug: string; title: string } | null
}

export interface CategoryItem {
  id: number
  name: string
  sort: number
  post_count: number
}

export interface TagItem {
  id: number
  name: string
  post_count: number
}

export interface ArchiveGroup {
  year: number
  month: number
  posts: { id: number; title: string; slug: string; published_at: string }[]
}

export interface PostListQuery {
  page?: number
  size?: number
  type?: 1 | 2 | 3
  category?: string
  tag?: string
  keyword?: string
  featured?: boolean
}

/* ===== M4 互动 ===== */

/** 前台评论树节点（仅已通过；email/ip 永不下发） */
export interface CommentNode {
  id: number
  nickname: string
  content: string
  is_admin: boolean
  created_at: string
  children: CommentNode[]
}

export interface LikeStatus {
  liked: boolean
  like_count: number
}

export interface CommentPostBody {
  nickname: string
  email?: string
  content: string
  /** 蜜罐字段：正常用户永远为空 */
  website?: string
}
