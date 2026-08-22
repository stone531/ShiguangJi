<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { TocItem } from './MarkdownView.vue'

/** 文章目录：h2/h3 树 + 滚动高亮（IntersectionObserver） */
const props = defineProps<{ items: TocItem[] }>()

const activeId = ref('')
let observer: IntersectionObserver | null = null

function observe() {
  observer?.disconnect()
  observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting) activeId.value = e.target.id
      }
    },
    { rootMargin: '-80px 0px -70% 0px' },
  )
  for (const item of props.items) {
    const el = document.getElementById(item.id)
    if (el) observer.observe(el)
  }
}

watch(
  () => props.items,
  async () => {
    await nextTick()
    observe()
  },
)

onMounted(observe)
onBeforeUnmount(() => observer?.disconnect()) // 红线：observer 必须清理
</script>

<template>
  <nav class="text-sm" aria-label="目录">
    <p class="font-display mb-3 text-xs font-semibold uppercase tracking-widest text-dim">目录</p>
    <ul class="space-y-1.5 border-l border-line">
      <li v-for="item in items" :key="item.id">
        <a
          :href="`#${item.id}`"
          class="block truncate text-dim transition-colors hover:text-ink"
          :class="[
            item.level === 3 ? 'pl-7' : 'pl-4',
            { 'text-accent! border-l-2 border-accent -ml-px': activeId === item.id },
          ]"
        >{{ item.text }}</a>
      </li>
    </ul>
  </nav>
</template>
