<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NAlert, NCard, NEmpty, NGrid, NGridItem, NSpin, useMessage } from 'naive-ui'
import { get } from '@/api/http'
import { listComments } from '@/api/comment'
import type { AdminCommentVO, StatsVO } from '@/types/api'

const message = useMessage()
const router = useRouter()
const loading = ref(true)
const stats = ref<StatsVO>({
  post_total: 0,
  post_published: 0,
  comment_total: 0,
  comment_pending: 0,
  view_total: 0,
  like_total: 0,
})
const recentComments = ref<AdminCommentVO[]>([])

/** 竞态守卫：只有最新一次请求的响应才落 state */
let seq = 0

async function load() {
  const current = ++seq
  loading.value = true
  try {
    // 六指标与待审列表并行取，互不依赖
    const [statsData, pending] = await Promise.all([
      get<StatsVO>('/admin/stats'),
      listComments({ status: 0, page: 1, size: 5 }).catch(() => null),
    ])
    if (current !== seq) return
    stats.value = statsData
    recentComments.value = pending?.list ?? []
  } catch (err) {
    if (current !== seq) return
    message.error(err instanceof Error ? err.message : '统计加载失败')
  } finally {
    if (current === seq) loading.value = false
  }
}

function goComments() {
  router.push('/admin/comments')
}

onMounted(load)
</script>

<template>
  <div class="dashboard">
    <n-alert
      v-if="stats.comment_pending > 0"
      type="warning"
      :bordered="false"
      style="margin-bottom: 16px; cursor: pointer"
      @click="goComments"
    >
      有 {{ stats.comment_pending }} 条评论待审核，点击前往处理
    </n-alert>

    <n-spin :show="loading">
      <n-grid :cols="4" :x-gap="16" :y-gap="16" item-responsive responsive="screen">
        <n-grid-item span="4 s:2 l:1">
          <n-card title="文章总数">
            <div class="stat-value">{{ stats.post_total }}</div>
            <div class="stat-sub">已发布 {{ stats.post_published }}</div>
          </n-card>
        </n-grid-item>
        <n-grid-item span="4 s:2 l:1">
          <n-card title="评论总数">
            <div class="stat-value">{{ stats.comment_total }}</div>
            <div class="stat-sub">待审核 {{ stats.comment_pending }}</div>
          </n-card>
        </n-grid-item>
        <n-grid-item span="4 s:2 l:1">
          <n-card title="总浏览">
            <div class="stat-value">{{ stats.view_total }}</div>
          </n-card>
        </n-grid-item>
        <n-grid-item span="4 s:2 l:1">
          <n-card title="总点赞">
            <div class="stat-value">{{ stats.like_total }}</div>
          </n-card>
        </n-grid-item>
      </n-grid>
    </n-spin>

    <n-card title="最近待审评论" style="margin-top: 16px">
      <n-empty v-if="recentComments.length === 0" description="暂无待审核评论" />
      <ul v-else class="comment-list">
        <li v-for="c in recentComments" :key="c.id" @click="goComments">
          <span class="comment-nickname">{{ c.nickname }}</span>
          <span class="comment-content">{{ c.content }}</span>
          <span class="comment-time">{{ new Date(c.created_at).toLocaleString() }}</span>
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

.stat-sub {
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.6;
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
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  cursor: pointer;
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
  color: #6a737d;
  font-size: 12px;
  flex-shrink: 0;
}
</style>
