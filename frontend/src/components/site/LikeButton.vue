<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getLikeStatus, toggleLike } from '@/api/public'

const props = defineProps<{ slug: string }>()

const liked = ref(false)
const count = ref(0)
const busy = ref(false)
/** 点击后的心跳动效标记 */
const pop = ref(false)

onMounted(async () => {
  try {
    const data = await getLikeStatus(props.slug)
    liked.value = data.liked
    count.value = data.like_count
  } catch {
    // 状态查询失败静默：按钮退化为未点赞初始态
  }
})

async function onToggle() {
  if (busy.value) return
  busy.value = true
  pop.value = true
  try {
    const data = await toggleLike(props.slug)
    liked.value = data.liked
    count.value = data.like_count
  } catch {
    // 失败回滚动效，不打断阅读
  } finally {
    busy.value = false
    setTimeout(() => (pop.value = false), 300)
  }
}
</script>

<template>
  <button
    class="group flex items-center gap-2 rounded-full border px-5 py-2 text-sm transition-all"
    :class="
      liked
        ? 'border-accent/50 bg-accent/10 text-accent'
        : 'border-line text-dim hover:border-accent/40 hover:text-accent'
    "
    :aria-pressed="liked"
    @click="onToggle"
  >
    <span
      class="transition-transform"
      :class="[liked ? 'scale-110' : 'group-hover:scale-110', pop ? 'animate-[like-pop_0.3s_ease]' : '']"
    >{{ liked ? '♥' : '♡' }}</span>
    <span>{{ count }}</span>
    <span class="text-xs opacity-70">{{ liked ? '已赞' : '喜欢' }}</span>
  </button>
</template>

<style scoped>
@keyframes like-pop {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.5);
  }
  100% {
    transform: scale(1.1);
  }
}
</style>
