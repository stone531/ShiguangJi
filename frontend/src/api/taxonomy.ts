// 后台分类/标签管理接口（M4）。
import { get, post, del } from './http'
import type { CategoryFull, TagFull } from '@/types/api'

/** 编辑器选项：全量分类（含空分类） */
export function listCategoriesAdmin() {
  return get<{ list: CategoryFull[] }>('/admin/categories')
}

/** 编辑器选项：全量标签（公开 /tags 只含有已发布文章的标签） */
export function listTagsAdmin() {
  return get<{ list: TagFull[] }>('/admin/tags')
}

/** 新建（id=0）或更新分类 */
export function saveCategory(name: string, sort = 0, id = 0) {
  return post<CategoryFull>('/admin/categories', { id, name, sort })
}

export function deleteCategory(id: number) {
  return del<null>(`/admin/categories/${id}`)
}

/** 新建（id=0）或更新标签；编辑器即时新建标签用 */
export function saveTag(name: string, id = 0) {
  return post<TagFull>('/admin/tags', { id, name })
}

export function deleteTag(id: number) {
  return del<null>(`/admin/tags/${id}`)
}

/** 把 source 标签合并到 target（移动关联后删除 source） */
export function mergeTag(id: number, targetId: number) {
  return post<null>(`/admin/tags/${id}/merge`, { target_id: targetId })
}
