/** 日期显示工具：全站统一 yyyy-MM-dd（本地时区） */

/** 格式化日期为 yyyy-MM-dd；接受 Date 或后端时间串，null/无效返回空串 */
export function fmtDate(d: string | Date | null | undefined): string {
  if (!d) return ''
  const t = typeof d === 'string' ? new Date(d) : d
  if (Number.isNaN(t.getTime())) return ''
  const p = (n: number) => String(n).padStart(2, '0')
  return `${t.getFullYear()}-${p(t.getMonth() + 1)}-${p(t.getDate())}`
}
