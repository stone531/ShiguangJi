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
