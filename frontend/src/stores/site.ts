// 前台站点信息：首次访问加载一次，全站共享（站名/赞赏/社交/统计）。
import { defineStore } from 'pinia'
import { getSitePublic } from '@/api/public'
import type { SitePublicVO } from '@/types/api'

export const useSiteStore = defineStore('site', {
  state: () => ({
    info: null as SitePublicVO | null,
    loading: false,
  }),
  getters: {
    siteName: (s) => s.info?.site_name ?? '时光集',
    siteDesc: (s) => s.info?.site_desc ?? '记录技术与生活',
  },
  actions: {
    /** 幂等加载；失败静默降级（页面以默认站名渲染，不阻塞首屏） */
    async ensure() {
      if (this.info || this.loading) return
      this.loading = true
      try {
        this.info = await getSitePublic()
      } catch {
        // 接口不可用时保持 null，getter 给兜底值
      } finally {
        this.loading = false
      }
    },
  },
})
