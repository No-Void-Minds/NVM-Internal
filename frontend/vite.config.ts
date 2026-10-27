import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    // Frontend dan API tampil satu origin, sama seperti nanti di belakang Nginx,
    // jadi cookie sesi tidak butuh pengaturan CORS (Sprint 1).
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
})