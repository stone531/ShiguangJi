<script setup lang="ts">
import { computed } from 'vue'
import { countChars, kchars, readingMinutes } from '@/utils/reading'
import { fmtDate } from '@/utils/date'

/**
 * 文章元信息行（spec §8）：日期 🕰️ ⏳阅读时长(字数) [创作✍️不易👨‍🌾请勿🚫广告]
 * 前台详情页与后台发布预览共用，保证两边完全一致。
 */
const props = defineProps<{
  date: string | Date | null
  content: string
  type?: number
  categoryName?: string | null
  categoryLink?: string
  views?: number
}>()

const typeLabel: Record<number, string> = { 1: '文章', 2: '图集', 3: '视频' }

const chars = computed(() => countChars(props.content))
const minutes = computed(() => readingMinutes(chars.value))
const kc = computed(() => kchars(chars.value))
</script>

<template>
  <div class="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 text-xs text-dim">
    <span v-if="type">{{ typeLabel[type] }}</span>
    <RouterLink v-if="categoryName && categoryLink" :to="categoryLink" class="text-accent hover:underline">
      {{ categoryName }}
    </RouterLink>
    <span v-else-if="categoryName">{{ categoryName }}</span>
    <span class="font-mono">{{ fmtDate(date) }}</span>
    <span aria-hidden="true">🕰️</span>
    <span>⏳{{ minutes }}分钟({{ kc }}千字)</span>
    <span v-if="views !== undefined">浏览 {{ views }}</span>
    <span class="noad-badge">创作✍️不易👨‍🌾请勿🚫广告</span>
  </div>
</template>
