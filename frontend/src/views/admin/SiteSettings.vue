<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  NButton, NDynamicInput, NForm, NFormItem, NInput, NInputNumber,
  NSwitch, NUpload, NUploadDragger, useMessage,
} from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import { getSiteSettings, saveSiteSettings } from '@/api/site'
import { uploadImage } from '@/api/post'
import type { SiteAdminVO } from '@/types/api'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)

const form = reactive<SiteAdminVO>({
  site_name: '',
  site_desc: '',
  avatar: '',
  wechat_qr: '',
  alipay_qr: '',
  donate_text: '',
  social_links: [],
  page_size: 10,
  comment_enabled: true,
  sensitive_words: '',
})

async function load() {
  loading.value = true
  try {
    const data = await getSiteSettings()
    Object.assign(form, data)
    form.social_links = Array.isArray(data.social_links) ? data.social_links.map((l) => ({ ...l })) : []
  } catch (e) {
    message.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

// 通过真实上传接口上传图片并回填 URL
function makeUploadRequest(field: keyof SiteAdminVO) {
  return async ({ file, onFinish, onError }: {
    file: UploadFileInfo
    onFinish: () => void
    onError: () => void
  }) => {
    if (!file.file) {
      onError()
      return
    }
    try {
      const result = await uploadImage(file.file)
      ;(form as unknown as Record<string, unknown>)[field as string] = result.url
      onFinish()
      message.success('上传成功')
    } catch (e) {
      onError()
      message.error(e instanceof Error ? e.message : '上传失败')
    }
  }
}

const avatarUpload = { customRequest: makeUploadRequest('avatar') }
const wechatUpload = { customRequest: makeUploadRequest('wechat_qr') }
const alipayUpload = { customRequest: makeUploadRequest('alipay_qr') }

function onRemove(field: keyof SiteAdminVO) {
  ;(form as unknown as Record<string, unknown>)[field as string] = ''
}

function defaultSocialLink(): { name: string; url: string; icon: string } {
  return { name: 'GitHub', url: '', icon: 'github' }
}

async function onSave() {
  if (!form.site_name.trim()) {
    message.warning('站点名不能为空')
    return
  }
  saving.value = true
  try {
    await saveSiteSettings({
      ...form,
      social_links: form.social_links.filter((l) => l.name.trim() && l.url.trim()),
    })
    message.success('已保存')
  } catch (e) {
    message.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="site-settings" :class="{ loading }">
    <n-form label-placement="top" class="settings-form">
      <div class="grid-2">
        <n-form-item label="站点名" required>
          <n-input v-model:value="form.site_name" placeholder="时光集" maxlength="50" />
        </n-form-item>
        <n-form-item label="站点简介">
          <n-input v-model:value="form.site_desc" placeholder="记录技术与生活" maxlength="200" />
        </n-form-item>
      </div>

      <n-form-item label="头像">
        <div class="upload-row">
          <n-upload
            :max="1"
            :default-upload="true"
            :custom-request="avatarUpload.customRequest"
            :on-remove="() => onRemove('avatar')"
            accept="image/*"
            list-type="image-card"
          >
            <n-upload-dragger v-if="!form.avatar" class="avatar-uploader">点击上传头像</n-upload-dragger>
          </n-upload>
          <img v-if="form.avatar" :src="form.avatar" alt="头像预览" class="preview-avatar">
        </div>
      </n-form-item>

      <div class="grid-2">
        <n-form-item label="每页条数（1-50）">
          <n-input-number v-model:value="form.page_size" :min="1" :max="50" style="width: 100%" />
        </n-form-item>
        <n-form-item label="评论开关">
          <n-switch v-model:value="form.comment_enabled">
            <template #checked>已开启</template>
            <template #unchecked>已关闭</template>
          </n-switch>
        </n-form-item>
      </div>

      <n-form-item label="评论敏感词（换行分隔，实时生效；命中即拒评，另拦截超 2 条链接）">
        <n-input
          v-model:value="form.sensitive_words"
          type="textarea"
          :rows="5"
          maxlength="5000"
          placeholder="每行一个词，如：&#10;垃圾&#10;广告"
        />
      </n-form-item>

      <n-form-item label="赞赏文案">
        <n-input v-model:value="form.donate_text" placeholder="写得不错，请我喝杯咖啡 ☕" maxlength="100" />
      </n-form-item>

      <div class="grid-2">
        <n-form-item label="微信收款码">
          <div class="upload-row">
            <n-upload
              :max="1"
              :default-upload="true"
              :custom-request="wechatUpload.customRequest"
              :on-remove="() => onRemove('wechat_qr')"
              accept="image/*"
              list-type="image-card"
            >
              <n-upload-dragger v-if="!form.wechat_qr" class="qr-uploader">上传微信收款码</n-upload-dragger>
            </n-upload>
            <div v-if="form.wechat_qr" class="qr-card">
              <img :src="form.wechat_qr" alt="微信收款码预览">
              <n-button size="tiny" quaternary type="error" @click="onRemove('wechat_qr')">移除</n-button>
            </div>
          </div>
        </n-form-item>
        <n-form-item label="支付宝收款码">
          <div class="upload-row">
            <n-upload
              :max="1"
              :default-upload="true"
              :custom-request="alipayUpload.customRequest"
              :on-remove="() => onRemove('alipay_qr')"
              accept="image/*"
              list-type="image-card"
            >
              <n-upload-dragger v-if="!form.alipay_qr" class="qr-uploader">上传支付宝收款码</n-upload-dragger>
            </n-upload>
            <div v-if="form.alipay_qr" class="qr-card">
              <img :src="form.alipay_qr" alt="支付宝收款码预览">
              <n-button size="tiny" quaternary type="error" @click="onRemove('alipay_qr')">移除</n-button>
            </div>
          </div>
        </n-form-item>
      </div>

      <n-form-item label="社交链接">
        <n-dynamic-input v-model:value="form.social_links" :on-create="defaultSocialLink">
          <template #default="{ value }">
            <div class="social-row">
              <n-input v-model:value="value.name" placeholder="名称（如 GitHub）" maxlength="30" />
              <n-input v-model:value="value.url" placeholder="https://..." maxlength="200" />
              <n-input v-model:value="value.icon" placeholder="图标（如 github）" maxlength="30" />
            </div>
          </template>
        </n-dynamic-input>
      </n-form-item>

      <div class="form-footer">
        <n-button type="primary" :loading="saving" @click="onSave">保存设置</n-button>
      </div>
    </n-form>
  </div>
</template>

<style scoped>
.site-settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 860px;
}
.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 24px;
}
.upload-row {
  display: flex;
  gap: 16px;
  align-items: center;
}
.preview-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
}
.qr-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
}
.qr-card img {
  width: 120px;
  height: 120px;
  object-fit: cover;
  border-radius: 4px;
}
.avatar-uploader,
.qr-uploader {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: #888;
}
.social-row {
  display: flex;
  flex: 1;
  gap: 8px;
  width: 100%;
}
.form-footer {
  display: flex;
  justify-content: flex-end;
}
</style>
