import { ref } from 'vue'
import { ApiError } from '../api/client'
import type { UploadHandle } from '../api/attachments'
import { precheck } from '../utils/attachment'

export type QueueItem = {
  key: number
  file: File
  status: 'waiting' | 'uploading' | 'failed'
  progress: number
  error: string | null
  // false untuk penolakan sebelum unggah (tipe atau ukuran): mengulang tidak akan mengubah hasilnya.
  retryable: boolean
}

// Antrean unggah lampiran task dan sub task (diangkat dari AttachmentSection untuk revisi 36). Satu berkas per
// waktu: progresnya jelas dan Wi-Fi kantor tidak dibanjiri unggahan paralel. announce diisi untuk aria-live.
export function useUploadQueue(
  upload: (file: File, onProgress: (p: number) => void) => UploadHandle,
  onDone: (message: string) => void,
) {
  const queue = ref<QueueItem[]>([])
  const uploading = ref(false)
  const announce = ref('')
  let nextKey = 0
  let current: UploadHandle | null = null

  function messageOf(err: unknown): string {
    if (err instanceof ApiError) return err.message
    return 'Gagal terhubung ke server. Periksa koneksi, lalu coba lagi.'
  }

  function add(files: File[]) {
    for (const file of files) {
      const error = precheck(file)
      queue.value.push({
        key: nextKey++,
        file,
        status: error ? 'failed' : 'waiting',
        progress: 0,
        error,
        retryable: false,
      })
      if (error) announce.value = error
    }
    void run()
  }

  async function run() {
    if (uploading.value) return
    uploading.value = true
    let done = 0
    try {
      for (let it = next(); it; it = next()) {
        const item = it
        item.status = 'uploading'
        try {
          current = upload(item.file, (p) => (item.progress = p))
          await current.promise
          done++
          dismiss(item.key)
        } catch (err) {
          if (err instanceof DOMException && err.name === 'AbortError') {
            dismiss(item.key)
            continue
          }
          item.status = 'failed'
          item.retryable = true
          item.error = `${item.file.name}: ${messageOf(err)}`
          announce.value = `Gagal mengunggah ${item.error}`
        }
      }
    } finally {
      uploading.value = false
      current = null
    }
    if (done) {
      const msg = done === 1 ? '1 berkas diunggah.' : `${done} berkas diunggah.`
      announce.value = msg
      onDone(msg)
    }
  }

  function next(): QueueItem | undefined {
    return queue.value.find((q) => q.status === 'waiting')
  }

  function dismiss(key: number) {
    queue.value = queue.value.filter((q) => q.key !== key)
  }

  function retry(key: number) {
    const item = queue.value.find((q) => q.key === key)
    if (!item) return
    item.status = 'waiting'
    item.progress = 0
    item.error = null
    void run()
  }

  function abort() {
    current?.abort()
  }

  return { queue, uploading, announce, add, dismiss, retry, abort }
}
