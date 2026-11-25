// Uji browser Sprint 13 (Bagian E), tanpa screenshot: console error, gulir horizontal di 360/768/1280/1920 px,
// tombol di bawah 44x44 px, keyboard (Tab, Enter, Escape, fokus kembali), dan bar aksi panel detail, di tema
// terang dan gelap. Hanya membuka halaman dan dialog lalu menutupnya; tidak ada data yang diubah.
//
// Cara menjalankan (backend di :8080 dan `npm run dev` frontend di :5173 sudah jalan, Microsoft Edge terpasang):
//   cd scripts/e2e && npm install
//   export DEV_ADMIN_PASSWORD=... DEV_STAFF_USERNAME=... DEV_STAFF_PASSWORD=... \
//          DEV_MANAGER_USERNAME=... DEV_MANAGER_PASSWORD=...
//   node uji13.mjs
// Variabel sama dengan yang dipakai scripts/dev-login.sh. Opsional: DEV_ADMIN_USERNAME (bawaan admin),
// E2E_BASE (bawaan http://localhost:5173), E2E_ONLY=popover (hanya uji popover profil, revisi 14),
// E2E_ONLY=subtaskform (hanya uji form tambah sub task yang dilipat, revisi 17),
// E2E_ONLY=subtaskresult (kelompok Arahan dan Hasil serta simpan catatan hasil, revisi 18; menulis catatan hasil
// ke satu sub task di database dev),
// E2E_ONLY=schedule (bagian Jadwal di form Buat task, revisi 19; tidak menyimpan apa pun),
// E2E_ONLY=todo (To Do pribadi di papan, revisi 28; menambah, mengubah, dan menghapus beberapa item
// To Do milik akun Staf di database dev, lalu membersihkannya),
// E2E_ONLY=activityfiles (lampiran Aktivitas Harian, revisi 24; mengunggah dua berkas kecil ke jam pertama
// milik akun Staf di database dev, lalu menghapusnya lagi),
// E2E_ONLY=activity (Aktivitas Harian, revisi 21; menulis satu sel aktivitas hari ini milik akun Staf dan
// satu sel lewat dialog Manajer di database dev, output-nya tautan file Drive karangan).
// E2E_ONLY=revision (revisi 22: dua kelompok PIC di kartu dan PIC sub task mengembalikan Ready ke On Progress;
// membuat satu task uji per lebar di database dev lewat API, dengan Manajer sebagai PIC task).
// E2E_ONLY=status (revisi 23: halaman Status hanya untuk Admin; Manajer dan Staf melihat 404 di /status dan tidak
// punya menu Status; tidak menyimpan apa pun).
// E2E_ONLY=createtask (revisi 26: pemilih berkas yang dibatalkan tidak menutup panel Buat task, panel tidak
// melompat saat pilih berkas diklik, Escape tetap menutup, dan halaman /task/baru; membuat satu task uji per lebar
// di database dev lewat form, dengan Manajer sebagai PIC).
// E2E_ONLY=timeline (revisi 30: grid dan agenda, keyboard dan popover tanggal, Manajer membuat satu kartu per
// kategori di hari ini lalu menghapusnya, Staf tanpa tombol tulis; alasan kehadiran terbuka sejak revisi 38).
// E2E_ONLY=chain (revisi 33: rantai sambung jam di kartu Aktivitas hari ini milik Staf, termasuk melintasi istirahat,
// 360 dan 1280 px; menulis lalu mengosongkan jam 11.00-12.00 sampai 16.00-17.00 hari ini).
// E2E_ONLY=centang (revisi 34: PIC task mencentang dan membatalkan centang sub task milik Manajer, nama pencentang
// tampil, Staf yang tidak terlibat dan Admin melihat kotak mati; membuat dua task uji lalu menghapusnya).
// E2E_ONLY=outputlinks (revisi 35: tiga tautan di Output jam 16.00-17.00 Staf yang disambung ke 17.00-18.00, di kartu
// papan, tabel /aktivitas, dan dialog detail tanpa gulir horizontal; jam 7 dan 8 Staf dikosongkan sesudahnya).
// E2E_ONLY=paste (revisi 36: tempel gambar di Hasil dan Arahan sub task masuk ke bagiannya, tempelan teks tetap teks,
// berkas di atas 50 MB ditolak; membuat satu task uji milik Manajer lalu menghapusnya).
// E2E_ONLY=aktivitasmundur (revisi 37: kartu Aktivitas mundur sampai 2 hari kerja dan batasnya; tidak menulis isian
// baru). E2E_ONLY=akses (revisi 38: drawer Ubah pengguna dengan dua centang akses, Staf dengan akses melihat tombol
// tulis Timeline dan Important Notes; akses Staf dikembalikan ke nilai semula).
// E2E_ONLY=beranda (revisi 39: login tanpa tujuan masuk Timeline, login dari tautan task kembali ke task itu, "/" dan
// tujuan ke situs lain berakhir di Timeline, urutan menu, logo ke Timeline; membuat satu task uji lalu menghapusnya).
// E2E_ONLY=completed (revisi 40: kolom Completed paling banyak 10 task dengan keterangan dan tautan Arsip; membuat 11
// task uji milik Manajer per lebar layar lalu menghapusnya).
// E2E_ONLY=wbdasar, wbkeyboard, wbponsel, wbadmin, atau wb1 untuk keempatnya (Whiteboard W1: papan akar, kartu
// papan, breadcrumb, alat klik dan seret, catatan, checklist, seret kartu, Tempat sampah, keyboard, layar sentuh,
// Admin hanya baca; kartu papan uji dipindah ke Tempat sampah di akhir).
// E2E_ONLY=wb2 untuk semua skenario Whiteboard W2 (atau satu per satu: wbhadir, wblangsung, wbseret, wbhapusedit,
// wbputus, wbundo, wbduatab) ditambah navbar (semua halaman, tiga peran, 768, 1024, 1279, dan 1280 px).
// E2E_ONLY=wb3 untuk semua skenario Whiteboard W3 (atau satu per satu: wbberkas, wbtautan, wbkolom, wbpenghubung,
// wbkomentar, wbgambar, wbtabel, wbpindah, wbzoom, wbdobel).
// Keluar dengan kode 1 bila ada yang gagal.
import fs from "node:fs";
import http from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import zlib from "node:zlib";
import { chromium } from "playwright-core";

const BASE = process.env.E2E_BASE || "http://localhost:5173";
const ONLY = process.env.E2E_ONLY;
const env = (name, fallback) => {
  const v = process.env[name] || fallback;
  if (!v) {
    console.error(
      `Variabel ${name} kosong. Lihat cara menjalankan di bagian atas berkas.`,
    );
    process.exit(2);
  }
  return v;
};
const ROLES = {
  admin: {
    user: env("DEV_ADMIN_USERNAME", "admin"),
    pass: env("DEV_ADMIN_PASSWORD"),
    pages: [
      "/task",
      "/arsip",
      "/aktivitas",
      "/timeline",
      "/pengguna",
      "/audit-log",
      "/pengaturan",
      "/profil",
      "/status",
      "TASK",
      "/tidak-ada",
    ],
  },
  manajer: {
    user: env("DEV_MANAGER_USERNAME"),
    pass: env("DEV_MANAGER_PASSWORD"),
    pages: [
      "/task",
      "/arsip",
      "/aktivitas",
      "/timeline",
      "/pengguna",
      "/audit-log",
      "/pengaturan",
      "/profil",
      "TASK",
      "/tidak-ada",
    ],
  },
  staf: {
    user: env("DEV_STAFF_USERNAME"),
    pass: env("DEV_STAFF_PASSWORD"),
    pages: [
      "/task",
      "/arsip",
      "/aktivitas",
      "/timeline",
      "/profil",
      "TASK",
      "/tidak-ada",
    ],
  },
};
const WIDTHS = [360, 768, 1280, 1920];
const THEMES = ["terang", "gelap"];

const fails = [];
const info = [];
let checks = 0;
const fail = (where, what) => fails.push(`${where} :: ${what}`);

const browser = await chromium.launch({ channel: "msedge", headless: true });

// Sesi per akun disimpan di scripts/e2e/.auth (gitignore) dan dipakai ulang antar run, supaya run beruntun tidak
// menghabiskan limiter login (10 percobaan per IP, satu kembali tiap 30 detik). Login sungguhan hanya bila sesi
// tersimpan tidak ada atau sudah tidak sah (dicek lewat /auth/me), sekali per akun. Skenario yang memang menguji
// login (form login, kembali ke tujuan asal) tetap login lewat form.
const AUTH_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), ".auth");
const sessions = {};
let realLogins = 0;
const authFile = (role) => path.join(AUTH_DIR, `${role}.json`);

async function sessionValid(state) {
  const ctx = await browser.newContext({ storageState: state });
  try {
    const res = await ctx.request.get(BASE + "/api/auth/me");
    return res.status() === 200;
  } finally {
    await ctx.close();
  }
}

async function freshLogin(role) {
  const ctx = await browser.newContext();
  const { user, pass } = ROLES[role];
  const res = await ctx.request.post(BASE + "/api/auth/login", {
    headers: { Origin: BASE },
    data: { username: user, password: pass },
  });
  realLogins++;
  if (res.status() !== 200) {
    console.error(`Login ${role} gagal: HTTP ${res.status()}`);
    process.exit(2);
  }
  sessions[role] = await ctx.storageState();
  await ctx.close();
  fs.mkdirSync(AUTH_DIR, { recursive: true });
  fs.writeFileSync(authFile(role), JSON.stringify(sessions[role]), { mode: 0o600 });
}

async function login(role) {
  try {
    const saved = JSON.parse(fs.readFileSync(authFile(role), "utf8"));
    if (await sessionValid(saved)) {
      sessions[role] = saved;
      return;
    }
  } catch {
    // Belum ada sesi tersimpan, atau berkasnya rusak: login sungguhan.
  }
  await freshLogin(role);
}

// Sesi kedaluwarsa di tengah run (401): login ulang sekali per akun, lalu sesi barunya dipakai skenario berikutnya.
const relogged = new Set();
async function refreshSession(role) {
  if (relogged.has(role)) return false;
  relogged.add(role);
  await freshLogin(role);
  return true;
}

async function open(role, width, theme) {
  const ctx = await browser.newContext({
    viewport: { width, height: width < 768 ? 800 : 900 },
    storageState: sessions[role],
  });
  await ctx.addInitScript((t) => {
    try {
      localStorage.setItem("nvm-tema", t);
    } catch {}
  }, theme);
  const page = await ctx.newPage();
  const errors = [];
  page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
  page.on("pageerror", (e) => errors.push(`pageerror ${e.message}`));
  return { ctx, page, errors };
}

// Pemeriksaan di satu tampilan: console, lebar, dan ukuran tombol yang terlihat.
async function audit(page, errors, where) {
  checks++;
  await page.waitForLoadState("networkidle");
  await page.waitForTimeout(150);
  if (errors.length)
    fail(where, `console: ${errors.splice(0).join(" | ").slice(0, 300)}`);
  const r = await page.evaluate(() => {
    const vw = window.innerWidth;
    const small = [];
    for (const b of document.querySelectorAll('button, [role="button"]')) {
      const s = getComputedStyle(b);
      const rect = b.getBoundingClientRect();
      if (
        s.visibility === "hidden" ||
        s.display === "none" ||
        rect.width <= 2 ||
        rect.height <= 2
      )
        continue;
      if (b.closest("[hidden], .sr-only") || b.offsetParent === null) continue;
      // Judul kartu task: area kliknya seluruh kartu lewat ::after (after:inset-0), jadi yang diukur kartunya.
      if (String(b.className).includes("after:inset-0")) {
        const card = b.closest("article")?.getBoundingClientRect();
        if (card && card.width >= 44 && card.height >= 44) continue;
      }
      // Pemicu profil memperluas area kliknya lewat ::after tanpa mengubah ukuran yang terlihat.
      const hit = b.hasAttribute("data-person-trigger")
        ? getComputedStyle(b, "::after")
        : null;
      const w = Math.max(rect.width, hit ? parseFloat(hit.width) || 0 : 0);
      const h = Math.max(rect.height, hit ? parseFloat(hit.height) || 0 : 0);
      if (w < 43.5 || h < 43.5) {
        const name = (b.getAttribute("aria-label") || b.textContent || "")
          .trim()
          .replace(/\s+/g, " ")
          .slice(0, 30);
        small.push(
          `"${name}" ${Math.round(rect.width)}x${Math.round(rect.height)}`,
        );
      }
    }
    return {
      sw: document.documentElement.scrollWidth,
      vw,
      small,
      theme: document.documentElement.dataset.theme,
    };
  });
  if (r.sw > r.vw) fail(where, `scrollWidth ${r.sw} > ${r.vw}`);
  if (r.small.length)
    fail(
      where,
      `tombol < 44px: ${[...new Set(r.small)].slice(0, 6).join(", ")}`,
    );
  return r;
}

// Bar aksi panel detail harus menempel di dasar panel: di awal, tengah, dan akhir gulir.
async function checkActionBar(page, where) {
  const res = await page.evaluate(async () => {
    const dialog = document.querySelector("dialog[open]");
    const bar = dialog?.querySelector(".sticky.mt-auto");
    if (!dialog || !bar) return { missing: true };
    const scroller = dialog.querySelector(".overflow-y-auto") || dialog;
    const out = [];
    for (const pos of ["awal", "tengah", "akhir"]) {
      scroller.scrollTop =
        pos === "awal"
          ? 0
          : pos === "tengah"
            ? scroller.scrollHeight / 2
            : scroller.scrollHeight;
      await new Promise((r) =>
        requestAnimationFrame(() => requestAnimationFrame(r)),
      );
      out.push({
        pos,
        gap: Math.round(
          dialog.getBoundingClientRect().bottom -
            bar.getBoundingClientRect().bottom,
        ),
      });
    }
    return { out };
  });
  checks++;
  // Panel hanya baca (tidak ada aksi untuk peran ini) memang tidak punya bar aksi.
  if (res.missing) return info.push(where);
  for (const o of res.out)
    if (Math.abs(o.gap) > 1)
      fail(where, `bar aksi tidak di dasar (${o.pos}, selisih ${o.gap}px)`);
}

// Tab sampai tombol utama, Enter membuka dialog, Escape menutup, fokus kembali ke pemicu.
async function checkKeyboard(page, where, matches) {
  checks++;
  for (let i = 0; i < 80; i++) {
    await page.keyboard.press("Tab");
    const hit = await page.evaluate((m) => {
      const el = document.activeElement;
      return (
        !!el &&
        el.tagName === "BUTTON" &&
        el.textContent.trim().replace(/\s+/g, " ").startsWith(m)
      );
    }, matches);
    if (hit) {
      await page.keyboard.press("Enter");
      const opened = await page
        .waitForSelector("dialog[open]", { timeout: 3000 })
        .then(
          () => true,
          () => false,
        );
      if (!opened)
        return fail(where, `Enter pada "${matches}" tidak membuka dialog`);
      await page.waitForTimeout(250);
      await page.keyboard.press("Escape");
      await page
        .waitForFunction(() => !document.querySelector("dialog[open]"), null, {
          timeout: 3000,
        })
        .catch(() => {});
      const back = await page.evaluate((m) => {
        const el = document.activeElement;
        return {
          closed: !document.querySelector("dialog[open]"),
          ok: !!el && el.textContent.trim().replace(/\s+/g, " ").startsWith(m),
        };
      }, matches);
      if (!back.closed) fail(where, "Escape tidak menutup dialog");
      else if (!back.ok) fail(where, `fokus tidak kembali ke "${matches}"`);
      return;
    }
  }
  fail(where, `Tab tidak sampai ke tombol "${matches}"`);
}

for (const role of Object.keys(ROLES)) await login(role);

for (const theme of ONLY ? [] : THEMES) {
  for (const [role, cfg] of Object.entries(ROLES)) {
    for (const width of WIDTHS) {
      const { ctx, page, errors } = await open(role, width, theme);
      // ID task pertama di papan, karena data dev berubah (T-0010 sudah tidak ada).
      await page.goto(BASE + "/task");
      await page.waitForLoadState("networkidle");
      const firstId = (
        await page
          .locator("article .specimen-tag")
          .first()
          .textContent()
          .catch(() => null)
      )?.trim();
      for (const raw of cfg.pages) {
        const path =
          raw === "TASK" ? (firstId ? `/task/${firstId}` : null) : raw;
        if (!path) continue;
        await page.goto(BASE + path);
        const r = await audit(
          page,
          errors,
          `${theme} ${role} ${width} ${path}`,
        );
        if (r.theme !== theme)
          fail(`${theme} ${role} ${width} ${path}`, `data-theme ${r.theme}`);
      }

      // Panel detail dari papan, lalu Detail sub task pertama bila ada.
      await page.goto(BASE + "/task");
      await page.waitForLoadState("networkidle");
      const card = page.locator("article h3 button").first();
      if (await card.count()) {
        await card.click();
        await page.waitForSelector("dialog[open]");
        await page.waitForTimeout(400);
        await audit(page, errors, `${theme} ${role} ${width} panel detail`);
        await checkActionBar(page, `${theme} ${role} ${width} panel detail`);
        const sub = page
          .locator("dialog[open] button[aria-controls^='sub-detail-']")
          .first();
        if (await sub.count()) {
          await sub.click();
          await page.waitForTimeout(250);
          await audit(
            page,
            errors,
            `${theme} ${role} ${width} detail sub task`,
          );
        }
        await page.keyboard.press("Escape");
        await page
          .waitForFunction(() => !document.querySelector("dialog[open]"))
          .catch(() => {});
      }

      // Form buat task (Manajer).
      if (role === "manajer") {
        await page.getByRole("button", { name: "Buat task" }).first().click();
        await page.waitForSelector("dialog[open]");
        await page.waitForTimeout(300);
        await audit(page, errors, `${theme} ${role} ${width} form buat task`);
        await page.keyboard.press("Escape");
        await page
          .waitForFunction(() => !document.querySelector("dialog[open]"))
          .catch(() => {});
      }

      // Keyboard sekali per peran per tema, di lebar desktop.
      if (width === 1280) {
        const target =
          role === "manajer"
            ? ["/task", "Buat task"]
            : role === "admin"
              ? ["/pengguna", "Tambah pengguna"]
              : ["/task", null];
        await page.goto(BASE + target[0]);
        await page.waitForLoadState("networkidle");
        let label = target[1];
        if (!label)
          label =
            (await page.locator("article h3 button").first().textContent())
              ?.trim()
              .replace(/\s+/g, " ") ?? "";
        if (label)
          await checkKeyboard(page, `${theme} ${role} keyboard`, label);
      }
      await ctx.close();
    }
  }
}
// Popover profil (revisi 14): klik dan keyboard, tutup lewat klik luar dan Escape, fokus kembali ke pemicu,
// tanpa gulir horizontal, dan di dalam panel detail Escape pertama hanya menutup popover.
const POP = '[popover] [role="dialog"]';
async function popoverState(page) {
  return page.evaluate((sel) => {
    const card = document.querySelector(sel);
    const el = document.activeElement;
    return {
      open: !!card && card.getBoundingClientRect().width > 0,
      focusInside: !!card && card.contains(el),
      focusTrigger:
        !!el &&
        el.hasAttribute("data-person-trigger") &&
        el.getAttribute("aria-expanded") === "false",
      panel: !!document.querySelector("dialog[open]"),
      sw: document.documentElement.scrollWidth,
      vw: window.innerWidth,
      name: card?.querySelector("h2")?.textContent?.trim() ?? "",
      loaded:
        !!card &&
        !card.querySelector('[role="status"]') &&
        !card.querySelector('[role="alert"]'),
    };
  }, POP);
}

async function checkPopover(width) {
  const { ctx, page, errors } = await open("manajer", width, "terang");
  const where = (what) => `popover ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  await page.goto(BASE + "/task?scope=all");
  await page.waitForLoadState("networkidle");
  const triggers = page.locator("article [data-person-trigger]");
  if ((await triggers.count()) === 0) {
    fail(where("papan"), "tidak ada bulatan PIC di papan");
    return ctx.close();
  }

  // Klik bulatan di kartu: popover terbuka, panel detail tidak ikut terbuka.
  await triggers.first().click();
  await page.waitForSelector(POP);
  await page.waitForTimeout(150);
  await page
    .waitForFunction(
      (sel) => !document.querySelector(sel + ' [role="status"]'),
      POP,
    )
    .catch(() => {});
  let st = await popoverState(page);
  expect(
    st.open && st.focusInside,
    "klik",
    "popover tidak terbuka atau fokus tidak pindah ke popover",
  );
  expect(!st.panel, "klik", "klik bulatan ikut membuka panel detail");
  expect(
    st.loaded,
    "klik",
    "kartu profil tidak termuat (galat atau masih memuat)",
  );
  expect(st.sw <= st.vw, "klik", `scrollWidth ${st.sw} > ${st.vw}`);

  // Klik di luar: popover tertutup, fokus kembali, dan kliknya tidak membuka apa pun.
  await page.mouse.click(Math.round(width / 2), 5);
  // Fokus kembali ke pemicu setelah urutan pointer selesai, bukan saat pointerdown.
  await page.waitForTimeout(100);
  st = await popoverState(page);
  expect(
    !st.open && st.focusTrigger && !st.panel,
    "klik luar",
    `tertutup ${!st.open}, fokus ${st.focusTrigger}, panel ${st.panel}`,
  );

  // Keyboard: Enter membuka, Escape menutup, fokus kembali.
  await triggers.first().focus();
  await page.keyboard.press("Enter");
  await page.waitForSelector(POP);
  await page.waitForTimeout(150);
  st = await popoverState(page);
  expect(
    st.open && st.focusInside,
    "Enter",
    "Enter tidak membuka popover atau fokus tidak pindah",
  );
  await page.keyboard.press("Escape");
  st = await popoverState(page);
  expect(
    !st.open && st.focusTrigger,
    "Escape",
    `tertutup ${!st.open}, fokus kembali ${st.focusTrigger}`,
  );

  // Pindah langsung ke pemicu lain (hanya kalau ada dua orang berbeda di papan, dan di lebar desktop).
  if (width >= 640 && (await triggers.count()) > 1) {
    await triggers.nth(0).click();
    await page.waitForSelector(POP);
    await page.waitForTimeout(150);
    const first = (await popoverState(page)).name;
    const other = triggers.nth(1);
    await other.click();
    await page.waitForTimeout(200);
    st = await popoverState(page);
    const expanded = await other.getAttribute("aria-expanded");
    expect(
      st.open && expanded === "true",
      "pindah",
      `popover tidak pindah (dari "${first}" ke "${st.name}")`,
    );
    await page.keyboard.press("Escape");
  }

  // Di dalam panel detail: klik luar dan Escape pertama hanya menutup popover.
  await page.locator("article h3 button").first().click();
  await page.waitForSelector("dialog[open]");
  // Detail dimuat setelah panel terbuka; tunggu pemicunya, bukan jeda tetap.
  await page
    .waitForSelector("dialog[open] [data-person-trigger]", { timeout: 5000 })
    .catch(() => {});
  const inPanel = page.locator("dialog[open] [data-person-trigger]").first();
  if (await inPanel.count()) {
    await inPanel.click();
    await page.waitForSelector(POP);
    await page.waitForTimeout(150);
    st = await popoverState(page);
    expect(
      st.open && st.panel,
      "panel",
      "popover tidak terbuka di dalam panel",
    );
    expect(st.sw <= st.vw, "panel", `scrollWidth ${st.sw} > ${st.vw}`);
    const box = await page.locator("dialog[open] h2").first().boundingBox();
    await page.mouse.click(box.x + 2, box.y + 2);
    st = await popoverState(page);
    expect(
      !st.open && st.panel,
      "panel klik luar",
      `popover tertutup ${!st.open}, panel tetap ${st.panel}`,
    );
    await inPanel.click();
    await page.waitForSelector(POP);
    await page.waitForTimeout(150);
    await page.keyboard.press("Escape");
    st = await popoverState(page);
    expect(
      !st.open && st.panel && st.focusTrigger,
      "panel Escape 1",
      `popover tertutup ${!st.open}, panel tetap ${st.panel}, fokus ${st.focusTrigger}`,
    );
    await page.keyboard.press("Escape");
    st = await popoverState(page);
    expect(!st.panel, "panel Escape 2", "Escape kedua tidak menutup panel");
  } else fail(where("panel"), "tidak ada pemicu profil di panel detail");

  checks++;
  if (errors.length)
    fail(where("console"), errors.splice(0).join(" | ").slice(0, 300));
  await ctx.close();
}

if (!ONLY || ONLY === "popover")
  for (const w of [360, 1280]) await checkPopover(w);

// Form tambah sub task di panel detail (revisi 17): tertutup saat panel dibuka, terbuka lewat klik dan Enter
// dengan fokus ke nama, Escape dan Batal menutupnya tanpa menutup panel, fokus kembali ke tombol, dan isian
// yang sudah diketik bertahan. Tidak ada yang disimpan.
async function checkSubtaskForm(width) {
  const { ctx, page, errors } = await open("manajer", width, "terang");
  const where = (what) => `form sub task ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  await page.goto(BASE + "/task?scope=all");
  await page.waitForLoadState("networkidle");
  // Task Requested atau On Progress: hanya di situ sub task boleh ditambah.
  const id = await page.evaluate(async () => {
    const b = await (await fetch("/api/board")).json();
    const t = b.columns
      .filter((c) => c.status === "requested" || c.status === "on_progress")
      .flatMap((c) => c.tasks)[0];
    return t?.id ?? null;
  });
  if (!id) {
    fail(where("papan"), "tidak ada task Requested atau On Progress");
    return ctx.close();
  }
  await page
    .locator("article", { has: page.locator(".specimen-tag", { hasText: id }) })
    .locator("h3 button")
    .click();
  await page.waitForSelector("dialog[open]");
  // Detail dimuat setelah panel terbuka; tunggu tombolnya, bukan jeda tetap.
  await page
    .waitForSelector("dialog[open] button[aria-controls='sub-new-form']", {
      timeout: 5000,
    })
    .catch(() => {});

  const state = () =>
    page.evaluate(() => {
      const trigger = [
        ...document.querySelectorAll(
          "dialog[open] button[aria-controls='sub-new-form']",
        ),
      ][0];
      const el = document.activeElement;
      return {
        panel: !!document.querySelector("dialog[open]"),
        form: !!document.getElementById("sub-new-form"),
        expanded: trigger?.getAttribute("aria-expanded"),
        hasTrigger: !!trigger,
        focusTitle: el?.id === "sub-new-title",
        focusTrigger: el === trigger,
        title: document.getElementById("sub-new-title")?.value ?? null,
        sw: document.documentElement.scrollWidth,
        vw: window.innerWidth,
      };
    });
  const trigger = page.locator(
    "dialog[open] button[aria-controls='sub-new-form']",
  );

  let st = await state();
  expect(st.hasTrigger, "awal", "tombol Tambah sub task tidak ada");
  expect(
    !st.form && st.expanded === "false",
    "awal",
    `form sudah terbuka (form ${st.form}, aria-expanded ${st.expanded})`,
  );

  await trigger.click();
  await page.waitForSelector("#sub-new-form");
  st = await state();
  expect(
    st.focusTitle && st.expanded === "true",
    "klik",
    `fokus ke nama ${st.focusTitle}, aria-expanded ${st.expanded}`,
  );
  expect(st.sw <= st.vw, "klik", `scrollWidth ${st.sw} > ${st.vw}`);

  await page.keyboard.type("Uji draf");
  await page.keyboard.press("Escape");
  await page.waitForTimeout(200);
  st = await state();
  expect(
    !st.form && st.panel && st.focusTrigger,
    "Escape",
    `form tertutup ${!st.form}, panel tetap ${st.panel}, fokus ke tombol ${st.focusTrigger}`,
  );

  // Keyboard: Enter pada tombol membuka lagi, isian tadi masih ada.
  await trigger.focus();
  await page.keyboard.press("Enter");
  await page.waitForSelector("#sub-new-form");
  st = await state();
  expect(st.focusTitle, "Enter", "Enter tidak memindahkan fokus ke nama");
  expect(
    st.title === "Uji draf",
    "draf",
    `isian tidak bertahan, dapat ${JSON.stringify(st.title)}`,
  );

  await page.getByRole("button", { name: "Batal", exact: true }).click();
  await page.waitForTimeout(200);
  st = await state();
  expect(
    !st.form && st.panel && st.focusTrigger,
    "Batal",
    `form tertutup ${!st.form}, panel tetap ${st.panel}, fokus ke tombol ${st.focusTrigger}`,
  );

  checks++;
  if (errors.length)
    fail(where("console"), errors.splice(0).join(" | ").slice(0, 300));
  await ctx.close();
}

if (!ONLY || ONLY === "subtaskform")
  for (const w of [360, 1280]) await checkSubtaskForm(w);

