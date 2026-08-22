<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listCategories, listPublicPosts, listTags } from '@/api/public'
import { useSiteStore } from '@/stores/site'
import { useTypewriter } from '@/composables/useTypewriter'
import ArticleCard from '@/components/site/ArticleCard.vue'
import DonateModal from '@/components/site/DonateModal.vue'
import type { CategoryItem, PostCard, TagItem } from '@/types/api'

const site = useSiteStore()

/* ===== Hero 打字机 ===== */
const phrases = computed(() => [site.siteDesc, '把折腾的过程记下来', '生活碎片与技术笔记', '欢迎常来看看 ☕'])
const { text: typed, start: startType } = useTypewriter(() => phrases.value)

/* ===== Bento 数据 ===== */
const featured = ref<PostCard[]>([])
const latest = ref<PostCard[]>([])
const tags = ref<TagItem[]>([])
const categories = ref<CategoryItem[]>([])

/* ===== 分类 Tab + 文章流（请求序号守卫防翻页竞态） ===== */
const activeCategory = ref('')
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
    const data = await listPublicPosts({
      page: page.value,
      size: pageSize.value,
      category: activeCategory.value || undefined,
    })
    if (current !== seq) return // 过期响应丢弃
    feed.value = data.list
    total.value = data.total
  } catch {
    if (current === seq) feedError.value = true
  } finally {
    if (current === seq) loadingFeed.value = false
  }
}

function selectCategory(name: string) {
  if (activeCategory.value === name) return
  activeCategory.value = name
  page.value = 1
  loadFeed()
}

function turnPage(p: number) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  loadFeed()
}

function fmtDate(d: string | null) {
  return d ? new Date(d).toLocaleDateString('zh-CN') : ''
}

const donateOpen = ref(false)

onMounted(() => {
  startType()
  loadFeed()
  void Promise.allSettled([
    listPublicPosts({ featured: true, size: 5 }),
    listPublicPosts({ size: 5 }),
    listTags(),
    listCategories(),
  ]).then(([f, l, t, c]) => {
    if (f.status === 'fulfilled') featured.value = f.value.list
    if (l.status === 'fulfilled') latest.value = l.value.list
    if (t.status === 'fulfilled') tags.value = t.value.list
    if (c.status === 'fulfilled') categories.value = c.value.list
  })
})
</script>

