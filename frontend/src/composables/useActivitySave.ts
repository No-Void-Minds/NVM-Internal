import { ref } from 'vue'
import { ApiError } from '../api/client'
import { saveActivity, type ActivityCell } from '../api/activities'

type Status = 'saving' | 'saved' | 'error'

export const cellKey = (userId: number, slot: number) => `${userId}:${slot}`

// Simpan otomatis satu sel: dipakai kartu di papan task dan halaman Aktivitas, supaya aturan simpan,
// status per baris, dan tombol Coba lagi hanya ada di satu tempat.
// apply juga menerima seluruh jam hari itu: menyimpan jam kepala mengubah isi jam yang tersambung ke jam itu
// (revisi 33), jadi pemanggil bisa ikut menyegarkannya. date adalah tanggal sel yang disimpan, supaya pemanggil
// bisa mengabaikan jawaban untuk tanggal yang sudah ditinggalkan (revisi 37).
export function useActivitySave(
  apply: (userId: number, slot: number, cell: ActivityCell, day: ActivityCell[], date: string) => void,
) {
  const status = ref<Record<string, Status>>({})
  const message = ref<Record<string, string>>({})
  // Nilai yang terakhir dicoba, supaya Coba lagi tidak perlu menunggu pengguna mengetik ulang.
  const attempted = ref<Record<string, { date: string; activity: string; output: string }>>({})

  async function save(userId: number, date: string, slot: number, activity: string, output: string) {
    const key = cellKey(userId, slot)
    attempted.value[key] = { date, activity, output }
    status.value[key] = 'saving'
    delete message.value[key]
    try {
      const res = await saveActivity({ user_id: userId, date, slot, activity, output })
      // Hanya sel yang baru disimpan yang ditimpa hasil server, supaya sel lain yang sedang diisi tidak
      // ikut kembali ke nilai lama.
      apply(userId, slot, res.cells[slot - 1], res.cells, res.date)
      status.value[key] = 'saved'
    } catch (err) {
      // Isian yang gagal sengaja tidak ditimpa: teks yang diketik tetap ada di layar untuk dicoba lagi.
      status.value[key] = 'error'
      message.value[key] = err instanceof ApiError ? err.message : 'Tidak bisa menyimpan. Coba lagi.'
    }
  }

  function retry(userId: number, slot: number) {
    const v = attempted.value[cellKey(userId, slot)]
    if (v) void save(userId, v.date, slot, v.activity, v.output)
  }

  return { status, message, save, retry }
}