// Detail sub task (revisi 18): kelompok Arahan dan Hasil tampil, dan Simpan hasil menyimpan catatan hasil.
// Satu-satunya bagian yang menulis data: catatan hasil satu sub task di task Requested atau On Progress.
async function checkSubtaskResult(width) {
  const { ctx, page, errors } = await open("manajer", width, "terang");
  const where = (what) => `hasil sub task ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  await page.goto(BASE + "/task?scope=all");
  await page.waitForLoadState("networkidle");
  const target = await page.evaluate(async () => {
    const b = await (await fetch("/api/board")).json();
    const open = b.columns
      .filter((c) => c.status === "requested" || c.status === "on_progress")
      .flatMap((c) => c.tasks)
      .filter((t) => t.subtask_total > 0);
    for (const t of open) {
      const d = (await (await fetch(`/api/tasks/${t.id}`)).json()).task;
      const st = d.subtasks.find((x) => x.can_edit);
      if (st) return { id: t.id, sid: st.id };
    }
    return null;
  });
  if (!target) {
    fail(
      where("data"),
      "tidak ada sub task yang bisa diubah di task Requested atau On Progress",
    );
    return ctx.close();
  }
  await page
    .locator("article", {
      has: page.locator(".specimen-tag", { hasText: target.id }),
    })
    .locator("h3 button")
    .click();
  await page.waitForSelector("dialog[open]");
  const detailBtn = page.locator(
    `dialog[open] button[aria-controls='sub-detail-${target.sid}']`,
  );
  await detailBtn.waitFor({ timeout: 5000 });
  await detailBtn.click();
  const box = page.locator(`#sub-detail-${target.sid}`);
  await box.waitFor();

  const groups = await box.evaluate((el) =>
    [...el.querySelectorAll("section > :first-child")].map((h) =>
      h.textContent.trim(),
    ),
  );
  expect(
    groups[0] === "Arahan" && groups[1] === "Hasil",
    "kelompok",
    `judul kelompok ${JSON.stringify(groups)}`,
  );

  const note = `Uji e2e revisi 18, ${width} px, ${new Date().toISOString()}`;
  await page.locator(`#sub-note-${target.sid}`).fill(note);
  await page.getByRole("button", { name: "Simpan hasil" }).click();
  await page
    .waitForFunction(
      async ([id, sid, text]) => {
        const d = (await (await fetch(`/api/tasks/${id}`)).json()).task;
        return d.subtasks.find((x) => x.id === sid)?.result_note === text;
      },
      [target.id, target.sid, note],
      { timeout: 5000, polling: 250 },
    )
    .then(
      () => expect(true, "simpan", ""),
      () =>
        expect(false, "simpan", "catatan hasil tidak tersimpan dalam 5 detik"),
    );
  const alert = await page.locator(`dialog[open] li [role='alert']`).count();
  expect(alert === 0, "simpan", "muncul galat setelah Simpan hasil");
  const sw = await page.evaluate(() => [
    document.documentElement.scrollWidth,
    window.innerWidth,
  ]);
  expect(sw[0] <= sw[1], "lebar", `scrollWidth ${sw[0]} > ${sw[1]}`);

  checks++;
  if (errors.length)
    fail(where("console"), errors.splice(0).join(" | ").slice(0, 300));
  await ctx.close();
}

if (ONLY === "subtaskresult")
  for (const w of [360, 1280]) await checkSubtaskResult(w);

// Bagian Jadwal di form Buat task (revisi 19): setiap pilihan bisa dipilih, panah kiri dan kanan memindah
// pilihan, tombol hari berganti aria-pressed, dan setiap tampilan lolos audit ukuran tombol dan lebar.
async function checkSchedule(width) {
  const { ctx, page, errors } = await open("manajer", width, "terang");
  const where = (what) => `jadwal ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  await page.goto(BASE + "/task");
  await page.waitForLoadState("networkidle");
  await page.getByRole("button", { name: "Buat task" }).first().click();
  await page.waitForSelector("dialog[open]");
  const radio = (label) =>
    page
      .locator(`dialog[open] input[name='task-schedule']`)
      .locator(`xpath=..`)
      .filter({ hasText: label });

  for (const label of ["Harian", "Mingguan", "Bulanan"]) {
    await radio(label).click();
    await page.waitForTimeout(100);
    await audit(page, errors, where(label));
    const preview = await page
      .locator("dialog[open] p[aria-live='polite']")
      .filter({ hasText: "Muncul berikutnya" })
      .count();
    expect(preview > 0, label, "kalimat pratinjau tidak muncul");
  }

  // Mingguan: hari bawaan dari tanggal mulai (hari ini), lalu tambah satu hari lewat tombol.
  await radio("Mingguan").click();
  const pressed = () =>
    page.evaluate(
      () =>
        [
          ...document.querySelectorAll(
            "dialog[open] [aria-label='Hari pengulangan'] button[aria-pressed='true']",
          ),
        ].length,
    );
  const before = await pressed();
  expect(before === 1, "hari bawaan", `hari menyala ${before}, mau 1`);
  const off = page
    .locator(
      "dialog[open] [aria-label='Hari pengulangan'] button[aria-pressed='false']",
    )
    .first();
  await off.focus();
  await page.keyboard.press("Enter");
  expect(
    (await pressed()) === 2,
    "toggle hari",
    "Enter pada tombol hari tidak menyalakannya",
  );

  // Keyboard: panah kanan dari Tidak berulang pindah ke Harian.
  await page
    .locator("dialog[open] input[name='task-schedule'][value='none']")
    .focus();
  await page.keyboard.press("Space");
  await page.keyboard.press("ArrowRight");
  const checked = await page.evaluate(
    () =>
      document.querySelector("dialog[open] input[name='task-schedule']:checked")
        ?.value,
  );
  expect(
    checked === "daily",
    "panah",
    `pilihan setelah panah kanan ${checked}, mau daily`,
  );

  checks++;
  if (errors.length)
    fail(where("console"), errors.splice(0).join(" | ").slice(0, 300));
  await ctx.close();
}

if (ONLY === "schedule") for (const w of [360, 1280]) await checkSchedule(w);

// Aktivitas Harian (revisi 21): kartu di papan (tabel diputar mulai 768 px), halaman /aktivitas lebar penuh
// dengan mode Minggu sebagai bawaan, kepala tabel yang lengket, dan dialog sel.
async function checkActivity(width) {
  const { ctx, page, errors } = await open("staf", width, "terang");
  const where = (what) => `aktivitas ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const teks = `E2E ${Date.now()}`;
  const terlihat = (l) => l.locator("visible=true").first();
  // Tabel Aktivitas diganti kerangka memuat setiap kali mode atau tanggal berubah, jadi pemeriksaan
  // menunggu jumlah kepala harinya, bukan menunggu jeda.
  const tungguKepala = (want, minimal = false) =>
    page
      .waitForFunction(
        (arg) => {
          const n = document.querySelectorAll("th[scope='colgroup']").length;
          return arg.minimal ? n >= arg.want : n === arg.want;
        },
        { want, minimal },
        { timeout: 8000 },
      )
      .then(
        () => true,
        () => false,
      );
  // Nama isian: tabel yang diputar memakai aria-label, kartu per jam memakai label sr-only.
  const namaFokus = () =>
    page.evaluate(() => {
      const el = document.activeElement;
      if (!el) return "";
      return (
        el.getAttribute("aria-label") ||
        document.querySelector(`label[for="${el.id}"]`)?.textContent ||
        ""
      ).trim();
    });

  await page.goto(BASE + "/task");
  await page.waitForLoadState("networkidle");
  const judul = page.getByRole("button", { name: /Aktivitas hari ini/ });
  if ((await judul.getAttribute("aria-expanded")) === "false")
    await judul.click();
  await audit(page, errors, where("kartu papan"));

  // Mulai 768 px: tabel tiga baris dengan jam sebagai kepala kolom. Di bawahnya: kartu per jam, tanpa tabel.
  const tabel = await page.locator("table:visible").count();
  const kepalaJam = await page
    .locator("th[scope='col']:visible")
    .filter({ hasText: "09.00-10.00" })
    .count();
  if (width >= 768) {
    expect(tabel === 1, "tabel", `tabel terlihat ${tabel}, mau 1`);
    expect(kepalaJam === 1, "tabel", "jam bukan kepala kolom");
  } else {
    expect(tabel === 0, "kartu", "tabel tidak boleh terlihat di bawah 768 px");
    expect(kepalaJam === 0, "kartu", "kepala kolom jam masih terlihat");
  }
  const kotak = await page.locator("textarea:visible").count();
  expect(kotak === 16, "isian", `textarea terlihat ${kotak}, mau 16`);

  // Urutan Tab per jam: 09.00 Aktivitas, 09.00 Output, lalu 10.00 Aktivitas.
  await terlihat(page.getByLabel(/^Aktivitas 09\.00-10\.00/)).fill(teks);
  await page.keyboard.press("Tab");
  const kedua = await namaFokus();
  expect(
    /^Output 09\.00-10\.00/.test(kedua),
    "keyboard",
    `Tab dari Aktivitas 09.00 ke "${kedua}", mau Output 09.00-10.00`,
  );
  // Sejak revisi 24 berkas jam 09.00 (pratinjau, hapus, tambah) ada di antaranya. Itu boleh, asal setiap
  // tab stop punya nama: input berkas tersembunyi tanpa nama dulu ikut jadi tab stop (revisi 26).
  let ketiga = "";
  const lewat = [];
  for (let i = 0; i < 12; i++) {
    await page.keyboard.press("Tab");
    ketiga = await namaFokus();
    if (/^Aktivitas 10\.00-11\.00/.test(ketiga)) break;
    lewat.push(ketiga);
  }
  expect(
    /^Aktivitas 10\.00-11\.00/.test(ketiga),
    "keyboard",
    `Tab dari Output 09.00 tidak sampai ke Aktivitas 10.00-11.00 (lewat: ${lewat.join(" | ")})`,
  );
  expect(
    lewat.every((n) => n.trim() !== ""),
    "keyboard",
    `tab stop tanpa nama di antara Output 09.00 dan Aktivitas 10.00 (lewat: ${lewat.join(" | ")})`,
  );

  // Simpan otomatis sudah berjalan saat isian kehilangan fokus di langkah Tab di atas.
  const tersimpan = await terlihat(page.getByText("Tersimpan", { exact: true }))
    .waitFor({ timeout: 5000 })
    .then(
      () => true,
      () => false,
    );
  expect(tersimpan, "simpan", "status Tersimpan tidak muncul setelah blur");

  // Halaman Aktivitas: bawaan mode Minggu, lebar penuh, dan isian dari papan ikut tampil.
  await page.goto(BASE + "/aktivitas");
  await audit(page, errors, where("halaman"));
  const minggu = await page
    .getByRole("button", { name: "Minggu", exact: true })
    .getAttribute("aria-pressed");
  expect(
    minggu === "true",
    "mode",
    `mode bawaan aria-pressed ${minggu}, mau true`,
  );
  const adaMinggu = await tungguKepala(5, true);
  const hari = await page.locator("th[scope='colgroup']").count();
  expect(
    adaMinggu,
    "minggu",
    `kepala hari ${hari}, mau minimal 5 (Senin sampai Jumat)`,
  );
  const muncul = await page.getByText(teks).first().count();
  expect(muncul > 0, "halaman", "isian dari papan tidak muncul di /aktivitas");
  const baca = await page
    .getByText("Ubah isianmu lewat tabel di papan task.")
    .count();
  expect(baca > 0, "hak", "keterangan hanya baca tidak muncul");

  // Kepala nama, kolom jam, dan label hari tetap terlihat setelah tabel digulir 200 px ke bawah dan 300 px ke
  // kanan. Sel pojok dan sel jam harus menutupi isi di belakangnya: elementFromPoint di tengahnya mengembalikan
  // sel itu sendiri, bukan kepala nama atau isi sel yang tergulir ke bawahnya.
  const lengket = await page.evaluate(async () => {
    const wadah = [...document.querySelectorAll("div")].find(
      (d) => d.scrollHeight > d.clientHeight + 40 && d.querySelector("table"),
    );
    if (!wadah) return { missing: true };
    wadah.scrollTop = 200;
    wadah.scrollLeft = 300;
    wadah.scrollIntoView({ block: "start" });
    await new Promise((r) =>
      requestAnimationFrame(() => requestAnimationFrame(r)),
    );
    const box = wadah.getBoundingClientRect();
    const kepala = wadah.querySelector("thead th:nth-child(2)");
    const pojok = wadah.querySelector("thead th:first-child");
    const tampak = (el) => {
      const b = el.getBoundingClientRect();
      return (
        b.top > box.top + pojok.getBoundingClientRect().height &&
        b.bottom < Math.min(box.bottom, window.innerHeight)
      );
    };
    const jamSemua = [...wadah.querySelectorAll("tbody th[scope='row']")];
    const jam = jamSemua.find(tampak);
    const kena = (el) => {
      if (!el) return "tidak ada";
      const b = el.getBoundingClientRect();
      const h = document.elementFromPoint(
        b.left + b.width / 2,
        b.top + b.height / 2,
      );
      return h === el || el.contains(h)
        ? "ok"
        : `${h?.tagName} "${(h?.textContent || "").trim().slice(0, 30)}"`;
    };
    const label = [
      ...wadah.querySelectorAll("th[scope='colgroup'] > span"),
    ].find(tampak);
    return {
      atas: kepala.getBoundingClientRect().top - box.top,
      kiri: (jam ?? jamSemua[0]).getBoundingClientRect().left - box.left,
      scrollTop: wadah.scrollTop,
      scrollLeft: wadah.scrollLeft,
      pojok: kena(pojok),
      jam: kena(jam),
      labelKiri: label ? label.getBoundingClientRect().left - box.left : null,
      labelKena: kena(label),
    };
  });
  expect(
    !lengket.missing,
    "lengket",
    "wadah tabel yang bergulir tidak ditemukan",
  );
  if (!lengket.missing) {
    expect(
      lengket.scrollTop > 0 && Math.abs(lengket.atas) <= 4,
      "lengket",
      `kepala nama bergeser ${Math.round(lengket.atas)} px setelah digulir`,
    );
    expect(
      Math.abs(lengket.kiri) <= 4,
      "lengket",
      `kolom jam bergeser ${Math.round(lengket.kiri)} px setelah digulir ke samping`,
    );
    expect(
      lengket.pojok === "ok",
      "pojok",
      `tengah sel pojok tertutup ${lengket.pojok}`,
    );
    expect(
      lengket.jam === "ok",
      "pojok",
      `tengah sel jam tertutup ${lengket.jam}`,
    );
    // Label hari hanya bisa diuji bila tabelnya memang bergulir ke samping di lebar ini.
    if (lengket.scrollLeft > 0) {
      expect(
        lengket.labelKiri !== null &&
          Math.abs(lengket.labelKiri) <= 4 &&
          lengket.labelKena === "ok",
        "label hari",
        `label hari di ${lengket.labelKiri} px dari tepi wadah (${lengket.labelKena}) setelah digulir ${lengket.scrollLeft} px`,
      );
    }
  }

  // Panel filter Orang di atas kepala tabel yang lengket: dibuka setelah tabel digulir ke bawah, item pertamanya
  // harus menjadi elemen teratas di titik tengahnya. Lalu Escape menutup dan fokus kembali ke tombol, dan klik
  // di luar panel menutupnya.
  if (width === 360 || width === 1280) {
    const tombolOrang = page.locator("button[aria-controls='aktivitas-orang']");
    await tombolOrang.click();
    const item = page.locator("#aktivitas-orang li label").first();
    await item.waitFor({ timeout: 3000 });
    await item.scrollIntoViewIfNeeded();
    const atas = await item.evaluate((el) => {
      const b = el.getBoundingClientRect();
      const h = document.elementFromPoint(
        b.left + b.width / 2,
        b.top + b.height / 2,
      );
      return h === el || el.contains(h)
        ? "ok"
        : `${h?.tagName} "${(h?.textContent || "").trim().slice(0, 30)}"`;
    });
    expect(atas === "ok", "panel orang", `item pertama panel tertutup ${atas}`);
    await page.keyboard.press("Escape");
    const tertutup = (await page.locator("#aktivitas-orang").count()) === 0;
    const fokusTombol = await tombolOrang.evaluate(
      (b) => document.activeElement === b,
    );
    expect(
      tertutup && fokusTombol,
      "panel orang",
      `Escape: tertutup ${tertutup}, fokus di tombol ${fokusTombol}`,
    );
    await tombolOrang.click();
    await page.locator("h1").first().click();
    const klikLuar = (await page.locator("#aktivitas-orang").count()) === 0;
    expect(klikLuar, "panel orang", "klik di luar panel tidak menutupnya");
    const label = (await tombolOrang.textContent())?.trim();
    expect(
      label === "Orang",
      "panel orang",
      `label tombol tanpa pilihan "${label}", mau "Orang"`,
    );
  }

  // Dialog sel dengan keyboard: Enter membuka, Escape menutup, fokus kembali ke sel yang sama.
  const sel = page.locator("tbody td button").first();
  const labelSel = await sel.getAttribute("aria-label");
  await sel.focus();
  await page.keyboard.press("Enter");
  const terbuka = await page
    .waitForSelector("dialog[open]", { timeout: 3000 })
    .then(
      () => true,
      () => false,
    );
  expect(terbuka, "dialog", "Enter pada sel tidak membuka dialog");
  if (terbuka) {
    const isian = await page.locator("dialog[open] textarea").count();
    expect(isian === 0, "dialog", `Staf dapat ${isian} isian di dialog, mau 0`);
    const ubah = await page
      .locator("dialog[open]")
      .getByRole("button", { name: "Ubah", exact: true })
      .count();
    expect(ubah === 0, "dialog", "Staf melihat tombol Ubah");
    await page.keyboard.press("Escape");
    await page
      .waitForFunction(() => !document.querySelector("dialog[open]"), null, {
        timeout: 3000,
      })
      .catch(() => {});
    await page.waitForTimeout(150);
    const kembali = await page.evaluate(
      () => document.activeElement?.getAttribute("aria-label") ?? "",
    );
    expect(
      kembali === labelSel,
      "dialog",
      `fokus kembali ke "${kembali}", mau "${labelSel}"`,
    );
  }

  // Petunjuk "Ubah isianmu..." di dialog hanya untuk isian milik Staf itu sendiri (sel yang diisi di atas).
  const petunjuk = "Ubah isianmu lewat tabel di papan task.";
  const tutupDialog = async () => {
    await page.keyboard.press("Escape");
    await page
      .waitForFunction(() => !document.querySelector("dialog[open]"), null, {
        timeout: 3000,
      })
      .catch(() => {});
  };
  const selSendiri = page
    .locator("tbody td", { hasText: teks })
    .first()
    .locator("button");
  const labelSendiri = (await selSendiri.getAttribute("aria-label")) ?? "";
  await selSendiri.click();
  await page.waitForSelector("dialog[open]");
  const adaPetunjuk = await page
    .locator("dialog[open]")
    .getByText(petunjuk)
    .count();
  expect(
    adaPetunjuk === 1,
    "petunjuk",
    "petunjuk tidak muncul di isian sendiri",
  );
  await tutupDialog();
  const nama = labelSendiri.replace(/^Lihat aktivitas /, "").split(",")[0];
  const labelLain = await page.evaluate(
    (n) =>
      [...document.querySelectorAll("tbody td button")]
        .map((b) => b.getAttribute("aria-label") ?? "")
        .find((l) => !l.startsWith(`Lihat aktivitas ${n},`)) ?? "",
    nama,
  );
  if (labelLain) {
    await page.locator(`tbody td button[aria-label="${labelLain}"]`).click();
    await page.waitForSelector("dialog[open]");
    const lainPetunjuk = await page
      .locator("dialog[open]")
      .getByText(petunjuk)
      .count();
    expect(
      lainPetunjuk === 0,
      "petunjuk",
      "petunjuk muncul di isian orang lain",
    );
    await tutupDialog();
  } else
    info.push(
      "aktivitas petunjuk: hanya satu Staf tampil, isian orang lain tidak diuji",
    );

  // Mode Hari: segmented berganti, tombol hari menyala, dan hanya satu kepala hari yang tersisa.
  await page.getByRole("button", { name: "Hari", exact: true }).click();
  await page.waitForURL(/mode=hari/, { timeout: 8000 });
  const adaSatuHari = await tungguKepala(1);
  const satuHari = await page.locator("th[scope='colgroup']").count();
  expect(adaSatuHari, "mode hari", `kepala hari ${satuHari}, mau 1`);
  await audit(page, errors, where("mode hari"));
  const tombolHari = page.locator(
    "[aria-label='Hari dalam minggu ini'] button",
  );
  const tanggalHari = await tombolHari.first().textContent();
  await tombolHari.first().click();
  await page
    .waitForFunction(
      () =>
        document
          .querySelector("[aria-label='Hari dalam minggu ini'] button")
          ?.getAttribute("aria-pressed") === "true",
      null,
      { timeout: 8000 },
    )
    .catch(() => {});
  await tungguKepala(1);
  const menyala = await tombolHari.first().getAttribute("aria-pressed");
  expect(
    !!tanggalHari?.trim(),
    "mode hari",
    "tombol hari tidak memuat nama hari dan tanggalnya",
  );
  expect(
    menyala === "true",
    "mode hari",
    "tombol hari tidak menyala setelah diklik",
  );
  expect(
    page.url().includes("mode=hari"),
    "url",
    `mode tidak tersimpan di URL: ${page.url()}`,
  );

  const lebar = await page.evaluate(() => [
    document.documentElement.scrollWidth,
    window.innerWidth,
  ]);
  expect(
    lebar[0] <= lebar[1],
    "lebar",
    `scrollWidth ${lebar[0]} > ${lebar[1]}`,
  );

  checks++;
  if (errors.length)
    fail(where("console"), errors.splice(0).join(" | ").slice(0, 300));
  await ctx.close();
}

// Manajer: dialog dibuka dalam mode lihat, Ubah membuka form, Batal kembali ke mode lihat, Simpan kembali ke
// mode lihat dengan isi terbaru, dan Escape di form yang berubah hanya kembali ke mode lihat.
async function checkActivityManager(width) {
  const { ctx, page, errors } = await open("manajer", width, "terang");
  const where = (what) => `aktivitas manajer ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const teks = `E2E manajer ${Date.now()}`;
  // Id file Drive karangan: thumbnail-nya pasti gagal, jadi yang diuji adalah area gambar yang disembunyikan.
  const drive =
    "https://drive.google.com/file/d/1E2EcontohTidakAda/view?usp=sharing";
  const dialog = page.locator("dialog[open]");
  const tombol = (name) => dialog.getByRole("button", { name, exact: true });
  const fokus = () =>
    page.evaluate(() => {
      const el = document.activeElement;
      return el?.id || el?.textContent?.trim() || "";
    });
  const textareaCount = () => dialog.locator("textarea").count();

  await page.goto(BASE + "/aktivitas");
  await page
    .waitForSelector("tbody td button", { timeout: 8000 })
    .catch(() => {});
  await audit(page, errors, where("halaman"));
  const sel = page.locator("tbody td button").first();
  const labelSel = await sel.getAttribute("aria-label");
  await sel.click();
  await page.waitForSelector("dialog[open]");
  expect(
    (await textareaCount()) === 0,
    "lihat",
    "dialog tidak dibuka dalam mode lihat",
  );
  expect(
    (await tombol("Ubah").count()) === 1,
    "lihat",
    "tombol Ubah tidak ada",
  );
  const petunjuk = await dialog
    .getByText("Ubah isianmu lewat tabel di papan task.")
    .count();
  expect(petunjuk === 0, "lihat", "petunjuk Staf muncul untuk Manajer");

  await tombol("Ubah").click();
  expect((await textareaCount()) === 2, "ubah", "Ubah tidak membuka form");
  expect(
    (await fokus()) === "sel-aktivitas",
    "ubah",
    `fokus di "${await fokus()}", mau sel-aktivitas`,
  );
  await audit(page, errors, where("form"));
  await tombol("Batal").click();
  expect(
    (await textareaCount()) === 0,
    "batal",
    "Batal tidak kembali ke mode lihat",
  );
  expect(
    (await fokus()) === "Ubah",
    "batal",
    `fokus setelah Batal di "${await fokus()}", mau Ubah`,
  );

  await tombol("Ubah").click();
  await dialog.locator("#sel-aktivitas").fill(teks);
  await dialog.locator("#sel-output").fill(drive);
  await tombol("Simpan").click();
  const lihatLagi = await page
    .waitForFunction(
      () => document.querySelectorAll("dialog[open] textarea").length === 0,
      null,
      {
        timeout: 5000,
      },
    )
    .then(
      () => true,
      () => false,
    );
  expect(lihatLagi, "simpan", "Simpan tidak kembali ke mode lihat");
  expect(
    (await dialog.getByText(teks).count()) === 1,
    "simpan",
    "isi terbaru tidak tampil di dialog",
  );
  expect(
    (await fokus()) === "Ubah",
    "simpan",
    `fokus setelah Simpan di "${await fokus()}", mau Ubah`,
  );
  expect(
    (await dialog.getByText("File Drive", { exact: true }).count()) === 1,
    "drive",
    "label File Drive tidak ada",
  );
  const tautan = await dialog.locator(`a[href="${drive}"]`).getAttribute("rel");
  expect(
    tautan === "noopener noreferrer",
    "drive",
    `URL lengkap tidak tampil sebagai tautan (rel ${tautan})`,
  );
  // Thumbnail gagal: area gambarnya hilang, tanpa ikon gambar rusak.
  const gambarHilang = await page
    .waitForFunction(() => !document.querySelector("dialog[open] img"), null, {
      timeout: 10000,
    })
    .then(
      () => true,
      () => false,
    );
  expect(gambarHilang, "drive", "area thumbnail yang gagal masih tampil");
  await audit(page, errors, where("lihat drive"));

  // Escape bertahap: form yang sudah berubah kembali ke mode lihat dulu, Escape berikutnya menutup.
  await tombol("Ubah").click();
  await dialog.locator("#sel-aktivitas").fill(teks + " belum disimpan");
  await page.keyboard.press("Escape");
  await page.waitForTimeout(150);
  const masihBuka = await page.locator("dialog[open]").count();
  expect(
    masihBuka === 1 && (await textareaCount()) === 0,
    "escape",
    "Escape pertama tidak kembali ke mode lihat",
  );
  expect(
    (await dialog.getByText(teks + " belum disimpan").count()) === 0,
    "escape",
    "perubahan yang dibatalkan tampil",
  );
  await page.keyboard.press("Escape");
  const tertutup = await page
    .waitForFunction(() => !document.querySelector("dialog[open]"), null, {
      timeout: 3000,
    })
    .then(
      () => true,
      () => false,
    );
  expect(tertutup, "escape", "Escape kedua tidak menutup dialog");
  await page.waitForTimeout(150);
  const kembali = await page.evaluate(
    () => document.activeElement?.getAttribute("aria-label") ?? "",
  );
  expect(
    kembali === labelSel,
    "escape",
    `fokus kembali ke "${kembali}", mau "${labelSel}"`,
  );

  const diTabel = await page
    .locator("tbody td", { hasText: teks })
    .locator("a", { hasText: "File Drive" })
    .count();
  expect(
    diTabel === 1,
    "tabel",
    "sel di tabel tidak menampilkan isi baru dan chip File Drive",
  );

  const lebar = await page.evaluate(() => [
    document.documentElement.scrollWidth,
    window.innerWidth,
  ]);
  expect(
    lebar[0] <= lebar[1],
    "lebar",
    `scrollWidth ${lebar[0]} > ${lebar[1]}`,
  );

  checks++;
  if (errors.length)
    fail(where("console"), errors.splice(0).join(" | ").slice(0, 300));
  await ctx.close();
}

if (ONLY === "activity") {
  for (const w of [360, 768, 1280, 1920]) await checkActivity(w);
  for (const w of [360, 1280]) await checkActivityManager(w);
}

// Revisi 22: kartu menampilkan kelompok "PIC" dan "Sub task", dan PIC sub task (bukan PIC task) mengembalikan
// task Ready ke On Progress dengan memilih sub task miliknya. Datanya disiapkan lewat API: Manajer PIC task,
// Staf memegang dua sub task, Manajer satu sub task; semuanya dicentang lalu task dipindah ke Ready.
async function apiAs(role) {
  let ctx = await browser.newContext({ storageState: sessions[role] });
  if ((await ctx.request.get(BASE + "/api/auth/me")).status() === 401 && (await refreshSession(role))) {
    await ctx.close();
    ctx = await browser.newContext({ storageState: sessions[role] });
  }
  const call = async (method, path, data) => {
    const res = await ctx.request.fetch(BASE + "/api" + path, {
      method,
      headers: { Origin: BASE },
      data,
    });
    const text = await res.text();
    return { status: res.status(), body: text ? JSON.parse(text) : null };
  };
  const me = (await call("GET", "/auth/me")).body.user;
  return { ctx, call, me };
}

async function setupRevisionTask(title) {
  const mgr = await apiAs("manajer");
  const staf = await apiAs("staf");
  const created = await mgr.call("POST", "/tasks", {
    title,
    urgency: "medium",
    pic_ids: [mgr.me.id],
    label_ids: [],
    subtasks: [
      { title: "Naskah E2E", pic_id: staf.me.id },
      { title: "Desain E2E", pic_id: staf.me.id },
      { title: "Foto E2E", pic_id: mgr.me.id },
    ],
  });
  if (created.status !== 201)
    throw new Error(
      `buat task ${created.status} ${JSON.stringify(created.body)}`,
    );
  const id = created.body.id;
  // GET /tasks/:id membungkus isinya di kunci "task".
  const detail = async () => (await mgr.call("GET", `/tasks/${id}`)).body.task;
  let d = await detail();
  await mgr.call("POST", `/tasks/${id}/move`, {
    version: d.version,
    to: "on_progress",
    note: "",
  });
  d = await detail();
  for (const st of d.subtasks) {
    const who = st.title === "Foto E2E" ? mgr : staf;
    const r = await who.call("PUT", `/tasks/${id}/subtasks/${st.id}/done`, {
      done: true,
    });
    if (r.status >= 300) throw new Error(`centang ${st.title} ${r.status}`);
  }
  d = await detail();
  const ready = await mgr.call("POST", `/tasks/${id}/move`, {
    version: d.version,
    to: "ready",
    note: "",
  });
  if (ready.status !== 200)
    throw new Error(`ke Ready ${ready.status} ${JSON.stringify(ready.body)}`);
  await staf.ctx.close();
  return { id, mgr, detail };
}

async function checkRevision(width) {
  const where = (what) => `revisi ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const title = `E2E revisi ${width} ${Date.now()}`;
  const { id, mgr, detail } = await setupRevisionTask(title);

  const { ctx, page, errors } = await open("staf", width, "terang");
  await page.goto(BASE + "/task");
  await page.waitForLoadState("networkidle");
  // Di layar sempit papan hanya menampilkan satu kolom; task uji ada di kolom Ready.
  const pilihKolom = page.locator("[aria-label='Pilih kolom']");
  if (await pilihKolom.isVisible())
    await pilihKolom.getByRole("button", { name: /^Ready/ }).click();
  const kartu = page.locator("article", {
    has: page.locator("h3", { hasText: title }),
  });
  await kartu.waitFor({ timeout: 8000 });
  await kartu.scrollIntoViewIfNeeded();

  // Dua kelompok berlabel: PIC (Manajer) dan Sub task (Staf dan Manajer, urut posisi sub task).
  const grup = async (label) =>
    kartu
      .locator("ul[aria-labelledby]")
      .evaluateAll(
        (uls, l) =>
          uls
            .filter(
              (u) =>
                document
                  .getElementById(u.getAttribute("aria-labelledby") ?? "")
                  ?.textContent?.trim() === l,
            )
            .map((u) => u.querySelectorAll("li").length),
        label,
      );
  const pic = await grup("PIC");
  const sub = await grup("Sub task");
  expect(
    pic.length === 1 && pic[0] === 1,
    "kartu",
    `kelompok PIC ${JSON.stringify(pic)}, mau satu kelompok berisi 1`,
  );
  expect(
    sub.length === 1 && sub[0] === 2,
    "kartu",
    `kelompok Sub task ${JSON.stringify(sub)}, mau satu kelompok berisi 2`,
  );
  await audit(page, errors, where("papan"));

  // Dialog Pindahkan: tujuan satu (On Progress), daftar sub task milik Staf terpilih semua, alasan opsional.
  await kartu.getByRole("button", { name: /Pindahkan ke On Progress/ }).click();
  await page.waitForSelector("dialog[open]");
  const dialog = page.locator("dialog[open]");
  const legend = await dialog
    .getByText("Sub task yang direvisi", { exact: true })
    .count();
  expect(legend === 1, "dialog", "judul daftar sub task tidak ada");
  const kotak = dialog.locator("input[type='checkbox']");
  const total = await kotak.count();
  const tercentang = await dialog
    .locator("input[type='checkbox']:checked")
    .count();
  expect(
    total === 2 && tercentang === 2,
    "dialog",
    `centang ${tercentang} dari ${total}, mau 2 dari 2`,
  );
  const foto = await dialog.getByText("Foto E2E").count();
  expect(foto === 0, "dialog", "sub task milik orang lain ikut ditawarkan");
  expect(
    (await dialog.getByLabel("Alasan revisi (opsional)").count()) === 1,
    "dialog",
    "textarea alasan tidak ada",
  );
  await audit(page, errors, where("dialog"));

  await dialog.getByLabel("Desain E2E").uncheck();
  await dialog
    .getByLabel("Alasan revisi (opsional)")
    .fill("Judul terlalu panjang");
  await page.getByRole("button", { name: "Kembalikan ke On Progress" }).click();
  const tutup = await page
    .waitForFunction(() => !document.querySelector("dialog[open]"), null, {
      timeout: 5000,
    })
    .then(
      () => true,
      () => false,
    );
  expect(
    tutup,
    "kembalikan",
    "dialog tidak tertutup setelah Kembalikan ke On Progress",
  );

  const d = await detail();
  const done = Object.fromEntries(
    d.subtasks.map((st) => [st.title, !!st.done_at]),
  );
  expect(
    d.status === "on_progress",
    "kembalikan",
    `status ${d.status}, mau on_progress`,
  );
  expect(
    !done["Naskah E2E"] && done["Desain E2E"] && done["Foto E2E"],
    "kembalikan",
    `centang setelah revisi ${JSON.stringify(done)}, mau hanya Naskah yang terbuka`,
  );
  const riwayat = d.history.find((h) => h.new_value?.reopened_subtasks);
  expect(
    riwayat?.new_value?.reason === "Judul terlalu panjang" &&
      riwayat.new_value.reopened_subtasks.length === 1,
    "riwayat",
    `riwayat revisi ${JSON.stringify(riwayat?.new_value)}`,
  );

  const lebar = await page.evaluate(() => [
    document.documentElement.scrollWidth,
    window.innerWidth,
  ]);
  expect(
    lebar[0] <= lebar[1],
    "lebar",
    `scrollWidth ${lebar[0]} > ${lebar[1]}`,
  );
  checks++;
  if (errors.length)
    fail(where("console"), errors.splice(0).join(" | ").slice(0, 300));
  await ctx.close();
  await mgr.ctx.close();
  console.log(`  task uji ${id}`);
}

