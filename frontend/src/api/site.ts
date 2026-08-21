import axios from 'axios'
import { http, put } from './http'
import type { Response, SiteSettingVO } from '@/types/api'

/** 判断是否为后端 404（接口尚未实现，M4 接入后移除）。
 *  拦截器把 AxiosError 包装成 Error 并挂到 cause，这里还原后判定。 */
function isNotFound(e: unknown): boolean {
  const cause = (e as { cause?: unknown })?.cause
  const target = axios.isAxiosError(e) ? e : axios.isAxiosError(cause) ? cause : null
  return target?.response?.status === 404
}

/** TODO(M4)：站点设置接口后端就绪后移除 mock 分支 */
export async function getSiteSettings(): Promise<SiteSettingVO> {
  try {
    // 直接用 http 实例拿原始 axios 错误（拦截器会包装丢失 status）
    const resp = await http.get<Response<SiteSettingVO>>('/admin/site')
    if (resp.data.code !== 0) throw new Error(resp.data.msg)
    return resp.data.data
  } catch (e) {
    if (isNotFound(e)) return defaultSettings()
    throw e
  }
}

export async function saveSiteSettings(s: SiteSettingVO): Promise<null> {
  try {
    return await put<null>('/admin/site', s)
  } catch (e) {
    if (isNotFound(e)) {
      console.warn('TODO(M4): 站点设置接口未实现，saveSiteSettings 为本地假成功')
      return null
    }
    throw e
  }
}

/** 探测站点设置接口是否已由后端提供（M4 后恒为 true） */
export async function isSiteApiAvailable(): Promise<boolean> {
  try {
    await http.get<Response<null>>('/admin/site')
    return true
  } catch (e) {
    return !isNotFound(e)
  }
}

/** spec §4.8 预置键默认值 */
function defaultSettings(): SiteSettingVO {
  return {
    site_name: '时光集',
    site_desc: '记录技术与生活',
    avatar: '',
    wechat_qr: '',
    alipay_qr: '',
    donate_text: '写得不错，请我喝杯咖啡 ☕',
    social_links: [],
    page_size: 10,
    comment_enabled: true,
  }
}
