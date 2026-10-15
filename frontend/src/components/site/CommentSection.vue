<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { createComment, listComments } from '@/api/public'
import { fmtDate } from '@/utils/date'
import type { CommentNode } from '@/types/api'

const props = defineProps<{ slug: string }>()

const comments = ref<CommentNode[]>([])
const total = ref(0)
const page = ref(1)
const size = 20
const loading = ref(false)

const form = reactive({
  nickname: '',
  email: '',
  content: '',
  /** 蜜罐：人类不可见不可填，机器人填了即被后端静默丢弃 */
  website: '',
})
const submitting = ref(false)
const successMsg = ref('')
const errorMsg = ref('')

onMounted(() => void reload())

/** 切换文章复用组件实例时由父组件 :key 重建，无需 watch slug */
async function reload() {
  loading.value = true
  try {
    const data = await listComments(props.slug, 1, size)
    comments.value = data.list
    total.value = data.total
    page.value = 1
  } catch {
    comments.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  const next = page.value + 1
  loading.value = true
  try {
    const data = await listComments(props.slug, next, size)
    comments.value.push(...data.list)
    page.value = next
  } catch {
    // 加载失败保持当前列表，用户可再次点击重试
  } finally {
    loading.value = false
  }
}

function validate(): string | null {
  const nickLen = [...form.nickname.trim()].length
  const contentLen = [...form.content.trim()].length
  if (nickLen < 1 || nickLen > 20) return '昵称需在 1-20 字符'
  if (contentLen < 1 || contentLen > 500) return '评论内容需在 1-500 字符'
  if (form.email && (!form.email.includes('@') || form.email.length > 100)) return '邮箱格式不正确'
  return null
}

async function submit() {
  successMsg.value = ''
  errorMsg.value = ''
  const invalid = validate()
  if (invalid) {
    errorMsg.value = invalid
    return
  }
  if (submitting.value) return
  submitting.value = true
  try {
    await createComment(props.slug, {
      nickname: form.nickname.trim(),
      email: form.email.trim() || undefined,
      content: form.content,
      website: form.website,
    })
    successMsg.value = '评论已提交，审核通过后展示 ✨'
    form.content = ''
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '提交失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="mt-12 border-t border-line pt-8">
    <h2 class="mb-6 text-sm font-semibold text-dim">
      评论 <span class="ml-1 text-accent">{{ total }}</span>
    </h2>

    <!-- 列表 -->
    <div v-if="loading && comments.length === 0" class="py-8 text-center text-sm text-dim">加载中…</div>
    <div v-else-if="comments.length === 0" class="py-8 text-center text-sm text-dim">
      还没有评论，来说点什么吧
    </div>
    <ul v-else class="space-y-6">
      <li v-for="c in comments" :key="c.id" class="border-l-2 border-line pl-4">
        <div class="flex items-center gap-2 text-sm">
          <span class="font-medium" :class="c.is_admin ? 'text-accent' : ''">{{ c.nickname }}</span>
          <span v-if="c.is_admin" class="rounded-full border border-accent/40 bg-accent/10 px-1.5 py-px text-[10px] text-accent">
            博主
          </span>
          <span class="text-xs text-dim">{{ fmtDate(c.created_at) }}</span>
        </div>
        <p class="mt-2 whitespace-pre-wrap break-words text-sm leading-relaxed">{{ c.content }}</p>

        <!-- 二级回复 -->
        <ul v-if="c.children.length" class="mt-4 space-y-4 border-l border-line/60 pl-4">
          <li v-for="ch in c.children" :key="ch.id">
            <div class="flex items-center gap-2 text-sm">
              <span class="font-medium" :class="ch.is_admin ? 'text-accent' : ''">{{ ch.nickname }}</span>
              <span
                v-if="ch.is_admin"
                class="rounded-full border border-accent/40 bg-accent/10 px-1.5 py-px text-[10px] text-accent"
              >博主</span>
              <span class="text-xs text-dim">{{ fmtDate(ch.created_at) }}</span>
            </div>
            <p class="mt-1.5 whitespace-pre-wrap break-words text-sm leading-relaxed text-dim">{{ ch.content }}</p>
          </li>
        </ul>
      </li>
    </ul>

    <button
      v-if="page * size < total"
      class="mx-auto mt-6 block rounded-full border border-line px-5 py-1.5 text-xs text-dim transition-colors hover:border-accent/40 hover:text-accent disabled:opacity-50"
      :disabled="loading"
      @click="loadMore"
    >加载更多评论</button>

    <!-- 表单 -->
    <form class="mt-10 border-t border-line pt-8" @submit.prevent="submit">
      <div class="grid gap-4 sm:grid-cols-2">
        <input
          v-model="form.nickname"
          type="text"
          placeholder="昵称 *"
          maxlength="20"
          class="w-full rounded-xl border border-line bg-white px-4 py-2.5 text-sm outline-none transition-colors placeholder:text-dim/60 focus:border-accent/50"
        >
        <input
          v-model="form.email"
          type="email"
          placeholder="邮箱（选填，不公开）"
          maxlength="100"
          class="w-full rounded-xl border border-line bg-white px-4 py-2.5 text-sm outline-none transition-colors placeholder:text-dim/60 focus:border-accent/50"
        >
      </div>
      <!-- 蜜罐：对人类隐藏，机器人默认填充会触发后端静默丢弃 -->
      <input v-model="form.website" type="text" name="website" tabindex="-1" autocomplete="off" class="hidden" aria-hidden="true">
      <textarea
        v-model="form.content"
        rows="4"
        :placeholder="'说点什么…（' + [...form.content].length + '/500）'"
        maxlength="500"
        class="mt-4 w-full resize-y rounded-xl border border-line bg-white px-4 py-3 text-sm outline-none transition-colors placeholder:text-dim/60 focus:border-accent/50"
      />
      <div class="mt-4 flex items-center justify-between">
        <p v-if="successMsg" class="text-xs text-accent">{{ successMsg }}</p>
        <p v-else-if="errorMsg" class="text-xs text-red-400">{{ errorMsg }}</p>
        <span v-else />
        <button
          type="submit"
          class="rounded-full bg-accent px-6 py-2 text-sm font-medium text-white transition-opacity hover:opacity-85 disabled:opacity-50"
          :disabled="submitting"
        >{{ submitting ? '提交中…' : '发表评论' }}</button>
      </div>
      <p class="mt-3 text-xs text-dim/70">评论需博主审核后展示 · 支持 Markdown 之外的纯文本换行</p>
    </form>
  </section>
</template>
