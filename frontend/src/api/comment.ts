import { get, put, post, del } from './http'
import type { PageData, AdminCommentVO } from '@/types/api'

export interface CommentQuery {
  status?: number
  page?: number
  size?: number
}

/** 后台评论列表（含 email/ip/所属文章，供审核参考） */
export function listComments(q: CommentQuery) {
  return get<PageData<AdminCommentVO>>('/admin/comments', q)
}

/** 审核：1=通过 / 2=驳回 */
export function reviewComment(id: number, status: 1 | 2) {
  return put<null>(`/admin/comments/${id}/review`, { status })
}

/** 博主回复：is_admin=1 自动通过，挂到目标所在顶级评论下 */
export function replyComment(id: number, content: string) {
  return post<null>(`/admin/comments/${id}/reply`, { content })
}

/** 软删评论 */
export function deleteComment(id: number) {
  return del<null>(`/admin/comments/${id}`)
}
