<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getPostBySlug, listPublicPosts } from '@/api/public'
import { useSiteStore } from '@/stores/site'
import MarkdownView from '@/components/site/MarkdownView.vue'
import type { TocItem } from '@/components/site/MarkdownView.vue'
import Toc from '@/components/site/Toc.vue'
import DonateModal from '@/components/site/DonateModal.vue'
import ArticleCard from '@/components/site/ArticleCard.vue'
import LikeButton from '@/components/site/LikeButton.vue'
import CommentSection from '@/components/site/CommentSection.vue'
import type { Post, PostCard } from '@/types/api'

const route = useRoute()
const site = useSiteStore()

const post = ref<Post | null>(null)
const prev = ref<{ slug: string; title: string } | null>(null)
const next = ref<{ slug: string; title: string } | null>(null)
const related = ref<PostCard[]>([])
const toc = ref<TocItem[]>([])
const loading = ref(true)
const notFound = ref(false)
const donateOpen = ref(false)
const tocDrawerOpen = ref(false)

const typeLabel: Record<number, string> = { 1: '文章', 2: '图片集', 3: '视频' }

let relatedSeq = 0

async function load(slug: string) {
  loading.value = true
  notFound.value = false
  post.value = null
  toc.value = []
  try {
    const data = await getPostBySlug(slug)
    post.value = data.post
    prev.value = data.prev
    next.value = data.next
    document.title = `${data.post.title} - ${site.siteName}`
    window.scrollTo({ top: 0 })
    void loadRelated(data.post)
  } catch {
    notFound.value = true
    document.title = `找不到内容 - ${site.siteName}`
  } finally {
    loading.value = false
  }
}

/** 相关文章：共同标签（取首个标签查列表，排除自身） */
async function loadRelated(p: Post) {
  const firstTag = p.tags?.[0]?.name
  if (!firstTag) {
    related.value = []
    return
  }
  const seq = ++relatedSeq
  try {
    const data = await listPublicPosts({ tag: firstTag, size: 4 })
    if (seq !== relatedSeq) return
    related.value = data.list.filter((c) => c.slug !== p.slug).slice(0, 3)
  } catch {
    related.value = []
  }
}

function fmtDate(d: string | null) {
  return d ? new Date(d).toLocaleDateString('zh-CN') : ''
}

// 相关文章/上下篇点击跳转时复用同组件实例，需响应 slug 变化
watch(
  () => route.params.slug,
  (slug) => {
    if (route.name === 'post' && typeof slug === 'string') void load(slug)
  },
)

onMounted(() => {
  site.ensure()
  const slug = route.params.slug
  if (typeof slug === 'string') void load(slug)
})
</script>

