<script setup lang="ts">
/** 404：顶层 catch-all（不套 SiteLayout，自带背景与返回入口） */
</script>

<template>
  <div class="aurora" aria-hidden="true" />
  <div class="noise" aria-hidden="true" />
  <main class="flex min-h-screen flex-col items-center justify-center px-4 text-center">
    <p class="font-display text-gradient text-8xl font-bold">404</p>
    <p class="mt-6 text-dim">这个页面去银河系旅行了</p>
    <div class="mt-8 flex gap-4">
      <RouterLink
        to="/"
        class="rounded-full bg-gradient-to-r from-accent to-accent-2 px-6 py-2 text-sm font-medium text-void transition-opacity hover:opacity-85"
      >回首页</RouterLink>
      <RouterLink to="/archive" class="glass rounded-full px-6 py-2 text-sm transition-colors hover:text-accent">看归档</RouterLink>
    </div>
  </main>
</template>