if (ONLY === "revision") for (const w of [360, 1280]) await checkRevision(w);

// Revisi 23: /status hanya untuk Admin. Peran lain melihat halaman 404 yang sudah ada dengan URL tetap /status,
// dan tautan Status tidak ada di top bar maupun menu Lainnya. Admin tetap melihat halamannya.
async function checkStatusPage(role, width) {
  const { ctx, page, errors } = await open(role, width, "terang");
  const where = (what) => `status ${role} ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  await page.goto(BASE + "/status");
  await page.waitForLoadState("networkidle");
  const judul = (await page.locator("h1").first().textContent())?.trim();
  const path = new URL(page.url()).pathname;
  // Semua tautan ke /status di DOM, termasuk menu Lainnya yang tersembunyi.
  const tautan = await page.locator("a[href='/status']").count();
  if (role === "admin") {
    expect(
      judul === "Status sistem",
      "halaman",
      `judul "${judul}", mau "Status sistem"`,
    );
    expect(tautan > 0, "menu", "Admin tidak punya tautan Status");
  } else {
    expect(
      judul === "Halaman tidak ditemukan",
      "404",
      `judul "${judul}", mau "Halaman tidak ditemukan"`,
    );
    expect(
      path === "/status",
      "404",
      `URL berubah ke ${path}, mau tetap /status`,
    );
    expect(
      tautan === 0,
      "menu",
      `${tautan} tautan Status masih ada di navigasi`,
    );
  }
  await audit(page, errors, where("audit"));
  await ctx.close();
}

if (ONLY === "status")
  for (const role of ["admin", "manajer", "staf"])
    for (const w of [360, 1280]) await checkStatusPage(role, w);

// Lampiran Aktivitas Harian (revisi 24): unggah lewat input berkas, paste gambar, pratinjau yang dibuka dan
// ditutup dengan keyboard, hapus, dan tanpa gulir horizontal halaman.
async function checkActivityFiles(width) {
  const { ctx, page, errors } = await open("staf", width, "terang");
  const where = (what) => `lampiran ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  // PNG 1x1 yang lolos pemeriksaan magic bytes di backend.
  const pngBase64 =
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg==";
  const nama = `uji-${Date.now()}.png`;

  await page.goto(BASE + "/task");
  await page.waitForLoadState("networkidle");
  const judul = page.getByRole("button", { name: /Aktivitas hari ini/ });
  if ((await judul.getAttribute("aria-expanded")) === "false")
    await judul.click();
  await page.waitForSelector("textarea:visible");

  // Unggah lewat input berkas jam pertama. Input-nya sr-only, jadi diisi langsung, bukan lewat klik tombol.
  const input = page.locator("input[type=file]").first();
  await input.setInputFiles({
    name: nama,
    mimeType: "image/png",
    buffer: Buffer.from(pngBase64, "base64"),
  });
  const munculLampiran = await page
    .getByRole("button", { name: `Pratinjau ${nama}` })
    .first()
    .waitFor({ timeout: 15000 })
    .then(
      () => true,
      () => false,
    );
  expect(munculLampiran, "unggah", "lampiran tidak muncul setelah diunggah");
  await audit(page, errors, where("kartu dengan lampiran"));

  // Paste gambar di textarea Aktivitas jam pertama: namanya dibuat sendiri (tangkapan-09.00-n.png).
  const sebelum = await page.locator("input[type=file]").count();
  expect(sebelum > 0, "paste", "input berkas tidak ditemukan");
  await page.evaluate(async (b64) => {
    const area = document.querySelector("textarea");
    if (!area) throw new Error("textarea tidak ada");
    const bin = atob(b64);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    const file = new File([bytes], "image.png", { type: "image/png" });
    const dt = new DataTransfer();
    dt.items.add(file);
    area.focus();
    area.dispatchEvent(
      new ClipboardEvent("paste", {
        clipboardData: dt,
        bubbles: true,
        cancelable: true,
      }),
    );
  }, pngBase64);
  const adaTangkapan = await page
    .getByRole("button", { name: /Pratinjau tangkapan-09\.00-\d+\.png/ })
    .first()
    .waitFor({ timeout: 15000 })
    .then(
      () => true,
      () => false,
    );
  expect(adaTangkapan, "paste", "gambar yang ditempel tidak terunggah");

  // Pratinjau dengan keyboard: Enter membuka dialog, Escape menutupnya.
  const pemicu = page
    .getByRole("button", { name: `Pratinjau ${nama}` })
    .first();
  await pemicu.focus();
  await page.keyboard.press("Enter");
  const terbuka = await page
    .waitForSelector("dialog[open]", { timeout: 5000 })
    .then(
      () => true,
      () => false,
    );
  expect(terbuka, "pratinjau", "Enter tidak membuka pratinjau");
  if (terbuka) {
    const gambar = await page
      .locator("dialog[open] img")
      .first()
      .waitFor({ timeout: 10000 })
      .then(
        () => true,
        () => false,
      );
    expect(gambar, "pratinjau", "gambar pratinjau tidak dimuat");
    await audit(page, errors, where("dialog pratinjau"));
    await page.keyboard.press("Escape");
    const tertutup = await page
      .waitForFunction(() => !document.querySelector("dialog[open]"), null, {
        timeout: 5000,
      })
      .then(
        () => true,
        () => false,
      );
    expect(tertutup, "pratinjau", "Escape tidak menutup pratinjau");
  }

  // Hapus: tombol hapus membuka konfirmasi, lalu lampirannya hilang dari sel.
  await page
    .getByRole("button", { name: `Hapus ${nama}` })
    .first()
    .click();
  await page.waitForSelector("dialog[open]");
  await page
    .locator("dialog[open]")
    .getByRole("button", { name: "Hapus", exact: true })
    .click();
  const hilang = await page
    .getByRole("button", { name: `Pratinjau ${nama}` })
    .first()
    .waitFor({ state: "detached", timeout: 15000 })
    .then(
      () => true,
      () => false,
    );
  expect(hilang, "hapus", "lampiran masih ada setelah dihapus");

  const lebar = await page.evaluate(() => [
    document.documentElement.scrollWidth,
    window.innerWidth,
  ]);
  expect(
    lebar[0] <= lebar[1],
    "lebar",
    `scrollWidth ${lebar[0]} > ${lebar[1]}`,
  );

  // Sisa tangkapan layar dibersihkan supaya database dev tidak menumpuk berkas uji.
  for (const tombol of await page
    .getByRole("button", { name: /^Hapus tangkapan-09\.00-\d+\.png$/ })
    .all()) {
    await tombol.click();
    await page.waitForSelector("dialog[open]");
    await page
      .locator("dialog[open]")
      .getByRole("button", { name: "Hapus", exact: true })
      .click();
    await page.waitForTimeout(500);
  }

  checks++;
  if (errors.length)
    fail(where("console"), errors.splice(0).join(" | ").slice(0, 300));
  await ctx.close();
}

if (ONLY === "activityfiles")
  for (const w of [360, 1280]) await checkActivityFiles(w);