<template>
  <!-- 加载/404 态 -->
  <div v-if="loading" class="glass mt-10 rounded-2xl p-16 text-center text-sm text-dim">加载中…</div>
  <div v-else-if="notFound" class="glass mt-10 rounded-2xl p-16 text-center">
    <p class="text-lg">内容不存在或未发布</p>
    <RouterLink to="/" class="mt-4 inline-block text-sm text-accent hover:underline">← 回首页</RouterLink>
  </div>

  <article v-else-if="post" class="mt-10">
    <!-- 头部元信息 -->
    <header class="text-center">
      <div class="flex items-center justify-center gap-2 text-xs text-dim">
        <span class="rounded-full border border-line px-2 py-0.5">{{ typeLabel[post.type] }}</span>
        <RouterLink v-if="post.category" :to="`/category/${post.category.name}`" class="text-accent hover:underline">
          {{ post.category.name }}
        </RouterLink>
        <span class="font-mono">{{ fmtDate(post.published_at) }}</span>
        <span>浏览 {{ post.views }}</span>
      </div>
      <h1 class="mx-auto mt-4 max-w-3xl text-3xl font-bold leading-snug">{{ post.title }}</h1>
      <div v-if="post.tags?.length" class="mt-4 flex flex-wrap justify-center gap-2">
        <RouterLink
          v-for="t in post.tags"
          :key="t.id"
          :to="`/tag/${t.name}`"
          class="rounded bg-white/5 px-2 py-0.5 text-xs text-dim transition-colors hover:text-accent"
        ># {{ t.name }}</RouterLink>
      </div>
    </header>

    <!-- 正文 + TOC -->
    <div class="mt-10 lg:grid lg:grid-cols-[1fr_200px] lg:gap-10">
      <div class="glass rounded-2xl p-6 sm:p-10">
        <!-- 视频：顶部 16:9 嵌入（域名白名单由后端保存时校验） -->
        <div v-if="post.type === 3 && post.video_url" class="mb-8 aspect-video overflow-hidden rounded-xl border border-line">
          <iframe :src="post.video_url" class="h-full w-full" allowfullscreen title="视频" />
        </div>

        <p v-if="post.summary" class="mb-8 border-l-2 border-accent pl-4 text-dim">{{ post.summary }}</p>

        <MarkdownView :source="post.content" @toc="(items: TocItem[]) => (toc = items)" />
      </div>

      <!-- 桌面端悬浮 TOC -->
      <aside v-if="toc.length" class="sticky top-20 hidden h-fit lg:block">
        <Toc :items="toc" />
      </aside>
    </div>

    <!-- 移动端 TOC 抽屉 -->
    <div v-if="toc.length" class="lg:hidden">
      <button
        class="glass fixed bottom-6 right-4 z-40 rounded-full px-4 py-2 text-sm text-dim shadow-lg"
        @click="tocDrawerOpen = true"
      >目录</button>
      <Teleport to="body">
        <Transition name="fade">
          <div v-if="tocDrawerOpen" class="fixed inset-0 z-50 bg-black/70" @click="tocDrawerOpen = false">
            <div class="glass absolute bottom-0 left-0 right-0 max-h-[60vh] overflow-y-auto rounded-t-2xl p-6" @click.stop>
              <Toc :items="toc" />
            </div>
          </div>
        </Transition>
      </Teleport>
    </div>

    <!-- 上/下一篇 -->
    <nav class="mt-10 grid gap-4 sm:grid-cols-2">
      <RouterLink v-if="prev" :to="`/post/${prev.slug}`" class="glass spotlight rounded-2xl p-5 transition-colors hover:border-accent/40">
        <p class="text-xs text-dim">上一篇</p>
        <p class="mt-1 truncate">{{ prev.title }}</p>
      </RouterLink>
      <RouterLink
        v-if="next"
        :to="`/post/${next.slug}`"
        class="glass spotlight rounded-2xl p-5 text-right transition-colors hover:border-accent/40 sm:col-start-2"
      >
        <p class="text-xs text-dim">下一篇</p>
        <p class="mt-1 truncate">{{ next.title }}</p>
      </RouterLink>
    </nav>

    <!-- 相关文章 -->
    <section v-if="related.length" class="mt-12">
      <h2 class="font-display mb-5 text-sm font-semibold uppercase tracking-widest text-dim">相关文章</h2>
      <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <ArticleCard v-for="p in related" :key="p.id" :post="p" />
      </div>
    </section>

    <!-- 点赞 + 赞赏 -->
    <section class="glass mt-12 rounded-2xl p-8 text-center">
      <div class="flex justify-center">
        <LikeButton :slug="post.slug" />
      </div>
      <p class="mt-6 text-3xl">☕</p>
      <p class="mt-3 text-dim">{{ site.info?.donate_text ?? '写得不错，请我喝杯咖啡' }}</p>
      <button
        class="mt-5 rounded-full bg-gradient-to-r from-accent to-accent-2 px-6 py-2 text-sm font-medium text-void transition-opacity hover:opacity-85"
        @click="donateOpen = true"
      >赞赏支持</button>
    </section>

    <!-- 评论（comment_enabled=false 整区隐藏） -->
    <CommentSection v-if="site.info?.comment_enabled !== false" :key="post.slug" :slug="post.slug" />
  </article>

  <DonateModal v-model="donateOpen" />
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
