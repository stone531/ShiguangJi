<script lang="ts">
export interface TocItem {
  id: string
  text: string
  level: number
}
</script>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import { nextTick, onMounted, ref, watch } from 'vue'
import Lightbox from './Lightbox.vue'

/**
 * Markdown 渲染管线（spec §2/§8 XSS 防御）：
 * markdown-it(html:false) → DOMPurify 净化 → v-html → Shiki 异步高亮（dark-plus）。
 * 标题注入锚点 id 并向父组件抛出 TOC；图片点击开 Lightbox。
 */

const props = defineProps<{ source: string }>()
const emit = defineEmits<{ (e: 'toc', items: TocItem[]): void }>()

const root = ref<HTMLElement | null>(null)
const rendered = ref('')
const lightboxSrc = ref<string | null>(null)

const md = new MarkdownIt({ html: false, linkify: true })

let headingSeq = 0
const toc: TocItem[] = []

// h2/h3 注入锚点并收集 TOC（id 用序号：中文标题无法直接做 URL slug）
md.renderer.rules.heading_open = (tokens, idx, options, _env, self) => {
  const token = tokens[idx]
  if (token.tag === 'h2' || token.tag === 'h3') {
    const id = `h-${headingSeq++}`
    toc.push({ id, text: tokens[idx + 1].content, level: Number(token.tag.slice(1)) })
    token.attrSet('id', id)
  }
  return self.renderToken(tokens, idx, options)
}

// DOMPurify 全局钩子：外链新窗口打开 + 图片懒加载（模块单次注册）
DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName === 'A' && node.getAttribute('href')?.startsWith('http')) {
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer')
  }
  if (node.tagName === 'IMG') {
    node.setAttribute('loading', 'lazy')
  }
})

let renderSeq = 0

async function render() {
  const seq = ++renderSeq
  headingSeq = 0
  toc.length = 0
  const raw = md.render(props.source)
  if (seq !== renderSeq) return
  rendered.value = DOMPurify.sanitize(raw)
  emit('toc', [...toc])
  await nextTick()
  void highlight(seq)
}

/* ===== Shiki 按需加载（动态 import，代码块随详情页 chunk 拆分） ===== */
let shikiReady: Promise<typeof import('shiki')> | null = null

const langAlias: Record<string, string> = {
  js: 'javascript', ts: 'typescript', sh: 'bash', shell: 'bash', zsh: 'bash',
  yml: 'yaml', py: 'python', rb: 'ruby', golang: 'go', 'c++': 'cpp', html: 'xml',
}

async function highlight(seq: number) {
  const el = root.value
  if (!el || seq !== renderSeq) return
  const blocks = Array.from(el.querySelectorAll('pre code'))
  if (!blocks.length) return
  let codeToHtml: typeof import('shiki').codeToHtml
  try {
    shikiReady ??= import('shiki')
    ;({ codeToHtml } = await shikiReady)
  } catch {
    return // 加载失败保持纯文本
  }
  await Promise.all(
    blocks.map(async (block) => {
      if (seq !== renderSeq) return
      const code = block.textContent ?? ''
      const m = /language-([\w+-]+)/.exec(block.className)
      const lang = m ? (langAlias[m[1]] ?? m[1]) : 'text'
      try {
        const html = await codeToHtml(code, { lang, theme: 'dark-plus' })
        if (seq !== renderSeq) return
        const tpl = document.createElement('template')
        tpl.innerHTML = html.trim()
        block.parentElement?.replaceWith(tpl.content)
      } catch {
        // 未知语言：保留默认转义渲染
      }
    }),
  )
}

function onContentClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (target.tagName === 'IMG') {
    lightboxSrc.value = (target as HTMLImageElement).currentSrc || (target as HTMLImageElement).src
  }
}

watch(() => props.source, () => void render())
onMounted(() => void render())
</script>

<template>
  <div ref="root" class="md-body" @click="onContentClick" v-html="rendered" />
  <Lightbox v-model="lightboxSrc" />
</template>
