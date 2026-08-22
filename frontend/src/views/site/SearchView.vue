<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listPublicPosts } from '@/api/public'
import ArticleCard from '@/components/site/ArticleCard.vue'
import type { PostCard } from '@/types/api'

/** 搜索：标题/摘要/标签模糊匹配（后端 LIKE）；q 与 URL 双向同步 + 300ms 防抖 */
const route = useRoute()
const router = useRouter()

const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const list = ref<PostCard[]>([])
const total = ref(0)
const loading = ref(false)
const searched = ref(false)
let debounceTimer: number | null = null
let seq = 0

async function search(keyword: string) {
  const current = ++seq
  if (!keyword.trim()) {
    list.value = []
    total.value = 0
    searched.value = false
    return
  }
  loading.value = true
  try {
    const data = await listPublicPosts({ keyword: keyword.trim(), size: 30 })
    if (current !== seq) return
    list.value = data.list
    total.value = data.total
    searched.value = true
  } catch {
    if (current === seq) {
      list.value = []
      total.value = 0
      searched.value = true
    }
  } finally {
    if (current === seq) loading.value = false
  }
}

function onInput() {
  if (debounceTimer !== null) clearTimeout(debounceTimer)
  debounceTimer = window.setTimeout(() => {
    // 同步到 URL（replace 不产生历史记录）
    void router.replace({ query: q.value.trim() ? { q: q.value.trim() } : {} })
  }, 300)
}

watch(
  () => route.query.q,
  (v) => {
    const kw = typeof v === 'string' ? v : ''
    if (kw !== q.value) q.value = kw
    void search(kw)
  },
)

onMounted(() => {
  if (q.value) void search(q.value)
})
onBeforeUnmount(() => {
  if (debounceTimer !== null) clearTimeout(debounceTimer) // 红线：定时器清理
})
</script>

<template>
  <header class="mx-auto mt-14 max-w-xl text-center">
    <h1 class="text-gradient text-2xl font-bold">搜索</h1>
    <div class="glass mt-6 flex items-center gap-2 rounded-full px-5 py-3">
      <span class="text-dim">🔍</span>
      <input
        v-model="q"
        type="search"
        placeholder="标题 / 摘要 / 标签…"
        class="w-full bg-transparent text-sm outline-none placeholder:text-dim/60"
        @input="onInput"
      />
    </div>
  </header>

  <div class="mt-10">
    <p v-if="searched && q" class="mb-5 text-center text-xs text-dim">找到 {{ total }} 篇相关内容</p>
    <div v-if="loading" class="glass rounded-2xl p-10 text-center text-sm text-dim">搜索中…</div>
    <div v-else-if="searched && !list.length" class="glass rounded-2xl p-10 text-center text-sm text-dim">
      没有找到与「{{ q }}」相关的内容
    </div>
    <div v-else-if="list.length" class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <ArticleCard v-for="p in list" :key="p.id" :post="p" />
    </div>
    <p v-else class="glass rounded-2xl p-10 text-center text-sm text-dim">输入关键词，回车或稍候开始搜索</p>
  </div>
</template>
