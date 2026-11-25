// Uji browser revisi 46: ubah To Do di kartu profil papan. Dobel klik (dan dobel ketuk di layar sentuh) membuka
// isian dengan kursor di posisi klik tanpa seleksi, Escape membatalkan, Enter menyimpan, isian selebar baris dan
// tumbuh mengikuti teks, kontrol baris tersembunyi selama mengetik, mode Atur tidak menyempitkan teks, dan teks
// panjang terbungkus utuh. Di 360 dan 1280 px. Membuat satu to do uji milik akun Staf lalu menghapusnya.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r46.mjs
import { chromium } from "playwright-core";
import { api, layout, realLogins, session } from "./sesi.mjs";

const BASE = process.env.E2E_BASE || "http://localhost:5173";
const browser = await chromium.launch({ channel: "msedge", headless: true });
let checks = 0;
const fails = [];
const expect = (ok, where, msg) => {
  checks++;
  if (!ok) fails.push(`${where}: ${msg}`);
};

const staf = await session(browser, BASE, "staf");
const sApi = await api(browser, BASE, staf.state);
const cleanup = async () => {
  const list = (await sApi.call("GET", "/todos")).body?.todos ?? [];
  for (const t of list) if (String(t.title).startsWith("E2E r46")) await sApi.call("DELETE", `/todos/${t.id}`);
};
const titleOf = async (id) => ((await sApi.call("GET", "/todos")).body?.todos ?? []).find((t) => t.id === id)?.title;

// Titik di tepi kiri huruf ke-n sebuah kata di dalam elemen, beserta indeks huruf itu di seluruh teks.
const pointAt = (loc, word, n) =>
  loc.evaluate(
    (el, [word, n]) => {
      const walk = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
      for (let t = walk.nextNode(); t; t = walk.nextNode()) {
        const i = t.textContent.indexOf(word);
        if (i < 0) continue;
        const r = document.createRange();
        r.setStart(t, i + n);
        r.setEnd(t, i + n + 1);
        const b = r.getBoundingClientRect();
        const lead = t.textContent.length - t.textContent.trimStart().length;
        return { x: b.left + 1, y: b.top + b.height / 2, index: i + n - lead };
      }
      return null;
    },
    [word, n],
  );

async function openTodo(page) {
  await page.goto(BASE + "/task");
  await page.waitForLoadState("networkidle");
  const panelBtn = page.getByRole("button", { name: /^To Do/ }).locator("visible=true");
  if (await panelBtn.count()) {
    await panelBtn.first().click();
    await page.getByRole("dialog", { name: "To Do" }).waitFor({ timeout: 8000 });
  }
}

