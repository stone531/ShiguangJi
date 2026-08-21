<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import {
  NAvatar,
  NButton,
  NDropdown,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NLayoutSider,
  NMenu,
  NModal,
  useDialog,
  useMessage,
  type FormInst,
  type FormItemRule,
  type FormRules,
  type MenuOption,
} from 'naive-ui'
import { changePassword } from '@/api/auth'
import { useAdminStore } from '@/stores/admin'

const router = useRouter()
const dialog = useDialog()
const message = useMessage()
const adminStore = useAdminStore()

/* ---------- 侧边导航 ---------- */
const collapsed = ref(false)

function renderEmoji(emoji: string) {
  return () => h(NIcon, null, { default: () => h('span', { style: 'font-size:16px' }, emoji) })
}

const menuOptions: MenuOption[] = [
  {
    label: () => h(RouterLink, { to: '/admin/dashboard' }, { default: () => '仪表盘' }),
    key: '/admin/dashboard',
    icon: renderEmoji('📊'),
  },
  {
    label: () => h(RouterLink, { to: '/admin/posts' }, { default: () => '文章管理' }),
    key: '/admin/posts',
    icon: renderEmoji('📝'),
  },
  {
    label: () => h(RouterLink, { to: '/admin/comments' }, { default: () => '评论管理' }),
    key: '/admin/comments',
    icon: renderEmoji('💬'),
  },
  {
    label: () => h(RouterLink, { to: '/admin/settings' }, { default: () => '站点设置' }),
    key: '/admin/settings',
    icon: renderEmoji('⚙️'),
  },
]

const activeKey = computed(() => router.currentRoute.value.path)

/* ---------- 顶栏用户下拉 ---------- */
const nickname = computed(() => adminStore.info?.nickname ?? '管理员')
const avatar = computed(() => adminStore.info?.avatar ?? '')

const userOptions = [
  { label: '修改密码', key: 'change-password' },
  { label: '退出登录', key: 'logout' },
]

function handleUserAction(key: string) {
  if (key === 'change-password') {
    pwdModal.show = true
  } else if (key === 'logout') {
    dialog.warning({
      title: '退出登录',
      content: '确定要退出登录吗？',
      positiveText: '退出',
      negativeText: '取消',
      onPositiveClick: () => {
        adminStore.clear()
        router.push('/admin/login')
      },
    })
  }
}

/* ---------- 修改密码弹窗 ---------- */
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

function resetPwdForm() {
  pwdModel.oldPassword = ''
  pwdModel.newPassword = ''
  pwdModel.confirmPassword = ''
  pwdFormRef.value?.restoreValidation()
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
    resetPwdForm()
  } catch (err) {
    message.error(err instanceof Error ? err.message : '修改失败')
  } finally {
    pwdModal.loading = false
  }
}
</script>

<template>
  <n-layout style="height: 100vh" has-sider>
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed="collapsed"
      :collapsed-width="64"
      :width="220"
      show-trigger
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="brand">
        <span v-if="!collapsed">数字花园后台</span>
        <span v-else>花</span>
      </div>
      <n-menu
        :value="activeKey"
        :options="menuOptions"
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="20"
      />
    </n-layout-sider>

    <n-layout>
      <n-layout-header bordered class="header">
        <div />
        <n-dropdown :options="userOptions" @select="handleUserAction">
          <div class="user-chip">
            <n-avatar round :size="28" :src="avatar">
              {{ nickname.slice(0, 1) }}
            </n-avatar>
            <span class="nickname">{{ nickname }}</span>
          </div>
        </n-dropdown>
      </n-layout-header>

      <n-layout-content content-style="padding: 24px" :native-scrollbar="false">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>

  <n-modal
    v-model:show="pwdModal.show"
    preset="card"
    title="修改密码"
    style="width: 420px"
    @after-leave="resetPwdForm"
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
</template>

<style scoped>
.brand {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 16px;
}

.header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}

.user-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.nickname {
  font-size: 14px;
}
</style>
