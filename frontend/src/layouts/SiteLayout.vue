<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useSiteStore } from '@/stores/site'

const site = useSiteStore()
const route = useRoute()

onMounted(() => {
  site.ensure()
})
</script>

<template>
  <div class="aurora" aria-hidden="true" />
  <div class="noise" aria-hidden="true" />

  <header class="glass sticky top-0 z-40 border-x-0 border-t-0">
    <div class="mx-auto flex h-14 max-w-6xl items-center gap-6 px-4">
      <RouterLink to="/" class="font-display shrink-0 text-lg font-semibold tracking-wide">
        <span class="text-gradient">{{ site.siteName }}</span>
      </RouterLink>
      <nav class="ml-auto flex items-center gap-5 text-sm">
        <RouterLink to="/" class="nav-link" :class="{ 'nav-link--active': route.path === '/' }">首页</RouterLink>
        <RouterLink to="/archive" class="nav-link" :class="{ 'nav-link--active': route.path.startsWith('/archive') }">归档</RouterLink>
        <RouterLink to="/about" class="nav-link" :class="{ 'nav-link--active': route.path.startsWith('/about') }">关于</RouterLink>
        <RouterLink to="/search" class="nav-link" :class="{ 'nav-link--active': route.path.startsWith('/search') }" aria-label="搜索">搜索</RouterLink>
      </nav>
    </div>
  </header>

  <main class="mx-auto min-h-[70vh] max-w-6xl px-4 pb-24">
    <RouterView />
  </main>

  <footer class="border-t border-line">
    <div class="mx-auto flex max-w-6xl flex-col items-center gap-3 px-4 py-8 text-xs text-dim sm:flex-row">
      <span>© {{ new Date().getFullYear() }} {{ site.siteName }} · {{ site.siteDesc }}</span>
      <nav v-if="site.info?.social_links?.length" class="flex gap-4 sm:ml-auto">
        <a
          v-for="l in site.info.social_links"
          :key="l.name"
          :href="l.url"
          target="_blank"
          rel="noopener noreferrer"
          class="transition-colors hover:text-ink"
        >{{ l.name }}</a>
      </nav>
    </div>
  </footer>
</template>

<style scoped>
.nav-link {
  color: var(--color-dim);
  transition: color 0.2s;
}

.nav-link:hover,
.nav-link--active {
  color: var(--color-ink);
}
</style>