try {
  await cleanup();
  for (const width of [360, 1280]) {
    const title = `E2E r46 ${width} rencana konten minggu depan untuk kampanye ramadan bersama tim desain dan tim iklan berbayar supaya jadwalnya rapi`;
    const made = (await sApi.call("POST", "/todos", { title })).body?.todo;
    expect(!!made, `${width}`, "to do uji gagal dibuat");
    if (!made) continue;

    for (const touch of width === 360 ? [false, true] : [false]) {
      const where = `${width}${touch ? " sentuh" : ""}`;
      const ctx = await browser.newContext({
        storageState: staf.state,
        viewport: { width, height: 800 },
        hasTouch: touch,
        isMobile: touch,
      });
      const page = await ctx.newPage();
      const errors = [];
      page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
      await openTodo(page);
      const cur = await titleOf(made.id);
      const btn = page.getByRole("button", { name: `Ubah ${cur}` }).locator("visible=true").first();
      await btn.waitFor({ timeout: 8000 });
      const box = page.getByRole("textbox", { name: "Ubah to do" });

      // Teks panjang terbungkus utuh di tampilan biasa: tidak meluap ke samping dan lebih dari satu baris.
      const wrap = await btn.evaluate((el) => ({
        over: el.scrollWidth > el.clientWidth + 1 || el.scrollHeight > el.clientHeight + 1,
        lines: Math.round(el.getBoundingClientRect().height / parseFloat(getComputedStyle(el).lineHeight)),
      }));
      expect(!wrap.over && wrap.lines >= 2, where, `teks panjang tidak terbungkus utuh (${JSON.stringify(wrap)})`);

      if (!touch) {
        // Klik sekali tidak membuka isian.
        await btn.click();
        await page.waitForTimeout(250);
        expect((await box.count()) === 0, where, "klik sekali membuka isian ubah");
      }

      // Dobel klik (atau dobel ketuk) di huruf ketiga kata "kampanye": kursor di sana, tanpa seleksi.
      let pt = await pointAt(btn, "kampanye", 2);
      if (touch) {
        await page.touchscreen.tap(pt.x, pt.y);
        await page.touchscreen.tap(pt.x, pt.y);
      } else await page.mouse.dblclick(pt.x, pt.y);
      await box.waitFor({ timeout: 5000 }).catch(() => {});
      const caret = await box.evaluate((el) => ({ s: el.selectionStart, e: el.selectionEnd, active: el === document.activeElement })).catch(() => null);
      expect(caret && caret.active && caret.s === caret.e && caret.s === pt.index, where, `kursor ${JSON.stringify(caret)}, mau ${pt.index} tanpa seleksi`);

      if (caret) {
        // Isian selebar baris, tumbuh mengikuti teks, dan kontrol baris tersembunyi selama mengetik.
        const fit = await box.evaluate((el) => {
          const row = el.closest("li").getBoundingClientRect();
          return { ratio: el.getBoundingClientRect().width / row.width, over: el.scrollHeight > el.clientHeight + 2 };
        });
        expect(fit.ratio > 0.75 && !fit.over, where, `isian ubah ${JSON.stringify(fit)}`);
        expect((await page.getByRole("button", { name: `Hapus ${cur}` }).locator("visible=true").count()) === 0, where, "tombol Hapus tampil selama mengetik");

        // Huruf yang diketik masuk di posisi kursor; Escape membatalkan.
        await page.keyboard.type("X");
        const v = await box.inputValue();
        expect(v === cur.slice(0, pt.index) + "X" + cur.slice(pt.index), where, `isi setelah ketik: "${v.slice(0, 80)}"`);
        await page.keyboard.press("Escape");
        await page.waitForTimeout(300);
        expect((await box.count()) === 0 && (await titleOf(made.id)) === cur, where, "Escape tidak membatalkan perubahan");
        const focus = await page.evaluate(() => document.activeElement?.getAttribute("aria-label") ?? "");
        expect(focus === `Ubah ${cur}`, where, `fokus setelah Escape di "${focus}"`);
      }

      if (!touch) {
        // Enter di judul yang terfokus membuka isian (kursor di akhir), Enter kedua menyimpan.
        await btn.focus();
        await page.keyboard.press("Enter");
        await box.waitFor({ timeout: 5000 });
        await page.keyboard.type(" Y");
        await page.keyboard.press("Enter");
        await page.waitForTimeout(500);
        expect((await titleOf(made.id)) === `${cur} Y`, where, `Enter tidak menyimpan: "${await titleOf(made.id)}"`);
        await sApi.call("PUT", `/todos/${made.id}`, { title: cur });
        await page.reload();
        await page.waitForLoadState("networkidle");
        await openTodo(page);
      }

      // Mode Atur: teks tetap selebar sebelumnya dan kontrol berada di bawah teks, bukan menyempitkannya.
      const b2 = page.getByRole("button", { name: `Ubah ${cur}` }).locator("visible=true").first();
      const before = await b2.boundingBox();
      await page.getByRole("button", { name: "Atur", exact: true }).locator("visible=true").first().click();
      const after = await b2.boundingBox();
      const up = await page.getByRole("button", { name: `Naikkan ${cur}` }).locator("visible=true").first().boundingBox();
      expect(after && before && after.width >= before.width - 1, where, `mode Atur menyempitkan teks ${before?.width} ke ${after?.width}`);
      expect(up && after && up.y >= after.y + after.height - 1, where, "kontrol mode Atur tidak di bawah teks");
      expect(up && up.width >= 43.5 && up.height >= 43.5, where, `tombol Naikkan ${up?.width}x${up?.height}`);
      pt = await pointAt(b2, "ramadan", 0);
      await page.mouse.dblclick(pt.x, pt.y);
      expect(await box.waitFor({ timeout: 5000 }).then(() => true, () => false), where, "dobel klik di mode Atur tidak membuka isian");
      await page.keyboard.press("Escape");

      const l = await layout(page);
      expect(!l.overflow, where, "gulir horizontal");
      expect(l.small.length === 0, where, `tombol di bawah 44 px: ${l.small.join(", ")}`);
      expect(errors.length === 0, where, `console: ${errors.join(" | ").slice(0, 200)}`);
      await ctx.close();
    }
  }
} finally {
  await cleanup();
  await sApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
