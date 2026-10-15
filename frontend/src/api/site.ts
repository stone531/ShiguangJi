import { get, put } from './http'
import type { SiteAdminVO } from '@/types/api'

/** 后台站点设置（含敏感词表；公开 /site 永不下发该键） */
export function getSiteSettings() {
  return get<SiteAdminVO>('/admin/site')
}

export function saveSiteSettings(s: SiteAdminVO) {
  return put<null>('/admin/site', s)
}
