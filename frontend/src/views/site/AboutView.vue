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
      class="mx-auto h-28 w-28 rounded-full border border-line object-cover"
    />
    <div v-else class="mx-auto flex h-28 w-28 items-center justify-center rounded-full border border-line bg-white/5 text-4xl">👤</div>

    <h1 class="text-gradient mt-6 text-3xl font-bold">{{ site.siteName }}</h1>
    <p class="mt-4 leading-loose text-dim">{{ site.siteDesc }}</p>

    <nav v-if="site.info?.social_links?.length" class="mt-8 flex flex-wrap justify-center gap-3">
      <a
        v-for="l in site.info.social_links"
        :key="l.name"
        :href="l.url"
        target="_blank"
        rel="noopener noreferrer"
        class="glass spotlight rounded-full px-5 py-2 text-sm transition-colors hover:text-accent"
      >{{ l.name }}</a>
    </nav>
  </section>

  <!-- 固定赞赏区 -->
  <section class="glass mx-auto mt-16 max-w-2xl rounded-2xl p-10 text-center">
    <p class="text-4xl">☕</p>
    <h2 class="mt-4 font-semibold">爱心捐赠</h2>
    <p class="mt-2 text-sm text-dim">{{ site.info?.donate_text ?? '写得不错，请我喝杯咖啡' }}</p>
    <button
      class="mt-6 rounded-full bg-gradient-to-r from-accent to-accent-2 px-6 py-2 text-sm font-medium text-void transition-opacity hover:opacity-85"
      @click="donateOpen = true"
    >打开收款码</button>
  </section>

  <DonateModal v-model="donateOpen" />
</template>
