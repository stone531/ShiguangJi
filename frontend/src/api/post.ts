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
