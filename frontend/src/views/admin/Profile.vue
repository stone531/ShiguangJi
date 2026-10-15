<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { NAvatar, NButton, NForm, NFormItem, NInput, NUpload, useMessage } from 'naive-ui'
import type { UploadCustomRequestOptions } from 'naive-ui'
import { getProfile, updateProfile } from '@/api/auth'
import { uploadImage } from '@/api/post'
import ChangePasswordForm from '@/components/admin/ChangePasswordForm.vue'
import { useAdminStore } from '@/stores/admin'

const message = useMessage()
const adminStore = useAdminStore()

const username = ref('')
const profile = reactive({ nickname: '', avatar: '' })
const saving = ref(false)

onMounted(async () => {
  try {
    const p = await getProfile()
    username.value = p.username
    profile.nickname = p.nickname
    profile.avatar = p.avatar
  } catch (e) {
    message.error(e instanceof Error ? e.message : '资料加载失败')
  }
})

// 头像上传（自定义请求调 uploadImage，复用 /admin/upload）
function avatarCustomRequest({ file, onFinish, onError }: UploadCustomRequestOptions) {
  const raw = file.file
  if (!raw) {
    onError()
    return
  }
  uploadImage(raw)
    .then((r) => {
      profile.avatar = r.url
      message.success('头像上传成功，记得保存')
      onFinish()
    })
    .catch((e: unknown) => {
      message.error(e instanceof Error ? e.message : '头像上传失败')
      onError()
    })
}

async function save() {
  const nickname = profile.nickname.trim()
  if (!nickname) {
    message.error('昵称不能为空')
    return
  }
  saving.value = true
  try {
    await updateProfile(nickname, profile.avatar)
    // 同步顶栏昵称/头像（保留 must_change_password 标记）
    if (adminStore.info) {
      adminStore.set({ ...adminStore.info, nickname, avatar: profile.avatar })
    }
    message.success('资料已保存')
  } catch (e) {
    message.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="profile-page">
    <div class="section">
      <h2>个人资料</h2>
      <n-form label-placement="top" class="profile-form">
        <n-form-item label="头像">
          <div class="avatar-row">
            <n-upload :custom-request="avatarCustomRequest" :max="1" accept="image/*" list-type="image-card">
              <div v-if="!profile.avatar" class="avatar-placeholder">点击上传</div>
            </n-upload>
            <n-avatar v-if="profile.avatar" round :size="64" :src="profile.avatar">
              {{ profile.nickname.slice(0, 1) }}
            </n-avatar>
          </div>
        </n-form-item>
        <n-form-item label="用户名">
          <n-input :value="username" disabled />
        </n-form-item>
        <n-form-item label="昵称" required>
          <n-input v-model:value="profile.nickname" placeholder="评论回复时展示的名称" maxlength="50" />
        </n-form-item>
        <n-button type="primary" :loading="saving" @click="save">保存资料</n-button>
      </n-form>
    </div>

    <div class="section">
      <h2>修改密码</h2>
      <ChangePasswordForm class="profile-form" />
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  max-width: 560px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.section {
  border: 1px solid rgba(0, 0, 0, 0.09);
  border-radius: 8px;
  padding: 16px;
}
.section h2 {
  margin: 0 0 16px;
}
.profile-form {
  max-width: 360px;
}
.avatar-row {
  display: flex;
  align-items: center;
  gap: 16px;
}
.avatar-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
}
</style>
