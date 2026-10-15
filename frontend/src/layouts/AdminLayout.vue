<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import {
  NAvatar,
  NDropdown,
  NIcon,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NLayoutSider,
  NMenu,
  NModal,
  useDialog,
  type MenuOption,
} from 'naive-ui'
import ChangePasswordForm from '@/components/admin/ChangePasswordForm.vue'
import { useAdminStore } from '@/stores/admin'

const router = useRouter()
const dialog = useDialog()
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
    label: () => h(RouterLink, { to: '/admin/profile' }, { default: () => '个人资料' }),
    key: '/admin/profile',
    icon: renderEmoji('👤'),
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
  { label: '个人资料', key: 'profile' },
  { label: '修改密码', key: 'change-password' },
  { label: '退出登录', key: 'logout' },
]

function handleUserAction(key: string) {
  if (key === 'profile') {
    router.push('/admin/profile')
  } else if (key === 'change-password') {
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

/* ---------- 修改密码弹窗（表单逻辑在 ChangePasswordForm） ---------- */
const pwdModal = reactive({ show: false })
const pwdFormRef = ref<InstanceType<typeof ChangePasswordForm> | null>(null)
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
    @after-leave="pwdFormRef?.reset()"
  >
    <ChangePasswordForm ref="pwdFormRef" @success="pwdModal.show = false" />
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
