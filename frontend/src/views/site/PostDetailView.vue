<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getPostBySlug, listPublicPosts } from '@/api/public'
import { useSiteStore } from '@/stores/site'
import MarkdownView from '@/components/site/MarkdownView.vue'
import type { TocItem } from '@/components/site/MarkdownView.vue'
import PostList from '@/components/site/PostList.vue'
import PostMetaLine from '@/components/site/PostMetaLine.vue'
import DonateModal from '@/components/site/DonateModal.vue'
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

let relatedSeq = 0

/** 顶部索引：仅 h2（spec §10，tao 式正文目录） */
const tocH2 = computed(() => toc.value.filter((t) => t.level === 2))

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
  <div v-if="loading" class="mt-10 py-16 text-center text-sm text-dim">加载中…</div>
  <div v-else-if="notFound" class="mt-10 py-16 text-center">
    <p class="text-lg">内容不存在或未发布</p>
    <RouterLink to="/" class="mt-4 inline-block text-sm text-accent hover:underline">← 回首页</RouterLink>
  </div>

  <article v-else-if="post" class="mt-10">
    <!-- 头部：元信息行 + 标题 + 标签 -->
    <header class="text-center">
      <PostMetaLine
        :date="post.published_at"
        :content="post.content"
        :type="post.type"
        :category-name="post.category?.name ?? null"
        :category-link="post.category ? `/category/${post.category.name}` : undefined"
        :views="post.views"
      />
      <h1 class="mx-auto mt-4 max-w-3xl text-3xl font-bold leading-snug">{{ post.title }}</h1>
      <div v-if="post.tags?.length" class="mt-4 flex flex-wrap justify-center gap-2 text-xs text-dim">
        <RouterLink v-for="t in post.tags" :key="t.id" :to="`/tag/${t.name}`" class="hover:text-accent">
          # {{ t.name }}
        </RouterLink>
      </div>
    </header>

    <!-- 顶部索引（h2 目录，tao 式） -->
    <nav v-if="tocH2.length" class="mx-auto mt-8 max-w-[42em] text-sm">
      <p class="mb-2 text-xs font-semibold text-dim">索引</p>
      <ol class="space-y-1">
        <li v-for="h in tocH2" :key="h.id">
          <a :href="`#${h.id}`" class="text-accent hover:underline">{{ h.text }}</a>
        </li>
      </ol>
    </nav>

    <!-- 正文（白底直接排版，视频置顶） -->
    <div class="mt-8">
      <div v-if="post.type === 3 && post.video_url" class="mx-auto mb-8 aspect-video max-w-[42em] overflow-hidden" style="border-radius: 4px">
        <iframe :src="post.video_url" class="h-full w-full" allowfullscreen title="视频" />
      </div>

      <p v-if="post.summary" class="mx-auto mb-8 max-w-[42em] border-l-2 border-line pl-4 text-dim">{{ post.summary }}</p>

      <MarkdownView :source="post.content" @toc="(items: TocItem[]) => (toc = items)" />
    </div>

    <!-- 上/下一篇（文字链接式） -->
    <nav class="mx-auto mt-12 flex max-w-[42em] justify-between gap-4 text-sm">
      <RouterLink v-if="prev" :to="`/post/${prev.slug}`" class="truncate text-dim hover:text-accent">← {{ prev.title }}</RouterLink>
      <span v-else />
      <RouterLink v-if="next" :to="`/post/${next.slug}`" class="truncate text-dim hover:text-accent">{{ next.title }} →</RouterLink>
    </nav>

    <!-- 相关文章（纯文字列表） -->
    <section v-if="related.length" class="mx-auto mt-12 max-w-[42em]">
      <h2 class="mb-3 text-sm font-semibold text-dim">相关文章</h2>
      <PostList :posts="related" />
    </section>

    <!-- 点赞 + 赞赏 -->
    <section class="mt-12 border-t border-line pt-10 text-center">
      <div class="flex justify-center">
        <LikeButton :slug="post.slug" />
      </div>
      <p class="mt-8 text-3xl">☕</p>
      <p class="mt-3 text-dim">{{ site.info?.donate_text ?? '写得不错，请我喝杯咖啡' }}</p>
      <button
        class="mt-5 rounded-full bg-accent px-6 py-2 text-sm font-medium text-white transition-opacity hover:opacity-85"
        @click="donateOpen = true"
      >赞赏支持</button>
    </section>

    <!-- 评论（comment_enabled=false 整区隐藏） -->
    <CommentSection v-if="site.info?.comment_enabled !== false" :key="post.slug" :slug="post.slug" />
  </article>

  <DonateModal v-model="donateOpen" />
</template>