// Revisi 26. Dialog pemilih berkas OS tidak bisa diklik Playwright, jadi pembatalannya ditiru dengan event cancel
// yang bubble dari input berkas, sama seperti yang dikirim browser.
async function checkCreateTask(width) {
  const { ctx, page, errors } = await open("manajer", width, "terang");
  const where = (what) => `buat task ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  await page.goto(BASE + "/task");
  await page.waitForLoadState("networkidle");
  await page.getByRole("button", { name: "Buat task" }).first().click();
  await page.waitForSelector("dialog[open]");
  expect(
    (await page
      .getByRole("link", { name: "Buka form Buat task di tab baru" })
      .count()) === 1,
    "tab baru",
    "tombol Buka di tab baru tidak ada di kepala panel",
  );
  await page
    .locator("dialog[open] summary", { hasText: "Detail tambahan" })
    .click();
  const zone = page.locator("dialog[open] label:has(input[type=file])");
  await zone.scrollIntoViewIfNeeded();
  await page.waitForTimeout(150);
  const scrolls = () =>
    page.evaluate(() => {
      const d = document.querySelector("dialog[open]");
      return [d.scrollTop, d.querySelector(".overflow-y-auto").scrollTop];
    });
  const before = await scrolls();
  const chooser = page.waitForEvent("filechooser");
  await zone.locator("span.underline").click();
  await chooser;
  await page.waitForTimeout(200);
  const after = await scrolls();
  expect(
    before.join() === after.join(),
    "pilih berkas",
    `scrollTop dialog dan wadah gulir ${before.join("/")} jadi ${after.join("/")}`,
  );

  await page.evaluate(() =>
    document
      .querySelector("dialog[open] input[type=file]")
      .dispatchEvent(new Event("cancel", { bubbles: true })),
  );
  await page.waitForTimeout(150);
  expect(
    (await page.locator("dialog[open]").count()) === 1,
    "batal pilih",
    "panel ikut tertutup",
  );
  await audit(page, errors, where("panel"));

  await page.keyboard.press("Escape");
  await page.waitForTimeout(150);
  expect(
    (await page.locator("dialog[open]").count()) === 0,
    "escape",
    "Escape tidak menutup panel",
  );

  // Halaman penuh: isi judul dan satu PIC, kirim, lalu harus pindah ke halaman task baru.
  await page.goto(BASE + "/task/baru");
  await page.waitForLoadState("networkidle");
  expect(
    (await page.locator("h1").first().textContent())?.trim() === "Buat task",
    "halaman",
    "judul halaman bukan Buat task",
  );
  await audit(page, errors, where("halaman"));
  const judul = `[DUMMY] E2E halaman Buat task ${width}`;
  await page.locator("#task-title").fill(judul);
  await page
    .getByRole("button", { name: /Pilih PIC/ })
    .first()
    .click();
  await page.locator("input[type=checkbox]").first().check();
  await page.getByRole("button", { name: "Buat task", exact: true }).click();
  await page.waitForURL(/\/task\/T-\d+$/, { timeout: 10000 }).catch(() => {});
  const path = new URL(page.url()).pathname;
  expect(/^\/task\/T-\d+$/.test(path), "kirim", `URL setelah kirim ${path}`);
  expect(
    (await page
      .locator("[role=status]")
      .filter({ hasText: "dibuat" })
      .count()) > 0,
    "pesan",
    "pesan task dibuat tidak muncul di halaman task",
  );
  await audit(page, errors, where("hasil"));
  await ctx.close();

  // Staf tidak boleh membuat task: 404 dengan URL tetap.
  const staf = await open("staf", width, "terang");
  await staf.page.goto(BASE + "/task/baru");
  await staf.page.waitForLoadState("networkidle");
  expect(
    (await staf.page.locator("h1").first().textContent())?.trim() ===
      "Halaman tidak ditemukan",
    "staf",
    "Staf tidak melihat 404 di /task/baru",
  );
  expect(
    new URL(staf.page.url()).pathname === "/task/baru",
    "staf",
    "URL Staf berubah",
  );
  await audit(staf.page, staf.errors, where("staf"));
  await staf.ctx.close();
}

if (ONLY === "createtask")
  for (const w of [360, 768, 1280]) await checkCreateTask(w);

// Jejak uji To Do dibersihkan lewat API, bukan lewat UI: kalau lari sebelumnya mati di tengah jalan,
// itemnya tetap tertinggal di akun staf dan membuat uji keadaan kosong gagal. Mengembalikan jumlah item
// milik pengguna yang bukan jejak uji, atau null kalau API-nya tidak bisa dibaca.
async function bersihkanTodoApi(ctx) {
  try {
    const r = await ctx.request.get(BASE + "/api/todos");
    if (!r.ok()) return null;
    const items = (await r.json()).todos ?? [];
    for (const it of items)
      if (String(it.title).startsWith("E2E "))
        await ctx.request.delete(`${BASE}/api/todos/${it.id}`, {
          headers: { Origin: BASE },
        });
    return items.filter((it) => !String(it.title).startsWith("E2E ")).length;
  } catch {
    return null;
  }
}

// To Do pribadi di papan (revisi 28): keadaan kosong, tambah lewat Enter, centang, ubah dengan keyboard,
// hapus lalu Urungkan, urutkan dengan tombol, baris berisi URL panjang, panel di lebar sedang, lembar di
// 360 px, tanpa gulir horizontal halaman. Semua locator memakai role dan nama aksesibel, bukan placeholder,
// supaya tetap berlaku setelah tampilannya berubah. Kartu To Do hanya muncul sendiri di lebar yang punya
// kolom kiri; di lebar lain panelnya harus dibuka dulu, dan itu yang membuat uji pertama gagal dengan
// timeout 30 detik di isian.
async function checkTodo(width) {
  const { ctx, page, errors } = await open("staf", width, "terang");
  const where = (what) => `todo ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const tag = `E2E ${Date.now()}`;
  const satu = `${tag} satu`;
  const dua = `${tag} dua`;
  const panjang = `${tag} tautan https://novoidminds.xyz/task?due=hari-ini&status=dikerjakan&pic=staf.uji&urut=deadline`;
  const vis = (l) => l.locator("visible=true").first();
  const ada = (l, ms = 8000) =>
    vis(l)
      .waitFor({ timeout: ms })
      .then(
        () => true,
        () => false,
      );
  const terlihat = async (l) => (await l.locator("visible=true").count()) > 0;
  // Permintaan yang gagal dicatat beserta metode dan URL-nya. Pesan console browser untuk ini hanya
  // "Failed to load resource", tanpa URL, jadi tanpa catatan ini galat 404 tidak bisa dilacak.
  const gagalReq = [];
  page.on("response", (r) => {
    if (r.status() >= 400)
      gagalReq.push(`${r.request().method()} ${r.url()} ${r.status()}`);
  });

  try {
    const sisaMilikSendiri = await bersihkanTodoApi(ctx);

    await page.goto(BASE + "/task");
    await page.waitForLoadState("networkidle");

    // Tombol To Do hanya ada di lebar yang memakai bar. Kalau ada, panelnya dibuka dulu dan ditunggu
    // sampai benar-benar terlihat, bukan ditunggu dengan jeda.
    const tombolPanel = page.getByRole("button", { name: /^To Do/ });
    const pakaiPanel = await terlihat(tombolPanel);
    if (pakaiPanel) {
      await vis(tombolPanel).click();
      const dialog = page.getByRole("dialog", { name: "To Do" });
      const terbuka = await ada(dialog);
      expect(
        terbuka,
        "panel",
        "panel To Do tidak terbuka setelah tombolnya diklik",
      );
      if (!terbuka) return;
      if (width < 640) {
        // Lembar dari bawah: menempel di tepi bawah layar.
        const box = await vis(dialog).boundingBox();
        const vh = await page.evaluate(() => window.innerHeight);
        expect(
          !!box && Math.abs(box.y + box.height - vh) <= 2,
          "lembar",
          `lembar tidak menempel di bawah (bawah ${box ? Math.round(box.y + box.height) : "?"} dari ${vh})`,
        );
      }
    }

    // Isian tambah: dicari lewat nama aksesibelnya. Kalau tidak ada, kartunya memang tidak tampil, dan
    // itu dilaporkan sebagai kegagalan, bukan dibiarkan jadi timeout.
    const isian = page.getByRole("textbox", { name: "Tulis to do" });
    const adaIsian = await ada(isian);
    expect(
      adaIsian,
      "kartu",
      `isian To Do tidak terlihat (panel dipakai: ${pakaiPanel})`,
    );
    if (!adaIsian) return;
    await audit(page, errors, where("kartu"));

    // Keputusan Sprint 11y: kartu kosong hanya menampilkan kepala, satu kalimat, dan isian. Diuji hanya
    // kalau akun stafnya memang tidak punya to do sungguhan, supaya uji ini tidak ikut gagal karena data.
    if (sisaMilikSendiri === 0) {
      expect(
        await terlihat(page.getByText("Belum ada to do.", { exact: true })),
        "kosong",
        "kalimat keadaan kosong tidak muncul",
      );
      expect(
        !(await terlihat(
          page.getByRole("button", { name: "Atur", exact: true }),
        )),
        "kosong",
        "tombol Atur tampil padahal belum ada item",
      );
      expect(
        !(await terlihat(page.getByText(/tersisa|Semua selesai/))),
        "kosong",
        "ringkasan tampil padahal belum ada item",
      );
    }

    // Tambah dua item lewat Enter. Isian dikosongkan dan tetap fokus, jadi item kedua cukup diketik lagi.
    await vis(isian).fill(satu);
    await vis(isian).press("Enter");
    expect(
      await ada(page.getByRole("button", { name: `Ubah ${satu}` })),
      "tambah",
      "item pertama tidak muncul",
    );
    const sisaIsian = await page.evaluate(() => {
      const el = document.activeElement;
      return {
        label: el?.getAttribute("aria-label") ?? "",
        value: el instanceof HTMLInputElement ? el.value : "?",
      };
    });
    expect(
      sisaIsian.label === "Tulis to do",
      "tambah",
      `fokus pindah ke "${sisaIsian.label}"`,
    );
    expect(
      sisaIsian.value === "",
      "tambah",
      `isian tidak dikosongkan, isinya "${sisaIsian.value}"`,
    );

    // Centang item baru harus kosong. Centangnya digambar dengan stroke-dashoffset penuh, jadi yang
    // diperiksa offsetnya, bukan ada atau tidaknya elemennya.
    const centangTersembunyi = await page.evaluate((t) => {
      const li = [...document.querySelectorAll("li")].find(
        (el) => el.querySelector(`button[aria-label="Ubah ${t}"]`) !== null,
      );
      const path = li?.querySelector("label svg path");
      if (!path) return "tidak ada path centang";
      const st = getComputedStyle(path);
      if (
        st.visibility === "hidden" ||
        st.display === "none" ||
        st.opacity === "0"
      )
        return "";
      const off = parseFloat(st.strokeDashoffset) || 0;
      const len = path.getTotalLength();
      return off >= len - 1
        ? ""
        : `stroke-dashoffset ${off} dari panjang ${Math.round(len)}`;
    }, satu);
    expect(
      centangTersembunyi === "",
      "centang baru",
      `centang tampil di item yang belum selesai: ${centangTersembunyi}`,
    );

    await page.keyboard.type(dua);
    await page.keyboard.press("Enter");
    expect(
      await ada(page.getByRole("button", { name: `Ubah ${dua}` })),
      "tambah",
      "item kedua tidak muncul",
    );

    // Baris dengan URL panjang: kontrol baris muncul sebagai lapisan, jadi tinggi barisnya tidak boleh
    // berubah saat hover atau focus, dan judulnya tetap dapat lebih dari separuh lebar kartu.
    await page.keyboard.type(panjang);
    await page.keyboard.press("Enter");
    const judulPanjang = page.getByRole("button", { name: `Ubah ${panjang}` });
    const adaPanjang = await ada(judulPanjang);
    expect(adaPanjang, "panjang", "item dengan URL panjang tidak muncul");
    if (!adaPanjang) return;
    const ukur = async () =>
      await page.evaluate((t) => {
        const b = document.querySelector(`button[aria-label="Ubah ${t}"]`);
        const li = b?.closest("li");
        const daftar = li?.closest("ol, ul");
        if (!b || !li || !daftar) return null;
        return {
          tinggi: Math.round(li.getBoundingClientRect().height),
          judul: Math.round(b.getBoundingClientRect().width),
          kartu: Math.round(daftar.getBoundingClientRect().width),
        };
      }, panjang);
    const sebelum = await ukur();
    await vis(judulPanjang).hover();
    const saatHover = await ukur();
    await vis(judulPanjang).focus();
    const saatFokus = await ukur();
    expect(
      !!sebelum && !!saatHover && !!saatFokus,
      "panjang",
      "baris URL panjang tidak bisa diukur",
    );
    if (sebelum && saatHover && saatFokus) {
      expect(
        Math.abs(saatHover.tinggi - sebelum.tinggi) <= 1 &&
          Math.abs(saatFokus.tinggi - sebelum.tinggi) <= 1,
        "panjang",
        `tinggi baris berubah: ${sebelum.tinggi} lalu hover ${saatHover.tinggi} lalu fokus ${saatFokus.tinggi}`,
      );
      expect(
        sebelum.judul * 2 >= sebelum.kartu,
        "panjang",
        `judul hanya ${sebelum.judul} px dari kartu ${sebelum.kartu} px`,
      );
    }

    // Mode Atur: kontrol urut dan hapus tampil di semua baris, termasuk di layar sentuh yang tanpa hover.
    const atur = page.getByRole("button", { name: "Atur", exact: true });
    expect(await ada(atur), "atur", "tombol Atur tidak ada");
    await vis(atur).click();
    expect(
      (await vis(atur).getAttribute("aria-pressed")) === "true",
      "atur",
      "tombol Atur tidak menandai dirinya aktif",
    );
    const tombolNaik = page.getByRole("button", { name: `Naikkan ${dua}` });
    expect(
      await ada(tombolNaik),
      "atur",
      "kontrol baris tidak muncul di mode Atur",
    );

    // Urutkan dengan tombol, lalu urutannya dibaca dari DOM.
    await vis(tombolNaik).click();
    const urut = await page
      .waitForFunction(
        (t) => {
          const judul = [
            ...document.querySelectorAll("li button[aria-label^='Ubah ']"),
          ]
            .filter((b) => b.textContent?.includes(t))
            .map((b) => b.textContent?.trim() ?? "");
          return judul.length >= 2 && judul[0].includes("dua") ? judul : null;
        },
        tag,
        { timeout: 8000 },
      )
      .then(
        () => true,
        () => false,
      );
    expect(
      urut,
      "urutkan",
      "item kedua tidak naik ke atas setelah tombol Naikkan",
    );

    // Ubah dengan keyboard: Enter membuka isian, Escape membatalkan, Enter kedua menyimpan.
    const judulSatu = page.getByRole("button", { name: `Ubah ${satu}` });
    await vis(judulSatu).focus();
    await page.keyboard.press("Enter");
    expect(
      await ada(page.getByRole("textbox", { name: "Ubah to do" })),
      "ubah",
      "isian ubah tidak terbuka",
    );
    await page.keyboard.press("Escape");
    const kembali = await ada(judulSatu);
    expect(
      kembali,
      "ubah",
      "Escape tidak membatalkan perubahan: judul tidak kembali, panelnya ikut tertutup",
    );
    if (!kembali) return;
    if (pakaiPanel)
      expect(
        await terlihat(page.getByRole("dialog", { name: "To Do" })),
        "ubah",
        "Escape di isian ubah ikut menutup panel To Do",
      );
    // Fokus kembali ke tombol judul, bukan hilang ke body.
    const fokusSetelahEsc = await page.evaluate(
      () => document.activeElement?.getAttribute("aria-label") ?? "",
    );
    expect(
      fokusSetelahEsc === `Ubah ${satu}`,
      "ubah",
      `fokus setelah Escape ada di "${fokusSetelahEsc}"`,
    );
    await vis(judulSatu).focus();
    await page.keyboard.press("Enter");
    // Isian ubah membuka dengan seluruh teks terseleksi, supaya judul bisa diganti tanpa menghapus dulu.
    const seleksi = await page.evaluate(() => {
      const el = document.activeElement;
      if (!(el instanceof HTMLInputElement))
        return `fokus bukan isian: ${el?.tagName}`;
      return el.selectionStart === 0 && el.selectionEnd === el.value.length
        ? ""
        : `seleksi ${el.selectionStart} sampai ${el.selectionEnd} dari ${el.value.length}`;
    });
    expect(
      seleksi === "",
      "ubah",
      `teks isian ubah tidak terseleksi: ${seleksi}`,
    );
    // End dulu: tanpa itu, mengetik akan mengganti seluruh judul yang terseleksi, bukan menambah di akhir.
    await page.keyboard.press("End");
    await page.keyboard.type(" diubah");
    await page.keyboard.press("Enter");
    const tersimpan = await ada(
      page.getByRole("button", { name: `Ubah ${satu} diubah` }),
    );
    expect(tersimpan, "ubah", "judul tidak tersimpan setelah Enter");
    if (!tersimpan) return;

    // Centang: item pindah ke kelompok Selesai, judulnya dicoret.
    const kotak = page.getByRole("checkbox", {
      name: `Tandai selesai ${satu} diubah`,
    });
    const adaKotak = await ada(kotak);
    expect(adaKotak, "centang", "kotak centang item tidak ada");
    if (!adaKotak) return;
    // Yang diklik labelnya, bukan input-nya: input kotak centang sr-only, jadi kotaknya tertutup label
    // itu sendiri dan check() pada input selalu tertahan ("label intercepts pointer events"). Klik di
    // label adalah yang dilakukan pemakai, dan label yang membawa nama aksesibelnya.
    await vis(kotak.locator("xpath=..")).click();
    expect(
      await ada(page.getByRole("button", { name: /^Selesai \(/ })),
      "centang",
      "kelompok Selesai tidak muncul",
    );
    const dicoret = await page
      .waitForFunction(
        (t) => {
          const el = [...document.querySelectorAll("li p")].find(
            (p) => p.textContent?.trim() === t,
          );
          return el
            ? getComputedStyle(el).textDecorationLine.includes("line-through")
            : false;
        },
        `${satu} diubah`,
        { timeout: 8000 },
      )
      .then(
        () => true,
        () => false,
      );
    expect(dicoret, "centang", "judul item selesai tidak dicoret");

    // Hapus lalu Urungkan: itemnya kembali.
    const tombolHapus = page.getByRole("button", { name: `Hapus ${dua}` });
    const adaHapus = await ada(tombolHapus);
    expect(adaHapus, "hapus", "tombol hapus tidak ada");
    if (!adaHapus) return;
    await vis(tombolHapus).click();
    expect(
      await ada(page.getByText("To do dihapus.")),
      "hapus",
      "pesan hapus tidak muncul",
    );
    const urungkan = page.getByRole("button", { name: "Urungkan" });
    const adaUrungkan = await ada(urungkan);
    expect(adaUrungkan, "hapus", "toast Urungkan tidak muncul");
    if (!adaUrungkan) return;
    await vis(urungkan).click();
    expect(
      await ada(page.getByRole("button", { name: `Ubah ${dua}` })),
      "urungkan",
      "item tidak kembali setelah Urungkan",
    );

    await audit(page, errors, where("setelah diisi"));
    const lebar = await page.evaluate(() => [
      document.documentElement.scrollWidth,
      window.innerWidth,
    ]);
    expect(
      lebar[0] <= lebar[1],
      "lebar",
      `scrollWidth ${lebar[0]} > ${lebar[1]}`,
    );
  } finally {
    // Dijalankan juga saat ujinya berhenti di tengah jalan, supaya akun staf tidak menumpuk item "E2E ".
    await bersihkanTodoApi(ctx);
    checks++;
    if (errors.length)
      fail(
        where("console"),
        [
          errors.splice(0).join(" | "),
          gagalReq.length ? `permintaan gagal: ${gagalReq.join(" | ")}` : "",
        ]
          .filter(Boolean)
          .join(" || ")
          .slice(0, 400),
      );
    await ctx.close();
  }
}

if (ONLY === "todo") for (const w of [360, 1280]) await checkTodo(w);

// Revisi 30: halaman Timeline. Kartu uji ditandai "E2E Timeline" dan dibersihkan lewat API di finally.
const hariKantor = new Intl.DateTimeFormat("en-CA", {
  timeZone: "Asia/Jakarta",
}).format(new Date());
async function bersihkanTimeline(mgr) {
  const res = await mgr.call(
    "GET",
    `/timeline/events?month=${hariKantor.slice(0, 7)}`,
  );
  for (const e of res.body?.events ?? [])
    if (
      e.title.startsWith("E2E Timeline") ||
      e.description.startsWith("E2E Timeline")
    )
      await mgr.call("DELETE", `/timeline/events/${e.id}`);
}

async function checkTimeline(width) {
  const where = (what) => `timeline ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const lebarAman = async (page, w) => {
    const [sw, vw] = await page.evaluate(() => [
      document.documentElement.scrollWidth,
      window.innerWidth,
    ]);
    expect(sw <= vw, w, `scrollWidth ${sw} > ${vw}`);
  };
  const grid = width >= 768;
  const mgr = await apiAs("manajer");
  const nama = mgr.me.name;
  const judul = {
    launching: `E2E Timeline ${width} L`,
    produksi: `E2E Timeline ${width} P`,
  };
  const ringkas = [judul.launching, judul.produksi, `${nama}: Sakit`];
  const { ctx, page, errors } = await open("manajer", width, "terang");
  const muat = async () => {
    await page.waitForLoadState("networkidle");
    await page
      .getByText("Memuat jadwal...")
      .waitFor({ state: "detached" })
      .catch(() => {});
  };
  const bukaHari = async () => {
    await page.locator(`[data-date="${hariKantor}"]`).first().click();
    await page.locator(POP).waitFor();
  };
  try {
    await page.goto(BASE + "/timeline");
    await muat();
    expect(
      (await page.getByRole("grid").count()) === (grid ? 1 : 0),
      "tampilan",
      grid ? "grid tidak tampil" : "grid tetap tampil di bawah 768 px",
    );
    expect(
      (await page.getByRole("list", { name: "Agenda bulan ini" }).count()) ===
        (grid ? 0 : 1),
      "tampilan",
      grid ? "agenda ikut tampil" : "agenda tidak tampil",
    );
    for (const c of ["Launching", "Produksi", "Kehadiran"])
      expect(
        (await page
          .getByRole("button", { name: c, exact: true })
          .getAttribute("aria-pressed")) === "true",
        "filter",
        `toggle ${c} tidak menyala bawaan`,
      );

    if (grid) {
      // Roving tabindex: satu tanggal di urutan Tab, panah pindah hari, Home dan End ke ujung minggu.
      const aktif = page.locator('[role="grid"] button[tabindex="0"]');
      expect(
        (await aktif.count()) === 1,
        "keyboard",
        "bukan tepat satu tanggal di urutan Tab",
      );
      await aktif.focus();
      const fokus = () =>
        page.evaluate(() => document.activeElement?.getAttribute("data-date"));
      const awal = await fokus();
      await page.keyboard.press("ArrowRight");
      const kanan = await fokus();
      expect(
        kanan && kanan > awal,
        "keyboard",
        `ArrowRight dari ${awal} ke ${kanan}`,
      );
      await page.keyboard.press("Home");
      const senin = await fokus();
      await page.keyboard.press("End");
      const minggu = await fokus();
      expect(
        new Date(senin + "T00:00:00Z").getUTCDay() === 1 &&
          new Date(minggu + "T00:00:00Z").getUTCDay() === 0,
        "keyboard",
        `Home ${senin}, End ${minggu}`,
      );
      // Panah di tepi bulan pindah ke bulan sebelumnya, lalu kembali.
      const bulan = hariKantor.slice(0, 7);
      await page.locator(`[data-date="${bulan}-01"]`).focus();
      await page.keyboard.press("ArrowLeft");
      await page.waitForTimeout(200);
      const sebelum = await fokus();
      expect(
        sebelum && sebelum.slice(0, 7) < bulan,
        "keyboard",
        `ArrowLeft dari tanggal 1 ke ${sebelum}`,
      );
      await page.keyboard.press("ArrowRight");
      await muat();
      expect(
        (await fokus()) === `${bulan}-01`,
        "keyboard",
        "tidak kembali ke tanggal 1",
      );
      // Enter membuka popover, Escape menutup dan fokus kembali ke sel.
      await page.locator(`[data-date="${hariKantor}"]`).focus();
      await page.keyboard.press("Enter");
      await page.locator(POP).waitFor();
      expect(
        await page.evaluate(
          (sel) =>
            document.querySelector(sel)?.contains(document.activeElement),
          POP,
        ),
        "popover",
        "fokus tidak pindah ke popover",
      );
      await page.keyboard.press("Escape");
      await page.waitForTimeout(150);
      expect(
        (await page.locator(POP).count()) === 0,
        "popover",
        "Escape tidak menutup popover",
      );
      expect(
        (await fokus()) === hariKantor,
        "popover",
        "fokus tidak kembali ke sel",
      );
    }

    // Manajer membuat satu kartu per kategori di hari ini.
    for (const kat of ["Launching", "Produksi", "Kehadiran"]) {
      await page.getByRole("button", { name: "Buat kartu" }).click();
      const dlg = page.locator("dialog[open]");
      await dlg.waitFor();
      await dlg.locator("label", { hasText: kat }).first().click();
      await dlg.getByLabel("Tanggal mulai (wajib)").fill(hariKantor);
      if (kat === "Kehadiran") {
        await dlg
          .locator("option", { hasText: nama })
          .first()
          .waitFor({ state: "attached" });
        await dlg.getByLabel("Nama (wajib)").selectOption({ label: nama });
        await dlg.locator("label", { hasText: "Sakit" }).click();
        await dlg.getByLabel("Keterangan").fill("E2E Timeline rahasia");
      } else {
        await dlg.getByLabel("Judul (wajib)").fill(judul[kat.toLowerCase()]);
      }
      await lebarAman(page, `form ${kat}`);
      await dlg.getByRole("button", { name: "Buat kartu" }).click();
      const ok = await page
        .getByText("Kartu dibuat.")
        .waitFor({ timeout: 5000 })
        .then(() => true)
        .catch(() => false);
      expect(ok, `buat ${kat}`, "pesan Kartu dibuat tidak muncul");
      if (!ok) {
        await page.keyboard.press("Escape");
        continue;
      }
      await muat();
    }

    await bukaHari();
    for (const r of ringkas)
      expect(
        (await page.locator(POP).getByText(r, { exact: true }).count()) === 1,
        "popover",
        `entri "${r}" tidak ada di popover`,
      );
    expect(
      (await page.locator(POP).getByText("E2E Timeline rahasia").count()) === 1,
      "privasi",
      "Manajer tidak melihat keterangan kehadiran",
    );
    await lebarAman(page, "popover");
    await audit(page, errors, where("popover Manajer"));
    await page.keyboard.press("Escape");

    // Staf: tanpa tombol tulis, dan alasan kehadiran Manajer disembunyikan server.
    const s = await open("staf", width, "terang");
    try {
      await s.page.goto(BASE + "/timeline");
      await s.page.waitForLoadState("networkidle");
      await s.page
        .getByText("Memuat jadwal...")
        .waitFor({ state: "detached" })
        .catch(() => {});
      expect(
        (await s.page.getByRole("button", { name: "Buat kartu" }).count()) ===
          0,
        "staf",
        "tombol Buat kartu tampil untuk Staf",
      );
      expect(
        (await s.page.getByLabel("Judul catatan baru").count()) === 0,
        "staf",
        "form catatan tampil untuk Staf",
      );
      await s.page.locator(`[data-date="${hariKantor}"]`).first().click();
      await s.page.locator(POP).waitFor();
      const pop = s.page.locator(POP);
      expect(
        (await pop
          .getByRole("button", { name: "Tambah di tanggal ini" })
          .count()) === 0 &&
          (await pop
            .getByRole("button", { name: /^(Ubah|Hapus) / })
            .count()) === 0,
        "staf",
        "tombol tulis tampil di popover Staf",
      );
      // Revisi 38: alasan dan keterangan kehadiran terbuka untuk semua pengguna.
      expect(
        (await pop.getByText(`${nama}: Sakit`, { exact: true }).count()) ===
          1 && (await pop.getByText("E2E Timeline rahasia").count()) === 1,
        "alasan terbuka",
        "Staf tidak melihat alasan atau keterangan kehadiran Manajer",
      );
      await lebarAman(s.page, "staf");
      await audit(s.page, s.errors, where("popover Staf"));
    } finally {
      await s.ctx.close();
    }

    // Manajer menghapus ketiganya lewat popover, dengan konfirmasi.
    await bukaHari();
    for (const r of ringkas) {
      const hapus = page
        .locator(POP)
        .getByRole("button", { name: `Hapus ${r}` });
      if (!(await hapus.count())) continue;
      await hapus.click();
      await page
        .locator(POP)
        .getByRole("button", { name: "Ya, hapus" })
        .click();
      // Tombol Hapus sudah hilang begitu konfirmasi tampil, jadi yang ditunggu teks entrinya sampai DELETE selesai.
      const entri = page.locator(POP).getByText(r, { exact: true });
      await entri.waitFor({ state: "detached", timeout: 5000 }).catch(() => {});
      expect(
        (await entri.count()) === 0,
        "hapus",
        `"${r}" masih ada setelah dihapus`,
      );
    }
    await page.keyboard.press("Escape");
    await audit(page, errors, where("setelah hapus"));
  } finally {
    await bersihkanTimeline(mgr);
    await mgr.ctx.close();
    await ctx.close();
  }
}

if (ONLY === "timeline") for (const w of [360, 1280]) await checkTimeline(w);

// Revisi 33: rantai sambung jam di kartu Aktivitas hari ini milik Staf. Menulis jam 11.00-12.00, 14.00-15.00,
// 15.00-16.00, dan 16.00-17.00 hari ini lewat API, lalu mengosongkannya lagi di akhir.
async function checkChain(width) {
  const staf = await apiAs("staf");
  const { ctx, page, errors } = await open("staf", width, "terang");
  const where = (what) => `rantai ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const kepala = `Uji rantai ${Date.now()}`;
  const simpan = (slot, activity) =>
    staf.call("PUT", "/activities", { slot, activity, output: "" });
  const bersihkan = async () => {
    for (const slot of [4, 6])
      await staf.call("PUT", "/activities/link", { slot, linked: false });
    for (const slot of [3, 5, 6, 7]) await simpan(slot, "");
  };
  // Tombol rantai jam itu, dicari dari nama aksesibelnya saja (revisi 33b menyebut jamnya di label), bukan
  // dari tempatnya di DOM: di layar lebar tombolnya sekarang ada di sel Aktivitas, bukan di kepala kolom.
  const tombol = (jam) =>
    page
      .getByRole("button", {
        name: new RegExp(
          `(Sambungkan|Lepas sambungan) jam ${jam.replace(/\./g, "\\.")}`,
        ),
      })
      .locator("visible=true")
      .first();

  try {
    await bersihkan();
    await simpan(5, kepala);
    await simpan(3, "Isi jam ketiga");
    const today = (await staf.call("GET", "/activities/me")).body.date;

    await page.goto(BASE + "/task");
    await page.waitForLoadState("networkidle");
    const judul = page.getByRole("button", { name: /Aktivitas hari ini/ });
    if ((await judul.getAttribute("aria-expanded")) === "false")
      await judul.click();
    await page.waitForSelector("textarea:visible");
    // Dibaca dari span ringkasan saja: textContent judul menempelkan tahun tanggal ke angkanya ("20264 dari").
    const terisi = async () =>
      Number(
        (await judul.locator("span").last().textContent()).match(
          /(\d+) dari/,
        )?.[1] ?? -1,
      );
    // Menunggu tombol jam itu sendiri, bukan aria-pressed mana pun: data dev bisa sudah punya rantai lain.
    const tunggu = (jam, pressed) =>
      tombol(jam)
        .and(page.locator(`[aria-pressed="${pressed}"]`))
        .waitFor({ timeout: 10000 });
    const awal = await terisi();

    // Ubahan revisi 33: jam setelah istirahat juga punya tombol rantai.
    expect(
      (await tombol("13.00-14.00").count()) === 1,
      "istirahat",
      "13.00-14.00 tidak punya tombol rantai",
    );
    // Revisi 33b: jam yang berisi tidak lagi menampilkan tombol mati, tombolnya tidak dirender sama sekali.
    expect(
      (await tombol("11.00-12.00").count()) === 0,
      "jam berisi",
      "jam berisi masih menampilkan tombol rantai",
    );

    // Sambung 15.00-16.00 lewat keyboard, lalu 16.00-17.00 lewat klik.
    await tombol("15.00-16.00").focus();
    await page.keyboard.press("Enter");
    await tunggu("15.00-16.00", true);
    await tombol("16.00-17.00").click();
    await tunggu("16.00-17.00", true);
    const lanjutan = page
      .getByText("Lanjutan 14.00-15.00")
      .filter({ visible: true });
    expect(
      (await lanjutan.count()) >= 2,
      "pasang",
      "label Lanjutan tidak muncul",
    );
    // Isi jam kepala harus tampil di jam tersambung. Sejak revisi 33b kotak lanjutan di tabel (mulai 768 px)
    // adalah textarea readonly, jadi isinya ada di value-nya dan tidak bisa dicari dengan getByText; di
    // kartu jam (di bawah 768 px) isinya masih teks biasa di kartu itu.
    const isiJam = async (jam) => {
      const kotak = page
        .getByRole("textbox", { name: `Aktivitas ${jam}` })
        .locator("visible=true")
        .first();
      if ((await kotak.count()) > 0) return await kotak.inputValue();
      const kartu = page.locator("li:visible").filter({ hasText: jam }).first();
      return (await kartu.count()) > 0 ? await kartu.innerText() : "";
    };
    const isiLanjutan = [
      await isiJam("15.00-16.00"),
      await isiJam("16.00-17.00"),
    ];
    expect(
      isiLanjutan.every((t) => t.includes(kepala)),
      "pasang",
      `isi jam kepala tidak tampil di jam tersambung: ${JSON.stringify(isiLanjutan)}`,
    );
    expect(
      (await terisi()) === awal + 2,
      "terisi",
      `jam terisi ${awal} jadi ${await terisi()}, mau +2`,
    );

    // Revisi 33b lanjutan: kotak jam lanjutan memakai isian yang sama dengan jam kepalanya, jadi tingginya
    // sama (selisih maksimal 1 px) dan lebarnya sama dengan kolom jam lain. Di bawah 768 px jam lanjutan
    // berupa teks, bukan isian, jadi pengukuran ini dilewati di sana.
    const ukurKotak = async (jam) => {
      const kotak = page
        .getByRole("textbox", { name: `Aktivitas ${jam}` })
        .locator("visible=true")
        .first();
      if ((await kotak.count()) === 0) return null;
      const box = await kotak.boundingBox();
      return box
        ? { tinggi: Math.round(box.height), lebar: Math.round(box.width) }
        : null;
    };
    const kotakKepala = await ukurKotak("14.00-15.00");
    const kotakLanjut = await ukurKotak("15.00-16.00");
    const kotakLain = await ukurKotak("17.00-18.00");
    if (kotakKepala && kotakLanjut && kotakLain) {
      expect(
        Math.abs(kotakLanjut.tinggi - kotakKepala.tinggi) <= 1,
        "ukuran",
        `tinggi kotak lanjutan ${kotakLanjut.tinggi} px, kepala ${kotakKepala.tinggi} px`,
      );
      expect(
        Math.abs(kotakLanjut.lebar - kotakLain.lebar) <= 1,
        "ukuran",
        `lebar kotak lanjutan ${kotakLanjut.lebar} px, kolom lain ${kotakLain.lebar} px`,
      );
    }
    // Keadaan tombol rantai dibedakan warna: yang tersambung memakai accent, yang belum tetap ink-muted.
    const warna = async (jam) => {
      const t = tombol(jam);
      return (await t.count()) === 0
        ? null
        : await t.evaluate((el) => getComputedStyle(el).color);
    };
    const warnaTersambung = await warna("15.00-16.00");
    const warnaBelum = await warna("13.00-14.00");
    expect(
      !!warnaTersambung && !!warnaBelum && warnaTersambung !== warnaBelum,
      "warna",
      `warna tombol tersambung ${warnaTersambung} dan belum ${warnaBelum}`,
    );

    // Status simpan pindah ke baris judul jam, jadi Tambah berkas merapat ke bawah Output.
    const jarakBerkas = async (jam) => {
      const out = page
        .getByRole("textbox", { name: `Output ${jam}` })
        .locator("visible=true")
        .first();
      const tombolBerkas = page
        .getByRole("button", {
          name: new RegExp(
            `^Tambah berkas untuk jam ${jam.replace(/\./g, "\\.")}`,
          ),
        })
        .locator("visible=true")
        .first();
      if ((await out.count()) === 0 || (await tombolBerkas.count()) === 0)
        return null;
      const a = await out.boundingBox();
      const b = await tombolBerkas.boundingBox();
      return a && b ? Math.round(b.y - (a.y + a.height)) : null;
    };
    const jarak = await jarakBerkas("14.00-15.00");
    expect(
      jarak !== null && jarak <= 12,
      "jarak berkas",
      `jarak Output ke Tambah berkas ${jarak} px`,
    );
    await audit(page, errors, where("tersambung"));

    // Halaman Aktivitas: satu sel untuk tiga jam.
    await page.goto(BASE + `/aktivitas?mode=hari&tanggal=${today}`);
    await page.waitForSelector("table td");
    expect(
      (await page
        .locator('td[rowspan="3"]')
        .filter({ hasText: kepala })
        .count()) === 1,
      "halaman",
      "rantai tidak tampil sebagai satu sel rowspan 3",
    );
    await audit(page, errors, where("halaman aktivitas"));

    // Lepas di tengah: 16.00-17.00 ikut lepas.
    await page.goto(BASE + "/task");
    await page.waitForSelector("textarea:visible");
    await tombol("15.00-16.00").click();
    await page.waitForFunction(
      () => !document.body.innerText.includes("Lanjutan 14.00-15.00"),
    );
    // Setelah 15.00-16.00 dilepas, 16.00-17.00 ikut lepas dan jam sebelumnya jadi kosong, jadi tombolnya
    // memang tidak dirender lagi (revisi 33b). Label Lanjutan yang hilang sudah ditunggu di atas.
    expect(
      (await tombol("16.00-17.00").count()) === 0,
      "lepas",
      "16.00-17.00 masih punya tombol rantai setelah 15.00-16.00 dilepas",
    );
    await audit(page, errors, where("dilepas"));

    // Melintasi istirahat: 13.00-14.00 menyambung ke 11.00-12.00 dan terhitung satu jam terisi.
    const sebelumIstirahat = await terisi();
    await tombol("13.00-14.00").click();
    await page.waitForFunction(() =>
      document.body.innerText.includes("Lanjutan 11.00-12.00"),
    );
    expect(
      (await tombol("13.00-14.00").getAttribute("aria-pressed")) === "true",
      "istirahat",
      "13.00-14.00 tidak tersambung",
    );
    expect(
      (await terisi()) === sebelumIstirahat + 1,
      "istirahat",
      `jam terisi ${sebelumIstirahat} jadi ${await terisi()}, mau +1`,
    );
    await audit(page, errors, where("melintasi istirahat"));
  } finally {
    await bersihkan();
    await staf.ctx.close();
    await ctx.close();
  }
}

if (ONLY === "chain") for (const w of [360, 1280]) await checkChain(w);

// Revisi 34: centang sub task. Task A: PIC task Staf, sub task dipegang Manajer, jadi Staf boleh mencentang
// sebagai PIC task. Task B: Staf tidak terlibat sama sekali, jadi kotaknya mati. Admin selalu mati.
async function checkCentang(width) {
  const where = (what) => `centang ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const mgr = await apiAs("manajer");
  const staf = await apiAs("staf");
  const made = [];
  const buat = async (title, pic, subPic) => {
    const r = await mgr.call("POST", "/tasks", {
      title,
      urgency: "medium",
      pic_ids: [pic],
      label_ids: [],
      subtasks: [{ title: `${title} sub`, pic_id: subPic }],
    });
    if (r.status !== 201) throw new Error(`buat task ${r.status}`);
    made.push(r.body.id);
    const d = (await mgr.call("GET", `/tasks/${r.body.id}`)).body.task;
    await mgr.call("POST", `/tasks/${r.body.id}/move`, {
      version: d.version,
      to: "on_progress",
      note: "",
    });
    return r.body.id;
  };
  const stamp = Date.now();
  const a = await buat(
    `E2E centang A ${width} ${stamp}`,
    staf.me.id,
    mgr.me.id,
  );
  const b = await buat(`E2E centang B ${width} ${stamp}`, mgr.me.id, mgr.me.id);
  const kotak = (page) =>
    page.locator('input[type="checkbox"][aria-busy]').first();
  try {
    const s = await open("staf", width, "terang");
    try {
      await s.page.goto(BASE + `/task/${a}`);
      await s.page.waitForLoadState("networkidle");
      expect(
        (await s.page
          .getByText(
            "Sub task bisa dicentang PIC sub task, PIC task, pembuat task, dan Manajer",
          )
          .count()) === 1,
        "teks bantuan",
        "teks bantuan sub task belum diganti",
      );
      expect(
        await kotak(s.page).isEnabled(),
        "PIC task",
        "kotak mati untuk PIC task",
      );
      await kotak(s.page).focus();
      await s.page.keyboard.press("Space");
      const dicentang = s.page.getByText(
        new RegExp(`Dicentang ${staf.me.name}`),
      );
      const ok = await dicentang
        .waitFor({ timeout: 8000 })
        .then(() => true)
        .catch(() => false);
      expect(ok, "pencentang", `tidak tampil "Dicentang ${staf.me.name}"`);
      await audit(s.page, s.errors, where("detail PIC task"));
      await kotak(s.page).click();
      await s.page.waitForLoadState("networkidle");
      await s.page.waitForTimeout(300);
      expect(
        !(await kotak(s.page).isChecked()),
        "batal",
        "centang tidak bisa dibatalkan PIC task",
      );

      await s.page.goto(BASE + `/task/${b}`);
      await s.page.waitForLoadState("networkidle");
      expect(
        await kotak(s.page).isDisabled(),
        "tidak terlibat",
        "kotak aktif untuk Staf yang tidak terlibat",
      );
      const tolak = await staf.call(
        "PUT",
        `/tasks/${b}/subtasks/${(await mgr.call("GET", `/tasks/${b}`)).body.task.subtasks[0].id}/done`,
        { done: true },
      );
      expect(
        tolak.status === 403,
        "tidak terlibat",
        `API centang Staf tidak terlibat ${tolak.status}, mau 403`,
      );
    } finally {
      await s.ctx.close();
    }
    const ad = await open("admin", width, "terang");
    try {
      await ad.page.goto(BASE + `/task/${a}`);
      await ad.page.waitForLoadState("networkidle");
      expect(
        await kotak(ad.page).isDisabled(),
        "admin",
        "kotak aktif untuk Admin",
      );
    } finally {
      await ad.ctx.close();
    }
  } finally {
    for (const id of made) {
      const d = (await mgr.call("GET", `/tasks/${id}`)).body?.task;
      if (d) await mgr.call("DELETE", `/tasks/${id}?version=${d.version}`);
    }
    await staf.ctx.close();
    await mgr.ctx.close();
  }
}

if (ONLY === "centang") for (const w of [360, 1280]) await checkCentang(w);

// Revisi 35: Output dengan beberapa tautan. Jam 16.00-17.00 milik Staf berisi tiga URL dan teks, jam
// 17.00-18.00 disambung ke sana, lalu diperiksa kartu papan (jam lanjutan), tabel /aktivitas Manajer, dan
// dialog detail jam tanpa gulir horizontal.
async function checkOutputLinks(width) {
  const where = (what) => `tautan ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const staf = await apiAs("staf");
  const panjang = "https://contoh.test/" + "a".repeat(180);
  const output = `Draf: https://drive.google.com/file/d/AbC1/view\nhttps://drive.google.com/file/d/AbC2/view, ${panjang}.`;
  const label = ["File Drive", "File Drive 2", "contoh.test"];
  const bersihkan = async () => {
    await staf.call("PUT", "/activities/link", { slot: 8, linked: false });
    for (const slot of [7, 8])
      await staf.call("PUT", "/activities", { slot, activity: "", output: "" });
  };
  const periksaTautan = async (scope, w) => {
    for (const l of label) {
      const a = scope
        .getByRole("link", { name: `${l}, buka di tab baru` })
        .first();
      const ada = (await a.count()) === 1;
      expect(ada, w, `tombol "${l}" tidak ada`);
      if (!ada) continue;
      expect(
        (await a.getAttribute("target")) === "_blank" &&
          (await a.getAttribute("rel")) === "noopener noreferrer" &&
          ((await a.getAttribute("title")) ?? "").startsWith("https://"),
        w,
        `atribut tombol "${l}" salah`,
      );
    }
  };
  try {
    await bersihkan();
    await staf.call("PUT", "/activities", {
      slot: 7,
      activity: "E2E tautan",
      output,
    });
    const link = await staf.call("PUT", "/activities/link", {
      slot: 8,
      linked: true,
    });
    expect(link.status < 300, "rantai", `sambung jam 8 ${link.status}`);

    const s = await open("staf", width, "terang");
    try {
      await s.page.goto(BASE + "/task");
      await s.page.waitForLoadState("networkidle");
      const judul = s.page.getByRole("button", { name: /Aktivitas hari ini/ });
      if ((await judul.getAttribute("aria-expanded")) === "false")
        await judul.click();
      await s.page.waitForTimeout(500);
      // Di layar lebar Output jam lanjutan ada di kotak data-slot 8; di layar sempit di kartu jam 17.00-18.00.
      const scope =
        width >= 768
          ? s.page.locator('[data-slot="8"][data-kind="output"]')
          : s.page.locator("body");
      await periksaTautan(scope, "kartu papan");
      await audit(s.page, s.errors, where("kartu papan"));
    } finally {
      await s.ctx.close();
    }

    const m = await open("manajer", width, "terang");
    try {
      await m.page.goto(BASE + "/aktivitas");
      await m.page.waitForLoadState("networkidle");
      const sel = m.page.getByRole("button", {
        name: new RegExp(`Lihat aktivitas ${staf.me.name}, 16\\.00-18\\.00`),
      });
      const adaSel = await sel
        .first()
        .waitFor({ timeout: 8000 })
        .then(() => true)
        .catch(() => false);
      expect(
        adaSel,
        "tabel",
        "sel rantai 16.00-18.00 tidak ditemukan di /aktivitas",
      );
      if (adaSel) {
        const td = m.page.locator("td", { has: sel.first() });
        await periksaTautan(td, "tabel");
        // Tengah sel tertutup chip tautan (z-10, sengaja), jadi dialog dibuka lewat keyboard: jalur Tab lalu Enter.
        await sel.first().focus();
        await m.page.keyboard.press("Enter");
        const dlg = m.page.locator("dialog[open]");
        await dlg.waitFor();
        await periksaTautan(dlg, "dialog");
        const [sw, cw] = await dlg.evaluate((d) => [
          d.scrollWidth,
          d.clientWidth,
        ]);
        expect(
          sw <= cw,
          "dialog",
          `dialog bisa digulir ke samping: scrollWidth ${sw} > ${cw}`,
        );
        await audit(m.page, m.errors, where("dialog"));
        await m.page.keyboard.press("Escape");
      }
    } finally {
      await m.ctx.close();
    }
  } finally {
    await bersihkan();
    await staf.ctx.close();
  }
}

if (ONLY === "outputlinks")
  for (const w of [360, 1280]) await checkOutputLinks(w);

// Revisi 36: tempel dan seret berkas di lampiran sub task. Gambar tempelan di Hasil masuk ke Hasil, di Arahan
// masuk ke Arahan, tempelan teks di Catatan hasil tetap teks, dan berkas di atas 50 MB ditolak dengan pesan.
async function checkPaste(width) {
  const where = (what) => `tempel ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const mgr = await apiAs("manajer");
  const title = `E2E tempel ${width} ${Date.now()}`;
  const made = await mgr.call("POST", "/tasks", {
    title,
    urgency: "medium",
    pic_ids: [mgr.me.id],
    label_ids: [],
    subtasks: [{ title: "Sub task tempel", pic_id: mgr.me.id }],
  });
  if (made.status !== 201) throw new Error(`buat task ${made.status}`);
  const id = made.body.id;
  const { ctx, page, errors } = await open("manajer", width, "terang");
  await ctx.grantPermissions(["clipboard-read", "clipboard-write"], {
    origin: BASE,
  });
  // Event paste sintetis berisi berkas, dikirim ke elemen yang sedang difokus seperti Ctrl+V sungguhan.
  const tempel = (name, type, size = 0) =>
    page.evaluate(
      ({ name, type, size }) => {
        const png =
          "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg==";
        const bytes = size
          ? new Uint8Array(size)
          : Uint8Array.from(atob(png), (c) => c.charCodeAt(0));
        const dt = new DataTransfer();
        dt.items.add(new File([bytes], name, { type }));
        document.activeElement.dispatchEvent(
          new ClipboardEvent("paste", {
            clipboardData: dt,
            bubbles: true,
            cancelable: true,
          }),
        );
      },
      { name, type, size },
    );
  const tempelan = /^Pratinjau tempel-\d{8}-\d{6}(-\d+)?\.png$/;
  try {
    await page.goto(BASE + `/task/${id}`);
    await page.waitForLoadState("networkidle");
    await page
      .getByRole("button", { name: "Detail", exact: true })
      .first()
      .click();
    const arahan = page
      .locator("section", {
        has: page.getByRole("heading", { name: "Arahan" }),
      })
      .last();
    const hasil = page
      .locator("section", { has: page.getByRole("heading", { name: "Hasil" }) })
      .last();
    await hasil.waitFor();
    if (width >= 768)
      expect(
        await hasil
          .getByText("Atau tempel (Ctrl+V) atau seret berkas ke sini.")
          .isVisible(),
        "petunjuk",
        "teks bantuan tempel tidak tampil",
      );

    // Gambar ditempel saat fokus di Catatan hasil: masuk ke Hasil, bukan ke Arahan.
    await hasil.getByLabel("Catatan hasil").focus();
    await tempel("image.png", "image/png");
    const diHasil = await hasil
      .getByRole("button", { name: tempelan })
      .first()
      .waitFor({ timeout: 10000 })
      .then(() => true)
      .catch(() => false);
    expect(diHasil, "Hasil", "gambar tempelan tidak masuk ke Hasil");
    expect(
      (await arahan.getByRole("button", { name: tempelan }).count()) === 0,
      "Hasil",
      "gambar ikut masuk ke Arahan",
    );

    // Gambar ditempel saat fokus di tombol Tambah berkas Arahan: masuk ke Arahan.
    await page.getByRole("button", { name: "Tambah berkas arahan" }).focus();
    await tempel("image.png", "image/png");
    const diArahan = await arahan
      .getByRole("button", { name: tempelan })
      .first()
      .waitFor({ timeout: 10000 })
      .then(() => true)
      .catch(() => false);
    expect(diArahan, "Arahan", "gambar tempelan tidak masuk ke Arahan");
    expect(
      (await hasil.getByRole("button", { name: tempelan }).count()) === 1,
      "Arahan",
      "Hasil ikut bertambah",
    );

    // Teks lewat Ctrl+V sungguhan tetap jadi teks di Catatan hasil, tanpa unggahan baru.
    const catatan = hasil.getByLabel("Catatan hasil");
    await catatan.fill("");
    await page.evaluate(() =>
      navigator.clipboard.writeText("Teks tempelan biasa"),
    );
    await catatan.focus();
    await page.keyboard.press("Control+V");
    expect(
      (await catatan.inputValue()) === "Teks tempelan biasa",
      "teks",
      `isi Catatan hasil "${await catatan.inputValue()}"`,
    );
    expect(
      (await hasil.locator("progress").count()) === 0,
      "teks",
      "tempelan teks memicu unggahan",
    );

    // Di atas batas 50 MB: ditolak sebelum diunggah, dengan pesan.
    await catatan.focus();
    await tempel("besar.png", "image/png", 51 * 1024 * 1024);
    const ditolak = await hasil
      .getByText(/besar\.png berukuran .* melebihi batas 50 MB/)
      // Pesannya juga ada di area aria-live untuk pembaca layar, jadi yang terlihat diambil yang pertama.
      .first()
      .waitFor({ timeout: 5000 })
      .then(() => true)
      .catch(() => false);
    expect(ditolak, "batas", "berkas di atas 50 MB tidak ditolak dengan pesan");
    await audit(page, errors, where("detail"));
  } finally {
    const d = (await mgr.call("GET", `/tasks/${id}`)).body?.task;
    if (d) await mgr.call("DELETE", `/tasks/${id}?version=${d.version}`);
    await mgr.ctx.close();
    await ctx.close();
  }
}

if (ONLY === "paste") for (const w of [360, 1280]) await checkPaste(w);

// Revisi 37: kartu Aktivitas mundur sampai 2 hari kerja. Tidak menulis isian baru: simpan di tanggal terawal
// memakai isi yang sudah ada, dan tanggal di luar rentang harus ditolak 403.
async function checkAktivitasMundur(width) {
  const where = (what) => `aktivitas mundur ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const staf = await apiAs("staf");
  const { ctx, page, errors } = await open("staf", width, "terang");
  try {
    const me = (await staf.call("GET", "/activities/me")).body;
    const editable = me.editable_dates;
    expect(
      editable.length === 3 && editable[0] === me.today,
      "api",
      `editable_dates ${JSON.stringify(editable)}`,
    );
    const earliest = editable[editable.length - 1];

    await page.goto(BASE + "/task");
    await page.waitForLoadState("networkidle");
    const judul = page
      .getByRole("button", { name: /Aktivitas (hari ini|kemarin)|Aktivitas,/ })
      .first();
    if ((await judul.getAttribute("aria-expanded")) === "false")
      await judul.click();
    const prev = page.getByRole("button", { name: /^Aktivitas sebelumnya/ });
    // Keterangan tanggal lalu sudah tampil sejak klik pertama, jadi yang ditunggu jawaban API tanggal barunya.
    const pindah = (btn) =>
      Promise.all([
        page.waitForResponse((r) =>
          r.url().includes("/api/activities/me?date="),
        ),
        btn.click(),
      ]).then(() => page.waitForLoadState("networkidle"));
    const next = page.getByRole("button", { name: /^Aktivitas berikutnya/ });
    expect(await next.isDisabled(), "hari ini", "Berikutnya aktif di hari ini");
    expect(
      (await judul.textContent()).includes("Aktivitas hari ini"),
      "hari ini",
      "judul bukan Aktivitas hari ini",
    );
    expect(
      (await page.getByText("Kamu sedang mengisi aktivitas").count()) === 0,
      "hari ini",
      "keterangan tampil di hari ini",
    );

    for (let i = 1; i < editable.length; i++) {
      expect(
        await prev.isEnabled(),
        `mundur ${i}`,
        "Sebelumnya mati sebelum batas",
      );
      await pindah(prev);
      await page
        .getByText("Kamu sedang mengisi aktivitas")
        .waitFor({ timeout: 8000 });
      await page.waitForLoadState("networkidle");
      expect(
        !(await judul.textContent()).includes("Aktivitas hari ini"),
        `mundur ${i}`,
        "judul tetap hari ini",
      );
      expect(
        /\d+ dari \d+ jam terisi/.test(await judul.textContent()),
        `mundur ${i}`,
        "hitungan jam hilang",
      );
    }
    expect(
      await prev.isDisabled(),
      "batas",
      "Sebelumnya masih aktif di batas paling awal",
    );
    expect(
      (await prev.getAttribute("aria-label")) ===
        "Aktivitas sebelumnya, sudah batas paling awal",
      "batas",
      `aria-label ${await prev.getAttribute("aria-label")}`,
    );
    await audit(page, errors, where("batas"));

    // Simpan di tanggal terawal dengan isi yang sama: diterima. Sehari sebelumnya: ditolak.
    const past = (
      await staf.call("GET", `/activities/me?date=${earliest}`)
    ).body.cells.find((c) => !c.linked);
    const ok = await staf.call("PUT", "/activities", {
      date: earliest,
      slot: past.slot,
      activity: past.activity,
      output: past.output,
    });
    expect(ok.status === 200, "simpan", `simpan di ${earliest}: ${ok.status}`);
    const before = new Date(earliest + "T00:00:00Z");
    before.setUTCDate(before.getUTCDate() - 1);
    const tolak = await staf.call("PUT", "/activities", {
      date: before.toISOString().slice(0, 10),
      slot: 1,
      activity: "x",
      output: "",
    });
    expect(
      tolak.status === 403,
      "simpan",
      `di luar rentang ${tolak.status}, mau 403`,
    );

    for (let i = 1; i < editable.length; i++) {
      await pindah(next);
      await page.waitForLoadState("networkidle");
    }
    await judul
      .filter({ hasText: "Aktivitas hari ini" })
      .waitFor({ timeout: 8000 });
    expect(
      await next.isDisabled(),
      "kembali",
      "Berikutnya aktif setelah kembali ke hari ini",
    );
  } finally {
    await staf.ctx.close();
    await ctx.close();
  }
}

if (ONLY === "aktivitasmundur")
  for (const w of [360, 1280]) await checkAktivitasMundur(w);

// Revisi 38: drawer Ubah pengguna dengan dua centang akses, dan Staf dengan akses melihat tombol tulis Timeline
// dan Important Notes. Akses Staf dikembalikan ke nilai semula di akhir.
async function checkAkses(width) {
  const where = (what) => `akses ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const mgr = await apiAs("manajer");
  const staf = await apiAs("staf");
  const users = (await mgr.call("GET", "/users")).body.users;
  const s = users.find((u) => u.id === staf.me.id);
  const put = (access) =>
    mgr.call("PUT", `/users/${s.id}`, {
      name: s.name,
      role: s.role,
      division_id: s.division_id,
      ...access,
    });
  const asli = {
    can_manage_timeline: s.can_manage_timeline,
    can_manage_notes: s.can_manage_notes,
  };
  const m = await open("manajer", width, "terang");
  try {
    await m.page.goto(BASE + "/pengguna");
    await m.page.waitForLoadState("networkidle");
    await m.page
      .getByRole("button", { name: `Aksi untuk ${s.name}` })
      .locator("visible=true")
      .first()
      .click();
    await m.page.getByRole("menuitem", { name: "Ubah" }).click();
    const dlg = m.page.locator("dialog[open]");
    await dlg.waitFor();
    for (const label of ["Kelola Timeline", "Kelola Important Notes"]) {
      const box = dlg.getByRole("checkbox", { name: new RegExp(label) });
      expect(
        (await box.count()) === 1 && (await box.isEnabled()),
        "drawer Staf",
        `centang "${label}" tidak ada atau mati`,
      );
    }
    // Ganti peran ke Manajer di form (tanpa menyimpan): keduanya tercentang dan nonaktif.
    await dlg.locator("label", { hasText: "Manajer" }).first().click();
    expect(
      (await dlg.getByText("Manajer selalu punya akses ini.").count()) === 1,
      "drawer Manajer",
      "teks Manajer tidak tampil",
    );
    const mbox = dlg.getByRole("checkbox", { name: /Kelola Timeline/ });
    expect(
      (await mbox.isDisabled()) && (await mbox.isChecked()),
      "drawer Manajer",
      "centang Manajer tidak tercentang dan nonaktif",
    );
    await audit(m.page, m.errors, where("drawer"));
    await m.page.keyboard.press("Escape");
  } finally {
    await m.ctx.close();
  }

  const st = await open("staf", width, "terang");
  try {
    const tulis = async () => {
      await st.page.goto(BASE + "/timeline");
      await st.page.waitForLoadState("networkidle");
      await st.page
        .getByText("Memuat jadwal...")
        .waitFor({ state: "detached" })
        .catch(() => {});
      await st.page
        .getByText("Memuat catatan...")
        .waitFor({ state: "detached" })
        .catch(() => {});
      return [
        await st.page.getByRole("button", { name: "Buat kartu" }).count(),
        await st.page.getByLabel("Judul catatan baru").count(),
      ];
    };
    let r = await put({
      can_manage_timeline: true,
      can_manage_notes: true,
    });
    expect(r.status === 204, "beri", `beri akses ${r.status}`);
    const [kartu, catatan] = await tulis();
    expect(
      kartu === 1 && catatan === 1,
      "dengan akses",
      `tombol Buat kartu ${kartu}, form catatan ${catatan}`,
    );
    await audit(st.page, st.errors, where("Staf dengan akses"));
    r = await put({
      can_manage_timeline: false,
      can_manage_notes: false,
    });
    const [k2, c2] = await tulis();
    expect(
      k2 === 0 && c2 === 0,
      "dicabut",
      `setelah dicabut: Buat kartu ${k2}, form catatan ${c2}`,
    );
  } finally {
    await put(asli);
    await st.ctx.close();
    await staf.ctx.close();
    await mgr.ctx.close();
  }
}

if (ONLY === "akses") for (const w of [360, 1280]) await checkAkses(w);

// Revisi 39: halaman awal Timeline dan urutan menu. Login lewat form sungguhan dua kali (tanpa tujuan di 360,
// dari tautan task di 1280), supaya limiter login tidak tersentuh berlebihan.
async function checkBeranda(width) {
  const where = (what) => `beranda ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const mgr = await apiAs("manajer");
  const made = await mgr.call("POST", "/tasks", {
    title: `E2E beranda ${width} ${Date.now()}`,
    urgency: "low",
    pic_ids: [mgr.me.id],
    label_ids: [],
    subtasks: [],
  });
  const taskId = made.body?.id;
  const ctx = await browser.newContext({
    viewport: { width, height: width < 768 ? 800 : 900 },
  });
  const page = await ctx.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(`pageerror ${e.message}`));
  const masuk = async () => {
    await page.getByLabel("Username").fill(ROLES.staf.user);
    await page.getByLabel("Password").fill(ROLES.staf.pass);
    // Limiter login (10 percobaan, satu kembali tiap 30 detik) bisa habis bila E2E dijalankan berulang;
    // jawaban 429 ditunggu sesuai Retry-After lalu dicoba sekali lagi, supaya hasilnya tidak bergantung jeda run.
    for (let i = 0; i < 2; i++) {
      const [res] = await Promise.all([
        page.waitForResponse((r) => r.url().endsWith("/api/auth/login")),
        page.getByRole("button", { name: "Masuk" }).click(),
      ]);
      if (res.status() !== 429) return;
      const wait = Math.min(Number(res.headers()["retry-after"] ?? 30), 90);
      info.push(`beranda ${width}: limiter login, menunggu ${wait} detik`);
      await page.waitForTimeout(wait * 1000 + 500);
    }
  };
  try {
    if (width < 768) {
      // Login tanpa tujuan: masuk ke Timeline.
      await page.goto(BASE + "/");
      await masuk();
      const ok = await page
        .waitForURL(/\/timeline$/, { timeout: 10000 })
        .then(() => true)
        .catch(() => false);
      expect(
        ok,
        "login",
        `setelah login tanpa tujuan ada di ${new URL(page.url()).pathname}`,
      );
    } else {
      // Login dari tautan task langsung: kembali ke task itu.
      await page.goto(BASE + `/task/${taskId}`);
      await page.waitForURL(/kembali=/, { timeout: 10000 });
      await masuk();
      const ok = await page
        .waitForURL(new RegExp(`/task/${taskId}$`), { timeout: 10000 })
        .then(() => true)
        .catch(() => false);
      expect(
        ok,
        "login",
        `setelah login dari tautan task ada di ${new URL(page.url()).pathname}`,
      );
    }
    // Sudah login: "/" ke Timeline, dan tujuan ke situs lain diabaikan.
    for (const path of ["/", "/?kembali=//contoh.test/curang"]) {
      await page.goto(BASE + path);
      await page.waitForURL(/\/timeline$/, { timeout: 10000 }).catch(() => {});
      expect(
        new URL(page.url()).pathname === "/timeline",
        "awal",
        `${path} membuka ${page.url()}`,
      );
    }

    // Urutan menu. Di layar lebar dari top bar, di ponsel dari bar bawah lalu menu Lainnya.
    const nav = page
      .getByRole("navigation", { name: "Navigasi utama" })
      .locator("visible=true")
      .first();
    const labels = (await nav.getByRole("link").allTextContents()).map((t) =>
      t.trim(),
    );
    const mau =
      width >= 768
        ? ["Timeline", "Task", "Aktivitas", "Arsip", "Profil"]
        : ["Timeline", "Task", "Arsip"];
    expect(
      JSON.stringify(labels) === JSON.stringify(mau),
      "urutan",
      `menu ${JSON.stringify(labels)}, mau ${JSON.stringify(mau)}`,
    );
    if (width < 768) {
      await nav.getByRole("button", { name: /Lainnya/ }).click();
      // Menu Lainnya berisi tautan biasa, bukan role menuitem.
      const lain = (
        await page.locator("#menu-lainnya a").allTextContents()
      ).map((t) => t.trim());
      expect(
        lain[0] === "Aktivitas" && lain.includes("Profil"),
        "Lainnya",
        `menu Lainnya ${JSON.stringify(lain)}`,
      );
      await page.keyboard.press("Escape");
    }

    // Logo ke Timeline, bisa dipakai lewat keyboard.
    await page.goto(BASE + "/task");
    await page.waitForLoadState("networkidle");
    const logo = page.getByRole("link", { name: /Internal, ke Timeline/ });
    await logo.focus();
    await page.keyboard.press("Enter");
    const keTimeline = await page
      .waitForURL(/\/timeline$/, { timeout: 10000 })
      .then(() => true)
      .catch(() => false);
    expect(keTimeline, "logo", "logo tidak membuka Timeline");
    await audit(page, errors, where("timeline"));
  } finally {
    const d = taskId
      ? (await mgr.call("GET", `/tasks/${taskId}`)).body?.task
      : null;
    if (d) await mgr.call("DELETE", `/tasks/${taskId}?version=${d.version}`);
    await mgr.ctx.close();
    await ctx.close();
  }
}

