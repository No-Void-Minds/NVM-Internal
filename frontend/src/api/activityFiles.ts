import { ApiError, notifyUnauthorized, request } from './client'
import type { ActivityFile } from './activities'

export type UploadHandle = { promise: Promise<ActivityFile>; abort: () => void }

type UploadBody = { file?: ActivityFile; error?: { code?: string; message?: string } } | null

// XMLHttpRequest dipakai karena fetch tidak bisa melaporkan progres unggahan, sama seperti lampiran task.
// userId 0 berarti jam milik sendiri, jadi tabel di papan tidak perlu mengirim id akunnya.
export function uploadActivityFile(
  userId: number,
  date: string,
  slot: number,
  file: File,
  onProgress: (fraction: number) => void,
): UploadHandle {
  const xhr = new XMLHttpRequest()
  const params = new URLSearchParams({ date, slot: String(slot) })
  if (userId) params.set('user_id', String(userId))
  const promise = new Promise<ActivityFile>((resolve, reject) => {
    xhr.open('POST', `/api/activities/files?${params}`)
    xhr.responseType = 'json'
    xhr.setRequestHeader('Accept', 'application/json')
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total)
    }
    xhr.onload = () => {
      const body = xhr.response as UploadBody
      if (xhr.status === 201 && body?.file) {
        resolve(body.file)
        return
      }
      if (xhr.status === 401) notifyUnauthorized()
      if (!body?.error?.code) {
        // Nginx membalas 413 dengan halaman HTML, bukan JSON dari backend.
        const message =
          xhr.status === 413 ? 'Berkas melebihi batas 95 MB.' : 'Server aplikasi tidak merespons.'
        reject(new ApiError(xhr.status, 'backend_unreachable', message))
        return
      }
      reject(new ApiError(xhr.status, body.error.code, body.error.message ?? 'Unggahan gagal. Coba lagi.'))
    }
    xhr.onerror = () =>
      reject(new ApiError(0, 'network', 'Tidak bisa terhubung. Periksa koneksi ke jaringan kantor.'))
    xhr.onabort = () => reject(new DOMException('Unggahan dibatalkan', 'AbortError'))
    const form = new FormData()
    form.append('file', file)
    xhr.send(form)
  })
  return { promise, abort: () => xhr.abort() }
}

export function activityFileUrl(id: string): string {
  return `/api/activities/files/${id}`
}

export function deleteActivityFile(id: string) {
  return request<void>('DELETE', `/activities/files/${id}`)
}
