import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  build: {
    // Font tidak pernah disisipkan sebagai data: URI. CSP produksi font-src 'self' memblokir data:, dan subset kecil
    // JetBrains Mono (di bawah batas 4 KB bawaan Vite) dulu tersisip sehingga gagal dimuat (terlihat di uji wss W2).
    assetsInlineLimit: (file) => (/\.(woff2?|ttf|otf)$/.test(file) ? false : undefined),
  },
  server: {
    // Frontend dan API tampil satu origin, sama seperti nanti di belakang Nginx,
    // jadi cookie sesi tidak butuh pengaturan CORS (Sprint 1).
    proxy: {
      '/api': {
        // VITE_API_TARGET hanya untuk uji restart backend (scripts/e2e/uji-restart.mjs), yang menjalankan backend
        // dan Vite kedua di port lain supaya backend dev yang sedang dipakai tidak ikut dimatikan.
        target: process.env.VITE_API_TARGET ?? 'http://127.0.0.1:8080',
        // Host asli dari browser harus sampai ke backend; cek Origin (CSRF) membandingkan keduanya.
        changeOrigin: false,
        // WebSocket Whiteboard ikut diteruskan, sama seperti location /api/whiteboard/ws di nginx.conf.
        ws: true,
      },
    },
  },
})