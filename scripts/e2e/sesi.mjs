// Sesi login bersama untuk berkas E2E kecil per revisi (uji-r45.mjs, uji-r46.mjs). Memakai sesi tersimpan yang sama
// dengan uji13.mjs di scripts/e2e/.auth (gitignore), jadi run beruntun tidak menghabiskan limiter login. Login
// sungguhan hanya bila sesi tersimpan tidak ada atau sudah tidak sah, sekali per akun, dengan variabel DEV_* yang
// sama dengan uji13.mjs.
import fs from "node:fs";
import path from "node:path";

const AUTH_DIR = path.join(import.meta.dirname, ".auth");
const ACCOUNTS = {
  staf: ["DEV_STAFF_USERNAME", "DEV_STAFF_PASSWORD"],
  manajer: ["DEV_MANAGER_USERNAME", "DEV_MANAGER_PASSWORD"],
};
export let realLogins = 0;

async function me(browser, base, state) {
  const ctx = await browser.newContext({ storageState: state });
  try {
    const res = await ctx.request.get(base + "/api/auth/me");
    return res.status() === 200 ? (await res.json()).user : null;
  } finally {
    await ctx.close();
  }
}

// Mengembalikan { state, user } untuk satu peran.
export async function session(browser, base, role) {
  const file = path.join(AUTH_DIR, `${role}.json`);
  try {
    const saved = JSON.parse(fs.readFileSync(file, "utf8"));
    const user = await me(browser, base, saved);
    if (user) return { state: saved, user };
  } catch {
    // Belum ada sesi tersimpan, atau berkasnya rusak: login sungguhan.
  }
  const [u, p] = ACCOUNTS[role].map((k) => process.env[k]);
  if (!u || !p) {
    console.error(`Sesi ${role} tidak sah dan ${ACCOUNTS[role].join("/")} kosong.`);
    process.exit(2);
  }
  const ctx = await browser.newContext();
  const res = await ctx.request.post(base + "/api/auth/login", {
    headers: { Origin: base },
    data: { username: u, password: p },
  });
  realLogins++;
  if (res.status() !== 200) {
    console.error(`Login ${role} gagal: HTTP ${res.status()}`);
    process.exit(2);
  }
  const state = await ctx.storageState();
  await ctx.close();
  fs.mkdirSync(AUTH_DIR, { recursive: true });
  fs.writeFileSync(file, JSON.stringify(state), { mode: 0o600 });
  return { state, user: await me(browser, base, state) };
}

// Klien API dengan cookie sesi peran itu; POST, PUT, dan DELETE membawa Origin seperti peramban.
export async function api(browser, base, state) {
  const ctx = await browser.newContext({ storageState: state });
  const call = async (method, url, data) => {
    const res = await ctx.request.fetch(base + "/api" + url, {
      method,
      headers: { Origin: base, "Content-Type": "application/json" },
      data: data === undefined ? undefined : JSON.stringify(data),
    });
    let body = null;
    try {
      body = await res.json();
    } catch {
      // Respons tanpa JSON (204).
    }
    return { status: res.status(), body };
  };
  return { ctx, call };
}

// Pemeriksaan tata letak dasar dengan aturan yang sama dengan audit uji13.mjs: tanpa gulir horizontal dan tombol
// terlihat minimal 44x44 px.
export async function layout(page) {
  return page.evaluate(() => {
    const small = [];
    for (const b of document.querySelectorAll('button, [role="button"]')) {
      const r = b.getBoundingClientRect();
      const s = getComputedStyle(b);
      if (s.display === "none" || s.visibility === "hidden" || r.width <= 2 || r.height <= 2) continue;
      if (b.closest("[hidden], .sr-only") || b.offsetParent === null) continue;
      // Judul kartu task membentang ke seluruh kartu lewat ::after.
      if (String(b.className).includes("after:inset-0")) continue;
      // Pemicu profil memperluas area kliknya lewat ::after tanpa mengubah ukuran yang terlihat.
      const hit = b.hasAttribute("data-person-trigger") ? getComputedStyle(b, "::after") : null;
      const w = Math.max(r.width, hit ? parseFloat(hit.width) || 0 : 0);
      const h = Math.max(r.height, hit ? parseFloat(hit.height) || 0 : 0);
      if (w < 43.5 || h < 43.5)
        small.push(`${(b.getAttribute("aria-label") || b.textContent || "").trim().slice(0, 30)} ${Math.round(r.width)}x${Math.round(r.height)}`);
    }
    return { overflow: document.documentElement.scrollWidth > window.innerWidth, small };
  });
}

// Rasio kontras WCAG dari warna teks dan latar terhitung elemen (latar dicari ke atas sampai tidak transparan).
export async function contrastOf(locator) {
  return locator.evaluate((el) => {
    const rgb = (c) => (c.match(/[\d.]+/g) || []).map(Number);
    let bg = el;
    while (bg && rgb(getComputedStyle(bg).backgroundColor)[3] === 0) bg = bg.parentElement;
    const lum = ([r, g, b]) =>
      [r, g, b]
        .map((v) => v / 255)
        .map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
        .reduce((s, v, i) => s + v * [0.2126, 0.7152, 0.0722][i], 0);
    const a = lum(rgb(getComputedStyle(el).color));
    const b = lum(rgb(getComputedStyle(bg || document.body).backgroundColor));
    return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
  });
}
