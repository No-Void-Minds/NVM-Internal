type ApiErrorBody = { error?: { code?: string; message?: string } }

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly retryAfterSeconds: number | null

  constructor(status: number, code: string, message: string, retryAfterSeconds: number | null = null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.retryAfterSeconds = retryAfterSeconds
  }
}

let unauthorizedHandler: (() => void) | null = null

// Dipasang oleh router supaya modul ini tidak mengimpor router secara langsung (menghindari import melingkar).
export function setUnauthorizedHandler(handler: () => void) {
  unauthorizedHandler = handler
}

async function request<T>(method: string, path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  let res: Response
  try {
    res = await fetch(`/api${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: 'same-origin',
      signal,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, 'network', 'Tidak bisa terhubung. Periksa koneksi ke jaringan kantor.')
  }

  if (res.status === 204) return undefined as T

  const data: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    const detail = (data as ApiErrorBody | null)?.error
    // Tanpa body JSON berarti backend tidak menjawab sama sekali, mis. proxy Vite atau Nginx gagal meneruskan.
    if (!detail?.code) {
      throw new ApiError(res.status, 'backend_unreachable', 'Server aplikasi tidak merespons.')
    }
    const retry = Number(res.headers.get('Retry-After'))
    if (res.status === 401) unauthorizedHandler?.()
    throw new ApiError(
      res.status,
      detail.code,
      detail.message ?? 'Terjadi kesalahan. Coba lagi.',
      Number.isFinite(retry) && retry > 0 ? retry : null,
    )
  }
  return data as T
}

export function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  return request<T>('GET', path, undefined, signal)
}

export function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('POST', path, body)
}
