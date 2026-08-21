<script setup lang="ts">
import { h, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NButton, NDataTable, NEmpty, NInput, NPopconfirm, NSelect, NTag, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { listPosts, deletePost, setPostStatus } from '@/api/post'
import type { Post } from '@/types/api'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const posts = ref<Post[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: false,
  onChange: (page: number) => {
    pagination.page = page
    load()
  },
})
const statusFilter = ref<number | null>(null)
const keyword = ref('')
const debouncedKeyword = ref('')

const statusOptions = [
  { label: '全部', value: -1 },
  { label: '草稿', value: 0 },
  { label: '已发布', value: 1 },
  { label: '回收站', value: 2 },
]

const typeLabel: Record<number, string> = { 1: '文章', 2: '图片集', 3: '视频' }
const statusLabel: Record<number, { text: string; type: 'default' | 'success' | 'error' }> = {
  0: { text: '草稿', type: 'default' },
  1: { text: '已发布', type: 'success' },
  2: { text: '回收站', type: 'error' },
}

// 竞态守卫：仅最新请求落 state
let seq = 0
async function load() {
  const current = ++seq
  loading.value = true
  try {
    const q = {
      page: pagination.page,
      size: pagination.pageSize,
      status: statusFilter.value === null || statusFilter.value < 0 ? undefined : statusFilter.value,
      keyword: debouncedKeyword.value || undefined,
    }
    const data = await listPosts(q)
    if (current !== seq) return // 过期响应丢弃
    posts.value = data.list
    pagination.itemCount = data.total
  } catch (e) {
    if (current !== seq) return
    message.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    if (current === seq) loading.value = false
  }
}

function onStatusFilter(value: number | null) {
  statusFilter.value = value === null || value < 0 ? null : value
  pagination.page = 1
  load()
}

// 关键词防抖 300ms（定时器清理红线）
let debounceTimer: number | null = null
function onKeywordInput(value: string) {
  keyword.value = value
  if (debounceTimer !== null) clearTimeout(debounceTimer)
  debounceTimer = window.setTimeout(() => {
    debounceTimer = null
    debouncedKeyword.value = value.trim()
    pagination.page = 1
    load()
  }, 300)
}
onBeforeUnmount(() => {
  if (debounceTimer !== null) clearTimeout(debounceTimer)
})

async function onToggleStatus(row: Post) {
  const target: 0 | 1 | 2 = row.status === 1 ? 2 : 1
  try {
    await setPostStatus(row.id, target)
    message.success(target === 1 ? '已发布' : '已移入回收站')
    // 当前页只剩这一条且不在第一页时回退页码，避免请求空页
    if (posts.value.length === 1 && pagination.page > 1) {
      pagination.page -= 1
    }
    load()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '操作失败')
  }
}

async function onDelete(row: Post) {
  try {
    await deletePost(row.id)
    message.success('已删除')
    // 当前页只剩这一条且不在第一页时回退页码，避免请求空页
    if (posts.value.length === 1 && pagination.page > 1) {
      pagination.page -= 1
    }
    load()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '删除失败')
  }
}

const columns: DataTableColumns<Post> = [
  {
    title: '标题',
    key: 'title',
    ellipsis: { tooltip: true },
    render: (row) =>
      h(
        NButton,
        { text: true, type: 'primary', onClick: () => router.push(`/admin/posts/${row.id}/edit`) },
        { default: () => row.title || '(无标题)' },
      ),
  },
  {
    title: '类型',
    key: 'type',
    width: 90,
    render: (row) => h(NTag, { size: 'small', bordered: false }, { default: () => typeLabel[row.type] ?? '未知' }),
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row) => {
      const s = statusLabel[row.status] ?? { text: '未知', type: 'default' as const }
      return h(NTag, { size: 'small', type: s.type }, { default: () => s.text })
    },
  },
  { title: '浏览', key: 'views', width: 70 },
  { title: '评论', key: 'comment_count', width: 70 },
  { title: '赞', key: 'like_count', width: 60 },
  { title: '更新时间', key: 'updated_at', width: 170, render: (row) => new Date(row.updated_at).toLocaleString() },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    render: (row) =>
      h('div', { class: 'row-actions' }, [
        h(NButton, { size: 'small', onClick: () => router.push(`/admin/posts/${row.id}/edit`) }, { default: () => '编辑' }),
        h(
          NButton,
          { size: 'small', type: row.status === 1 ? 'warning' : 'primary', onClick: () => onToggleStatus(row) },
          { default: () => (row.status === 1 ? '回收站' : '发布') },
        ),
        h(
          NPopconfirm,
          { onPositiveClick: () => onDelete(row) },
          {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }),
            default: () => `确认删除「${row.title || '(无标题)'}」？`,
          },
        ),
      ]),
  },
]

onMounted(load)
</script>

<template>
  <div class="post-list">
    <div class="toolbar">
      <n-select
        :value="statusFilter === null ? -1 : statusFilter"
        :options="statusOptions"
        style="width: 130px"
        @update:value="onStatusFilter"
      />
      <n-input
        :value="keyword"
        placeholder="关键词搜索"
        clearable
        style="width: 220px"
        @update:value="onKeywordInput"
      />
      <n-button type="primary" @click="router.push('/admin/posts/new')">新建文章</n-button>
    </div>
    <n-data-table
      remote
      :columns="columns"
      :data="posts"
      :loading="loading"
      :pagination="pagination"
      :row-key="(row: Post) => row.id"
    >
      <template #empty>
        <n-empty description="暂无文章" />
      </template>
    </n-data-table>
  </div>
</template>

<style scoped>
.post-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
}
.toolbar > :last-child {
  margin-left: auto;
}
.row-actions {
  display: flex;
  gap: 8px;
}
</style>