if (ONLY === "beranda") for (const w of [360, 1280]) await checkBeranda(w);

// Revisi 40: kolom Completed paling banyak 10 task. 11 task uji berjudul unik diselesaikan Manajer, papan
// disaring dengan kata kunci itu, lalu keterangan "Menampilkan 10 dari 11 task." dan tautan Arsip diperiksa.
async function checkBatasCompleted(width) {
  const where = (what) => `completed ${width} ${what}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const mgr = await apiAs("manajer");
  const kunci = `E2E batas ${width} ${Date.now()}`;
  const made = [];
  const { ctx, page, errors } = await open("manajer", width, "terang");
  try {
    for (let i = 1; i <= 11; i++) {
      const r = await mgr.call("POST", "/tasks", {
        title: `${kunci} nomor ${i}`,
        urgency: "low",
        pic_ids: [mgr.me.id],
        label_ids: [],
        subtasks: [],
      });
      if (r.status !== 201) throw new Error(`buat task ${r.status}`);
      made.push(r.body.id);
      let v = (await mgr.call("GET", `/tasks/${r.body.id}`)).body.task.version;
      for (const to of ["on_progress", "ready", "completed"]) {
        const m = await mgr.call("POST", `/tasks/${r.body.id}/move`, {
          version: v,
          to,
          note: "",
        });
        if (m.status !== 200)
          throw new Error(
            `pindah ke ${to} ${m.status} ${JSON.stringify(m.body)}`,
          );
        v = m.body.version;
      }
    }
    const query = new URLSearchParams({ scope: "all", q: kunci }).toString();
    await page.goto(BASE + `/task?${query}`);
    await page.waitForLoadState("networkidle");
    const pilih = page.locator("[aria-label='Pilih kolom']");
    if (await pilih.isVisible())
      await pilih.getByRole("button", { name: /^Completed/ }).click();
    const kolom = page.locator("section", {
      has: page.locator("#kolom-completed"),
    });
    const kartu = await kolom.locator("article").count();
    expect(
      kartu === 10,
      "kartu",
      `kolom Completed berisi ${kartu} kartu, mau 10`,
    );
    const lencana = (
      await kolom.locator("#kolom-completed .font-mono").textContent()
    )?.trim();
    expect(lencana?.startsWith("11"), "lencana", `lencana ${lencana}, mau 11`);
    const ket = kolom.getByText("Menampilkan 10 dari 11 task.");
    expect(
      (await ket.count()) === 1,
      "keterangan",
      "keterangan Menampilkan 10 dari 11 task tidak tampil",
    );
    // Yang paling baru selesai (nomor 11) ada paling atas, nomor 1 tergeser keluar.
    const pertama = (
      await kolom.locator("article h3").first().textContent()
    )?.trim();
    expect(
      pertama === `${kunci} nomor 11`,
      "urutan",
      `kartu teratas "${pertama}"`,
    );
    expect(
      (await kolom.getByText(`${kunci} nomor 1`, { exact: true }).count()) ===
        0,
      "urutan",
      "task tertua masih tampil",
    );
    await audit(page, errors, where("papan"));

    const tautan = kolom.getByRole("link", { name: "Lihat semua di Arsip" });
    await tautan.focus();
    await page.keyboard.press("Enter");
    await page.waitForURL(/\/arsip\?/, { timeout: 10000 }).catch(() => {});
    const url = new URL(page.url());
    // Untuk Manajer "Semua task" adalah bawaan, jadi scope tidak ikut di URL papan maupun Arsip.
    expect(
      url.pathname === "/arsip" && url.searchParams.get("q") === kunci,
      "Arsip",
      `tautan membuka ${page.url()}`,
    );
  } finally {
    for (const id of made) {
      const d = (await mgr.call("GET", `/tasks/${id}`)).body?.task;
      if (d) await mgr.call("DELETE", `/tasks/${id}?version=${d.version}`);
    }
    await mgr.ctx.close();
    await ctx.close();
  }
}

if (ONLY === "completed")
  for (const w of [360, 1280]) await checkBatasCompleted(w);

// Whiteboard W1 (docs/whiteboard.md). Setiap skenario membuat satu papan uji berjudul unik sebagai kartu di papan
// akar, bekerja di dalamnya, lalu memindahkan kartu papan itu ke Tempat sampah di akhir (data dev; dihapus permanen
// job harian setelah 30 hari).
async function wbSetup(role, title) {
  const api = await apiAs(role);
  // Posisi acak jauh dari tengah supaya kartu uji tidak menumpuk dengan kartu lain di papan akar dev.
  const x = 4000 + Math.floor(Math.random() * 4000);
  const r = await api.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title, x, y: x, w: 200, h: 120 });
  if (r.status !== 201) throw new Error(`buat papan ${r.status} ${JSON.stringify(r.body)}`);
  return { api, card: r.body.card.id, board: r.body.card.ref.id };
}

async function wbCleanup(api, card) {
  await api.call("POST", "/whiteboard/boards/root/cards/delete", { ids: [card] });
}

async function wbCard(api, board, kind, x, y, content, extra = {}) {
  const r = await api.call("POST", `/whiteboard/boards/${board}/cards`, { kind, x, y, w: 240, h: 120, content, ...extra });
  if (r.status !== 201) throw new Error(`buat kartu ${r.status} ${JSON.stringify(r.body)}`);
  return r.body.card;
}

async function wbCards(api, board) {
  return (await api.call("GET", `/whiteboard/boards/${board}`)).body.board.cards;
}

async function wbSaved(page) {
  await page
    .waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, {
      timeout: 10000,
    })
    .catch(() => {});
  return (await page.locator("[data-testid=wb-save]").textContent())?.trim();
}

const wbNote = (text) => ({ doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text }] }] } });

// Alat: di 768 px ke atas dari sidebar, di bawahnya lewat tombol Tambah dan lembar alat.
async function wbTool(page, width, label) {
  if (width >= 768) {
    await page.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: label, exact: true }).click();
  } else {
    await page.getByRole("button", { name: "Tambah", exact: true }).click();
    await page.locator("dialog[open]").getByRole("button", { name: label, exact: true }).click();
  }
}

async function wbOpenTrash(page, width) {
  if (width >= 768) {
    await page.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Tempat sampah" }).click();
  } else {
    await page.locator("main").getByRole("button", { name: /^(Tambah|Menu papan)$/ }).click();
    await page.locator("dialog[open]").getByRole("button", { name: "Tempat sampah" }).click();
  }
  await page.getByRole("heading", { name: "Tempat sampah" }).waitFor({ timeout: 10000 });
}

// Halaman kanvas tidak boleh menggulir: tinggi dan lebar dokumen sama dengan layar.
async function wbNoScroll(page) {
  return page.evaluate(
    () =>
      document.documentElement.scrollHeight <= window.innerHeight + 1 &&
      document.documentElement.scrollWidth <= window.innerWidth,
  );
}

// Papan akar, kartu papan dan membukanya, breadcrumb dan tombol kembali browser, alat klik dan seret, catatan,
// checklist, seret kartu, hapus dan Urungkan, panel Tempat sampah, riwayat, ganti judul, cari, urutan navbar.
async function checkWbDasar(width) {
  const where = (w) => `wbdasar ${width} ${w}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const api = await apiAs("staf");
  const kunci = `E2E wb ${width} ${Date.now()}`;
  const { ctx, page, errors } = await open("staf", width, "terang");
  let boardCard = null;
  try {
    // Urutan menu: Timeline, Task, Aktivitas, Whiteboard, Arsip. Di bawah 1280 px Whiteboard ada di menu Lainnya.
    await page.goto(BASE + "/whiteboard");
    await page.waitForLoadState("networkidle");
    if (width >= 1280) {
      const names = (await page.locator('header nav[aria-label="Navigasi utama"] a').allTextContents()).map((t) => t.trim());
      const i = names.indexOf("Whiteboard");
      expect(i > 0 && names[i - 1] === "Aktivitas" && names[i + 1] === "Arsip", "navbar", `urutan menu ${names.join(", ")}`);
    } else {
      await page.getByRole("button", { name: /Lainnya/ }).first().click();
      const links = (await page.locator("#menu-lainnya a").allTextContents()).map((t) => t.trim());
      expect(links.includes("Whiteboard"), "navbar", `menu Lainnya: ${links.join(", ")}`);
      await page.keyboard.press("Escape");
    }
    expect((await page.getByRole("heading", { level: 1 }).textContent())?.trim().length > 0, "akar", "judul papan akar kosong");
    expect(await wbNoScroll(page), "akar", "halaman Whiteboard bisa digulir");
    await audit(page, errors, where("papan akar"));

    // Papan baru dari alat Papan: kartunya langsung minta judul.
    await wbTool(page, width, "Papan");
    const titleInput = page.getByLabel("Judul papan", { exact: true });
    await titleInput.waitFor({ timeout: 10000 });
    await titleInput.fill(kunci);
    await titleInput.press("Enter");
    await wbSaved(page);
    const rootCards = (await api.call("GET", "/whiteboard/boards/root")).body.board.cards;
    boardCard = rootCards.find((c) => c.kind === "board" && c.ref?.title === kunci);
    expect(!!boardCard, "papan baru", "kartu papan baru tidak tersimpan dengan judulnya");
    if (!boardCard) return;
    const boardId = boardCard.ref.id;

    // Klik sekali hanya memilih kartu papan; dobel klik membukanya di /whiteboard/:id.
    // Tengah kartu: papan akar dev berisi banyak papan uji, jadi Pas ke layar bisa memperkecil sampai 10%.
    await page.locator(`[data-card="${boardCard.id}"]`).click();
    await page.waitForTimeout(300);
    expect(!page.url().endsWith(boardId) && (await page.locator(`[data-card="${boardCard.id}"]`).getAttribute("aria-label")).includes("terpilih"), "klik papan", "klik sekali membuka atau tidak memilih kartu papan");
    await page.locator(`[data-card="${boardCard.id}"]`).dblclick();
    await page.waitForURL(new RegExp(`/whiteboard/${boardId}$`));
    await page.waitForLoadState("networkidle");
    expect((await page.getByRole("heading", { level: 1 }).textContent())?.trim() === kunci, "buka papan", "judul papan anak");
    if (width >= 768) {
      const crumbs = (await page.getByRole("navigation", { name: "Lokasi papan" }).getByRole("link").allTextContents()).map((t) => t.trim());
      expect(crumbs.length === 1 && crumbs[0].length > 0, "breadcrumb", `breadcrumb ${crumbs.join(" > ")}`);
    } else {
      expect(await page.getByRole("link", { name: /^Kembali ke / }).isVisible(), "kembali", "tombol kembali ponsel tidak ada");
    }
    expect(await page.getByText("Papan masih kosong").isVisible(), "kosong", "keterangan papan kosong tidak tampil");
    expect(await wbNoScroll(page), "papan anak", "halaman papan bisa digulir");

    // Catatan dari alat: editor terbuka, ketik, tebal, Escape menutup dan menyimpan.
    await wbTool(page, width, "Catatan");
    const editor = page.getByRole("textbox", { name: "Isi catatan" });
    await editor.waitFor({ timeout: 10000 });
    await audit(page, errors, where("editor catatan"));
    await editor.pressSequentially("Catatan uji E2E");
    await page.getByRole("button", { name: "Tebal" }).click();
    await editor.pressSequentially(" tebal");
    await page.keyboard.press("Escape");
    await editor.waitFor({ state: "detached", timeout: 5000 }).catch(() => {});
    expect((await wbSaved(page)) === "Tersimpan", "simpan catatan", "penanda tidak kembali Tersimpan");
    let cards = await wbCards(api, boardId);
    const note = cards.find((c) => c.kind === "note");
    const doc = JSON.stringify(note?.content ?? {});
    expect(doc.includes("Catatan uji E2E") && doc.includes('"bold"'), "isi catatan", `isi tersimpan ${doc.slice(0, 200)}`);
    expect((await page.locator(`[data-card="${note?.id}"] strong`).count()) === 1, "render", "teks tebal tidak dirender");

    // Checklist: judul, satu butir, tutup, lalu centang tanpa membuka editor.
    await wbTool(page, width, "Checklist");
    await page.getByLabel("Judul checklist").fill("Rilis E2E");
    await page.getByRole("button", { name: "Tambah butir" }).click();
    const item = page.getByLabel("Teks butir 1");
    await item.waitFor({ timeout: 10000 });
    await item.fill("Butir satu");
    await page.keyboard.press("Escape");
    await wbSaved(page);
    await page.getByRole("checkbox", { name: "Selesai: Butir satu" }).check();
    await wbSaved(page);
    cards = await wbCards(api, boardId);
    const list = cards.find((c) => c.kind === "checklist");
    expect(
      list?.content.title === "Rilis E2E" && list.content.items[0]?.text === "Butir satu" && list.content.items[0]?.done,
      "checklist",
      `isi checklist ${JSON.stringify(list?.content)}`,
    );

    // Seret alat Judul ke titik tertentu di kanvas (sidebar hanya ada mulai 768 px).
    const canvas = page.locator('[role="application"]');
    if (width >= 768) {
      const tool = await page.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Judul", exact: true }).boundingBox();
      const cb = await canvas.boundingBox();
      await page.mouse.move(tool.x + tool.width / 2, tool.y + tool.height / 2);
      await page.mouse.down();
      await page.mouse.move(cb.x + 60, cb.y + cb.height - 80, { steps: 6 });
      await page.mouse.move(cb.x + 80, cb.y + cb.height - 80, { steps: 3 });
      await page.mouse.up();
      const headingInput = page.getByLabel("Teks judul");
      await headingInput.waitFor({ timeout: 10000 });
      // Diukur sebelum mengetik: setelah editor ditutup, kanvas menggeser kartu yang terpotong ke tengah layar.
      const hb = await page.locator("article", { has: headingInput }).boundingBox();
      await headingInput.fill("Judul diseret");
      await headingInput.press("Enter");
      await wbSaved(page);
      const heading = (await wbCards(api, boardId)).find((c) => c.kind === "heading");
      expect(heading?.content.text === "Judul diseret" && hb && Math.abs(hb.x + hb.width / 2 - (cb.x + 80)) < 60, "seret alat", `judul diseret di ${JSON.stringify(hb)}`);
    }

    // Seret catatan 96 px ke kanan dengan mouse, dipegang di pojok kiri atas (checklist menumpuk di kanan bawahnya).
    const zoom = parseInt((await page.locator("[data-testid=wb-zoom]").textContent())?.trim() ?? "100", 10) / 100;
    const box = await page.locator(`[data-card="${note.id}"]`).boundingBox();
    await page.mouse.move(box.x + 6, box.y + 6);
    await page.mouse.down();
    await page.mouse.move(box.x + 6 + 48 * zoom, box.y + 6, { steps: 4 });
    await page.mouse.move(box.x + 6 + 96 * zoom, box.y + 6, { steps: 4 });
    await page.mouse.up();
    await wbSaved(page);
    const moved = (await wbCards(api, boardId)).find((c) => c.id === note.id);
    expect(Math.abs(moved.x - note.x - 96) <= 8, "seret", `x ${note.x} menjadi ${moved.x}, mau sekitar +96`);

    // Hapus, Urungkan dari toast, lalu hapus lagi dan pulihkan dari panel Tempat sampah.
    await page.locator(`[data-card="${note.id}"]`).click({ position: { x: 6, y: 6 } });
    await page.getByRole("button", { name: "Hapus 1 kartu" }).click();
    await page.getByRole("button", { name: "Urungkan" }).click();
    await page.locator(`[data-card="${note.id}"]`).waitFor({ timeout: 10000 });
    await page.locator(`[data-card="${note.id}"]`).click({ position: { x: 6, y: 6 } });
    await page.getByRole("button", { name: "Hapus 1 kartu" }).click();
    await page.locator(`[data-card="${note.id}"]`).waitFor({ state: "detached", timeout: 10000 });
    await page.getByRole("button", { name: "Tutup" }).first().click().catch(() => {});
    await wbOpenTrash(page, width);
    await audit(page, errors, where("tempat sampah"));
    const restore = page.locator("dialog[open]").getByRole("button", { name: "Pulihkan 1 kartu" }).first();
    expect((await restore.count()) > 0, "tempat sampah", "batch kartu tidak ada di Tempat sampah");
    await restore.click();
    await page.locator(`[data-card="${note.id}"]`).waitFor({ timeout: 10000 });
    await page.keyboard.press("Escape");
    await page.waitForFunction(() => !document.querySelector("dialog[open]"));

    // Riwayat papan sebagai panel.
    await page.getByRole("button", { name: "Riwayat" }).click();
    await page.getByText("mengubah isi catatan").first().waitFor({ timeout: 10000 });
    await audit(page, errors, where("riwayat"));
    await page.keyboard.press("Escape");
    await page.waitForFunction(() => !document.querySelector("dialog[open]"));

    // Ganti judul papan dari bar atas.
    await page.getByRole("button", { name: "Ubah judul papan" }).click();
    const title = page.getByLabel("Judul papan", { exact: true });
    await title.fill(`${kunci} baru`);
    await title.press("Enter");
    await wbSaved(page);
    expect((await page.getByRole("heading", { level: 1 }).textContent())?.trim() === `${kunci} baru`, "judul", "judul baru tidak tampil");

    // Kembali ke papan akar lewat breadcrumb atau tombol kembali; kartu papan menampilkan judul baru dan ringkasan.
    if (width >= 768) await page.getByRole("navigation", { name: "Lokasi papan" }).getByRole("link").first().click();
    else await page.getByRole("link", { name: /^Kembali ke / }).click();
    await page.waitForURL(/\/whiteboard$/);
    const cardText = await page.locator(`[data-card="${boardCard.id}"]`).textContent();
    // Catatan dan checklist, ditambah judul yang diseret dari sidebar mulai 768 px.
    const isi = width >= 768 ? "3 kartu" : "2 kartu";
    expect(cardText.includes(`${kunci} baru`) && cardText.includes(isi), "ringkasan", `kartu papan: ${cardText}, mau ${isi}`);
    // Tombol kembali browser membuka papan anak lagi.
    await page.goBack();
    await page.waitForURL(new RegExp(`/whiteboard/${boardId}$`));
    expect(true, "kembali browser", "");

    // Cari judul papan.
    if (width < 768) await page.getByRole("button", { name: "Cari papan" }).click();
    await page.getByLabel("Cari judul papan").fill(`${kunci} baru`);
    await page.getByLabel("Cari judul papan").press("Enter");
    const hit = page.getByRole("list", { name: "Hasil pencarian papan" }).getByRole("link").first();
    await hit.waitFor({ timeout: 10000 });
    await audit(page, errors, where("cari"));
    expect((await hit.textContent()).includes(`${kunci} baru`), "cari", "hasil cari tidak memuat papan uji");
  } finally {
    if (boardCard) await wbCleanup(api, boardCard.id);
    await api.ctx.close();
    await ctx.close();
  }
}

