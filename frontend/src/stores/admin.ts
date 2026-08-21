import { defineStore } from 'pinia'
import type { AdminInfo } from '@/types/api'

const STORAGE_KEY = 'admin_info'

export const useAdminStore = defineStore('admin', {
  state: () => ({
    info: restore(),
  }),
  getters: {
    isLoggedIn: (s) => s.info !== null,
  },
  actions: {
    set(info: AdminInfo) {
      this.info = info
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(info))
    },
    clear() {
      this.info = null
      sessionStorage.removeItem(STORAGE_KEY)
    },
    /** 修改密码成功后清除强制改密标记 */
    patchMustChangePassword() {
      if (this.info) {
        this.info = { ...this.info, must_change_password: false }
        sessionStorage.setItem(STORAGE_KEY, JSON.stringify(this.info))
      }
    },
  },
})

function restore(): AdminInfo | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as AdminInfo) : null
  } catch {
    return null
  }
}
