/** 阅读元信息计算：剥离 markdown 标记后按字符数统计（spec §8） */

/** 字数：去掉代码块/行内代码/图片/链接语法/markdown 符号/空白后的字符数 */
export function countChars(markdown: string): number {
  return markdown
    .replace(/```[\s\S]*?```/g, ' ') // 围栏代码块
    .replace(/`[^`\n]*`/g, ' ') // 行内代码
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ') // 图片
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1') // 链接取文字部分
    .replace(/^\s{0,3}#{1,6}\s+/gm, ' ') // 标题符号
    .replace(/[*_~>|]+/g, ' ') // 强调/引用等标记符号
    .replace(/\s+/g, '')
    .length
}

/** 预计阅读时长（分钟，1 位小数）：按 300 字/分钟 */
export function readingMinutes(chars: number): number {
  return Math.round((chars / 300) * 10) / 10
}

/** 千字显示（1 位小数），如 812 字 → 0.8 */
export function kchars(chars: number): string {
  return (chars / 1000).toFixed(1)
}