// Keyboard: Tab ke kartu, panah berpindah, Shift+panah menggeser 8 px, Enter dan Escape, Ctrl+A, Delete, Enter
// membuka kartu papan.
async function checkWbKeyboard(width) {
  const where = (w) => `wbkeyboard ${width} ${w}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const { api, card, board } = await wbSetup("staf", `E2E wbkb ${width} ${Date.now()}`);
  const a = await wbCard(api, board, "note", 0, 0, wbNote("Kartu A"));
  const b = await wbCard(api, board, "note", 320, 0, wbNote("Kartu B"));
  const sub = await wbCard(api, board, "board", 0, 200, undefined, { title: "Sub papan" });
  const { ctx, page, errors } = await open("staf", width, "gelap");
  try {
    await page.goto(BASE + `/whiteboard/${board}`);
    await page.waitForLoadState("networkidle");
    let reached = false;
    for (let i = 0; i < 40 && !reached; i++) {
      await page.keyboard.press("Tab");
      reached = await page.evaluate((id) => document.activeElement?.getAttribute("data-card") === id, a.id);
    }
    expect(reached, "tab", "Tab tidak sampai ke kartu pertama");
    const ring = await page.evaluate(() => getComputedStyle(document.activeElement).outlineStyle);
    expect(ring === "dashed", "fokus", `cincin fokus ${ring}`);
    await page.keyboard.press("ArrowRight");
    expect(await page.evaluate((id) => document.activeElement?.getAttribute("data-card") === id, b.id), "panah", "ArrowRight tidak ke kartu B");
    await page.keyboard.press("Shift+ArrowDown");
    await page.waitForTimeout(600);
    await wbSaved(page);
    const after = (await wbCards(api, board)).find((c) => c.id === b.id);
    expect(after.y === b.y + 8, "shift panah", `y ${b.y} menjadi ${after.y}`);

    await page.keyboard.press("Enter");
    const editor = page.getByRole("textbox", { name: "Isi catatan" });
    await editor.waitFor({ timeout: 10000 });
    await page.keyboard.press("Escape");
    await editor.waitFor({ state: "detached", timeout: 5000 }).catch(() => {});
    expect((await editor.count()) === 0, "escape", "Escape tidak menutup editor");
    expect(await page.evaluate((id) => document.activeElement?.getAttribute("data-card") === id, b.id), "escape fokus", "fokus tidak kembali ke kartu setelah Escape");

    await page.keyboard.press("Control+a");
    expect(await page.getByRole("button", { name: "Hapus 3 kartu" }).isVisible(), "ctrl a", "Ctrl+A tidak memilih 3 kartu");
    await page.keyboard.press("Escape");
    expect((await page.getByRole("button", { name: /^Hapus \d+ kartu$/ }).count()) === 0, "escape pilihan", "Escape tidak membatalkan pilihan");

    await page.keyboard.press("ArrowLeft");
    await page.keyboard.press("Delete");
    await page.getByRole("button", { name: "Urungkan" }).waitFor({ timeout: 10000 });
    const left = await wbCards(api, board);
    expect(left.length === 2 && !left.some((c) => c.id === a.id), "delete", `kartu tersisa ${left.length}`);
    await audit(page, errors, where("papan"));

    // Enter pada kartu papan membukanya.
    await page.locator(`[data-card="${sub.id}"]`).focus();
    await page.keyboard.press("Enter");
    await page.waitForURL(new RegExp(`/whiteboard/${sub.ref.id}$`), { timeout: 10000 }).catch(() => {});
    expect(page.url().endsWith(`/whiteboard/${sub.ref.id}`), "enter papan", `Enter pada kartu papan membuka ${page.url()}`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await ctx.close();
  }
}

// Layar sentuh: ketuk kartu memilih, ketuk lagi membuka editor, ketuk kanvas menutup; Tambah, Catatan; ketuk kartu
// papan membukanya; geser satu jari dan pinch.
async function checkWbPonsel(width) {
  const where = (w) => `wbponsel ${width} ${w}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const { api, card: holder, board } = await wbSetup("staf", `E2E wbhp ${width} ${Date.now()}`);
  const a = await wbCard(api, board, "note", 0, 0, wbNote("Kartu sentuh"));
  const sub = await wbCard(api, board, "board", 0, 400, undefined, { title: "Sub sentuh" });
  const ctx = await browser.newContext({
    viewport: { width, height: 800 },
    hasTouch: true,
    isMobile: width < 768,
    storageState: sessions.staf,
  });
  const page = await ctx.newPage();
  const errors = [];
  page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
  page.on("pageerror", (e) => errors.push(`pageerror ${e.message}`));
  try {
    await page.goto(BASE + `/whiteboard/${board}`);
    await page.waitForLoadState("networkidle");
    await audit(page, errors, where("papan"));
    const editor = page.getByRole("textbox", { name: "Isi catatan" });
    const canvas = page.locator('[role="application"]');
    const cb = await canvas.boundingBox();
    // Titik kosong di kanvas: tepi kanan bagian tengah, jauh dari kartu (di kiri), bar pilihan (di atas), dan
    // kontrol tampilan (di bawah).
    const empty = { x: cb.x + cb.width - 12, y: cb.y + cb.height / 2 };
    const card = page.locator(`[data-card="${a.id}"]`);
    // Titik ketuk di bawah kartu: bar pilihan yang muncul setelah ketukan pertama menutupi bagian atasnya di ponsel.
    const cardBox = await card.boundingBox();
    const tapAt = { x: 12, y: cardBox.height - 12 };
    await card.tap({ position: tapAt });
    await card.tap({ position: tapAt });
    await editor.waitFor({ timeout: 10000 }).catch(() => {});
    expect(await editor.isVisible().catch(() => false), "ketuk dua kali", "ketukan kedua tidak membuka editor");
    await page.touchscreen.tap(empty.x, empty.y);
    await editor.waitFor({ state: "detached", timeout: 5000 }).catch(() => {});
    expect((await editor.count()) === 0, "ketuk kanvas", "ketuk kanvas tidak menutup editor");

    if (width < 768) {
      await page.getByRole("button", { name: "Tambah", exact: true }).tap();
      await page.locator("dialog[open]").getByRole("button", { name: "Catatan", exact: true }).tap();
    } else {
      await page.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Catatan", exact: true }).tap();
    }
    await editor.waitFor({ timeout: 10000 });
    await editor.pressSequentially("Dari ponsel");
    await page.touchscreen.tap(empty.x, empty.y);
    await editor.waitFor({ state: "detached", timeout: 5000 }).catch(() => {});
    await wbSaved(page);
    const cards = await wbCards(api, board);
    expect(cards.some((c) => JSON.stringify(c.content).includes("Dari ponsel")), "catatan", "catatan dari ponsel tidak tersimpan");

    // Geser satu jari dan pinch, lewat PointerEvent sentuh sintetis di kanvas kosong.
    const before = await card.boundingBox();
    const zoomBefore = await page.locator("[data-testid=wb-zoom]").textContent();
    await page.evaluate(
      ({ x, y }) => {
        const el = document.querySelector('[role="application"]');
        const fire = (type, id, cx, cy) =>
          el.dispatchEvent(
            new PointerEvent(type, { pointerId: id, pointerType: "touch", clientX: cx, clientY: cy, bubbles: true, isPrimary: id === 21 }),
          );
        fire("pointerdown", 21, x, y);
        fire("pointermove", 21, x - 40, y + 30);
        fire("pointermove", 21, x - 80, y + 60);
        fire("pointerup", 21, x - 80, y + 60);
        fire("pointerdown", 22, x - 160, y);
        fire("pointerdown", 23, x - 120, y);
        fire("pointermove", 23, x - 40, y);
        fire("pointermove", 23, x, y);
        fire("pointerup", 23, x, y);
        fire("pointerup", 22, x - 160, y);
      },
      empty,
    );
    const afterBox = await card.boundingBox();
    const zoomAfter = await page.locator("[data-testid=wb-zoom]").textContent();
    expect(afterBox && before && (afterBox.x !== before.x || afterBox.y !== before.y), "geser", "kanvas tidak bergeser");
    expect(parseInt(zoomAfter, 10) > parseInt(zoomBefore, 10), "pinch", `zoom ${zoomBefore} menjadi ${zoomAfter}`);
    await audit(page, errors, where("setelah pinch"));

    // Dobel ketuk kartu papan membukanya.
    await page.getByRole("button", { name: "Pas ke layar" }).tap();
    const subBox = await page.locator(`[data-card="${sub.id}"]`).boundingBox();
    await page.locator(`[data-card="${sub.id}"]`).tap({ position: { x: 20, y: subBox.height - 12 } });
    await page.locator(`[data-card="${sub.id}"]`).tap({ position: { x: 20, y: subBox.height - 12 } });
    await page.waitForURL(new RegExp(`/whiteboard/${sub.ref.id}$`), { timeout: 10000 }).catch(() => {});
    expect(page.url().endsWith(`/whiteboard/${sub.ref.id}`), "ketuk papan", `ketuk kartu papan membuka ${page.url()}`);
  } finally {
    await wbCleanup(api, holder);
    await api.ctx.close();
    await ctx.close();
  }
}

// Admin hanya membaca: tanpa alat tulis, tanpa ubah judul, Tempat sampah tanpa Pulihkan; API tulis 403.
async function checkWbAdmin(width) {
  const where = (w) => `wbadmin ${width} ${w}`;
  const expect = (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
  const { api, card, board } = await wbSetup("staf", `E2E wbadm ${width} ${Date.now()}`);
  await wbCard(api, board, "note", 0, 0, wbNote("Kartu Staf"));
  const admin = await apiAs("admin");
  const { ctx, page, errors } = await open("admin", width, "terang");
  try {
    await page.goto(BASE + `/whiteboard/${board}`);
    await page.waitForLoadState("networkidle");
    expect((await page.getByRole("button", { name: /^(Catatan|Judul|Checklist|Papan|Tambah)$/ }).count()) === 0, "papan", "alat tulis tampil untuk Admin");
    expect((await page.getByRole("button", { name: "Ubah judul papan" }).count()) === 0, "papan", "tombol ubah judul tampil untuk Admin");
    expect((await page.locator("[data-card]").count()) === 1, "papan", "kartu tidak terlihat oleh Admin");
    expect(await wbNoScroll(page), "papan", "halaman bisa digulir");
    await audit(page, errors, where("papan"));
    await wbOpenTrash(page, width);
    expect((await page.locator("dialog[open]").getByRole("button", { name: /^Pulihkan/ }).count()) === 0, "sampah", "tombol Pulihkan tampil untuk Admin");
    await audit(page, errors, where("tempat sampah"));
    const w = await admin.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "note", x: 0, y: 0, w: 240, h: 120 });
    expect(w.status === 403, "api", `POST kartu oleh Admin ${w.status}`);
    const d = await admin.call("POST", "/whiteboard/boards/root/cards/delete", { ids: [card] });
    expect(d.status === 403, "api", `hapus kartu papan oleh Admin ${d.status}`);
    const t = await admin.call("PUT", "/whiteboard/boards/root", { title: "x", version: 1 });
    expect(t.status === 403, "api", `ganti judul papan akar oleh Admin ${t.status}`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await admin.ctx.close();
    await ctx.close();
  }
}

if (ONLY === "wbdasar") for (const w of [360, 1280]) await checkWbDasar(w);
if (ONLY === "wbkeyboard") for (const w of [360, 1280]) await checkWbKeyboard(w);
if (ONLY === "wbponsel") for (const w of [360, 1280]) await checkWbPonsel(w);
if (ONLY === "wbadmin") for (const w of [360, 1280]) await checkWbAdmin(w);
if (ONLY === "wb1")
  for (const w of [360, 1280]) {
    await checkWbDasar(w);
    await checkWbKeyboard(w);
    await checkWbPonsel(w);
    await checkWbAdmin(w);
  }


// Whiteboard W2: kolaborasi langsung dengan beberapa konteks browser (akun berbeda) dan dua tab akun yang sama.
// Setiap skenario memakai papan uji sendiri (kartu papan di papan akar) yang dipindah ke Tempat sampah di akhir.
async function wbPage(ctx, board, errors) {
  const page = await ctx.newPage();
  page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
  page.on("pageerror", (e) => errors.push(`pageerror ${e.message}`));
  await page.goto(BASE + `/whiteboard/${board}`);
  await page.waitForLoadState("networkidle");
  // Tunggu WebSocket tersambung: penanda tidak lagi "Terputus".
  await page
    .waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 })
    .catch(() => {});
  return page;
}

async function wbContext(role, width) {
  return browser.newContext({ viewport: { width, height: width < 768 ? 800 : 900 }, storageState: sessions[role] });
}

function wbExpect(name, width) {
  const where = (w) => `${name} ${width} ${w}`;
  return (cond, w, msg) => {
    checks++;
    if (!cond) fail(where(w), msg);
  };
}

async function cardLabelOf(page, id) {
  return (await page.locator(`[data-card="${id}"]`).getAttribute("aria-label").catch(() => "")) ?? "";
}

