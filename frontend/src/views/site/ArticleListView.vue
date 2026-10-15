<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { listPublicPosts } from '@/api/public'
import { useSiteStore } from '@/stores/site'
import PostList from '@/components/site/PostList.vue'
import type { PostCard } from '@/types/api'

/** 分类/标签筛选列表页（/category/:name 与 /tag/:name 复用） */
const props = defineProps<{ mode: 'category' | 'tag' }>()

const route = useRoute()
const site = useSiteStore()

const name = ref('')
const list = ref<PostCard[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const failed = ref(false)
let seq = 0

const pageSize = () => site.info?.page_size ?? 10
const totalPages = ref(1)

async function load() {
  const current = ++seq
  loading.value = true
  failed.value = false
  try {
    const data = await listPublicPosts({
      page: page.value,
      size: pageSize(),
      [props.mode]: name.value,
    })
    if (current !== seq) return
    list.value = data.list
    total.value = data.total
    totalPages.value = Math.max(1, Math.ceil(data.total / pageSize()))
  } catch {
    if (current === seq) failed.value = true
  } finally {
    if (current === seq) loading.value = false
  }
}

function turnPage(p: number) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  void load()
}

watch(
  () => [route.params.name, props.mode] as const,
  ([n]) => {
    if (typeof n === 'string') {
      name.value = n
      page.value = 1
      void load()
    }
  },
)

onMounted(() => {
  site.ensure()
  const n = route.params.name
  if (typeof n === 'string') {
    name.value = n
    void load()
  }
})
</script>

<template>
  <header class="mt-12 text-center">
    <p class="text-xs text-dim">{{ mode === 'category' ? '分类' : '标签' }}</p>
    <h1 class="mt-2 text-2xl font-bold">{{ name }}</h1>
    <p class="mt-2 text-xs text-dim">共 {{ total }} 篇</p>
  </header>

  <div v-if="failed" class="mt-8 py-10 text-center text-sm text-dim">
    加载失败，<button class="text-accent hover:underline" @click="load">重试</button>
  </div>
  <div v-else-if="loading && !list.length" class="mt-8 py-10 text-center text-sm text-dim">加载中…</div>
  <div v-else-if="!list.length" class="mt-8 py-10 text-center text-sm text-dim">
    该{{ mode === 'category' ? '分类' : '标签' }}下暂无内容
    <RouterLink to="/" class="ml-2 text-accent hover:underline">回首页 →</RouterLink>
  </div>
  <div v-else class="mt-8">
    <PostList :posts="list" />
  </div>

  <nav v-if="totalPages > 1" class="mt-8 flex items-center justify-center gap-4 text-sm">
    <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page <= 1" @click="turnPage(page - 1)">← 上一页</button>
    <span class="font-mono text-dim">{{ page }} / {{ totalPages }}</span>
    <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page >= totalPages" @click="turnPage(page + 1)">下一页 →</button>
  </nav>
</template>
