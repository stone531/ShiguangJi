import axios from 'axios'
import { http, put, post, del } from './http'
import type { Response, PageData, CommentItem } from '@/types/api'

export interface CommentQuery {
  status?: number
  page?: number
  size?: number
}

/** 判断是否为后端 404（接口尚未实现，M4 接入后移除）。
 *  拦截器把 AxiosError 包装成 Error 并挂到 cause，这里还原后判定。 */
function isNotFound(e: unknown): boolean {
  const cause = (e as { cause?: unknown })?.cause
  const target = axios.isAxiosError(e) ? e : axios.isAxiosError(cause) ? cause : null
  return target?.response?.status === 404
}

/** TODO(M4)：评论接口后端就绪后移除 mock 分支 */
export async function listComments(q: CommentQuery): Promise<PageData<CommentItem>> {
  try {
    // 直接用 http 实例拿原始 axios 错误（拦截器会包装丢失 status）
    const resp = await http.get<Response<PageData<CommentItem>>>('/admin/comments', { params: q })
    if (resp.data.code !== 0) throw new Error(resp.data.msg)
    return resp.data.data
  } catch (e) {
    if (isNotFound(e)) return mockComments(q)
    throw e
  }
}

export async function reviewComment(id: number, status: 0 | 1 | 2): Promise<null> {
  try {
    return await put<null>(`/admin/comments/${id}/status`, { status })
  } catch (e) {
    if (isNotFound(e)) return mockSuccess(`reviewComment(${id}, ${status})`)
    throw e
  }
}

export async function replyComment(id: number, content: string): Promise<null> {
  try {
    return await post<null>(`/admin/comments/${id}/reply`, { content })
  } catch (e) {
    if (isNotFound(e)) return mockSuccess(`replyComment(${id})`)
    throw e
  }
}

export async function deleteComment(id: number): Promise<null> {
  try {
    return await del<null>(`/admin/comments/${id}`)
  } catch (e) {
    if (isNotFound(e)) return mockSuccess(`deleteComment(${id})`)
    throw e
  }
}

function mockSuccess(action: string): null {
  console.warn(`TODO(M4): 评论接口未实现，${action} 为本地假成功`)
  return null
}

function mockComments(q: CommentQuery): PageData<CommentItem> {
  const all: CommentItem[] = [
    {
      id: 1, post_id: 101, post_title: 'Vue 3 组合式 API 实践', parent_id: 0,
      nickname: '小明', content: '写得真好，受益匪浅 😊 已经照着重构了项目！', is_admin: false,
      status: 0, created_at: '2026-08-20T10:23:00+08:00',
    },
    {
      id: 2, post_id: 102, post_title: 'Go 错误处理指北', parent_id: 0,
      nickname: 'Gopher', content: '请问 errors.Is 和 errors.As 的适用场景分别是什么？查了文档还是有点懵 🤔', is_admin: false,
      status: 0, created_at: '2026-08-20T14:05:00+08:00',
    },
    {
      id: 3, post_id: 103, post_title: '我的 2026 开发环境', parent_id: 0,
      nickname: '路人甲', content: '[违规内容已屏蔽]', is_admin: false,
      status: 0, created_at: '2026-08-21T09:12:00+08:00',
    },
    {
      id: 4, post_id: 101, post_title: 'Vue 3 组合式 API 实践', parent_id: 0,
      nickname: '阿强', content: '自动保存那一段太实用了，之前丢过稿子的人狂喜 🎉', is_admin: false,
      status: 1, created_at: '2026-08-19T20:41:00+08:00',
    },
    {
      id: 5, post_id: 102, post_title: 'Go 错误处理指北', parent_id: 4,
      nickname: '站长', content: '简单说：Is 比较哨兵错误，As 提取自定义类型，下篇展开讲 😉', is_admin: true,
      status: 1, created_at: '2026-08-19T21:00:00+08:00',
    },
    {
      id: 6, post_id: 104, post_title: '搬家到静态博客的一年', parent_id: 0,
      nickname: '键盘侠', content: '这也叫教程？[违规内容已屏蔽] 建议重写', is_admin: false,
      status: 2, created_at: '2026-08-18T11:30:00+08:00',
    },
  ]
  const filtered = q.status === undefined ? all : all.filter((c) => c.status === q.status)
  const page = q.page ?? 1
  const size = q.size ?? 10
  const start = (page - 1) * size
  return { list: filtered.slice(start, start + size), total: filtered.length, page, size }
}
