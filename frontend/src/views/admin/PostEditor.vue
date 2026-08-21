<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import {
  NButton, NForm, NFormItem, NInput, NRadioButton, NRadioGroup,
  NSelect, NSwitch, NTag, NUpload, useDialog, useMessage,
} from 'naive-ui'
import type { UploadCustomRequestOptions } from 'naive-ui'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import { getPost, createPost, updatePost, uploadImage } from '@/api/post'
import type { PostPayload } from '@/api/post'
import { useAutoSave } from '@/composables/useAutoSave'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const isEdit = computed(() => route.params.id !== undefined)
const postId = computed(() => (isEdit.value ? Number(route.params.id) : null))

const form = reactive({
  title: '',
  slug: '',
  summary: '',
  cover: '',
  content: '',
  type: 1 as 1 | 2 | 3,
  video_url: '',
  // TODO(M4): 分类接口就绪后接入真实选项，当前静态占位
  category_id: null as number | null,
  is_top: false,
  is_featured: false,
})

const categoryOptions = [
  // TODO(M4): 替换为 /api/categories 真实数据
  { label: '未分类（占位）', value: 0, disabled: true },
]

const slugError = ref('')
function validateSlug() {
  if (!form.slug) {
    slugError.value = 'slug 必填（用于文章 URL，如 my-first-post）'
  } else if (!/^[a-z0-9-]+$/.test(form.slug)) {
    slugError.value = 'slug 只能包含小写字母、数字和连字符'
  } else {
    slugError.value = ''
  }
}

const ALLOWED_VIDEO_HOSTS = ['player.bilibili.com', 'www.youtube.com', 'www.youtube-nocookie.com', 'player.youku.com']
const videoUrlError = ref('')
function validateVideoUrl() {
  if (form.type !== 3 || !form.video_url) {
    videoUrlError.value = ''
    return
  }
  try {
    const host = new URL(form.video_url).hostname
    videoUrlError.value = ALLOWED_VIDEO_HOSTS.includes(host)
      ? ''
      : '仅支持 player.bilibili.com / www.youtube.com / www.youtube-nocookie.com / player.youku.com 域名的嵌入地址'
  } catch {
    videoUrlError.value = '请输入合法的视频嵌入 URL（以 http(s):// 开头）'
  }
}

// 封面上传（自定义请求调 uploadImage）
function coverCustomRequest({ file, onFinish, onError }: UploadCustomRequestOptions) {
  const raw = file.file
  if (!raw) {
    onError()
    return
  }
  uploadImage(raw)
    .then((r) => {
      form.cover = r.url
      message.success('封面上传成功')
      onFinish()
    })
    .catch((e: unknown) => {
      message.error(e instanceof Error ? e.message : '封面上传失败')
      onError()
    })
}

// 编辑器图片上传：md-editor-v3 onUploadImg
async function handleUploadImg(files: File[], callback: (urls: string[]) => void) {
  const urls: string[] = []
  for (const f of files) {
    try {
      const r = await uploadImage(f)
      urls.push(r.thumb_url || r.url)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '图片上传失败')
    }
  }
  callback(urls)
}

// ---------- 自动保存（防丢稿） ----------
const draftKey = computed(() => (isEdit.value ? `post:${postId.value}` : 'post:new'))
const { readDraft, writeDraft, clearDraft, start, lastSavedAt } = useAutoSave(draftKey.value)
// 组件实例不因 :id 变化重建（同一组件复用），draftKey 用固定值注册一次即可满足本路由场景

const savedSnapshot = ref('')
function snapshot(): string {
  return JSON.stringify(form)
}
function isDirty(): boolean {
  return snapshot() !== savedSnapshot.value
}

function beginAutoSave() {
  savedSnapshot.value = snapshot()
  start(() => ({ title: form.title, content: form.content }))
}

// 挂载：编辑态先回填服务器内容，再检测本地草稿
onMounted(async () => {
  if (isEdit.value && postId.value !== null) {
    try {
      const p = await getPost(postId.value)
      form.title = p.title
      form.slug = p.slug
      form.summary = p.summary
      form.cover = p.cover
      form.content = p.content
      form.type = p.type
      form.video_url = p.video_url
      form.category_id = p.category_id
      form.is_top = p.is_top
      form.is_featured = p.is_featured
    } catch (e) {
      message.error(e instanceof Error ? e.message : '文章加载失败')
    }
  }
  const draft = readDraft()
  if (draft && (draft.title !== form.title || draft.content !== form.content)) {
    const when = new Date(draft.savedAt).toLocaleTimeString()
    dialog.warning({
      title: '检测到未保存草稿',
      content: `存在 ${when} 保存的本地草稿（标题：${draft.title || '无'}），是否恢复？`,
      positiveText: '恢复草稿',
      negativeText: '丢弃',
      onPositiveClick: () => {
        form.title = draft.title
        form.content = draft.content
        beginAutoSave()
      },
      onNegativeClick: () => {
        clearDraft()
        beginAutoSave()
      },
    })
  } else {
    beginAutoSave()
  }
})

const lastSavedAtText = computed(() =>
  lastSavedAt.value ? `上次自动保存于 ${new Date(lastSavedAt.value).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}` : '',
)

// ---------- 保存 ----------
const saving = ref(false)
function buildPayload(status: 0 | 1): PostPayload {
  return {
    title: form.title,
    slug: form.slug,
    summary: form.summary,
    cover: form.cover,
    content: form.content,
    type: form.type,
    video_url: form.video_url,
    category_id: null, // TODO(M4): 分类接口接入后传真实值
    status,
    is_top: form.is_top,
    is_featured: form.is_featured,
    tag_ids: [], // TODO(M4): 标签接口接入后传真实值
  }
}

