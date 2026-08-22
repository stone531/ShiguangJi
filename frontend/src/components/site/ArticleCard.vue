<script setup lang="ts">
import type { PostCard } from '@/types/api'

defineProps<{ post: PostCard }>()

const typeLabel: Record<number, string> = { 1: '文章', 2: '图片集', 3: '视频' }

/** 聚光悬停：把光标位置写进 CSS 变量，theme.css 的 .spotlight 消费 */
function onMove(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const r = el.getBoundingClientRect()
  el.style.setProperty('--mx', `${e.clientX - r.left}px`)
  el.style.setProperty('--my', `${e.clientY - r.top}px`)
}

function fmtDate(d: string | null) {
  return d ? new Date(d).toLocaleDateString('zh-CN') : ''
}
</script>

<template>
  <RouterLink
    :to="`/post/${post.slug}`"
    class="spotlight glass block overflow-hidden rounded-2xl transition-all duration-300 hover:-translate-y-1"
    @mousemove="onMove"
  >
    <div v-if="post.cover" class="aspect-video overflow-hidden">
      <img :src="post.cover" :alt="post.title" class="h-full w-full object-cover" loading="lazy" />
    </div>
    <div class="p-5">
      <div class="flex items-center gap-2 text-xs text-dim">
        <span class="rounded-full border border-line px-2 py-0.5">{{ typeLabel[post.type] }}</span>
        <span v-if="post.category" class="text-accent">{{ post.category.name }}</span>
        <span v-if="post.is_top" class="text-amber-400">置顶</span>
        <span class="ml-auto font-mono">{{ fmtDate(post.published_at) }}</span>
      </div>
      <h3 class="mt-2 truncate text-lg font-semibold">{{ post.title }}</h3>
      <p class="mt-2 line-clamp-2 text-sm leading-relaxed text-dim">{{ post.summary }}</p>
      <div v-if="post.tags.length" class="mt-3 flex flex-wrap gap-1.5">
        <span
          v-for="t in post.tags"
          :key="t.id"
          class="rounded bg-white/5 px-1.5 py-0.5 text-xs text-dim"
        ># {{ t.name }}</span>
      </div>
      <div class="mt-3 flex gap-4 text-xs text-dim">
        <span>浏览 {{ post.views }}</span>
        <span>喜欢 {{ post.like_count }}</span>
        <span>评论 {{ post.comment_count }}</span>
      </div>
    </div>
  </RouterLink>
</template>
