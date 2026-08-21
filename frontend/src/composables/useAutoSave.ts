import { onBeforeUnmount, ref } from 'vue'

export interface DraftPayload {
  title: string
  content: string
  savedAt: number
}

/** 自动保存：定时暂存 localStorage；提交失败降级；恢复时提示 */
export function useAutoSave(key: string, intervalMs = 30_000) {
  const storageKey = `draft:${key}`
  const lastSavedAt = ref<number | null>(null)
  let timer: number | null = null

  function readDraft(): DraftPayload | null {
    try {
      const raw = localStorage.getItem(storageKey)
      return raw ? (JSON.parse(raw) as DraftPayload) : null
    } catch {
      return null
    }
  }

  function writeDraft(title: string, content: string) {
    try {
      const payload: DraftPayload = { title, content, savedAt: Date.now() }
      localStorage.setItem(storageKey, JSON.stringify(payload))
      lastSavedAt.value = payload.savedAt
    } catch {
      /* localStorage 满等情况忽略 */
    }
  }

  function clearDraft() {
    localStorage.removeItem(storageKey)
    lastSavedAt.value = null
  }

  function start(getCurrent: () => { title: string; content: string }) {
    stop()
    timer = window.setInterval(() => {
      const cur = getCurrent()
      if (cur.title || cur.content) writeDraft(cur.title, cur.content)
    }, intervalMs)
  }

  function stop() {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  onBeforeUnmount(stop) // 红线：定时器必须清理

  return { readDraft, writeDraft, clearDraft, start, stop, lastSavedAt }
}
