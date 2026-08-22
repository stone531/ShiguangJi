<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'

/** 图片灯箱：modelValue 为图片地址（null 关闭）；Esc 可关 */
const props = defineProps<{ modelValue: string | null }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string | null): void }>()

function onKey(e: KeyboardEvent) {
  if (props.modelValue && e.key === 'Escape') close()
}

function close() {
  emit('update:modelValue', null)
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey)) // 红线：监听必须清理
</script>

<template>
  <Teleport to="body">
    <Transition name="zoom">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex cursor-zoom-out items-center justify-center bg-black/85 p-6"
        @click="close"
      >
        <img :src="modelValue" alt="图片预览" class="max-h-full max-w-full rounded-lg" />
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.zoom-enter-active,
.zoom-leave-active {
  transition: opacity 0.2s ease;
}

.zoom-enter-from,
.zoom-leave-to {
  opacity: 0;
}
</style>