async function save(status: 0 | 1) {
  if (!form.title.trim()) {
    message.error('标题不能为空')
    return
  }
  validateSlug()
  validateVideoUrl()
  if (slugError.value || videoUrlError.value) {
    message.error('请修正表单错误后再保存')
    return
  }
  saving.value = true
  try {
    if (isEdit.value && postId.value !== null) {
      await updatePost(postId.value, buildPayload(status))
      message.success(status === 1 ? '已发布' : '已存草稿')
    } else {
      const created = await createPost(buildPayload(status))
      message.success(status === 1 ? '已发布' : '已存草稿')
      // H1：先清草稿并同步 snapshot，再 replace，避免 onBeforeRouteLeave 误弹「未保存的更改」
      clearDraft()
      savedSnapshot.value = snapshot()
      router.replace(`/admin/posts/${created.id}/edit`)
      return
    }
    clearDraft()
    savedSnapshot.value = snapshot()
  } catch (e) {
    // 保存失败（含 401）：立即本地兜底，防丢稿
    writeDraft(form.title, form.content)
    message.warning('保存失败，内容已本地暂存')
  } finally {
    saving.value = false
  }
}

// ---------- 离开守卫 ----------
onBeforeRouteLeave(() => {
  if (!isDirty()) return true
  return new Promise<boolean>((resolve) => {
    dialog.warning({
      title: '未保存的更改',
      content: '当前文章有未保存的修改，确定离开吗？',
      positiveText: '离开',
      negativeText: '留下',
      onPositiveClick: () => resolve(true),
      onNegativeClick: () => resolve(false),
      onClose: () => resolve(false),
      onMaskClick: () => resolve(false),
    })
  })
})
</script>

<template>
  <div class="post-editor">
    <div class="editor-header">
      <h2>{{ isEdit ? '编辑文章' : '新建文章' }}</h2>
      <div class="header-right">
        <span v-if="lastSavedAtText" class="save-hint">{{ lastSavedAtText }}</span>
        <n-button :loading="saving" @click="save(0)">存草稿</n-button>
        <n-button type="primary" :loading="saving" @click="save(1)">发布</n-button>
      </div>
    </div>

    <div class="meta-grid">
      <n-form label-placement="top">
        <n-form-item label="标题" required>
          <n-input v-model:value="form.title" placeholder="文章标题" maxlength="200" />
        </n-form-item>
        <n-form-item label="Slug" :validation-status="slugError ? 'error' : undefined" :feedback="slugError">
          <n-input v-model:value="form.slug" placeholder="my-first-post" @blur="validateSlug" />
        </n-form-item>
        <n-form-item label="摘要">
          <n-input v-model:value="form.summary" type="textarea" :rows="2" maxlength="500" placeholder="可空" />
        </n-form-item>
        <n-form-item label="类型">
          <n-radio-group v-model:value="form.type">
            <n-radio-button :value="1">文章</n-radio-button>
            <n-radio-button :value="2">图片集</n-radio-button>
            <n-radio-button :value="3">视频</n-radio-button>
          </n-radio-group>
        </n-form-item>
        <n-form-item
          v-if="form.type === 3"
          label="视频嵌入地址"
          :validation-status="videoUrlError ? 'error' : undefined"
          :feedback="videoUrlError"
        >
          <n-input v-model:value="form.video_url" placeholder="https://player.bilibili.com/player.html?..." @blur="validateVideoUrl" />
        </n-form-item>
        <n-form-item label="分类">
          <!-- TODO(M4): 分类接口就绪后接入真实数据，提交时 category_id 传 null -->
          <n-select v-model:value="form.category_id" :options="categoryOptions" placeholder="分类（M4 接入）" clearable disabled />
        </n-form-item>
        <n-form-item label="标签">
          <!-- TODO(M4): 标签接口就绪后接入，提交时 tag_ids 传 [] -->
          <n-tag :bordered="false" type="default">标签选择将在 M4 提供</n-tag>
        </n-form-item>
        <div class="switch-row">
          <n-form-item label="置顶">
            <n-switch v-model:value="form.is_top" />
          </n-form-item>
          <n-form-item label="推荐">
            <n-switch v-model:value="form.is_featured" />
          </n-form-item>
        </div>
        <n-form-item label="封面">
          <div class="cover-row">
            <n-upload :custom-request="coverCustomRequest" :max="1" accept="image/*" list-type="image-card">
              <div v-if="!form.cover" class="cover-placeholder">点击上传</div>
            </n-upload>
            <img v-if="form.cover" :src="form.cover" class="cover-preview" alt="封面预览" />
          </div>
        </n-form-item>
      </n-form>
    </div>

    <MdEditor
      v-model="form.content"
      theme="dark"
      language="zh-CN"
      style="height: 60vh"
      @on-upload-img="handleUploadImg"
    />
  </div>
</template>

<style scoped>
.post-editor {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.editor-header h2 {
  margin: 0;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.save-hint {
  font-size: 12px;
  opacity: 0.7;
}
.meta-grid {
  border: 1px solid rgba(255, 255, 255, 0.09);
  border-radius: 8px;
  padding: 16px;
}
.switch-row {
  display: flex;
  gap: 32px;
}
.cover-row {
  display: flex;
  align-items: center;
  gap: 16px;
}
.cover-preview {
  max-width: 200px;
  max-height: 120px;
  border-radius: 6px;
}
.cover-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
}
</style>