// Kehadiran: tiga orang saling melihat avatar dan kursor bernama; orang yang diam 3 menit tampil redup.
async function checkWbHadir(width) {
  const expect = wbExpect("wbhadir", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbhadir ${width} ${Date.now()}`);
  await wbCard(api, board, "note", 0, 0, wbNote("Kartu hadir"));
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  const ca = await wbContext("admin", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await cm.newPage();
    pm.on("console", (m) => m.type() === "error" && errors.push(m.text()));
    // Jam Manajer dipercepat untuk menguji status diam, tanpa menunggu 3 menit sungguhan.
    await pm.clock.install();
    await pm.goto(BASE + `/whiteboard/${board}`);
    await pm.waitForLoadState("networkidle");
    const pa = await wbPage(ca, board, errors);
    await ps.locator("[data-testid=wb-presence]").nth(1).waitFor({ timeout: 10000 });
    expect((await ps.locator("[data-testid=wb-presence]").count()) === 2, "avatar", "Staf tidak melihat dua orang lain");
    expect((await pa.locator("[data-testid=wb-presence]").count()) === 2, "avatar", "Admin tidak melihat dua orang lain");

    // Kursor Manajer terlihat di layar Staf dengan nama.
    const cb = await pm.locator('[role="application"]').boundingBox();
    await pm.mouse.move(cb.x + cb.width / 2, cb.y + cb.height / 2);
    await pm.clock.runFor(100);
    await pm.mouse.move(cb.x + cb.width / 2 + 30, cb.y + cb.height / 2 + 20, { steps: 3 });
    const cursor = ps.locator("[data-testid=wb-cursor]").first();
    await cursor.waitFor({ timeout: 10000 }).catch(() => {});
    expect((await cursor.count()) > 0, "kursor", "kursor Manajer tidak terlihat");

    // Manajer diam 3 menit: avatarnya di layar Staf menyebut diam.
    await pm.clock.runFor(185000);
    await ps
      .waitForFunction(() => [...document.querySelectorAll("[data-testid=wb-presence]")].some((b) => b.getAttribute("aria-label")?.includes("diam")), null, { timeout: 10000 })
      .catch(() => {});
    const labels = await ps.locator("[data-testid=wb-presence]").evaluateAll((els) => els.map((e) => e.getAttribute("aria-label")));
    expect(labels.some((l) => l.includes("diam")), "diam", `label avatar: ${labels.join(" | ")}`);
    await audit(ps, errors, `wbhadir ${width} papan`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
    await cm.close();
    await ca.close();
  }
}

// Edit langsung: Staf mengetik, Manajer melihat teks berubah dan label "sedang mengedit", tidak bisa membuka editor
// kartu itu, tetapi bisa mengganti warnanya; kedua perubahan tersimpan.
async function checkWbLangsung(width) {
  const expect = wbExpect("wblangsung", width);
  const { api, card, board } = await wbSetup("staf", `E2E wblangsung ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Awal"));
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    await ps.locator(`[data-card="${note.id}"]`).click({ position: { x: 6, y: 6 } });
    await ps.keyboard.press("Enter");
    const editor = ps.getByRole("textbox", { name: "Isi catatan" });
    await editor.waitFor({ timeout: 10000 });
    await editor.press("End");
    await editor.pressSequentially(" ditambah Staf");
    await pm.waitForFunction((id) => document.querySelector(`[data-card="${id}"]`)?.textContent?.includes("ditambah Staf"), note.id, { timeout: 10000 }).catch(() => {});
    expect((await pm.locator(`[data-card="${note.id}"]`).textContent()).includes("ditambah Staf"), "draf", "teks Staf tidak terlihat langsung");
    expect((await cardLabelOf(pm, note.id)).includes("sedang mengedit"), "label", "label sedang mengedit tidak ada");

    // Manajer mencoba mengedit: ditolak dengan nama pemegang. Mengganti warna tetap boleh.
    await pm.locator(`[data-card="${note.id}"]`).click({ position: { x: 6, y: 6 } });
    await pm.keyboard.press("Enter");
    await pm.getByText(/sedang mengedit kartu ini/).first().waitFor({ timeout: 5000 }).catch(() => {});
    expect((await pm.getByRole("textbox", { name: "Isi catatan" }).count()) === 0, "kunci", "Manajer bisa membuka editor kartu yang dikunci");
    await pm.getByLabel("Warna kartu terpilih").selectOption("kuning");
    await wbSaved(pm);
    await ps.keyboard.press("Escape");
    await wbSaved(ps);
    const saved = (await wbCards(api, board)).find((c) => c.id === note.id);
    expect(saved.color === "kuning" && JSON.stringify(saved.content).includes("ditambah Staf"), "tersimpan", `warna ${saved.color}, isi ${JSON.stringify(saved.content).slice(0, 120)}`);
    await audit(pm, errors, `wblangsung ${width} papan`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
    await cm.close();
  }
}

// Seret langsung: Manajer melihat kartu bergerak sebelum Staf melepas. Dua orang memegang kartu yang sama, yang
// kedua dibatalkan dengan label.
async function checkWbSeret(width) {
  const expect = wbExpect("wbseret", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbseret ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Seret aku"));
  await wbCard(api, board, "note", 600, 400, wbNote("Jangkar"));
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    const before = await pm.locator(`[data-card="${note.id}"]`).boundingBox();
    const sb = await ps.locator(`[data-card="${note.id}"]`).boundingBox();
    await ps.mouse.move(sb.x + 6, sb.y + 6);
    await ps.mouse.down();
    await ps.mouse.move(sb.x + 40, sb.y + 30, { steps: 5 });
    await ps.mouse.move(sb.x + 80, sb.y + 60, { steps: 5 });
    await pm.waitForTimeout(400);
    const during = await pm.locator(`[data-card="${note.id}"]`).boundingBox();
    expect(during && (during.x !== before.x || during.y !== before.y), "langsung", "kartu tidak bergerak di layar Manajer sebelum dilepas");
    expect((await cardLabelOf(pm, note.id)).includes("sedang memindahkan"), "label", "label sedang memindahkan tidak ada");

    // Manajer mencoba menyeret kartu yang sama selama Staf masih memegangnya.
    const mb = await pm.locator(`[data-card="${note.id}"]`).boundingBox();
    await pm.mouse.move(mb.x + 6, mb.y + 6);
    await pm.mouse.down();
    await pm.mouse.move(mb.x + 60, mb.y + 60, { steps: 5 });
    await pm.getByText(/sedang memindahkan kartu ini/).first().waitFor({ timeout: 5000 }).catch(() => {});
    await pm.mouse.up();
    expect((await pm.getByText(/sedang memindahkan kartu ini/).count()) > 0, "rebut", "seret kedua tidak dibatalkan dengan label");
    await ps.mouse.up();
    await wbSaved(ps);
    await wbSaved(pm);
    // Posisi akhir hanya hasil seret Staf (74 px layar ke kanan), bukan ditambah seret Manajer yang ditolak.
    const z = parseInt((await ps.locator("[data-testid=wb-zoom]").textContent()) ?? "100", 10) / 100;
    const saved = (await wbCards(api, board)).find((c) => c.id === note.id);
    expect(Math.abs(saved.x - note.x - 74 / z) <= 16, "posisi", `posisi akhir ${saved.x}, ${saved.y}, mau sekitar ${Math.round(74 / z)} (zoom ${z})`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
    await cm.close();
  }
}

// Kartu yang sedang diedit dihapus orang lain: yang mengedit melihat dialog dengan nama penghapus, dan Pulihkan
// kartu mengembalikan ketikan terakhirnya.
async function checkWbHapusEdit(width) {
  const expect = wbExpect("wbhapusedit", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbhapus ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Awal"));
  const mgr = await apiAs("manajer");
  const errors = [];
  const cs = await wbContext("staf", width);
  try {
    const ps = await wbPage(cs, board, errors);
    await ps.locator(`[data-card="${note.id}"]`).click({ position: { x: 6, y: 6 } });
    await ps.keyboard.press("Enter");
    const editor = ps.getByRole("textbox", { name: "Isi catatan" });
    await editor.waitFor({ timeout: 10000 });
    await editor.press("End");
    await editor.pressSequentially(" ketikan terakhir");
    await ps.waitForTimeout(500);
    const del = await mgr.call("POST", `/whiteboard/boards/${board}/cards/delete`, { ids: [note.id] });
    expect(del.status === 200, "hapus", `hapus oleh Manajer ${del.status}`);
    const dialog = ps.locator("dialog[open]", { hasText: "dipindah ke Tempat sampah oleh" });
    await dialog.waitFor({ timeout: 10000 }).catch(() => {});
    expect((await dialog.count()) === 1, "dialog", "dialog kartu dihapus tidak muncul");
    await audit(ps, errors, `wbhapusedit ${width} dialog`);
    await dialog.getByRole("button", { name: "Pulihkan kartu" }).click();
    await ps.locator(`[data-card="${note.id}"]`).waitFor({ timeout: 10000 });
    const back = (await wbCards(api, board)).find((c) => c.id === note.id);
    expect(back && JSON.stringify(back.content).includes("ketikan terakhir"), "pulih", `isi setelah pulih ${JSON.stringify(back?.content).slice(0, 120)}`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await mgr.ctx.close();
    await cs.close();
  }
}

// Putus: Staf offline, penanda Terputus tampil, perubahannya menunggu; Manajer mengubah kartu lain; setelah online
// lagi keduanya sama dan penanda kembali Tersimpan.
async function checkWbPutus(width) {
  const expect = wbExpect("wbputus", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbputus ${width} ${Date.now()}`);
  const a = await wbCard(api, board, "note", 0, 0, wbNote("Kartu A"));
  const b = await wbCard(api, board, "note", 400, 0, wbNote("Kartu B"));
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    await cs.setOffline(true);
    // Seret kartu A selagi offline: REST gagal dan masuk antrean.
    const ab = await ps.locator(`[data-card="${a.id}"]`).boundingBox();
    await ps.mouse.move(ab.x + 6, ab.y + 6);
    await ps.mouse.down();
    await ps.mouse.move(ab.x + 60, ab.y + 6, { steps: 6 });
    await ps.mouse.up();
    await ps.locator("[data-testid=wb-offline]").waitFor({ timeout: 10000 }).catch(() => {});
    expect(await ps.locator("[data-testid=wb-offline]").isVisible(), "penanda", "penanda Terputus tidak tampil");
    expect((await ps.locator("[data-testid=wb-save]").textContent()).includes("Terputus"), "status", "status bukan Terputus");
    // Manajer mengubah warna kartu B selama Staf putus.
    await pm.locator(`[data-card="${b.id}"]`).click({ position: { x: 6, y: 6 } });
    await pm.getByLabel("Warna kartu terpilih").selectOption("biru");
    await wbSaved(pm);
    await cs.setOffline(false);
    await ps.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 40000 }).catch(() => {});
    expect((await ps.locator("[data-testid=wb-save]").textContent()).trim() === "Tersimpan", "pulih", "penanda tidak kembali Tersimpan");
    const saved = await wbCards(api, board);
    const sa = saved.find((c) => c.id === a.id);
    expect(sa.x !== a.x, "antrean", "seret selama offline hilang");
    // Staf melihat warna dari Manajer, Manajer melihat posisi baru kartu A.
    await ps.waitForFunction((id) => document.querySelector(`[data-card="${id}"]`)?.className.includes("bg-completed-bg"), b.id, { timeout: 10000 }).catch(() => {});
    expect((await ps.locator(`[data-card="${b.id}"]`).getAttribute("class")).includes("bg-completed-bg"), "sinkron", "Staf tidak melihat warna dari Manajer");
    const pa = await pm.locator(`[data-card="${a.id}"]`).boundingBox();
    const pmZoom = parseInt((await pm.locator("[data-testid=wb-zoom]").textContent()) ?? "100", 10) / 100;
    expect(pa && Math.abs(pa.x - (await pm.locator(`[data-card="${b.id}"]`).boundingBox()).x - (sa.x - b.x) * pmZoom) < 4, "sinkron", "Manajer tidak melihat posisi baru kartu A");
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
    await cm.close();
  }
}

// Undo hanya aksi sendiri: Staf memindah lalu mewarnai, Manajer mengganti warnanya; undo warna dilewati dengan
// pesan, undo pindah mengembalikan posisi, warna Manajer tetap. Salin, tempel, dan duplikat ikut diuji.
async function checkWbUndo(width) {
  const expect = wbExpect("wbundo", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbundo ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Undo"));
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    const loc = ps.locator(`[data-card="${note.id}"]`);
    await loc.click({ position: { x: 6, y: 6 } });
    await ps.keyboard.press("Shift+ArrowRight");
    await ps.keyboard.press("Shift+ArrowRight");
    await ps.waitForTimeout(600);
    await wbSaved(ps);
    await ps.getByLabel("Warna kartu terpilih").selectOption("kuning");
    await wbSaved(ps);
    await pm.waitForTimeout(300);
    await pm.locator(`[data-card="${note.id}"]`).click({ position: { x: 6, y: 6 } });
    await pm.getByLabel("Warna kartu terpilih").selectOption("hijau");
    await wbSaved(pm);
    await ps.waitForFunction((id) => document.querySelector(`[data-card="${id}"]`)?.className.includes("bg-progress-bg"), note.id, { timeout: 10000 }).catch(() => {});
    await loc.focus();
    await ps.keyboard.press("Control+z");
    await ps.getByText("Sebagian langkah tidak dibatalkan").first().waitFor({ timeout: 5000 }).catch(() => {});
    expect((await ps.getByText("Sebagian langkah tidak dibatalkan").count()) > 0, "lewat", "pesan langkah dilewati tidak tampil");
    await ps.keyboard.press("Control+z");
    await wbSaved(ps);
    let saved = (await wbCards(api, board)).find((c) => c.id === note.id);
    expect(saved.x === note.x && saved.color === "hijau", "undo", `setelah undo x ${saved.x} warna ${saved.color}`);
    await ps.keyboard.press("Control+Shift+z");
    await wbSaved(ps);
    saved = (await wbCards(api, board)).find((c) => c.id === note.id);
    expect(saved.x === note.x + 16, "redo", `setelah redo x ${saved.x}`);

    // Duplikat dan salin-tempel membuat kartu baru dengan isi yang sama.
    await loc.click({ position: { x: 6, y: 6 } });
    await ps.keyboard.press("Control+d");
    await ps.waitForFunction((n) => document.querySelectorAll("[data-card]").length >= n, 2, { timeout: 10000 }).catch(() => {});
    await wbSaved(ps);
    const cards = await wbCards(api, board);
    expect(cards.length === 2 && cards.every((c) => JSON.stringify(c.content).includes("Undo")), "duplikat", `kartu ${cards.length}`);
    await ps.keyboard.press("Control+z");
    await wbSaved(ps);
    expect((await wbCards(api, board)).length === 1, "undo duplikat", "undo duplikat tidak menghapus salinannya");
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
    await cm.close();
  }
}

// Dua tab akun yang sama: kunci berpindah lewat "Edit di sini"; undo tab kedua tidak membatalkan aksi tab pertama.
async function checkWbDuaTab(width) {
  const expect = wbExpect("wbduatab", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbtab ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Dua tab"));
  const errors = [];
  const cs = await wbContext("staf", width);
  try {
    const t1 = await wbPage(cs, board, errors);
    const t2 = await wbPage(cs, board, errors);
    await t1.bringToFront();
    await t1.locator(`[data-card="${note.id}"]`).click({ position: { x: 6, y: 6 } });
    await t1.keyboard.press("Shift+ArrowDown");
    await t1.waitForTimeout(600);
    await wbSaved(t1);
    await t1.keyboard.press("Enter");
    await t1.getByRole("textbox", { name: "Isi catatan" }).waitFor({ timeout: 10000 });
    await t2.bringToFront();
    await t2.waitForFunction((id) => document.querySelector(`[data-card="${id}"]`)?.getAttribute("aria-label")?.includes("tab lain"), note.id, { timeout: 10000 }).catch(() => {});
    expect((await cardLabelOf(t2, note.id)).includes("tab lain"), "label", "tab kedua tidak menandai kartu diedit di tab lain");
    expect((await t2.getByRole("button", { name: "Urungkan langkah" }).isDisabled().catch(() => true)), "undo", "tab kedua bisa mengurungkan aksi tab pertama");
    await t2.locator(`[data-card="${note.id}"]`).click({ position: { x: 6, y: 6 } });
    await t2.keyboard.press("Enter");
    const prompt = t2.locator("dialog[open]", { hasText: "tab lain" });
    await prompt.waitFor({ timeout: 10000 });
    await prompt.getByRole("button", { name: "Edit di sini" }).click();
    await t2.getByRole("textbox", { name: "Isi catatan" }).waitFor({ timeout: 10000 }).catch(() => {});
    expect((await t2.getByRole("textbox", { name: "Isi catatan" }).count()) === 1, "pindah", "editor tidak pindah ke tab kedua");
    expect((await t1.getByRole("textbox", { name: "Isi catatan" }).count()) === 0, "pindah", "editor tab pertama masih terbuka");
    await audit(t2, errors, `wbduatab ${width} tab kedua`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
  }
}

// Navbar baru: semua halaman untuk ketiga peran di 768, 1024, 1279, dan 1280 px: tanpa gulir horizontal, menu atas
// tidak meluap, dan di bawah 1280 px bar bawah plus menu Lainnya memuat semua item peran itu.
async function checkNavbar() {
  const expectAll = {
    admin: ["Timeline", "Task", "Aktivitas", "Whiteboard", "Arsip", "Pengguna", "Pengaturan", "Audit Log", "Status", "Profil"],
    manajer: ["Timeline", "Task", "Aktivitas", "Whiteboard", "Arsip", "Pengguna", "Pengaturan", "Audit Log", "Profil"],
    staf: ["Timeline", "Task", "Aktivitas", "Whiteboard", "Arsip", "Profil"],
  };
  const pages = { Timeline: "/timeline", Task: "/task", Aktivitas: "/aktivitas", Whiteboard: "/whiteboard", Arsip: "/arsip", Pengguna: "/pengguna", Pengaturan: "/pengaturan", "Audit Log": "/audit-log", Profil: "/profil" };
  for (const role of ["admin", "manajer", "staf"]) {
    for (const width of [768, 1024, 1279, 1280]) {
      const where = (w) => `navbar ${role} ${width} ${w}`;
      const { ctx, page, errors } = await open(role, width, "terang");
      try {
        for (const [label, path] of Object.entries(pages)) {
          if (!expectAll[role].includes(label)) continue;
          await page.goto(BASE + path);
          await page.waitForLoadState("networkidle");
          const r = await page.evaluate(() => {
            const nav = document.querySelector('header nav[aria-label="Navigasi utama"]');
            const right = nav?.nextElementSibling;
            const navVisible = !!nav && getComputedStyle(nav).display !== "none";
            const last = nav?.lastElementChild?.getBoundingClientRect();
            return {
              sw: document.documentElement.scrollWidth,
              vw: window.innerWidth,
              navVisible,
              navOverflow: navVisible ? nav.scrollWidth > nav.clientWidth + 1 : false,
              overlap: navVisible && right && last ? last.right > right.getBoundingClientRect().left + 1 : false,
              bottom: !!document.querySelector("nav.fixed") && getComputedStyle(document.querySelector("nav.fixed")).display !== "none",
            };
          });
          checks++;
          if (r.sw > r.vw) fail(where(path), `scrollWidth ${r.sw} > ${r.vw}`);
          checks++;
          if (r.navOverflow || r.overlap) fail(where(path), "menu atas meluap atau menabrak blok kanan");
          checks++;
          if ((width >= 1280) !== r.navVisible || (width < 1280) !== r.bottom) fail(where(path), `menu atas ${r.navVisible}, bar bawah ${r.bottom}`);
        }
        // Semua item peran itu ada: di menu atas (1280) atau di bar bawah plus menu Lainnya.
        let items;
        if (width >= 1280) {
          items = (await page.locator('header nav[aria-label="Navigasi utama"] a').allTextContents()).map((t) => t.trim());
        } else {
          const bottom = (await page.locator("nav.fixed a").allTextContents()).map((t) => t.trim());
          await page.locator("nav.fixed").getByRole("button", { name: /Lainnya/ }).click();
          const more = (await page.locator("#menu-lainnya a").allTextContents()).map((t) => t.trim());
          await page.keyboard.press("Escape");
          items = [...bottom, ...more];
        }
        checks++;
        const missing = expectAll[role].filter((x) => !items.includes(x));
        if (missing.length) fail(where("menu"), `tidak ada di menu: ${missing.join(", ")} (ada: ${items.join(", ")})`);
        await audit(page, errors, where("audit"));
      } finally {
        await ctx.close();
      }
    }
  }
}

if (ONLY === "wbhadir") for (const w of [360, 1280]) await checkWbHadir(w);
if (ONLY === "wblangsung") for (const w of [360, 1280]) await checkWbLangsung(w);
if (ONLY === "wbseret") for (const w of [360, 1280]) await checkWbSeret(w);
if (ONLY === "wbhapusedit") for (const w of [360, 1280]) await checkWbHapusEdit(w);
if (ONLY === "wbputus") for (const w of [360, 1280]) await checkWbPutus(w);
if (ONLY === "wbundo") for (const w of [360, 1280]) await checkWbUndo(w);
if (ONLY === "wbduatab") for (const w of [360, 1280]) await checkWbDuaTab(w);
if (ONLY === "navbar") await checkNavbar();
if (ONLY === "wb2") {
  for (const w of [360, 1280]) {
    await checkWbHadir(w);
    await checkWbLangsung(w);
    await checkWbSeret(w);
    await checkWbHapusEdit(w);
    await checkWbPutus(w);
    await checkWbUndo(w);
    await checkWbDuaTab(w);
  }
  await checkNavbar();
}

// Whiteboard W3: berkas, tautan, kolom, garis penghubung, komentar, gambar bebas, tabel, dan pindah papan. Setiap
// skenario memakai papan uji sendiri yang dipindah ke Tempat sampah di akhir.

// PNG kecil yang sah (zlib bawaan Node), supaya backend bisa membuat thumbnail.
function testPng(w = 40, h = 20) {
  const crcTable = Array.from({ length: 256 }, (_, n) => {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    return c >>> 0;
  });
  const crc = (buf) => {
    let c = 0xffffffff;
    for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
  };
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const td = Buffer.concat([Buffer.from(type), data]);
    const c = Buffer.alloc(4);
    c.writeUInt32BE(crc(td));
    return Buffer.concat([len, td, c]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr[8] = 8;
  ihdr[9] = 2;
  const raw = Buffer.alloc((w * 3 + 1) * h, 0x80);
  for (let y = 0; y < h; y++) raw[y * (w * 3 + 1)] = 0;
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", zlib.deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

async function wbBoardView(api, board) {
  return (await api.call("GET", `/whiteboard/boards/${board}`)).body.board;
}

async function wbSelect(page, id, opts = {}) {
  // Tengah kartu: pojok kiri atas bisa tertutup bar pilihan di 360 px.
  await page.locator(`[data-card="${id}"]`).click(opts);
}

// Unggah lewat alat Berkas (pemilih berkas), tempel tangkapan layar, berkas SVG ditolak sebelum diunggah, gambar
// tampil dari thumbnail, berkas bisa diunduh sebagai lampiran.
async function checkWbBerkas(width) {
  const expect = wbExpect("wbberkas", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbberkas ${width} ${Date.now()}`);
  const errors = [];
  const cs = await wbContext("staf", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const [chooser] = await Promise.all([ps.waitForEvent("filechooser"), wbTool(ps, width, "Berkas")]);
    await chooser.setFiles([
      { name: "poster.png", mimeType: "image/png", buffer: testPng(400, 100) },
      { name: "catatan.txt", mimeType: "text/plain", buffer: Buffer.from("isi berkas uji") },
    ]);
    await ps.waitForFunction(() => document.querySelectorAll("[data-card]").length >= 2, null, { timeout: 15000 }).catch(() => {});
    const cards = await wbCards(api, board);
    const img = cards.find((c) => c.kind === "image");
    const txt = cards.find((c) => c.kind === "file");
    expect(img?.file?.thumb && txt?.file?.name === "catatan.txt", "unggah", `kartu: ${cards.map((c) => c.kind).join(", ")}`);
    if (img) {
      await ps.waitForFunction((id) => document.querySelector(`[data-card="${id}"] img`)?.naturalWidth > 0, img.id, { timeout: 10000 }).catch(() => {});
      expect(await ps.locator(`[data-card="${img.id}"] img`).evaluate((el) => el.naturalWidth > 0 && el.alt === "poster.png"), "thumbnail", "thumbnail gambar tidak tampil atau alt salah");
      expect(Math.abs(img.h - Math.round((img.w * 100) / 400)) <= 1, "rasio", `tinggi kartu gambar ${img.h} untuk lebar ${img.w}`);
    }
    if (txt) {
      const href = await ps.locator(`[data-card="${txt.id}"] a`, { hasText: "Unduh berkas" }).getAttribute("href");
      const res = await api.ctx.request.get(BASE + href);
      expect(res.status() === 200 && (res.headers()["content-disposition"] ?? "").startsWith("attachment") && (await res.text()) === "isi berkas uji", "unduh", `unduh ${res.status()} ${res.headers()["content-disposition"]}`);
    }

    // Tempel tangkapan layar di kanvas.
    await ps.locator(".wb-canvas").click({ position: { x: 20, y: 200 } });
    await ps.evaluate((b64) => {
      const bytes = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
      const dt = new DataTransfer();
      dt.items.add(new File([bytes], "image.png", { type: "image/png" }));
      document.dispatchEvent(new ClipboardEvent("paste", { clipboardData: dt, bubbles: true }));
    }, testPng().toString("base64"));
    await ps.waitForFunction(() => document.querySelectorAll("[data-card]").length >= 3, null, { timeout: 15000 }).catch(() => {});
    const after = await wbCards(api, board);
    expect(after.some((c) => c.kind === "image" && c.file?.name?.startsWith("tempel-")), "tempel", `tempel: ${after.map((c) => c.file?.name).join(", ")}`);

    // SVG ditolak di peramban (dan di server, lihat test Go).
    const [ch2] = await Promise.all([ps.waitForEvent("filechooser"), wbTool(ps, width, "Berkas")]);
    await ch2.setFiles([{ name: "logo.svg", mimeType: "image/svg+xml", buffer: Buffer.from("<svg xmlns='http://www.w3.org/2000/svg'/>") }]);
    await ps.getByText(/logo\.svg tidak bisa diunggah/).first().waitFor({ timeout: 5000 }).catch(() => {});
    expect((await ps.getByText(/logo\.svg tidak bisa diunggah/).count()) > 0, "svg", "SVG tidak ditolak dengan pesan");
    const svg = await api.ctx.request.post(BASE + `/api/whiteboard/boards/${board}/files?x=0&y=0&w=240&h=160`, {
      headers: { Origin: BASE },
      multipart: { file: { name: "logo.png", mimeType: "image/png", buffer: Buffer.from("<svg xmlns='http://www.w3.org/2000/svg'><script>alert(1)</script></svg>") } },
    });
    expect(svg.status() === 400, "svg server", `SVG berkedok PNG diterima server: ${svg.status()}`);
    expect(await wbNoScroll(ps), "gulir", "halaman bisa digulir");
    await audit(ps, errors, `wbberkas ${width} papan`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
  }
}

// Tautan ke server lokal: kartu tetap dibuat, pratinjau diblokir karena alamatnya jaringan internal, dan server
// lokal tidak pernah dihubungi backend. Tautan dibuka di tab baru tanpa opener.
async function checkWbTautan(width) {
  const expect = wbExpect("wbtautan", width);
  let hits = 0;
  const srv = http.createServer((_, res) => {
    hits++;
    res.setHeader("Content-Type", "text/html");
    res.end('<title>Rahasia internal</title><meta property="og:image" content="/x.png">');
  });
  await new Promise((r) => srv.listen(0, "127.0.0.1", r));
  const url = `http://127.0.0.1:${srv.address().port}/halaman`;
  const { api, card, board } = await wbSetup("staf", `E2E wbtautan ${width} ${Date.now()}`);
  const errors = [];
  const cs = await wbContext("staf", width);
  try {
    const ps = await wbPage(cs, board, errors);
    await wbTool(ps, width, "Tautan");
    const dlg = ps.locator("dialog[open]");
    await dlg.getByLabel(/Alamat/).fill(url);
    await dlg.getByRole("button", { name: "Tambah tautan" }).click();
    await ps.getByText(/tanpa pratinjau/).first().waitFor({ timeout: 15000 }).catch(() => {});
    const link = (await wbCards(api, board)).find((c) => c.kind === "link");
    expect(link?.preview?.status === "blocked", "ssrf", `status pratinjau ${link?.preview?.status}`);
    expect((await ps.getByText(/jaringan internal/).count()) > 0, "pesan", "alasan pratinjau tidak ditampilkan");
    expect(hits === 0, "ssrf", `server lokal dihubungi ${hits} kali`);
    if (link) {
      const a = ps.locator(`[data-card="${link.id}"] a`, { hasText: "Buka tautan" });
      expect((await a.getAttribute("href")) === url && (await a.getAttribute("rel"))?.includes("noopener") && (await a.getAttribute("target")) === "_blank", "buka", "atribut tautan salah");
      expect((await ps.locator(`[data-card="${link.id}"]`).textContent()).includes("Pratinjau tidak tersedia"), "kartu", "kartu tidak menyebut pratinjau tidak tersedia");
    }
    // Skema selain http dan https ditolak dengan pesan di dialog.
    await wbTool(ps, width, "Tautan");
    await dlg.getByLabel(/Alamat/).fill("javascript:alert(1)");
    await dlg.getByRole("button", { name: "Tambah tautan" }).click();
    await dlg.getByRole("alert").waitFor({ timeout: 10000 }).catch(() => {});
    expect((await dlg.getByRole("alert").count()) > 0, "skema", "alamat javascript: tidak ditolak di dialog");
    await audit(ps, errors, `wbtautan ${width} dialog`);
    await ps.keyboard.press("Escape");
    await audit(ps, errors, `wbtautan ${width} papan`);
  } finally {
    srv.close();
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
  }
}

async function dragTo(page, fromSel, to) {
  const a = await page.locator(fromSel).boundingBox();
  await page.mouse.move(a.x + 8, a.y + 8);
  await page.mouse.down();
  await page.mouse.move(a.x + 30, a.y + 30, { steps: 4 });
  await page.mouse.move(to.x, to.y, { steps: 8 });
  await page.mouse.up();
}

// Kolom: seret kartu masuk kolom (tersusun di bawah judul), seret keluar lagi, dan hapus kolom ikut menghapus isinya.
async function checkWbKolom(width) {
  const expect = wbExpect("wbkolom", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbkolom ${width} ${Date.now()}`);
  const col = await wbCard(api, board, "column", 0, 0, { title: "Ide" });
  const a = await wbCard(api, board, "note", 320, 0, wbNote("Masuk kolom"));
  const errors = [];
  const cs = await wbContext("staf", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const cb = await ps.locator(`[data-card="${col.id}"]`).boundingBox();
    await dragTo(ps, `[data-card="${a.id}"]`, { x: cb.x + cb.width / 2, y: cb.y + cb.height - 10 });
    await wbSaved(ps);
    let saved = (await wbCards(api, board)).find((c) => c.id === a.id);
    expect(saved?.column_id === col.id, "masuk", `column_id ${saved?.column_id}`);
    const nb = await ps.locator(`[data-card="${a.id}"]`).boundingBox();
    const cb2 = await ps.locator(`[data-card="${col.id}"]`).boundingBox();
    expect(nb && nb.y > cb2.y + 20 && nb.x >= cb2.x && nb.x + nb.width <= cb2.x + cb2.width + 1, "tumpuk", "kartu tidak tersusun di dalam kolom");
    expect((await cardLabelOf(ps, col.id)).startsWith("Kolom Ide"), "label", await cardLabelOf(ps, col.id));

    await dragTo(ps, `[data-card="${a.id}"]`, { x: cb2.x + cb2.width + 120, y: cb2.y + 40 });
    await wbSaved(ps);
    saved = (await wbCards(api, board)).find((c) => c.id === a.id);
    expect(!saved?.column_id, "keluar", `masih di kolom ${saved?.column_id}`);

    // Masukkan lagi lewat API bersama kartu kedua; Shift+Panah atas menaikkan kartu kedua di kolom.
    const b2 = await wbCard(api, board, "note", 320, 300, wbNote("Kedua"));
    await api.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: a.id, column_id: col.id, col_order: 1 }, { id: b2.id, column_id: col.id, col_order: 2 }] });
    await ps.reload();
    await ps.waitForLoadState("networkidle");
    await wbSelect(ps, b2.id);
    await ps.keyboard.press("Shift+ArrowUp");
    await wbSaved(ps);
    const ord = await wbCards(api, board);
    const oa = ord.find((c) => c.id === a.id)?.col_order;
    const ob = ord.find((c) => c.id === b2.id)?.col_order;
    expect(ob < oa, "keyboard", `urutan setelah Shift+ArrowUp: a ${oa}, kedua ${ob}`);
    // Hapus kolom: isinya ikut ke Tempat sampah dan kembali saat Urungkan.
    // Kolom dipilih lewat area judulnya; tengah kolom berisi kartu isinya.
    const colBox = await ps.locator(`[data-card="${col.id}"]`).boundingBox();
    await ps.locator(`[data-card="${col.id}"]`).click({ position: { x: colBox.width - 16, y: 16 } });
    await ps.keyboard.press("Delete");
    await ps.getByRole("button", { name: "Urungkan", exact: true }).waitFor({ timeout: 10000 }).catch(() => {});
    expect((await wbCards(api, board)).length === 0, "hapus", "isi kolom tidak ikut terhapus");
    await ps.getByRole("button", { name: "Urungkan", exact: true }).click();
    await wbSaved(ps);
    await ps.waitForTimeout(500);
    const back = await wbCards(api, board);
    expect(back.length === 3 && back.find((c) => c.id === a.id)?.column_id === col.id, "pulih", `setelah urungkan: ${back.length} kartu`);
    expect(await wbNoScroll(ps), "gulir", "halaman bisa digulir");
    await audit(ps, errors, `wbkolom ${width} papan`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
  }
}

// Garis penghubung lewat keyboard dan bar pilihan: pilih dua kartu, Hubungkan, ubah label dan ujung di dialog,
// orang lain melihat garisnya langsung, lalu hapus.
async function checkWbPenghubung(width) {
  const expect = wbExpect("wbpenghubung", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbgaris ${width} ${Date.now()}`);
  const a = await wbCard(api, board, "note", 0, 0, wbNote("Awal"));
  const b = await wbCard(api, board, "note", 0, 260, wbNote("Akhir"));
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    await wbSelect(ps, a.id);
    await wbSelect(ps, b.id, { modifiers: ["Shift"] });
    await ps.getByLabel("Aksi lain untuk kartu terpilih").selectOption("connect");
    const dlg = ps.locator("dialog[open]");
    await dlg.getByLabel("Label").waitFor({ timeout: 10000 });
    await dlg.getByLabel("Label").fill("lalu");
    await dlg.getByLabel("Label").press("Tab");
    await dlg.getByLabel("Garis", { exact: true }).selectOption("dashed");
    await wbSaved(ps);
    await audit(ps, errors, `wbpenghubung ${width} dialog`);
    await dlg.getByRole("button", { name: "Tutup" }).click();
    const view = await wbBoardView(api, board);
    const conn = view.connectors[0];
    expect(view.connectors.length === 1 && conn.label === "lalu" && conn.style.line === "dashed" && conn.from === a.id && conn.to === b.id, "simpan", JSON.stringify(view.connectors));
    await pm.waitForFunction(() => document.querySelectorAll("[data-testid=wb-connector]").length === 1, null, { timeout: 10000 }).catch(() => {});
    expect((await pm.locator("[data-testid=wb-connector]").count()) === 1, "langsung", "garis tidak muncul di layar Manajer");
    expect((await pm.locator("[data-testid=wb-connector] text").textContent())?.trim() === "lalu", "label", "label garis tidak tampil");

    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
    await wbSelect(ps, a.id);
    await ps.getByLabel("Aksi lain untuk kartu terpilih").selectOption("lines");
    await dlg.getByRole("button", { name: "Hapus garis" }).click();
    await wbSaved(ps);
    await pm.waitForFunction(() => document.querySelectorAll("[data-testid=wb-connector]").length === 0, null, { timeout: 10000 }).catch(() => {});
    expect((await pm.locator("[data-testid=wb-connector]").count()) === 0, "hapus", "garis masih tampil di layar Manajer");
    expect((await wbBoardView(api, board)).connectors.length === 0, "hapus api", "garis masih ada");
    await audit(ps, errors, `wbpenghubung ${width} papan`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
    await cm.close();
  }
}

// Komentar: Staf menulis, lencana jumlah muncul di layar Manajer, Manajer menghapus komentar Staf, Admin hanya
// membaca (tanpa form).
async function checkWbKomentar(width) {
  const expect = wbExpect("wbkomentar", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbkomen ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Bahas ini"));
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  const ca = await wbContext("admin", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    await wbSelect(ps, note.id);
    await ps.getByLabel("Aksi lain untuk kartu terpilih").selectOption("comments");
    const panel = ps.locator("dialog[open]");
    await panel.getByLabel("Komentar baru").fill("Ukuran gambarnya cek lagi");
    await panel.getByRole("button", { name: "Kirim komentar" }).click();
    await panel.getByText("Ukuran gambarnya cek lagi").waitFor({ timeout: 10000 }).catch(() => {});
    expect((await panel.getByText("Ukuran gambarnya cek lagi").count()) === 1, "kirim", "komentar tidak tampil di panel");
    await audit(ps, errors, `wbkomentar ${width} panel`);
    await pm.locator("[data-testid=wb-comment-badge]").waitFor({ timeout: 10000 }).catch(() => {});
    expect((await pm.locator("[data-testid=wb-comment-badge]").textContent())?.trim() === "1", "lencana", "lencana komentar tidak muncul di layar Manajer");

    const pa = await wbPage(ca, board, errors);
    await pa.locator("[data-testid=wb-comment-badge]").click();
    const pPanel = pa.locator("dialog[open]");
    await pPanel.getByText("Ukuran gambarnya cek lagi").waitFor({ timeout: 10000 }).catch(() => {});
    expect((await pPanel.getByText("Ukuran gambarnya cek lagi").count()) === 1 && (await pPanel.getByLabel("Komentar baru").count()) === 0, "admin", "Admin tidak bisa membaca atau masih melihat form");
    await audit(pa, errors, `wbkomentar ${width} admin`);

    await pm.locator("[data-testid=wb-comment-badge]").click();
    const mPanel = pm.locator("dialog[open]");
    expect((await mPanel.getByRole("button", { name: /^Ubah/ }).count()) === 0, "ubah", "Manajer bisa mengubah komentar orang lain");
    await mPanel.getByRole("button", { name: /^Hapus/ }).click();
    await mPanel.getByText("Belum ada komentar.").waitFor({ timeout: 10000 }).catch(() => {});
    await ps.keyboard.press("Escape");
    await ps.waitForFunction(() => !document.querySelector("[data-testid=wb-comment-badge]"), null, { timeout: 10000 }).catch(() => {});
    expect((await ps.locator("[data-testid=wb-comment-badge]").count()) === 0, "hapus", "lencana masih tampil setelah Manajer menghapus");
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
    await cm.close();
    await ca.close();
  }
}

// Gambar bebas tanpa kartu (revisi A): alat Gambar menyalakan mode pena dan toolbar melayang; coretan di atas kartu
// lain tersimpan sebagai objek tanpa bingkai, Manajer melihat goresan saat masih digambar, dua goresan satu sesi
// menjadi satu objek, penghapus Manajer pada objek yang dikunci ditolak dengan nama pemegang, Escape keluar,
// klik di luar garis tembus ke kartu di bawahnya, dan objek bisa difokus lalu dihapus dengan Delete.
async function checkWbGambar(width) {
  const expect = wbExpect("wbgambar", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbgambar ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Di bawah coretan"));
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  const mgr = await apiAs("manajer");
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    await wbTool(ps, width, "Gambar");
    const bar = ps.locator("[data-testid=wb-penbar]");
    await bar.waitFor({ timeout: 10000 });
    expect((await ps.locator('[aria-live="polite"]').textContent()).includes("Mode pena aktif"), "umumkan", "mode pena tidak diumumkan");
    await bar.getByRole("button", { name: "Pena biru" }).click();
    await audit(ps, errors, `wbgambar ${width} pena`);
    const nb = await ps.locator(`[data-card="${note.id}"]`).boundingBox();
    // Goresan mendatar melintasi kartu catatan.
    const y0 = nb.y + nb.height / 2;
    await ps.mouse.move(nb.x + 10, y0);
    await ps.mouse.down();
    for (let i = 1; i <= 12; i++) await ps.mouse.move(nb.x + 10 + i * 12, y0 + (i % 2) * 4);
    await pm.waitForFunction(() => document.querySelectorAll("[data-testid=wb-ink-live] path").length > 0, null, { timeout: 5000 }).catch(() => {});
    expect((await pm.locator("[data-testid=wb-ink-live] path").count()) > 0, "langsung", "goresan tidak terlihat Manajer saat masih digambar");
    await ps.mouse.up();
    await wbSaved(ps);
    await ps.mouse.move(nb.x + 10, y0 + 30);
    await ps.mouse.down();
    await ps.mouse.move(nb.x + 60, y0 + 40, { steps: 5 });
    await ps.mouse.up();
    await wbSaved(ps);
    await ps.waitForTimeout(400);
    let ink = (await wbCards(api, board)).filter((c) => c.kind === "drawing");
    const d = ink[0];
    expect(ink.length === 1 && d.content.strokes.length === 2 && d.content.strokes[0].color === "biru", "objek", `objek ${ink.length}, goresan ${d?.content?.strokes?.length}`);
    expect(d && d.content.strokes.every((s) => s.points.every((p) => Number.isInteger(p) && p >= 0 && p <= 4000)), "titik", "titik relatif tidak sah");
    const look = await ps.locator(`[data-card="${d.id}"]`).evaluate((el) => {
      const s = getComputedStyle(el);
      return { bg: s.backgroundColor, border: s.borderTopWidth, z: Number(s.zIndex) };
    });
    expect(look.bg === "rgba(0, 0, 0, 0)" && look.border === "0px", "tanpa bingkai", JSON.stringify(look));
    const cardZ = await ps.locator(`[data-card="${note.id}"]`).evaluate((el) => Number(getComputedStyle(el).zIndex));
    expect(look.z > cardZ, "lapisan", `coretan z ${look.z}, kartu z ${cardZ}`);
    expect((await cardLabelOf(pm, d.id)).includes("sedang mengedit"), "kunci", "objek sesi pena tidak tampil terkunci");
    const er = await mgr.call("POST", `/whiteboard/boards/${board}/cards/${d.id}/drawing`, { op: "remove", index: 0 });
    expect(er.status === 409 && /sedang mengubah/.test(er.body?.error?.message ?? ""), "penghapus", `penghapus Manajer ${er.status}`);

    await ps.keyboard.press("Escape");
    await bar.waitFor({ state: "detached", timeout: 5000 }).catch(() => {});
    expect((await bar.count()) === 0, "keluar", "toolbar pena masih tampil setelah Escape");
    // Klik di kartu jauh dari garis memilih kartu itu, bukan coretannya.
    await ps.locator(`[data-card="${note.id}"]`).click({ position: { x: nb.width - 12, y: 12 } });
    expect((await cardLabelOf(ps, note.id)).includes("terpilih"), "tembus", "klik di luar garis tidak sampai ke kartu");
    await ps.mouse.click(nb.x + 70, y0 + 2);
    expect((await cardLabelOf(ps, d.id)).startsWith("Gambar bebas oleh"), "label", await cardLabelOf(ps, d.id));
    expect((await cardLabelOf(ps, d.id)).includes("terpilih"), "pilih", "klik di garis tidak memilih coretan");
    await audit(ps, errors, `wbgambar ${width} terpilih`);
    await ps.locator(`[data-card="${d.id}"]`).focus();
    await ps.keyboard.press("Delete");
    await wbSaved(ps);
    ink = (await wbCards(api, board)).filter((c) => c.kind === "drawing");
    expect(ink.length === 0, "hapus", "objek gambar tidak terhapus dengan Delete");
    const bad = await api.call("POST", `/whiteboard/boards/${board}/drawings`, { stroke: { color: "tinta", width: 3, points: [0, 0] } });
    expect(bad.status === 400, "validasi", `goresan tebal 3 diterima: ${bad.status}`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await mgr.ctx.close();
    await cs.close();
    await cm.close();
  }
}

// Tabel: alat Tabel membuka editor sel; Enter pindah ke baris berikut; tambah baris; Manajer melihat isinya dan
// ditolak 409 saat mengubah tabel yang dikunci Staf.
async function checkWbTabel(width) {
  const expect = wbExpect("wbtabel", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbtabel ${width} ${Date.now()}`);
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  const mgr = await apiAs("manajer");
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    await wbTool(ps, width, "Tabel");
    const title = ps.getByLabel("Judul tabel");
    await title.waitFor({ timeout: 10000 });
    await title.fill("Harga paket");
    await title.press("Enter");
    const cell = ps.getByLabel("Baris 1, kolom 1");
    await cell.waitFor({ timeout: 10000 });
    await cell.fill("Harga");
    await cell.press("Enter");
    expect(await ps.getByLabel("Baris 2, kolom 1").evaluate((el) => el === document.activeElement), "enter", "Enter tidak pindah ke baris berikut");
    await ps.getByRole("button", { name: "Tambah baris" }).click();
    await ps.getByLabel("Baris 3, kolom 1").waitFor({ timeout: 10000 }).catch(() => {});
    await wbSaved(ps);
    let t = (await wbCards(api, board)).find((c) => c.kind === "table");
    expect(t?.content?.rows?.length === 3 && t.content.rows[0][0] === "Harga" && t.content.title === "Harga paket", "simpan", JSON.stringify(t?.content));
    await pm.waitForFunction((id) => document.querySelector(`[data-card="${id}"]`)?.textContent?.includes("Harga"), t?.id, { timeout: 10000 }).catch(() => {});
    expect((await pm.locator(`[data-card="${t.id}"]`).textContent()).includes("Harga"), "langsung", "isi tabel tidak tampil di layar Manajer");
    const op = await mgr.call("POST", `/whiteboard/boards/${board}/cards/${t.id}/table`, { op: "set", row: 0, col: 1, text: "x" });
    expect(op.status === 409 && /sedang mengubah/.test(op.body?.error?.message ?? ""), "kunci", `op Manajer saat dikunci ${op.status}`);
    await audit(ps, errors, `wbtabel ${width} edit`);
    await ps.keyboard.press("Escape");
    await wbSaved(ps);
    const big = await mgr.call("POST", `/whiteboard/boards/${board}/cards/patch`, {
      cards: [{ id: t.id, content: { header: true, rows: [Array(21).fill("")] }, base_content_version: t.content_version + 5 }],
    });
    expect(big.status === 400 || big.status === 409, "validasi", `tabel 21 kolom ${big.status}`);
    await audit(ps, errors, `wbtabel ${width} papan`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await mgr.ctx.close();
    await cs.close();
    await cm.close();
  }
}

// Pindah papan: lewat dialog Pindahkan ke papan, dan dengan melepas kartu di atas kartu papan.
async function checkWbPindah(width) {
  const expect = wbExpect("wbpindah", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbpindah ${width} ${Date.now()}`);
  const child = (await api.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "board", title: "Tujuan", x: 0, y: 0, w: 200, h: 120 })).body.card;
  const a = await wbCard(api, board, "note", 320, 0, wbNote("Pindah lewat menu"));
  const b = await wbCard(api, board, "note", 320, 200, wbNote("Pindah lewat seret"));
  const errors = [];
  const cs = await wbContext("staf", width);
  try {
    const ps = await wbPage(cs, board, errors);
    await wbSelect(ps, a.id);
    await ps.getByLabel("Aksi lain untuk kartu terpilih").selectOption("move");
    const dlg = ps.locator("dialog[open]");
    await dlg.getByRole("radio", { name: "Tujuan", exact: true }).check();
    await audit(ps, errors, `wbpindah ${width} dialog`);
    await dlg.getByRole("button", { name: "Pindahkan", exact: true }).click();
    await ps.getByRole("button", { name: "Buka papan" }).waitFor({ timeout: 10000 }).catch(() => {});
    const cb = await ps.locator(`[data-card="${child.id}"]`).boundingBox();
    await dragTo(ps, `[data-card="${b.id}"]`, { x: cb.x + cb.width / 2, y: cb.y + cb.height / 2 });
    await ps.waitForTimeout(800);
    const inChild = await wbCards(api, child.ref.id);
    expect(inChild.length === 2, "pindah", `isi papan Tujuan ${inChild.length}`);
    expect((await wbCards(api, board)).length === 1, "asal", "kartu masih di papan asal");
    await audit(ps, errors, `wbpindah ${width} papan`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
  }
}

// Zoom roda: Ctrl+wheel memperbesar dan memperkecil di sekitar kursor (titik di bawah kursor tetap), wheel biasa
// dan Shift+wheel menggeser kanvas, halaman tidak tergulir, zoom browser tidak berubah, klik persentase ke 100%.
async function checkWbZoom(width) {
  const expect = wbExpect("wbzoom", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbzoom ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Titik acuan"));
  const errors = [];
  const cs = await wbContext("staf", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const el = ps.locator(`[data-card="${note.id}"]`);
    const zoomText = async () => Number((await ps.locator("[data-testid=wb-zoom]").textContent()).replace(/\D/g, ""));
    const env = () => ps.evaluate(() => ({ sy: window.scrollY, sx: window.scrollX, scale: window.visualViewport?.scale ?? 1, dpr: window.devicePixelRatio }));
    const before = await env();
    const b0 = await el.boundingBox();
    // Kursor di dalam kartu, sedikit dari pojok: setelah zoom, titik dunia di bawah kursor tetap di tempatnya.
    const cx = b0.x + 30;
    const cy = b0.y + 20;
    const z0 = await zoomText();
    await ps.mouse.move(cx, cy);
    await ps.keyboard.down("Control");
    await ps.mouse.wheel(0, -100);
    await ps.keyboard.up("Control");
    await ps.waitForTimeout(150);
    const z1 = await zoomText();
    const b1 = await el.boundingBox();
    expect(z1 > z0 && z1 <= z0 * 1.35, "ctrl wheel", `zoom ${z0} -> ${z1}`);
    const fx = (cx - b0.x) / b0.width;
    const fy = (cy - b0.y) / b0.height;
    const drift = Math.hypot(b1.x + fx * b1.width - cx, b1.y + fy * b1.height - cy);
    expect(drift < 2, "kursor tetap", `titik bergeser ${drift.toFixed(1)} px`);
    await ps.keyboard.down("Control");
    for (let i = 0; i < 40; i++) await ps.mouse.wheel(0, 120);
    await ps.keyboard.up("Control");
    await ps.waitForTimeout(150);
    expect((await zoomText()) === 10, "batas", `zoom minimum ${await zoomText()}`);
    await ps.locator("[data-testid=wb-zoom]").click();
    expect((await zoomText()) === 100, "100%", `klik persentase ${await zoomText()}`);
    await ps.getByRole("button", { name: "Pas ke layar" }).click();
    await ps.waitForTimeout(150);

    const p0 = await el.boundingBox();
    await ps.mouse.move(cx, cy);
    await ps.mouse.wheel(0, 120);
    await ps.waitForTimeout(150);
    const p1 = await el.boundingBox();
    expect(Math.abs(p1.y - (p0.y - 120)) < 2 && Math.abs(p1.x - p0.x) < 1, "geser", `wheel biasa ${p0.y} -> ${p1.y}`);
    await ps.keyboard.down("Shift");
    await ps.mouse.wheel(0, 80);
    await ps.keyboard.up("Shift");
    await ps.waitForTimeout(150);
    const p2 = await el.boundingBox();
    expect(Math.abs(p2.x - (p1.x - 80)) < 2 && Math.abs(p2.y - p1.y) < 1, "shift", `Shift+wheel ${p1.x} -> ${p2.x}`);
    const after = await env();
    expect(after.sy === 0 && after.sx === 0 && after.scale === before.scale && after.dpr === before.dpr, "halaman", JSON.stringify({ before, after }));
    expect(await wbNoScroll(ps), "gulir", "halaman bisa digulir");
    await audit(ps, errors, `wbzoom ${width} papan`);
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
  }
}

// Dobel klik untuk mengedit: kursor di posisi klik untuk catatan, judul, butir checklist, dan sel tabel; klik
// sekali hanya memilih; centang tetap sekali klik; dobel klik area kosong tidak membuat kartu; kartu yang dikunci
// orang lain terbuka hanya baca dengan nama pemegang.
async function checkWbDobel(width) {
  const expect = wbExpect("wbdobel", width);
  const { api, card, board } = await wbSetup("staf", `E2E wbdobel ${width} ${Date.now()}`);
  const note = await wbCard(api, board, "note", 0, 0, wbNote("Halo dunia"));
  const head = await wbCard(api, board, "heading", 0, 160, { text: "Rencana besar", level: 1 }, { w: 320, h: 64 });
  const list = await wbCard(api, board, "checklist", 0, 260, { title: "Tugas", items: [{ id: crypto.randomUUID(), text: "Beli kertas", done: false }] });
  const tb = await wbCard(api, board, "table", 0, 420, { title: "", header: false, rows: [["Harga", "Isi"]] }, { w: 360, h: 120 });
  const errors = [];
  const cs = await wbContext("staf", width);
  const cm = await wbContext("manajer", width);
  try {
    const ps = await wbPage(cs, board, errors);
    const pm = await wbPage(cm, board, errors);
    // Titik tengah sebuah kata di teks tampilan.
    const wordAt = (sel, word) =>
      ps.evaluate(
        ([sel, word]) => {
          const el = document.querySelector(sel);
          const walk = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
          for (let n = walk.nextNode(); n; n = walk.nextNode()) {
            const i = n.textContent.indexOf(word);
            if (i < 0) continue;
            const r = document.createRange();
            r.setStart(n, i);
            r.setEnd(n, i + 1);
            const b = r.getBoundingClientRect();
            return { x: b.left + 1, y: b.top + b.height / 2 };
          }
          return null;
        },
        [sel, word],
      );
    const nCount = async () => (await wbCards(api, board)).length;
    const before = await nCount();

    // Klik sekali hanya memilih, tanpa editor.
    await ps.locator(`[data-card="${note.id}"]`).click();
    expect((await ps.getByRole("textbox", { name: "Isi catatan" }).count()) === 0, "klik sekali", "klik sekali membuka editor");

    // Catatan: dobel klik di depan "dunia", ketik, teks masuk di posisi itu.
    let pt = await wordAt(`[data-card="${note.id}"]`, "dunia");
    await ps.mouse.dblclick(pt.x, pt.y);
    await ps.getByRole("textbox", { name: "Isi catatan" }).waitFor({ timeout: 10000 });
    // Fokus TipTap diterapkan di frame berikutnya.
    await ps.waitForFunction(() => document.activeElement?.classList.contains("ProseMirror-focused"), null, { timeout: 5000 }).catch(() => {});
    await ps.keyboard.type("X");
    await ps.keyboard.press("Escape");
    await wbSaved(ps);
    let saved = await wbCards(api, board);
    expect(JSON.stringify(saved.find((c) => c.id === note.id).content).includes("Halo Xdunia"), "catatan", JSON.stringify(saved.find((c) => c.id === note.id).content).slice(0, 160));

    // Judul: dobel klik di depan "besar".
    pt = await wordAt(`[data-card="${head.id}"]`, "besar");
    await ps.mouse.dblclick(pt.x, pt.y);
    await ps.getByLabel("Teks judul").waitFor({ timeout: 10000 });
    await ps.keyboard.type("Y");
    await ps.keyboard.press("Enter");
    await wbSaved(ps);
    saved = await wbCards(api, board);
    expect(saved.find((c) => c.id === head.id).content.text === "Rencana Ybesar", "judul", saved.find((c) => c.id === head.id).content.text);

    // Butir checklist: dobel klik di depan "kertas"; centang tetap sekali klik tanpa membuka editor.
    pt = await wordAt(`[data-card="${list.id}"] [data-item-text]`, "kertas");
    await ps.mouse.dblclick(pt.x, pt.y);
    await ps.getByLabel("Teks butir 1").waitFor({ timeout: 10000 });
    expect(await ps.getByLabel("Teks butir 1").evaluate((el) => el === document.activeElement && el.selectionStart === 5), "butir", "kursor butir tidak di posisi klik");
    await ps.keyboard.type("Z");
    await ps.keyboard.press("Escape");
    await wbSaved(ps);
    await ps.getByRole("checkbox", { name: /Selesai: Beli/ }).click();
    await wbSaved(ps);
    saved = await wbCards(api, board);
    const item = saved.find((c) => c.id === list.id).content.items[0];
    expect(item.text === "Beli Zkertas" && item.done, "checklist", JSON.stringify(item));
    expect((await ps.getByLabel("Teks butir 1").count()) === 0, "centang", "centang membuka editor");

    // Sel tabel: dobel klik di depan "Isi".
    pt = await wordAt(`[data-card="${tb.id}"] [data-cell-text="0,1"]`, "Isi");
    await ps.mouse.dblclick(pt.x, pt.y);
    await ps.getByLabel("Baris 1, kolom 2").waitFor({ timeout: 10000 });
    expect(await ps.getByLabel("Baris 1, kolom 2").evaluate((el) => el === document.activeElement && el.selectionStart === 0), "sel", "kursor sel tidak di posisi klik");
    await ps.keyboard.type("W");
    await ps.keyboard.press("Escape");
    await wbSaved(ps);
    saved = await wbCards(api, board);
    expect(saved.find((c) => c.id === tb.id).content.rows[0][1] === "WIsi", "tabel", JSON.stringify(saved.find((c) => c.id === tb.id).content.rows));

    // Dobel klik area kosong kanvas tidak membuat kartu.
    const cb = await ps.locator(".wb-canvas").boundingBox();
    await ps.mouse.dblclick(cb.x + cb.width - 20, cb.y + cb.height / 2);
    await ps.waitForTimeout(400);
    expect((await nCount()) === before, "kosong", "dobel klik area kosong membuat kartu");

    // Manajer mengedit catatan; dobel klik Staf membuka tampilan hanya baca dengan nama pemegang.
    await pm.locator(`[data-card="${note.id}"]`).click();
    await pm.keyboard.press("Enter");
    await pm.getByRole("textbox", { name: "Isi catatan" }).waitFor({ timeout: 10000 });
    await ps.waitForFunction((id) => document.querySelector(`[data-card="${id}"]`)?.getAttribute("aria-label")?.includes("sedang mengedit"), note.id, { timeout: 10000 }).catch(() => {});
    pt = await wordAt(`[data-card="${note.id}"]`, "Halo");
    await ps.mouse.dblclick(pt.x, pt.y);
    await ps.locator("[data-testid=wb-readonly]").waitFor({ timeout: 5000 }).catch(() => {});
    expect(/Hanya baca: \S.* sedang mengedit/.test((await ps.locator("[data-testid=wb-readonly]").textContent()) ?? ""), "hanya baca", "tampilan hanya baca tanpa nama pemegang");
    expect((await ps.getByRole("textbox", { name: "Isi catatan" }).count()) === 0, "kunci", "editor terbuka di kartu yang dikunci");
    await audit(ps, errors, `wbdobel ${width} papan`);
    await pm.keyboard.press("Escape");
  } finally {
    await wbCleanup(api, card);
    await api.ctx.close();
    await cs.close();
    await cm.close();
  }
}

if (ONLY === "wbdobel") for (const w of [360, 1280]) await checkWbDobel(w);
if (ONLY === "wbzoom") for (const w of [360, 1280]) await checkWbZoom(w);
if (ONLY === "wbberkas") for (const w of [360, 1280]) await checkWbBerkas(w);
if (ONLY === "wbtautan") for (const w of [360, 1280]) await checkWbTautan(w);
if (ONLY === "wbkolom") for (const w of [360, 1280]) await checkWbKolom(w);
if (ONLY === "wbpenghubung") for (const w of [360, 1280]) await checkWbPenghubung(w);
if (ONLY === "wbkomentar") for (const w of [360, 1280]) await checkWbKomentar(w);
if (ONLY === "wbgambar") for (const w of [360, 1280]) await checkWbGambar(w);
if (ONLY === "wbtabel") for (const w of [360, 1280]) await checkWbTabel(w);
if (ONLY === "wbpindah") for (const w of [360, 1280]) await checkWbPindah(w);
if (ONLY === "wb3")
  for (const w of [360, 1280]) {
    await checkWbBerkas(w);
    await checkWbTautan(w);
    await checkWbKolom(w);
    await checkWbPenghubung(w);
    await checkWbKomentar(w);
    await checkWbGambar(w);
    await checkWbTabel(w);
    await checkWbPindah(w);
    await checkWbZoom(w);
    await checkWbDobel(w);
  }

await browser.close();
console.log(
  `pemeriksaan: ${checks}, gagal: ${fails.length}, panel tanpa bar aksi (hanya baca): ${info.length}, login sungguhan: ${realLogins}`,
);
if (info.length)
  console.log(
    "  tanpa bar: " +
      [...new Set(info.map((i) => i.split(" ").slice(1, 3).join(" ")))].join(
        ", ",
      ),
  );
for (const f of fails) console.log("FAIL " + f);
process.exitCode = fails.length ? 1 : 0;
