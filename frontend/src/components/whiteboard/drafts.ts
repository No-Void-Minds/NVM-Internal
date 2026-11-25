import { ref, watch, type Ref } from 'vue'

// Draf lokal untuk isian teks yang nilainya juga datang dari server (siaran perubahan orang lain atau jawaban operasi
// lain). Tanpa draf, render ulang komponen menimpa nilai input (Vue menyetel ulang value setiap patch) dan ketikan
// yang belum dikirim hilang. Draf yang belum diubah pengguna ikut nilai server; yang sudah diubah dipertahankan.
export function useDrafts(source: () => Record<string, string>): Ref<Record<string, string>> {
  const drafts = ref<Record<string, string>>({})
  let known: Record<string, string> = {}
  watch(
    source,
    (next) => {
      const out: Record<string, string> = {}
      for (const [k, v] of Object.entries(next)) {
        const d = drafts.value[k]
        out[k] = d !== undefined && d !== known[k] ? d : v
      }
      drafts.value = out
      known = { ...next }
    },
    { immediate: true },
  )
  return drafts
}
