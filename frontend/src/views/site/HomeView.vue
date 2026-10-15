<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listPublicPosts } from '@/api/public'
import { useSiteStore } from '@/stores/site'
import PostList from '@/components/site/PostList.vue'
import type { PostCard } from '@/types/api'

const site = useSiteStore()

/* ===== 纯列表 + 分页（请求序号守卫防翻页竞态） ===== */
const page = ref(1)
const feed = ref<PostCard[]>([])
const total = ref(0)
const loadingFeed = ref(false)
const feedError = ref(false)
let seq = 0

const pageSize = computed(() => site.info?.page_size ?? 10)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

async function loadFeed() {
  const current = ++seq
  loadingFeed.value = true
  feedError.value = false
  try {
    const data = await listPublicPosts({ page: page.value, size: pageSize.value })
    if (current !== seq) return // 过期响应丢弃
    feed.value = data.list
    total.value = data.total
  } catch {
    if (current === seq) feedError.value = true
  } finally {
    if (current === seq) loadingFeed.value = false
  }
}

function turnPage(p: number) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  loadFeed()
}

onMounted(() => {
  site.ensure()
  loadFeed()
})
</script>

<template>
  <!-- 极简头部：头像 + 站名 + 一行简介 -->
  <header class="flex flex-col items-center py-14 text-center">
    <img
      v-if="site.info?.avatar"
      :src="site.info.avatar"
      alt="博主头像"
      class="h-16 w-16 rounded-full object-cover"
    />
    <h1 class="mt-4 text-2xl font-bold tracking-wide">{{ site.siteName }}</h1>
    <p class="mt-2 text-sm text-dim">{{ site.siteDesc }}</p>
  </header>

  <!-- 文章列表 -->
  <div v-if="feedError" class="py-10 text-center text-sm text-dim">
    加载失败，<button class="text-accent hover:underline" @click="loadFeed">重试</button>
  </div>
  <div v-else-if="loadingFeed && !feed.length" class="py-10 text-center text-sm text-dim">加载中…</div>
  <div v-else-if="!feed.length" class="py-10 text-center text-sm text-dim">这里还没有内容</div>
  <PostList v-else :posts="feed" />

  <!-- 分页 -->
  <nav v-if="totalPages > 1" class="mt-8 flex items-center justify-center gap-4 text-sm">
    <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page <= 1" @click="turnPage(page - 1)">← 上一页</button>
    <span class="font-mono text-dim">{{ page }} / {{ totalPages }}</span>
    <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page >= totalPages" @click="turnPage(page + 1)">下一页 →</button>
  </nav>
</template>
