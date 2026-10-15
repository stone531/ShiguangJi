<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useSiteStore } from '@/stores/site'
import DonateModal from '@/components/site/DonateModal.vue'

/** 关于页：个人介绍 + 社交链接 + 固定赞赏区（spec §6.1） */
const site = useSiteStore()
const donateOpen = ref(false)

onMounted(() => site.ensure())
</script>

<template>
  <section class="mx-auto mt-16 max-w-2xl text-center">
    <img
      v-if="site.info?.avatar"
      :src="site.info.avatar"
      alt="头像"
      class="mx-auto h-24 w-24 rounded-full object-cover"
    />
    <div v-else class="mx-auto flex h-24 w-24 items-center justify-center rounded-full border border-line text-4xl">👤</div>

    <h1 class="mt-6 text-2xl font-bold">{{ site.siteName }}</h1>
    <p class="mt-4 leading-loose text-dim">{{ site.siteDesc }}</p>

    <nav v-if="site.info?.social_links?.length" class="mt-8 flex flex-wrap justify-center gap-3">
      <a
        v-for="l in site.info.social_links"
        :key="l.name"
        :href="l.url"
        target="_blank"
        rel="noopener noreferrer"
        class="rounded-full border border-line px-5 py-2 text-sm transition-colors hover:border-accent hover:text-accent"
      >{{ l.name }}</a>
    </nav>
  </section>

  <!-- 固定赞赏区 -->
  <section class="mx-auto mt-16 max-w-2xl border-t border-line pt-10 text-center">
    <p class="text-4xl">☕</p>
    <h2 class="mt-4 font-semibold">爱心捐赠</h2>
    <p class="mt-2 text-sm text-dim">{{ site.info?.donate_text ?? '写得不错，请我喝杯咖啡' }}</p>
    <button
      class="mt-6 rounded-full bg-accent px-6 py-2 text-sm font-medium text-white transition-opacity hover:opacity-85"
      @click="donateOpen = true"
    >打开收款码</button>
  </section>

  <DonateModal v-model="donateOpen" />
</template>
