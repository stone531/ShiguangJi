<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue'
import {
  NAlert, NButton, NDataTable, NDrawer, NDrawerContent, NEmpty, NInput, NModal,
  NPopconfirm, NTag, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { listComments, reviewComment, replyComment, deleteComment } from '@/api/comment'
import type { CommentItem } from '@/types/api'

const message = useMessage()

const loading = ref(false)
const comments = ref<CommentItem[]>([])
const activeStatus = ref<0 | 1 | 2>(0)
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

// 竞态守卫：仅最新请求落 state
let seq = 0
async function load() {
  const current = ++seq
  loading.value = true
  try {
    const data = await listComments({ status: activeStatus.value, page: pagination.page, size: pagination.pageSize })
    if (current !== seq) return // 过期响应丢弃
    comments.value = data.list
    pagination.itemCount = data.total
  } catch (e) {
    if (current !== seq) return
    message.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    if (current === seq) loading.value = false
  }
}

function onTabChange(status: number) {
  activeStatus.value = status as 0 | 1 | 2
  pagination.page = 1
  load()
}

// 查看全文抽屉
const drawerVisible = ref(false)
const detail = ref<CommentItem | null>(null)
function openDetail(row: CommentItem) {
  detail.value = row
  drawerVisible.value = true
}

// 回复弹窗
const replyVisible = ref(false)
const replyTarget = ref<CommentItem | null>(null)
const replyContent = ref('')
function openReply(row: CommentItem) {
  replyTarget.value = row
  replyContent.value = ''
  replyVisible.value = true
}
async function submitReply() {
  if (!replyTarget.value) return
  const content = replyContent.value.trim()
  if (!content) {
    message.warning('回复内容不能为空')
    return
  }
  try {
    await replyComment(replyTarget.value.id, content)
    message.success('回复已提交')
    replyVisible.value = false
    load()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '回复失败')
  }
}

async function onReview(row: CommentItem, status: 1 | 2) {
  try {
    await reviewComment(row.id, status)
    message.success(status === 1 ? '已通过' : '已驳回')
    if (comments.value.length === 1 && pagination.page > 1) pagination.page -= 1
    load()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '操作失败')
  }
}

async function onDelete(row: CommentItem) {
  try {
    await deleteComment(row.id)
    message.success('已删除')
    if (comments.value.length === 1 && pagination.page > 1) pagination.page -= 1
    load()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '删除失败')
  }
}

function truncate(text: string, max = 60): string {
  return text.length > max ? `${text.slice(0, max)}…` : text
}

const statusTag: Record<number, { text: string; type: 'warning' | 'success' | 'error' }> = {
  0: { text: '待审核', type: 'warning' },
  1: { text: '已通过', type: 'success' },
  2: { text: '已驳回', type: 'error' },
}

const columns: DataTableColumns<CommentItem> = [
  {
    title: '昵称',
    key: 'nickname',
    width: 120,
    render: (row) =>
      h('div', { class: 'nick-cell' }, [
        h('span', null, row.nickname),
        row.is_admin ? h(NTag, { size: 'tiny', type: 'info', bordered: false }, { default: () => '站长' }) : null,
      ]),
  },
  {
    title: '内容',
    key: 'content',
    render: (row) =>
      h(
        NButton,
        { text: true, class: 'content-cell', onClick: () => openDetail(row) },
        { default: () => truncate(row.content) },
      ),
  },
  { title: '时间', key: 'created_at', width: 170, render: (row) => new Date(row.created_at).toLocaleString() },
  {
    title: '操作',
    key: 'actions',
    width: 240,
    render: (row) =>
      h('div', { class: 'row-actions' }, [
        row.status !== 1
          ? h(NButton, { size: 'small', type: 'primary', onClick: () => onReview(row, 1) }, { default: () => '通过' })
          : null,
        row.status !== 2
          ? h(NButton, { size: 'small', type: 'warning', onClick: () => onReview(row, 2) }, { default: () => '驳回' })
          : null,
        h(NButton, { size: 'small', onClick: () => openReply(row) }, { default: () => '回复' }),
        h(
          NPopconfirm,
          { onPositiveClick: () => onDelete(row) },
          {
            trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '删除' }),
            default: () => `确认删除「${row.nickname}」的评论？`,
          },
        ),
      ]),
  },
]

onMounted(load)
</script>

<template>
  <div class="comment-manage">
    <n-alert type="info" :show-icon="true">
      评论接口后端将在 M4 提供，当前展示示例数据
    </n-alert>
    <n-tabs :value="activeStatus" type="line" @update:value="onTabChange">
      <n-tab v-for="s in [0, 1, 2]" :key="s" :name="s" :tab="statusTag[s].text" />
    </n-tabs>
    <n-data-table
      remote
      :columns="columns"
      :data="comments"
      :loading="loading"
      :pagination="pagination"
      :row-key="(row: CommentItem) => row.id"
    >
      <template #empty>
        <n-empty description="暂无评论" />
      </template>
    </n-data-table>

    <n-drawer v-model:show="drawerVisible" :width="420">
      <n-drawer-content :title="detail ? `评论详情 · ${detail.nickname}` : '评论详情'">
        <div v-if="detail" class="detail-body">
          <div class="detail-meta">
            <n-tag size="small" :type="statusTag[detail.status]?.type ?? 'default'">
              {{ statusTag[detail.status]?.text ?? '未知' }}
            </n-tag>
            <span>{{ new Date(detail.created_at).toLocaleString() }}</span>
          </div>
          <div class="detail-block">
            <div class="detail-label">所属文章</div>
            <div>{{ detail.post_title }}（#{{ detail.post_id }}）</div>
          </div>
          <div class="detail-block">
            <div class="detail-label">评论内容</div>
            <div class="detail-content">{{ detail.content }}</div>
          </div>
          <div v-if="detail.parent_id > 0" class="detail-block">
            <div class="detail-label">回复对象</div>
            <div>顶级评论 #{{ detail.parent_id }}</div>
          </div>
        </div>
      </n-drawer-content>
    </n-drawer>

    <n-modal
      v-model:show="replyVisible"
      preset="dialog"
      title="回复评论"
      positive-text="提交回复"
      negative-text="取消"
      @positive-click="submitReply"
    >
      <div v-if="replyTarget" class="reply-quote">回复 {{ replyTarget.nickname }}：{{ truncate(replyTarget.content, 40) }}</div>
      <n-input
        v-model:value="replyContent"
        type="textarea"
        placeholder="输入回复内容（500 字以内）"
        maxlength="500"
        show-count
        :rows="4"
      />
    </n-modal>
  </div>
</template>

<style scoped>
.comment-manage {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.row-actions {
  display: flex;
  gap: 8px;
}
.nick-cell {
  display: flex;
  gap: 6px;
  align-items: center;
}
.content-cell {
  text-align: left;
  white-space: normal;
}
.detail-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.detail-meta {
  display: flex;
  gap: 12px;
  align-items: center;
  color: #888;
}
.detail-label {
  font-size: 12px;
  color: #888;
  margin-bottom: 4px;
}
.detail-content {
  white-space: pre-wrap;
  word-break: break-all;
}
.reply-quote {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: rgba(128, 128, 128, 0.12);
  border-radius: 4px;
  color: #aaa;
  font-size: 13px;
}
</style>
