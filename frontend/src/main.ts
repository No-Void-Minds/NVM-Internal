import { createApp } from 'vue'
import { createPinia } from 'pinia'
import '@fontsource/ibm-plex-sans/400.css'
import '@fontsource/ibm-plex-sans/500.css'
import '@fontsource/ibm-plex-sans/600.css'
import '@fontsource/ibm-plex-mono/400.css'
import './style.css'
import App from './App.vue'
import { router } from './router'

const app = createApp(App)
app.use(createPinia())
app.use(router)

// Tunggu pengecekan sesi pertama selesai, supaya layout aplikasi tidak sempat berkedip sebelum halaman Masuk.
void router.isReady().then(() => app.mount('#app'))
