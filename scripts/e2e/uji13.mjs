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
// E2E_ONLY=activity (Aktivitas Harian, revisi 21; menulis satu sel aktivitas hari ini milik akun Staf dan
// satu sel lewat dialog Manajer di database dev, output-nya tautan file Drive karangan).
// E2E_ONLY=revision (revisi 22: dua kelompok PIC di kartu dan PIC sub task mengembalikan Ready ke On Progress;
// membuat satu task uji per lebar di database dev lewat API, dengan Manajer sebagai PIC task).
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
      "/pengguna",
      "/audit-log",
      "/pengaturan",
      "/profil",
      "/status",
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
      "/profil",
      "/status",
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
  await page.keyboard.press("Tab");
  const ketiga = await namaFokus();
  expect(
    /^Aktivitas 10\.00-11\.00/.test(ketiga),
    "keyboard",
    `Tab dari Output 09.00 ke "${ketiga}", mau Aktivitas 10.00-11.00`,
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
