<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NCard,
  NForm,
  NFormItem,
  NInput,
  NButton,
  NModal,
  useMessage,
  type FormInst,
  type FormRules,
  type FormItemRule,
} from 'naive-ui'
import { login, changePassword } from '@/api/auth'
import { useAdminStore } from '@/stores/admin'

const router = useRouter()
const message = useMessage()
const adminStore = useAdminStore()

const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const model = reactive({
  username: '',
  password: '',
})

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    {
      validator: (_: FormItemRule, v: string) => v.length >= 3 && v.length <= 50,
      message: '用户名长度 3-50 个字符',
      trigger: 'blur',
    },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    {
      validator: (_: FormItemRule, v: string) => v.length >= 8 && v.length <= 72,
      message: '密码长度 8-72 个字符',
      trigger: 'blur',
    },
  ],
}

async function handleLogin() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const info = await login(model.username, model.password)
    adminStore.set(info)
    model.password = ''
    if (info.must_change_password) {
      pwdModal.show = true
    } else {
      router.push('/admin/dashboard')
    }
  } catch (err) {
    message.error(err instanceof Error ? err.message : '登录失败')
  } finally {
    loading.value = false
  }
}

/* ----- 强制改密弹窗 ----- */
const pwdModal = reactive({ show: false, loading: false })
const pwdFormRef = ref<FormInst | null>(null)
const pwdModel = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const pwdRules: FormRules = {
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
      validator: (_: FormItemRule, v: string) => v === pwdModel.newPassword,
      message: '两次输入的密码不一致',
      trigger: 'blur',
    },
  ],
}

async function handleChangePassword() {
  const valid = await pwdFormRef.value?.validate().catch(() => false)
  if (!valid) return
  pwdModal.loading = true
  try {
    await changePassword(pwdModel.oldPassword, pwdModel.newPassword)
    message.success('密码修改成功')
    pwdModal.show = false
    adminStore.patchMustChangePassword()
    pwdModel.oldPassword = ''
    pwdModel.newPassword = ''
    pwdModel.confirmPassword = ''
    router.push('/admin/dashboard')
  } catch (err) {
    message.error(err instanceof Error ? err.message : '修改失败')
  } finally {
    pwdModal.loading = false
  }
}
</script>

<template>
  <div class="login-page">
    <n-card class="login-card" title="时光集">
      <n-form ref="formRef" :model="model" :rules="rules" label-placement="top">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="model.username" placeholder="用户名" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="model.password"
            type="password"
            show-password-on="click"
            placeholder="密码"
            @keyup.enter="handleLogin"
          />
        </n-form-item>
        <n-button type="primary" block :loading="loading" :disabled="loading" @click="handleLogin">
          登 录
        </n-button>
      </n-form>
    </n-card>

    <n-modal
      v-model:show="pwdModal.show"
      :mask-closable="false"
      :close-on-esc="false"
      preset="card"
      title="首次登录请修改密码"
      style="width: 420px"
    >
      <n-form ref="pwdFormRef" :model="pwdModel" :rules="pwdRules" label-placement="top">
        <n-form-item label="旧密码" path="oldPassword">
          <n-input v-model:value="pwdModel.oldPassword" type="password" show-password-on="click" />
        </n-form-item>
        <n-form-item label="新密码" path="newPassword">
          <n-input v-model:value="pwdModel.newPassword" type="password" show-password-on="click" />
        </n-form-item>
        <n-form-item label="确认新密码" path="confirmPassword">
          <n-input
            v-model:value="pwdModel.confirmPassword"
            type="password"
            show-password-on="click"
            @keyup.enter="handleChangePassword"
          />
        </n-form-item>
        <n-button
          type="primary"
          block
          :loading="pwdModal.loading"
          :disabled="pwdModal.loading"
          @click="handleChangePassword"
        >
          确认修改
        </n-button>
      </n-form>
    </n-modal>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #101014;
}
.login-card {
  width: 400px;
}
</style>
