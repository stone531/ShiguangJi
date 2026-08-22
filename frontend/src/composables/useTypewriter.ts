import { onBeforeUnmount, ref } from 'vue'

/**
 * 打字机效果：短语逐字打出 → 停顿 → 逐字删除 → 下一条（Hero 轮播）。
 * 轻量 CSS/JS 实现（spec §6.1.1 轻量原则，不引动效库）。
 */
export function useTypewriter(phrases: () => string[], typeMs = 110, holdMs = 2000) {
  const text = ref('')
  let idx = 0
  let charIdx = 0
  let deleting = false
  let timer: number | null = null

  function tick() {
    const list = phrases()
    if (list.length === 0) {
      timer = null
      return
    }
    const current = list[idx % list.length]
    if (!deleting) {
      charIdx++
      text.value = current.slice(0, charIdx)
      if (charIdx >= current.length) {
        deleting = true
        timer = window.setTimeout(tick, holdMs)
        return
      }
      timer = window.setTimeout(tick, typeMs)
    } else {
      charIdx--
      text.value = current.slice(0, charIdx)
      if (charIdx <= 0) {
        deleting = false
        idx++
      }
      timer = window.setTimeout(tick, typeMs / 2)
    }
  }

  function start() {
    stop()
    tick()
  }

  function stop() {
    if (timer !== null) {
      clearTimeout(timer)
      timer = null
    }
  }

  onBeforeUnmount(stop) // 红线：定时器必须清理

  return { text, start, stop }
}
