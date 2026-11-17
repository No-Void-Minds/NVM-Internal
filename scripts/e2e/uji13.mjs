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
// E2E_BASE (bawaan http://localhost:5173), E2E_ONLY=popover (hanya uji popover profil, revisi 14).
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
    pages: ["/task", "/arsip", "/profil", "/status", "TASK", "/tidak-ada"],
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

for (const theme of ONLY === "popover" ? [] : THEMES) {
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
        const sub = page.locator("dialog[open] button[aria-expanded]").first();
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
  const inPanel = page.locator("dialog[open] [data-person-trigger]").first();
  if (await inPanel.count()) {
    await inPanel.click();
    await page.waitForSelector(POP);
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
