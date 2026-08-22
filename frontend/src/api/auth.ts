import { get, post, put } from './http'
import type { AdminInfo, AdminProfileVO } from '@/types/api'

export function login(username: string, password: string) {
  return post<AdminInfo>('/admin/login', { username, password })
}

export function changePassword(oldPassword: string, newPassword: string) {
  return post<null>('/admin/password', {
    old_password: oldPassword,
    new_password: newPassword,
  })
}

export function getProfile() {
  return get<AdminProfileVO>('/admin/profile')
}

export function updateProfile(nickname: string, avatar: string) {
  return put<null>('/admin/profile', { nickname, avatar })
}
