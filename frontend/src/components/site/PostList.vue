<script setup lang="ts">
import type { PostCard } from '@/types/api'
import { fmtDate } from '@/utils/date'

/** 纯文字文章列表（tao 式）：标题 + 灰色 mono 日期；图集/视频加灰色小标记 */
defineProps<{ posts: PostCard[] }>()

const typeTag: Record<number, string> = { 2: '图集', 3: '视频' }
</script>

<template>
  <ul>
    <li v-for="p in posts" :key="p.id">
      <RouterLink
        :to="`/post/${p.slug}`"
        class="group flex items-baseline gap-3 border-b border-line py-3"
      >
        <span class="truncate transition-colors group-hover:text-accent">{{ p.title }}</span>
        <span v-if="typeTag[p.type]" class="shrink-0 text-xs text-dim">{{ typeTag[p.type] }}</span>
        <span class="ml-auto shrink-0 font-mono text-xs text-dim">{{ fmtDate(p.published_at) }}</span>
      </RouterLink>
    </li>
  </ul>
</template>
