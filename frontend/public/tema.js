// Dipasang sinkron di <head> sebelum halaman digambar, supaya pengguna mode gelap tidak melihat kilatan terang.
// Pilihan tersimpan per browser; bila belum memilih, mengikuti pengaturan sistem operasi.
;(function () {
  var tema = null
  try {
    tema = localStorage.getItem('nvm-tema')
  } catch {
    // Penyimpanan bisa diblokir (mode privat, kebijakan browser): pakai pengaturan sistem.
  }
  if (tema !== 'terang' && tema !== 'gelap') {
    tema = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'gelap' : 'terang'
  }
  document.documentElement.setAttribute('data-theme', tema)
})()
