// Cek token CSS: setiap variabel desain yang dirujuk aplikasi harus punya nilai di CSS hasil build untuk setiap tema
// yang mendefinisikannya di src/style.css.
//
// Alasannya: Tailwind v4 hanya mengeluarkan variabel @theme yang dirujuk utilitas atau CSS. Variabel yang hanya
// dirujuk lewat inline style, binding :style, string di TypeScript (mis. peta warna, getPropertyValue), atau nama
// yang dirakit dinamis tidak terlihat oleh Tailwind dan bisa hilang diam-diam dari build. Itu terjadi pada warna
// kolaborasi Whiteboard: nilai mode terang indeks 1 dan 3 sampai 8 tidak ada di build produksi, label kursor
// transparan, padahal hitungan kontras dari berkas sumber lolos.
//
// Yang diperiksa:
//   - rujukan: setiap --nama di src/**/*.{vue,ts} dan di blok CSS biasa style.css (bukan definisinya);
//   - nama yang dirakit dinamis (--sesuatu-${...}) ditolak, karena tidak bisa diperiksa; tulis nama lengkapnya
//     atau pakai aturan CSS statis (seperti [data-collab='n'] di style.css);
//   - untuk token yang didefinisikan di @theme atau :root (terang) dan :root[data-theme='gelap'] (gelap), build
//     harus punya definisinya di aturan :root atau :host untuk tema yang sama.
//
// Cara menjalankan: cd frontend && node scripts/cek-token.mjs   (membangun CSS ke folder sementara)
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

// Definisi token di sumber: blok @theme dan :root (terang) serta :root[data-theme='gelap'] (gelap). Aturan lain
// (mis. .wb-card-krem > *) adalah timpaan lokal, bukan sumber token.
export function sourceTokens(css) {
  const out = new Map()
  const add = (body, theme) => {
    for (const m of body.matchAll(/(--[a-z][a-z0-9-]*)\s*:/g)) {
      if (!out.has(m[1])) out.set(m[1], new Set())
      out.get(m[1]).add(theme)
    }
  }
  const noComments = css.replace(/\/\*[\s\S]*?\*\//g, '')
  for (const m of noComments.matchAll(/(@theme(?:\s+\w+)?|:root\[data-theme='gelap'\]|:root)\s*\{([^{}]*)\}/g)) {
    add(m[2], m[1].includes('gelap') ? 'gelap' : 'terang')
  }
  return out
}

// Definisi token di CSS hasil build (diminifikasi, bisa di dalam @layer). Selector :root atau :host tanpa
// data-theme berarti terang; dengan data-theme=gelap berarti gelap.
export function builtTokens(css) {
  const out = new Map()
  for (const m of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const sel = m[1].trim()
    if (!/:root|:host/.test(sel)) continue
    const theme = /data-theme=["']?gelap/.test(sel) ? 'gelap' : 'terang'
    for (const d of m[2].matchAll(/(--[a-z][a-z0-9-]*)\s*:/g)) {
      if (!out.has(d[1])) out.set(d[1], new Set())
      out.get(d[1]).add(theme)
    }
  }
  return out
}

// Rujukan token di satu berkas: setiap --nama (dalam var(), string, atau getPropertyValue). Definisi di CSS
// (--nama: nilai) tidak dihitung sebagai rujukan. Nama yang dirakit dari template string dilaporkan terpisah.
export function references(text) {
  const refs = new Set()
  const dynamic = []
  const lines = text.split('\n')
  lines.forEach((line, i) => {
    for (const m of line.matchAll(/--[a-z][a-z0-9-]*-?\$\{/g)) dynamic.push({ line: i + 1, text: m[0] })
    for (const m of line.matchAll(/--[a-z][a-z0-9-]*[a-z0-9]/g)) {
      const after = line.slice(m.index + m[0].length)
      if (/^\s*:/.test(after) && !/^\s*:\s*['"`]/.test(after)) continue
      refs.add(m[0])
    }
  })
  return { refs, dynamic }
}

// Hasil cek: token yang dirujuk tetapi tidak punya nilai di build untuk tema yang didefinisikan sumber.
export function compare(source, built, refs) {
  const missing = []
  for (const name of [...refs].sort()) {
    const themes = source.get(name)
    if (!themes) continue
    for (const t of themes) if (!built.get(name)?.has(t)) missing.push(`${name} (${t})`)
  }
  return missing
}

function walk(dir, out = []) {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name)
    if (e.isDirectory()) walk(p, out)
    else if (/\.(vue|ts)$/.test(e.name)) out.push(p)
  }
  return out
}

function main() {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
  const styleFile = path.join(root, 'src/style.css')
  const style = fs.readFileSync(styleFile, 'utf8')
  const source = sourceTokens(style)

  const refs = new Set()
  const dynamic = []
  // Blok CSS biasa di style.css juga merujuk token; definisi di blok token sendiri tidak dihitung.
  for (const r of references(style.replace(/\/\*[\s\S]*?\*\//g, '')).refs) refs.add(r)
  for (const file of walk(path.join(root, 'src'))) {
    const r = references(fs.readFileSync(file, 'utf8'))
    r.refs.forEach((x) => refs.add(x))
    for (const d of r.dynamic) dynamic.push(`${path.relative(root, file)}:${d.line} ${d.text}`)
  }

  const out = fs.mkdtempSync(path.join(os.tmpdir(), 'nvm-token-'))
  try {
    execFileSync(process.execPath, [path.join(root, 'node_modules/vite/bin/vite.js'), 'build', '--outDir', out, '--emptyOutDir', '--logLevel', 'error'], {
      cwd: root,
      stdio: ['ignore', 'ignore', 'inherit'],
    })
    const assets = path.join(out, 'assets')
    const css = fs
      .readdirSync(assets)
      .filter((f) => f.endsWith('.css'))
      .map((f) => fs.readFileSync(path.join(assets, f), 'utf8'))
      .join('\n')
    const missing = compare(source, builtTokens(css), refs)
    const used = [...refs].filter((r) => source.has(r)).length
    if (dynamic.length || missing.length) {
      for (const d of dynamic) console.error(`Nama token dirakit dinamis (tidak bisa diperiksa): ${d}`)
      for (const m of missing) console.error(`Token dirujuk aplikasi tetapi tidak ada di CSS build: ${m}`)
      process.exit(1)
    }
    console.log(`Token CSS: ${used} token dirujuk, semuanya ada di CSS build untuk setiap temanya.`)
  } finally {
    fs.rmSync(out, { recursive: true, force: true })
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) main()
