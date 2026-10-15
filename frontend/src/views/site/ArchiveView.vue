<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getArchive } from '@/api/public'
import type { ArchiveGroup } from '@/types/api'

/** 归档：按年月分组的时间线（spec §6.1） */
const groups = ref<ArchiveGroup[]>([])
const loading = ref(true)
const failed = ref(false)

async function load() {
  loading.value = true
  failed.value = false
  try {
    groups.value = (await getArchive()).list
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <header class="mt-12 text-center">
    <h1 class="text-2xl font-bold">归档</h1>
    <p class="mt-2 text-xs text-dim">共 {{ groups.reduce((n, g) => n + g.posts.length, 0) }} 篇</p>
  </header>

  <div v-if="failed" class="mt-8 py-10 text-center text-sm text-dim">
    加载失败，<button class="text-accent hover:underline" @click="load">重试</button>
  </div>
  <div v-else-if="loading" class="mt-8 py-10 text-center text-sm text-dim">加载中…</div>
  <div v-else-if="!groups.length" class="mt-8 py-10 text-center text-sm text-dim">还没有发布过内容</div>

  <section v-else class="archive-timeline mx-auto mt-10 max-w-2xl">
    <div v-for="group in groups" :key="`${group.year}-${group.month}`" class="relative pl-8">
      <h2 class="font-display sticky top-0 z-10 mb-4 inline-block rounded bg-void/90 px-2 py-1 text-lg font-semibold">
        {{ group.year }} 年 {{ group.month }} 月
        <span class="ml-1 font-mono text-xs text-dim">{{ group.posts.length }}</span>
      </h2>
      <ul>
        <li v-for="p in group.posts" :key="p.id">
          <RouterLink
            :to="`/post/${p.slug}`"
            class="group flex items-baseline gap-4 rounded-lg px-2 py-2.5 transition-colors hover:bg-black/[0.03]"
          >
            <span class="font-mono text-xs text-dim">{{ new Date(p.published_at).toLocaleDateString('zh-CN') }}</span>
            <span class="truncate transition-colors group-hover:text-accent">{{ p.title }}</span>
          </RouterLink>
        </li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
/* 时间线：左竖线 + 年月分组节点 */
.archive-timeline > div {
  border-left: 1px solid var(--color-line);
  margin-bottom: 2rem;
}

.archive-timeline > div::before {
  content: '';
  position: absolute;
  left: -4px;
  top: 8px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-accent);
}
</style>
