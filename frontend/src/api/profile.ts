import { ApiError, apiGet, apiPut, notifyUnauthorized, request } from './client'
import type { TaskStatus } from '../utils/task'

// Sama dengan backend (account.MaxAvatarSize). Dicek di browser supaya berkas besar tidak diunggah dulu baru ditolak.
export const maxAvatarBytes = 2 * 1024 * 1024
export const avatarTypes = ['image/jpeg', 'image/png']

export type ProfileInput = { name: string; job_title: string }

export function updateProfile(input: ProfileInput) {
  return apiPut<void>('/me/profile', input)
}

export function deleteAvatar() {
  return request<void>('DELETE', '/me/avatar')
}

export function avatarUrl(userId: number, avatarId: string): string {
  // ?v= membuat URL berganti setiap foto berganti, jadi browser boleh menyimpan foto selamanya.
  return `/api/users/${userId}/avatar?v=${encodeURIComponent(avatarId)}`
}

export type TaskCounts = { counts: Record<TaskStatus, number>; completed_days: number }

export function getTaskCounts(signal?: AbortSignal): Promise<TaskCounts> {
  return apiGet<TaskCounts>('/me/task-counts', signal)
}

type UploadBody = { avatar_id?: string; error?: { code?: string; message?: string } } | null

// XMLHttpRequest dipakai karena fetch tidak bisa melaporkan progres unggahan.
export function uploadAvatar(file: File, onProgress: (fraction: number) => void): Promise<string> {
  return new Promise<string>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('PUT', '/api/me/avatar')
    xhr.responseType = 'json'
    xhr.setRequestHeader('Accept', 'application/json')
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total)
    }
    xhr.onload = () => {
      const body = xhr.response as UploadBody
      if (xhr.status === 200 && body?.avatar_id) {
        resolve(body.avatar_id)
        return
      }
      if (xhr.status === 401) notifyUnauthorized()
      if (!body?.error?.code) {
        // Nginx membalas 413 dengan halaman HTML, bukan JSON dari backend.
        const message = xhr.status === 413 ? 'Foto melebihi batas 2 MB.' : 'Server aplikasi tidak merespons.'
        reject(new ApiError(xhr.status, 'backend_unreachable', message))
        return
      }
      reject(new ApiError(xhr.status, body.error.code, body.error.message ?? 'Unggahan gagal. Coba lagi.'))
    }
    xhr.onerror = () =>
      reject(new ApiError(0, 'network', 'Tidak bisa terhubung. Periksa koneksi ke jaringan kantor.'))
    const form = new FormData()
    form.append('file', file)
    xhr.send(form)
  })
}
