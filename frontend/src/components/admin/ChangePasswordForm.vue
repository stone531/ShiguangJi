<!-- 修改密码表单：AdminLayout 顶栏弹窗与 Profile 页共用 -->
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, useMessage } from 'naive-ui'
import type { FormInst, FormItemRule, FormRules } from 'naive-ui'
import { changePassword } from '@/api/auth'
import { useAdminStore } from '@/stores/admin'

const emit = defineEmits<{ success: [] }>()
const message = useMessage()
const adminStore = useAdminStore()

const formRef = ref<FormInst | null>(null)
const saving = ref(false)
const model = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const rules: FormRules = {
  oldPassword: [{ required: true, message: '请输入旧密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    {
      validator: (_: FormItemRule, v: string) => v.length >= 8 && v.length <= 72,
      message: '密码长度 8-72 个字符',
      trigger: 'blur',
    },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_: FormItemRule, v: string) => v === model.newPassword,
      message: '两次输入的密码不一致',
      trigger: 'blur',
    },
  ],
}

function reset() {
  model.oldPassword = ''
  model.newPassword = ''
  model.confirmPassword = ''
  formRef.value?.restoreValidation()
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    await changePassword(model.oldPassword, model.newPassword)
    message.success('密码修改成功')
    adminStore.patchMustChangePassword()
    reset()
    emit('success')
  } catch (err) {
    message.error(err instanceof Error ? err.message : '修改失败')
  } finally {
    saving.value = false
  }
}

defineExpose({ reset })
</script>

<template>
  <n-form ref="formRef" :model="model" :rules="rules" label-placement="top">
    <n-form-item label="旧密码" path="oldPassword">
      <n-input v-model:value="model.oldPassword" type="password" show-password-on="click" />
    </n-form-item>
    <n-form-item label="新密码" path="newPassword">
      <n-input v-model:value="model.newPassword" type="password" show-password-on="click" />
    </n-form-item>
    <n-form-item label="确认新密码" path="confirmPassword">
      <n-input
        v-model:value="model.confirmPassword"
        type="password"
        show-password-on="click"
        @keyup.enter="submit"
      />
    </n-form-item>
    <n-button type="primary" block :loading="saving" :disabled="saving" @click="submit">
      确认修改
    </n-button>
  </n-form>
</template>
