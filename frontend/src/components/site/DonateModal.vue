<script setup lang="ts">
import { ref, watch } from 'vue'
import { useSiteStore } from '@/stores/site'

/** 赞赏弹窗：微信/支付宝收款码 Tab 切换（spec §6.2 爱心捐赠） */
const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const site = useSiteStore()
const tab = ref<'wechat' | 'alipay'>('wechat')

watch(
  () => props.modelValue,
  (open) => {
    // 打开时按已配置的收款码选默认 Tab
    if (open) {
      const info = site.info
      tab.value = info?.wechat_qr ? 'wechat' : 'alipay'
    }
  },
)

function close() {
  emit('update:modelValue', false)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
        @click.self="close"
      >
        <div class="w-full max-w-sm rounded-lg border border-line bg-white p-6 shadow-lg">
          <div class="flex items-center justify-between">
            <h3 class="font-semibold">{{ site.info?.donate_text ?? '请我喝杯咖啡 ☕' }}</h3>
            <button class="text-dim transition-colors hover:text-ink" aria-label="关闭" @click="close">✕</button>
          </div>

          <div class="mt-4 flex justify-center gap-2">
            <button
              class="rounded-full px-4 py-1.5 text-sm transition-colors"
              :class="tab === 'wechat' ? 'bg-accent/20 text-accent' : 'text-dim hover:text-ink'"
              :disabled="!site.info?.wechat_qr"
              @click="tab = 'wechat'"
            >微信</button>
            <button
              class="rounded-full px-4 py-1.5 text-sm transition-colors"
              :class="tab === 'alipay' ? 'bg-accent/20 text-accent' : 'text-dim hover:text-ink'"
              :disabled="!site.info?.alipay_qr"
              @click="tab = 'alipay'"
            >支付宝</button>
          </div>

          <div class="mt-4">
            <img
              v-if="tab === 'wechat' && site.info?.wechat_qr"
              :src="site.info.wechat_qr"
              alt="微信收款码"
              class="mx-auto w-52 rounded-xl"
            />
            <img
              v-else-if="tab === 'alipay' && site.info?.alipay_qr"
              :src="site.info.alipay_qr"
              alt="支付宝收款码"
              class="mx-auto w-52 rounded-xl"
            />
            <p v-else class="py-10 text-center text-sm text-dim">博主还没有配置收款码 ～</p>
          </div>

          <p class="mt-4 text-center text-xs text-dim">长按识别二维码 · 金额随缘</p>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