<template>
  <!-- Hero -->
  <section class="flex flex-col items-center py-20 text-center">
    <img
      v-if="site.info?.avatar"
      :src="site.info.avatar"
      alt="博主头像"
      class="h-20 w-20 rounded-full border border-line object-cover"
    />
    <h1 class="font-display mt-6 text-4xl font-bold tracking-wide sm:text-5xl">
      <span class="text-gradient">{{ site.siteName }}</span>
    </h1>
    <p class="mt-5 h-7 font-mono text-base text-dim">
      {{ typed }}<span class="type-caret" aria-hidden="true" />
    </p>
    <div class="mt-8 flex gap-4">
      <a href="#feed" class="rounded-full bg-gradient-to-r from-accent to-accent-2 px-6 py-2 text-sm font-medium text-void transition-opacity hover:opacity-85">开始阅读</a>
      <RouterLink to="/about" class="glass rounded-full px-6 py-2 text-sm transition-colors hover:text-accent">了解我</RouterLink>
    </div>
  </section>

  <!-- Bento Grid（spec §6.1.1 版式） -->
  <section class="grid grid-cols-1 gap-4 sm:grid-cols-3">
    <!-- 关于我（大格） -->
    <div class="glass spotlight rounded-2xl p-6 sm:row-span-2">
      <h2 class="font-display text-sm font-semibold uppercase tracking-widest text-dim">关于我</h2>
      <p class="mt-3 leading-relaxed">{{ site.siteDesc }}</p>
      <nav v-if="site.info?.social_links?.length" class="mt-4 flex flex-wrap gap-2">
        <a
          v-for="l in site.info.social_links"
          :key="l.name"
          :href="l.url"
          target="_blank"
          rel="noopener noreferrer"
          class="rounded-full border border-line px-3 py-1 text-xs text-dim transition-colors hover:border-accent hover:text-accent"
        >{{ l.name }}</a>
      </nav>
      <RouterLink to="/about" class="mt-5 inline-block text-sm text-accent hover:underline">更多 →</RouterLink>
    </div>

    <!-- 精选文章（宽格） -->
    <div class="glass spotlight rounded-2xl p-6 sm:col-span-2">
      <h2 class="font-display text-sm font-semibold uppercase tracking-widest text-dim">精选文章</h2>
      <ul class="mt-3 divide-y divide-line">
        <li v-for="p in featured" :key="p.id">
          <RouterLink :to="`/post/${p.slug}`" class="flex items-baseline gap-3 py-2.5 transition-colors hover:text-accent">
            <span class="truncate">{{ p.title }}</span>
            <span class="ml-auto shrink-0 font-mono text-xs text-dim">{{ fmtDate(p.published_at) }}</span>
          </RouterLink>
        </li>
        <li v-if="!featured.length" class="py-2.5 text-sm text-dim">暂无精选内容</li>
      </ul>
    </div>

    <!-- 标签云 -->
    <div class="glass spotlight rounded-2xl p-6">
      <h2 class="font-display text-sm font-semibold uppercase tracking-widest text-dim">标签云</h2>
      <div class="mt-3 flex flex-wrap gap-2">
        <RouterLink
          v-for="t in tags"
          :key="t.id"
          :to="`/tag/${t.name}`"
          class="rounded bg-white/5 px-2 py-1 text-dim transition-colors hover:bg-accent/20 hover:text-accent"
          :style="{ fontSize: `${Math.min(1.05, 0.75 + t.post_count * 0.08)}rem` }"
        ># {{ t.name }}</RouterLink>
        <span v-if="!tags.length" class="text-sm text-dim">暂无标签</span>
      </div>
    </div>

    <!-- 赞赏 -->
    <button class="glass spotlight rounded-2xl p-6 text-left transition-colors hover:border-accent/40" @click="donateOpen = true">
      <h2 class="font-display text-sm font-semibold uppercase tracking-widest text-dim">赞赏</h2>
      <p class="mt-3 text-2xl">☕</p>
      <p class="mt-2 text-sm text-dim">{{ site.info?.donate_text ?? '写得不错，请我喝杯咖啡' }}</p>
      <p class="mt-3 text-xs text-accent">点此打开二维码 →</p>
    </button>

    <!-- 最新文章（宽格） -->
    <div class="glass spotlight rounded-2xl p-6 sm:col-span-2">
      <h2 class="font-display text-sm font-semibold uppercase tracking-widest text-dim">最新文章</h2>
      <ul class="mt-3 divide-y divide-line">
        <li v-for="p in latest" :key="p.id">
          <RouterLink :to="`/post/${p.slug}`" class="flex items-baseline gap-3 py-2.5 transition-colors hover:text-accent">
            <span class="truncate">{{ p.title }}</span>
            <span class="ml-auto shrink-0 font-mono text-xs text-dim">{{ fmtDate(p.published_at) }}</span>
          </RouterLink>
        </li>
        <li v-if="!latest.length" class="py-2.5 text-sm text-dim">暂无内容</li>
      </ul>
    </div>

    <!-- 统计 -->
    <div class="glass spotlight rounded-2xl p-6">
      <h2 class="font-display text-sm font-semibold uppercase tracking-widest text-dim">统计</h2>
      <dl class="mt-4 space-y-3 text-sm">
        <div class="flex justify-between"><dt class="text-dim">文章</dt><dd class="font-mono">{{ site.info?.post_count ?? 0 }}</dd></div>
        <div class="flex justify-between"><dt class="text-dim">浏览</dt><dd class="font-mono">{{ site.info?.view_count ?? 0 }}</dd></div>
        <div class="flex justify-between"><dt class="text-dim">获赞</dt><dd class="font-mono">{{ site.info?.like_count ?? 0 }}</dd></div>
      </dl>
    </div>
  </section>

  <!-- 分类 Tab + 文章流 -->
  <section id="feed" class="mt-16 scroll-mt-20">
    <nav class="flex flex-wrap gap-2">
      <button
        class="rounded-full px-4 py-1.5 text-sm transition-colors"
        :class="activeCategory === '' ? 'bg-accent/20 text-accent' : 'glass text-dim hover:text-ink'"
        @click="selectCategory('')"
      >全部</button>
      <button
        v-for="c in categories"
        :key="c.id"
        class="rounded-full px-4 py-1.5 text-sm transition-colors"
        :class="activeCategory === c.name ? 'bg-accent/20 text-accent' : 'glass text-dim hover:text-ink'"
        @click="selectCategory(c.name)"
      >{{ c.name }} <span class="font-mono text-xs opacity-60">{{ c.post_count }}</span></button>
    </nav>

    <div v-if="feedError" class="glass mt-6 rounded-2xl p-10 text-center text-sm text-dim">
      加载失败，<button class="text-accent hover:underline" @click="loadFeed">重试</button>
    </div>
    <div v-else-if="loadingFeed && !feed.length" class="glass mt-6 rounded-2xl p-10 text-center text-sm text-dim">加载中…</div>
    <div v-else-if="!feed.length" class="glass mt-6 rounded-2xl p-10 text-center text-sm text-dim">这里还没有内容</div>
    <TransitionGroup v-else name="card" tag="div" class="mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <ArticleCard v-for="p in feed" :key="p.id" :post="p" />
    </TransitionGroup>

    <!-- 分页 -->
    <nav v-if="totalPages > 1" class="mt-8 flex items-center justify-center gap-4 text-sm">
      <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page <= 1" @click="turnPage(page - 1)">← 上一页</button>
      <span class="font-mono text-dim">{{ page }} / {{ totalPages }}</span>
      <button class="text-dim transition-colors hover:text-ink disabled:opacity-30" :disabled="page >= totalPages" @click="turnPage(page + 1)">下一页 →</button>
    </nav>
  </section>

  <DonateModal v-model="donateOpen" />
</template>

<style scoped>
.type-caret {
  display: inline-block;
  width: 2px;
  height: 1.1em;
  margin-left: 2px;
  background: var(--color-accent);
  vertical-align: text-bottom;
  animation: caret-blink 1s step-end infinite;
}

@keyframes caret-blink {
  50% { opacity: 0; }
}

.card-enter-active {
  transition: all 0.3s ease;
}

.card-enter-from {
  opacity: 0;
  transform: translateY(12px);
}
</style>
