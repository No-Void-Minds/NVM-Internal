import { ref } from 'vue'

export type Tema = 'terang' | 'gelap'

// Nilai awal dipasang /tema.js sebelum halaman digambar (pilihan tersimpan, atau pengaturan sistem).
const theme = ref<Tema>(document.documentElement.dataset.theme === 'gelap' ? 'gelap' : 'terang')

export function useTheme() {
  function toggle() {
    theme.value = theme.value === 'gelap' ? 'terang' : 'gelap'
    document.documentElement.dataset.theme = theme.value
    try {
      localStorage.setItem('nvm-tema', theme.value)
    } catch {
      // Penyimpanan diblokir: pilihan tetap berlaku sampai halaman dimuat ulang.
    }
  }
  return { theme, toggle }
}
