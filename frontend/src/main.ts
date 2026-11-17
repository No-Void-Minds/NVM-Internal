import { createApp } from 'vue'
import { createPinia } from 'pinia'
// D-DIN dimuat lewat @font-face di style.css dari /fonts. JetBrains Mono khusus ID task, kode, dan angka tabular.
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'
import './style.css'
import App from './App.vue'
import { router } from './router'

const app = createApp(App)
app.use(createPinia())
app.use(router)

// Tunggu pengecekan sesi pertama selesai, supaya layout aplikasi tidak sempat berkedip sebelum halaman Masuk.
void router.isReady().then(() => app.mount('#app'))
