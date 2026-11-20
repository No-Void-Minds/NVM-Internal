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
// kategori di hari ini lalu menghapusnya, Staf tanpa tombol tulis dan tanpa alasan kehadiran Manajer).
// E2E_ONLY=chain (revisi 33: rantai sambung jam di kartu Aktivitas hari ini milik Staf, termasuk melintasi istirahat,
// 360 dan 1280 px; menulis lalu mengosongkan jam 11.00-12.00 sampai 16.00-17.00 hari ini).
// Keluar dengan kode 1 bila ada yang gagal.
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

// Login sekali per peran lewat API (limiter login 10 percobaan per IP), lalu sesinya dipakai ulang di setiap konteks.
const sessions = {};
async function login(role) {
  const ctx = await browser.newContext();
  const { user, pass } = ROLES[role];
  const res = await ctx.request.post(BASE + "/api/auth/login", {
    headers: { Origin: BASE },
    data: { username: user, password: pass },
  });
  if (res.status() !== 200) {
    console.error(`Login ${role} gagal: HTTP ${res.status()}`);
    process.exit(2);
  }
  sessions[role] = await ctx.storageState();
  await ctx.close();
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
  const ctx = await browser.newContext({ storageState: sessions[role] });
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
      expect(
        (await pop
          .getByText(`${nama}: Tidak hadir`, { exact: true })
          .count()) === 1 &&
          (await pop.getByText(`${nama}: Sakit`).count()) === 0 &&
          (await pop.getByText("E2E Timeline rahasia").count()) === 0,
        "privasi",
        "Staf melihat alasan atau keterangan kehadiran Manajer",
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
    expect(
      (await page.getByText(kepala).filter({ visible: true }).count()) >= 2,
      "pasang",
      "isi jam kepala tidak tampil di jam tersambung",
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

await browser.close();
console.log(
  `pemeriksaan: ${checks}, gagal: ${fails.length}, panel tanpa bar aksi (hanya baca): ${info.length}`,
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
