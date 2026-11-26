// Uji browser revisi 48: kolom kiri papan task (kartu profil dan To Do) bisa dijangkau dari posisi gulir mana pun.
// Di 1920 dan 1600 px (kolom kiri di samping papan): 15 to do, papan digulir ke tengah, kolom kiri tetap di dalam
// layar, isian Tulis to do terlihat, roda di kolom kiri mencapai item terakhir tanpa menggeser halaman, Tab ke item
// di luar area terlihat menggulirkan area itu, tanpa gulir horizontal dan tombol di bawah 44 px. Di 1280 dan 360 px
// (kolom kiri di atas papan) panel To Do tidak berubah. Membuat 15 to do milik Staf dan 12 task (PIC Staf) supaya
// papan cukup panjang, lalu menghapus semuanya.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r48.mjs
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
const mgr = await session(browser, BASE, "manajer");
const sApi = await api(browser, BASE, staf.state);
const mApi = await api(browser, BASE, mgr.state);
const todos = [];
const tasks = [];

try {
  for (let i = 1; i <= 15; i++) {
    const r = await sApi.call("POST", "/todos", { title: `E2E r48 ${String(i).padStart(2, "0")} rapikan bahan` });
    if (r.body?.todo) todos.push(r.body.todo);
  }
  for (let i = 1; i <= 12; i++) {
    const r = await mApi.call("POST", "/tasks", { title: `E2E r48 task ${i}`, urgency: "low", pic_ids: [staf.user.id] });
    if (r.status === 201) tasks.push(r.body.id);
  }
  expect(todos.length === 15 && tasks.length === 12, "data", `to do ${todos.length}, task ${tasks.length}`);
  const last = todos.at(-1)?.title;

  for (const width of [1920, 1600]) {
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width, height: 900 } });
    const page = await ctx.newPage();
    const errors = [];
    page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
    await page.goto(BASE + "/task");
    await page.waitForLoadState("networkidle");
    const side = page.getByTestId("board-side");
    expect(await side.isVisible(), `${width}`, "kolom kiri tidak tampil di samping papan");
    const lastBtn = side.getByRole("button", { name: `Ubah ${last}` });
    await lastBtn.waitFor({ state: "attached", timeout: 8000 });

    // Papan digulir ke tengah halaman.
    const mid = await page.evaluate(() => {
      const y = Math.round((document.documentElement.scrollHeight - innerHeight) / 2);
      scrollTo(0, y);
      return { y: scrollY, max: document.documentElement.scrollHeight - innerHeight };
    });
    expect(mid.y > 100, `${width}`, `halaman tidak cukup panjang untuk digulir ke tengah (${JSON.stringify(mid)})`);
    await page.waitForTimeout(200);
    const inView = (sel) =>
      page.evaluate((sel) => {
        const r = (typeof sel === "string" ? document.querySelector(sel) : null)?.getBoundingClientRect();
        return r ? r.top >= -0.5 && r.bottom <= innerHeight + 0.5 && r.height > 0 : false;
      }, sel);
    expect(await inView("[data-testid=board-side]"), `${width}`, "kolom kiri keluar dari layar saat papan di tengah");
    expect(
      await page.getByRole("textbox", { name: "Tulis to do" }).locator("visible=true").evaluate((el) => {
        const r = el.getBoundingClientRect();
        return r.top >= 0 && r.bottom <= innerHeight;
      }),
      `${width}`,
      "isian Tulis to do tidak terlihat",
    );

    // Roda di area daftar To Do: item terakhir tercapai, halaman tidak bergeser.
    const lb = await page.getByRole("textbox", { name: "Tulis to do" }).locator("visible=true").boundingBox();
    await page.mouse.move(lb.x + 40, lb.y - 60);
    for (let i = 0; i < 12; i++) await page.mouse.wheel(0, 400);
    await page.waitForTimeout(300);
    const after = await page.evaluate(() => scrollY);
    expect(after === mid.y, `${width}`, `halaman ikut bergeser saat kolom kiri digulir (${mid.y} lalu ${after})`);
    const lastSeen = await lastBtn.evaluate((el) => {
      const r = el.getBoundingClientRect();
      const box = document.querySelector("[data-testid=board-side]").getBoundingClientRect();
      return r.bottom <= Math.min(innerHeight, box.bottom) + 0.5 && r.top >= box.top - 0.5;
    });
    expect(lastSeen, `${width}`, "item to do terakhir tidak tercapai dengan menggulir kolom kiri");

    // Keyboard: area daftar digulir ke atas, lalu Tab dari item ke-14 ke item terakhir menggulirkan area itu.
    await page.evaluate((title) => {
      const btn = [...document.querySelectorAll("[data-testid=board-side] button")].find((b) => b.getAttribute("aria-label") === `Ubah ${title}`);
      let el = btn.parentElement;
      while (el && getComputedStyle(el).overflowY !== "auto") el = el.parentElement;
      el.scrollTop = 0;
    }, last);
    await side.getByRole("button", { name: `Ubah ${todos.at(-2).title}` }).evaluate((el) => el.focus({ preventScroll: true }));
    await page.keyboard.press("Tab");
    await page.keyboard.press("Tab");
    const focusSeen = await page.evaluate(() => {
      const el = document.activeElement;
      const r = (el.closest("li") ?? el).getBoundingClientRect();
      let s = el.parentElement;
      while (s && getComputedStyle(s).overflowY !== "auto") s = s.parentElement;
      const box = s.getBoundingClientRect();
      return { label: el.getAttribute("aria-label") ?? el.textContent.trim().slice(0, 40), ok: r.top >= box.top - 1 && r.bottom <= box.bottom + 1 };
    });
    expect(focusSeen.ok, `${width}`, `fokus keyboard di luar area terlihat (${focusSeen.label})`);
    expect((await page.evaluate(() => scrollY)) === mid.y, `${width}`, "halaman bergeser saat fokus keyboard masuk daftar");

    const l = await layout(page);
    expect(!l.overflow, `${width}`, "gulir horizontal");
    expect(l.small.length === 0, `${width}`, `tombol di bawah 44 px: ${l.small.join(", ")}`);
    expect(errors.length === 0, `${width}`, `console: ${errors.join(" | ").slice(0, 200)}`);
    await ctx.close();
  }

  // Di bawah breakpoint kolom kiri tidak tampil; panel To Do tetap seperti sebelumnya (daftar dibatasi 22rem dan
  // baris tambah adalah baris terakhir daftar).
  for (const width of [1280, 360]) {
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width, height: 800 } });
    const page = await ctx.newPage();
    await page.goto(BASE + "/task");
    await page.waitForLoadState("networkidle");
    expect(!(await page.getByTestId("board-side").isVisible()), `${width}`, "kolom kiri tampil di bawah breakpoint");
    await page.getByRole("button", { name: /^To Do/ }).locator("visible=true").first().click();
    const dlg = page.getByRole("dialog", { name: "To Do" });
    await dlg.waitFor({ timeout: 8000 });
    const shape = await dlg.getByRole("textbox", { name: "Tulis to do" }).evaluate((el) => {
      const ul = el.closest("ul");
      return { list: ul?.className.includes("max-h-[22rem]") ?? false, lastRow: ul?.lastElementChild?.contains(el) ?? false };
    });
    expect(shape.list && shape.lastRow, `${width}`, `panel To Do berubah: ${JSON.stringify(shape)}`);
    const l = await layout(page);
    expect(!l.overflow, `${width}`, "gulir horizontal");
    await ctx.close();
  }
} finally {
  for (const t of todos) await sApi.call("DELETE", `/todos/${t.id}`);
  for (const id of tasks) {
    const t = (await mApi.call("GET", `/tasks/${id}`)).body?.task;
    if (t) await mApi.call("DELETE", `/tasks/${id}?version=${t.version}`);
  }
  await sApi.ctx.close();
  await mApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
