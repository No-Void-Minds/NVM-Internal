// Aturan yang sama dengan backend (password_handlers.go), dicek lebih dulu supaya pengguna tidak menunggu respons server.
export const MIN_PASSWORD_LENGTH = 8
const MAX_PASSWORD_BYTES = 72

export function validateNewPassword(current: string, next: string, confirm: string): string | null {
  if (!current) return 'Password lama wajib diisi.'
  if ([...next].length < MIN_PASSWORD_LENGTH) return `Password baru minimal ${MIN_PASSWORD_LENGTH} karakter.`
  if (new TextEncoder().encode(next).length > MAX_PASSWORD_BYTES) {
    return 'Password baru terlalu panjang, maksimal 72 karakter.'
  }
  if (next !== confirm) return 'Ulangi password baru belum sama.'
  if (next === current) return 'Password baru harus berbeda dari password lama.'
  return null
}
