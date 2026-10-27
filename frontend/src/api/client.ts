type ApiErrorBody = { error?: { code?: string; message?: string } }

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export async function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  let res: Response
  try {
    res = await fetch(`/api${path}`, {
      headers: { Accept: 'application/json' },
      credentials: 'same-origin',
      signal,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, 'network', 'Tidak bisa terhubung. Periksa koneksi ke jaringan kantor.')
  }

  const body: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    const detail = (body as ApiErrorBody | null)?.error
    // Tanpa body JSON berarti backend tidak menjawab sama sekali, mis. proxy Vite atau Nginx gagal meneruskan.
    if (!detail?.code) {
      throw new ApiError(res.status, 'backend_unreachable', 'Server aplikasi tidak merespons.')
    }
    throw new ApiError(res.status, detail.code, detail.message ?? 'Terjadi kesalahan. Coba lagi.')
  }
  return body as T
}