<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { NAlert, NCard, NEmpty, NGrid, NGridItem, NSpin, useMessage } from 'naive-ui'
import { get } from '@/api/http'

interface StatsVO {
  post_count?: number
  comment_count?: number
  like_count?: number
  total_views?: number
  /** M1 后端占位字段：存在时表示统计未实现，前端展示占位 0 */
  todo?: unknown
}

interface RecentComment {
  id: number
  nickname: string
  content: string
  created_at: string
}

const message = useMessage()
const loading = ref(true)
const stats = ref<StatsVO>({ post_count: 0, comment_count: 0, like_count: 0, total_views: 0 })
const recentComments = ref<RecentComment[]>([])

/** 竞态守卫：只有最新一次请求的响应才落 state */
let seq = 0

async function load() {
  const current = ++seq
  loading.value = true
  try {
    const data = await get<StatsVO>('/admin/stats')
    if (current !== seq) return // 过期响应丢弃
    stats.value = data
  } catch (err) {
    if (current !== seq) return
    message.error(err instanceof Error ? err.message : '统计加载失败')
  } finally {
    if (current === seq) loading.value = false
  }
}

function statValue(v: number | undefined): number {
  // M2：后端 stats 为 {todo} 占位，统一显示 0
  return v ?? 0
}

onMounted(load)
</script>

<template>
  <div class="dashboard">
    <n-alert type="info" :bordered="false" style="margin-bottom: 16px">
      统计接口将在 M4 提供，当前为占位数据
    </n-alert>

    <n-spin :show="loading">
      <n-grid :cols="4" :x-gap="16" :y-gap="16" item-responsive responsive="screen">
        <n-grid-item span="4 s:2 l:1">
          <n-card title="文章数">
            <div class="stat-value">{{ statValue(stats.post_count) }}</div>
          </n-card>
        </n-grid-item>
        <n-grid-item span="4 s:2 l:1">
          <n-card title="评论数">
            <div class="stat-value">{{ statValue(stats.comment_count) }}</div>
          </n-card>
        </n-grid-item>
        <n-grid-item span="4 s:2 l:1">
          <n-card title="点赞数">
            <div class="stat-value">{{ statValue(stats.like_count) }}</div>
          </n-card>
        </n-grid-item>
        <n-grid-item span="4 s:2 l:1">
          <n-card title="总浏览">
            <div class="stat-value">{{ statValue(stats.total_views) }}</div>
          </n-card>
        </n-grid-item>
      </n-grid>
    </n-spin>

    <n-card title="最近待审评论" style="margin-top: 16px">
      <n-empty v-if="recentComments.length === 0" description="暂无待审核评论（M4 数据接入）" />
      <ul v-else class="comment-list">
        <li v-for="c in recentComments" :key="c.id">
          <span class="comment-nickname">{{ c.nickname }}</span>
          <span class="comment-content">{{ c.content }}</span>
          <span class="comment-time">{{ c.created_at }}</span>
        </li>
      </ul>
    </n-card>
  </div>
</template>

<style scoped>
.stat-value {
  font-size: 32px;
  font-weight: 600;
}

.comment-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.comment-list li {
  display: flex;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.comment-nickname {
  font-weight: 500;
  flex-shrink: 0;
}

.comment-content {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.comment-time {
  color: rgba(255, 255, 255, 0.45);
  font-size: 12px;
  flex-shrink: 0;
}
</style>
