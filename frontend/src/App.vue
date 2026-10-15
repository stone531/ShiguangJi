<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NDialogProvider, zhCN, dateZhCN } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'

/* 后台浅色主题：主色与前台强调色同源（Solarized Blue #268bd2） */
const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#268bd2',
    primaryColorHover: '#4da3e0',
    primaryColorPressed: '#1e6fa8',
    primaryColorSuppl: '#268bd2',
  },
}
</script>

<template>
  <n-config-provider :theme-overrides="themeOverrides" :locale="zhCN" :date-locale="dateZhCN">
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
