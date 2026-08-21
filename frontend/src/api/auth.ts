import { post } from './http'
import type { AdminInfo } from '@/types/api'

export function login(username: string, password: string) {
  return post<AdminInfo>('/admin/login', { username, password })
}

export function changePassword(oldPassword: string, newPassword: string) {
  return post<null>('/admin/password', {
    old_password: oldPassword,
    new_password: newPassword,
  })
}
