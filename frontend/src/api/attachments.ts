import { ApiError, notifyUnauthorized, request } from './client'

export type AttachmentKind = 'brief' | 'deliverable'

export type TaskAttachment = {
  id: string
  kind: AttachmentKind
  name: string
  mime_type: string
  size_bytes: number
  uploaded_at: string
  uploaded_by: string
}

export type UploadHandle = { promise: Promise<TaskAttachment>; abort: () => void }

type UploadBody = { attachment?: TaskAttachment; error?: { code?: string; message?: string } } | null

// XMLHttpRequest dipakai karena fetch tidak bisa melaporkan progres unggahan (S5-07).
export function uploadAttachment(
  taskId: string,
  kind: AttachmentKind,
  file: File,
  onProgress: (fraction: number) => void,
): UploadHandle {
  const xhr = new XMLHttpRequest()
  const promise = new Promise<TaskAttachment>((resolve, reject) => {
    xhr.open('POST', `/api/tasks/${taskId}/attachments?kind=${kind}`)
    xhr.responseType = 'json'
    xhr.setRequestHeader('Accept', 'application/json')
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total)
    }
    xhr.onload = () => {
      const body = xhr.response as UploadBody
      if (xhr.status === 201 && body?.attachment) {
        resolve(body.attachment)
        return
      }
      if (xhr.status === 401) notifyUnauthorized()
      if (!body?.error?.code) {
        // Nginx (Sprint 7) membalas 413 dengan halaman HTML, bukan JSON dari backend.
        const message =
          xhr.status === 413 ? 'Berkas melebihi batas 50 MB.' : 'Server aplikasi tidak merespons.'
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

export function attachmentUrl(taskId: string, id: string): string {
  return `/api/tasks/${taskId}/attachments/${id}`
}

export function deleteAttachment(taskId: string, id: string) {
  return request<void>('DELETE', `/tasks/${taskId}/attachments/${id}`)
}
