// Audit luapan di 768 dan 1024 px (gerbang akhir Whiteboard W5): halaman utama dan semua panel Whiteboard untuk Staf
// dan Manajer. Lolos bila tidak ada gulir horizontal, tidak ada tombol terlihat di bawah 44 px, tidak ada galat
// console, dan halaman Whiteboard tidak bisa digulir. Membuat satu papan uji berisi bentuk dan garis lalu memindahkannya
// ke Tempat sampah.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-lebar.mjs
import { chromium } from "playwright-core";
import { api, layout, realLogins, session } from "./sesi.mjs";

const BASE = process.env.E2E_BASE || "http://localhost:5173";
const WIDTHS = (process.env.E2E_WIDTHS || "768,1024").split(",").map(Number);
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

// Papan uji: bentuk, kelas UML, catatan, dan garis siku.
const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E lebar ${Date.now()}`, x: 9500, y: 9500, w: 200, h: 120 })).body.card;
const board = holder.ref.id;
const shape = (extra) => ({ rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text: "Langkah", align: "center", font_size: 16, ...extra });
const mk = async (kind, x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind, x, y, w, h, content })).body.card;
const a = await mk("shape", 0, 0, 180, 80, shape({ shape: "process" }));
const b = await mk("shape", 0, 220, 180, 120, shape({ shape: "decision", text: "Lanjut?" }));
const c = await mk("shape", 320, 200, 220, 180, shape({ shape: "uml_class", name: "Pesanan", attributes: "+ id: int", methods: "+ bayar()" }));
await mk("note", 320, 0, 240, 120, { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Catatan" }] }] } });
await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: a.id, to: b.id, geometry: { from: { port: "bottom" }, to: { port: "top" }, route: "orthogonal", points: [] }, style: { start: "none", end: "arrow", line: "solid", color: "" }, label: "ya" });
await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: b.id, to: c.id, geometry: { from: { port: "right" }, to: { port: "left" }, route: "curved", points: [] }, style: { start: "diamond", end: "triangle", line: "dashed", color: "" } });

async function check(page, where, errors, wb = false) {
  await page.waitForLoadState("networkidle");
  await page.waitForTimeout(200);
  const l = await layout(page);
  expect(!l.overflow, where, "gulir horizontal");
  expect(l.small.length === 0, where, `tombol di bawah 44 px: ${l.small.join(", ")}`);
  if (wb) {
    const scroll = await page.evaluate(() => document.documentElement.scrollHeight > window.innerHeight + 1);
    expect(!scroll, where, "halaman Whiteboard bisa digulir");
  }
  expect(errors.length === 0, where, `console: ${errors.splice(0).join(" | ").slice(0, 200)}`);
}
const close = async (page) => {
  await page.keyboard.press("Escape");
  await page.waitForTimeout(150);
};

try {
  for (const width of WIDTHS) {
    for (const [role, sess] of [["Staf", staf], ["Manajer", mgr]]) {
      const ctx = await browser.newContext({ storageState: sess.state, viewport: { width, height: 900 } });
      const page = await ctx.newPage();
      const errors = [];
      page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
      page.on("pageerror", (e) => errors.push(e.message));
      const at = (w) => `${width} ${role} ${w}`;
      for (const path of ["/task", "/arsip", "/aktivitas", "/timeline"]) {
        await page.goto(BASE + path);
        await check(page, at(path), errors);
      }
      await page.goto(BASE + "/whiteboard");
      await check(page, at("whiteboard akar"), errors, true);
      await page.goto(BASE + `/whiteboard/${board}`);
      await page.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
      await check(page, at("papan bentuk"), errors, true);

      // Bentuk terpilih: pegangan, port, dan bar pilihan.
      await page.locator(`[data-card="${b.id}"]`).click();
      await check(page, at("bentuk terpilih"), errors, true);
      await page.getByLabel("Aksi lain untuk kartu terpilih").selectOption("shapestyle");
      await page.getByRole("dialog", { name: "Gaya bentuk" }).waitFor({ timeout: 5000 });
      await check(page, at("gaya bentuk"), errors);
      await close(page);
      await page.getByLabel("Aksi lain untuk kartu terpilih").selectOption("lines");
      await page.getByRole("dialog", { name: "Garis penghubung" }).waitFor({ timeout: 5000 });
      await check(page, at("panel garis"), errors);
      await close(page);
      await close(page);

      // Palet bentuk dan template lewat sidebar alat (768 ke atas), ekspor dari bar atas.
      await page.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Bentuk", exact: true }).click();
      await page.getByRole("dialog", { name: "Pilih bentuk" }).waitFor({ timeout: 5000 });
      await check(page, at("palet bentuk"), errors);
      await close(page);
      await page.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Template", exact: true }).click();
      await page.getByRole("dialog", { name: "Buat papan dari template" }).waitFor({ timeout: 5000 });
      await check(page, at("template"), errors);
      await close(page);
      await page.getByRole("button", { name: "Ekspor", exact: true }).click();
      await page.getByRole("dialog", { name: "Ekspor papan" }).waitFor({ timeout: 5000 });
      await check(page, at("ekspor"), errors);
      await close(page);
      await page.getByRole("button", { name: "Daftar", exact: true }).click();
      await page.getByTestId("wb-list-item").first().waitFor({ timeout: 5000 });
      await check(page, at("tampilan daftar"), errors, true);
      await ctx.close();
    }
  }
} finally {
  await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: [holder.id] });
  await sApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
