// 前台公开接口（spec §5.1）。与 admin 接口共用 http 单例与拦截器。
import { get, post } from './http'
import type {
  ArchiveGroup,
  CategoryItem,
  CommentNode,
  CommentPostBody,
  LikeStatus,
  PageData,
  PostCard,
  PostDetailData,
  PostListQuery,
  SitePublicVO,
  TagItem,
} from '@/types/api'

export function getSitePublic() {
  return get<SitePublicVO>('/site')
}

export function listPublicPosts(q: PostListQuery) {
  return get<PageData<PostCard>>('/posts', q)
}

export function getPostBySlug(slug: string) {
  return get<PostDetailData>(`/posts/${slug}`)
}

export function listCategories() {
  return get<{ list: CategoryItem[] }>('/categories')
}

export function listTags() {
  return get<{ list: TagItem[] }>('/tags')
}

export function getArchive() {
  return get<{ list: ArchiveGroup[] }>('/archive')
}

/** 评论树（仅已通过），两级结构 */
export function listComments(slug: string, page = 1, size = 20) {
  return get<PageData<CommentNode>>(`/posts/${slug}/comments`, { page, size })
}

/** 发表评论：先审后显；蜜罐命中时后端假成功 */
export function createComment(slug: string, body: CommentPostBody) {
  return post<{ id: number; message: string }>(`/posts/${slug}/comments`, body)
}

/** 点赞状态（指纹匿名标识） */
export function getLikeStatus(slug: string) {
  return get<LikeStatus>(`/posts/${slug}/like`)
}

/** 切换点赞 */
export function toggleLike(slug: string) {
  return post<LikeStatus>(`/posts/${slug}/like`)
}
