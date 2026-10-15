<script setup lang="ts">
import { NButton, NModal } from 'naive-ui'
import MarkdownView from '@/components/site/MarkdownView.vue'
import PostMetaLine from '@/components/site/PostMetaLine.vue'

/**
 * 发布前全屏预览（spec §12）：复刻前台详情页最终形态——
 * 标题 + 元信息行（实时字数/时长）+ 标签 + 视频/正文，所见即所得。
 */
defineProps<{
  show: boolean
  title: string
  content: string
  type: 1 | 2 | 3
  videoUrl: string
  categoryName: string | null
  tags: string[]
}>()

const emit = defineEmits<{ (e: 'update:show', v: boolean): void; (e: 'confirm'): void }>()

function close() {
  emit('update:show', false)
}
</script>

<template>
  <NModal :show="show" @update:show="(v: boolean) => emit('update:show', v)">
    <div class="fixed inset-0 z-50 overflow-y-auto bg-white">
      <div class="mx-auto max-w-3xl px-4 pb-16 pt-10">
        <header class="text-center">
          <PostMetaLine
            :date="new Date().toISOString()"
            :content="content"
            :type="type"
            :category-name="categoryName"
          />
          <h1 class="mx-auto mt-4 max-w-3xl text-3xl font-bold leading-snug">{{ title || '（无标题）' }}</h1>
          <div v-if="tags.length" class="mt-4 flex flex-wrap justify-center gap-2 text-xs text-dim">
            <span v-for="t in tags" :key="t"># {{ t }}</span>
          </div>
        </header>

        <div class="mt-8">
          <div v-if="type === 3 && videoUrl" class="mx-auto mb-8 aspect-video max-w-[42em] overflow-hidden" style="border-radius: 4px">
            <iframe :src="videoUrl" class="h-full w-full" allowfullscreen title="视频预览" />
          </div>
          <MarkdownView :source="content" />
        </div>

        <div class="mt-12 flex justify-center gap-4 border-t border-line pt-8">
          <NButton size="large" @click="close">返回修改</NButton>
          <NButton size="large" type="primary" @click="emit('confirm')">确认发布</NButton>
        </div>
      </div>
    </div>
  </NModal>
</template>
