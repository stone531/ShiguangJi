import axios from 'axios'
import type { AxiosResponse } from 'axios'
import type { Response } from '@/types/api'
import { useAdminStore } from '@/stores/admin'
import router from '@/router'

export const http = axios.create({
  baseURL: '/api',
  timeout: 15_000,
  withCredentials: true,
  // CSRF 纵深防御：写接口要求该头（后端 RequireXHR 校验）
  headers: { 'X-Requested-With': 'XMLHttpRequest' },
})

http.interceptors.response.use(
  (resp: AxiosResponse<Response>) => {
    const body = resp.data
    if (body.code !== 0) {
      if (body.code === 20001 || body.code === 20002) {
        useAdminStore().clear()
        router.push('/admin/login')
      }
      return Promise.reject(new Error(body.msg))
    }
    return resp
  },
  (error) => {
    if (error.response?.status === 401) {
      useAdminStore().clear()
      router.push('/admin/login')
    }
    // 保留原始 AxiosError 供调用方按需判定（如取 HTTP 状态码）；message 供 UI 展示
    const wrapped = new Error(error.response?.data?.msg ?? '网络错误，请稍后重试')
    ;(wrapped as Error & { cause?: unknown }).cause = error
    return Promise.reject(wrapped)
  },
)

/** 统一取 data 的请求方法 */
export async function get<T>(url: string, params?: object): Promise<T> {
  const resp = await http.get<Response<T>>(url, { params })
  return resp.data.data
}

export async function post<T>(url: string, body?: object): Promise<T> {
  const resp = await http.post<Response<T>>(url, body)
  return resp.data.data
}

export async function put<T>(url: string, body?: object): Promise<T> {
  const resp = await http.put<Response<T>>(url, body)
  return resp.data.data
}

export async function del<T>(url: string): Promise<T> {
  const resp = await http.delete<Response<T>>(url)
  return resp.data.data
}
