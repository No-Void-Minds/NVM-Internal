<template>
  <!-- Logo resmi NVM (sama dengan compro). Nama aksesibel diberikan pemanggil lewat teks di sebelahnya. -->
  <span class="brand-logo" aria-hidden="true"></span>
</template>

<style scoped>
/* nvm-logo.png putih di atas transparan. Dipakai sebagai mask, bentuk aslinya diisi warna teks tema, jadi logo
   benar di mode terang dan gelap tanpa salinan yang diwarnai ulang (cara compro, BrandLogo.vue). Tinggi diatur
   pemanggil lewat kelas h-*. */
.brand-logo {
  display: block;
  aspect-ratio: 4084 / 1512;
  background-color: var(--color-ink);
  -webkit-mask: url('/brand/nvm-logo.png') center / contain no-repeat;
  mask: url('/brand/nvm-logo.png') center / contain no-repeat;
}

@media (forced-colors: active) {
  .brand-logo {
    forced-color-adjust: none;
    background-color: CanvasText;
  }
}
</style>
