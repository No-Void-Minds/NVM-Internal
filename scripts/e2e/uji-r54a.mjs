// Uji browser revisi 54a: pustaka bentuk BPMN dan palet baru.
// - 1280 px: proses BPMN kecil (event mulai, task pengguna, gateway eksklusif, event selesai) dari palet dan API,
//   sequence flow lewat preset di panel garis, message flow ke anotasi. Ikon varian tergambar, varian diganti lewat
//   Gaya bentuk, muat ulang tetap sama, klien kedua (Manajer) melihat bentuk dan variannya, ekspor PNG tanpa galat.
// - Palet di 1280 dan 360 px: cari lintas kelompok, kelompok dilipat dan dibuka (opsi terlipat tidak ikut navigasi
//   panah), Terakhir dipakai bertahan setelah muat ulang (localStorage), pratinjau ikut hover dan panah, tanpa
//   gulir horizontal dan tombol minimal 44 px.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r54a.mjs
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
const manajer = await session(browser, BASE, "manajer");
const sApi = await api(browser, BASE, staf.state);
const holders = [];
const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
const shape = (s, variant, text) => ({ shape: s, ...(variant ? { variant } : {}), rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text, align: "center", font_size: 16 });
const boardOf = async (b) => (await sApi.call("GET", `/whiteboard/boards/${b}`)).body.board;
const marksOf = (page, id) => page.locator(`[data-card="${id}"] [data-mark]`).count();
const markD = (page, id) => page.locator(`[data-card="${id}"] [data-mark]`).evaluateAll((els) => els.map((e) => e.getAttribute("d")).join("|"));

async function openPalette(page, W) {
  if (W >= 768) await page.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Bentuk", exact: true }).click();
  else {
    await page.getByRole("button", { name: "Tambah", exact: true }).click();
    await page.locator("dialog[open]").getByRole("button", { name: "Bentuk", exact: true }).click();
  }
  const dlg = page.getByRole("dialog", { name: "Pilih bentuk" });
  await dlg.waitFor({ timeout: 5000 });
  // Animasi buka dialog (skala) selesai dulu sebelum ukuran tombol diukur.
  await page.waitForTimeout(300);
  return dlg;
}
const previewText = (dlg) => dlg.getByTestId("wb-shape-preview").textContent().then((t) => (t ?? "").trim());

try {
  // Palet di 1280 dan 360.
  for (const W of [1280, 360]) {
    const where = (s) => `${W} palet ${s}`;
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54a palet ${W} ${Date.now()}`, x: 9900, y: 9400 + holders.length * 200, w: 200, h: 120 })).body.card;
    holders.push(holder);
    const board = holder.ref.id;
    const errors = [];
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: W, height: 800 } });
    const ps = await ctx.newPage();
    ps.on("pageerror", (e) => errors.push(String(e)));
    await ps.goto(BASE + `/whiteboard/${board}`);
    await saved(ps);
    // Mulai tanpa riwayat bentuk supaya hasilnya pasti.
    await ps.evaluate(() => {
      for (const k of Object.keys(localStorage)) if (k.startsWith("nvm-wb-bentuk-terakhir:")) localStorage.removeItem(k);
    });

    let dlg = await openPalette(ps, W);
    const lay = await layout(ps);
    expect(!lay.overflow, where("gulir"), "gulir horizontal di palet");
    expect(lay.small.length === 0, where("tap"), lay.small.join(", "));
    const box = dlg.getByRole("combobox", { name: "Cari bentuk" });
    await box.fill("gateway");
    const gw = await dlg.getByRole("option").allTextContents();
    expect(gw.length === 4 && gw.every((t) => /Gateway/.test(t)), where("cari"), `hasil cari gateway: ${gw.join(", ")}`);
    expect((await previewText(dlg)).startsWith("Gateway eksklusif"), where("pratinjau"), await previewText(dlg));
    await box.press("ArrowDown");
    expect((await previewText(dlg)).startsWith("Gateway paralel"), where("pratinjau panah"), await previewText(dlg));
    await dlg.getByRole("option", { name: "Gateway inklusif" }).hover();
    await ps.mouse.move(1, 1, { steps: 1 }).catch(() => {});
    await dlg.getByRole("option", { name: "Gateway inklusif" }).hover();
    expect((await previewText(dlg)).startsWith("Gateway inklusif"), where("pratinjau hover"), await previewText(dlg));
    await box.fill("waktu");
    const tm = await dlg.getByRole("option").allTextContents();
    expect(tm.length === 2 && tm.every((t) => /waktu/.test(t)), where("cari varian"), tm.join(", "));

    // Lipat kelompok BPMN: opsinya hilang dan panah melompatinya.
    await box.fill("");
    const bpmn = dlg.getByTestId("wb-shape-group").filter({ hasText: /BPMN \(/ });
    await bpmn.click();
    expect((await bpmn.getAttribute("aria-expanded")) === "false", where("lipat"), "tombol kelompok tidak aria-expanded false");
    expect((await dlg.locator('[role=option][data-shape^="bpmn_"]').count()) === 0, where("lipat"), "opsi BPMN masih tampil");
    await box.focus();
    await box.press("End");
    const last = await dlg.locator('[role=option][aria-selected="true"]').getAttribute("data-shape");
    expect(!last?.startsWith("bpmn_"), where("lipat panah"), `opsi terakhir ${last}`);
    await bpmn.click();
    expect((await dlg.locator('[role=option][data-shape^="bpmn_"]').count()) > 20, where("buka"), "opsi BPMN tidak kembali");

    // Pilih dua bentuk, lalu Terakhir dipakai berisi keduanya (yang terbaru di depan) dan bertahan sesudah muat ulang.
    await box.fill("event mulai pesan");
    await dlg.getByRole("option", { name: "Event mulai pesan" }).click();
    await saved(ps);
    dlg = await openPalette(ps, W);
    await dlg.getByRole("combobox", { name: "Cari bentuk" }).fill("task layanan");
    await dlg.getByRole("combobox", { name: "Cari bentuk" }).press("Enter");
    await saved(ps);
    await ps.reload();
    await saved(ps);
    dlg = await openPalette(ps, W);
    const firstGroup = (await dlg.getByTestId("wb-shape-group").first().textContent())?.trim() ?? "";
    const recent = await dlg.getByRole("group", { name: /^Terakhir dipakai/ }).getByRole("option").allTextContents();
    expect(firstGroup.startsWith("Terakhir dipakai") && recent[0]?.trim() === "Task layanan" && recent[1]?.trim() === "Event mulai pesan", where("terakhir"), `${firstGroup} | ${recent.join(", ")}`);
    await ps.keyboard.press("Escape");
    const cards = (await boardOf(board)).cards.filter((c) => c.kind === "shape");
    expect(cards.some((c) => c.content.shape === "bpmn_start" && c.content.variant === "message") && cards.some((c) => c.content.shape === "bpmn_task" && c.content.variant === "service"), where("tersimpan"), JSON.stringify(cards.map((c) => c.content.shape + ":" + c.content.variant)));
    expect(errors.length === 0, where("galat"), errors.join(" | "));
    await ctx.close();
  }

  // Proses BPMN kecil di 1280.
  {
    const W = 1280;
    const where = (s) => `${W} proses ${s}`;
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54a proses ${Date.now()}`, x: 9900, y: 9400 + holders.length * 200, w: 200, h: 120 })).body.card;
    holders.push(holder);
    const board = holder.ref.id;
    const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
    const start = await add(0, 100, 56, 56, shape("bpmn_start", "none", "Pesanan masuk"));
    const task = await add(140, 83, 160, 90, shape("bpmn_task", "user", "Periksa stok"));
    const gw = await add(380, 96, 64, 64, shape("bpmn_gateway", "exclusive", "Stok ada?"));
    const end = await add(540, 100, 56, 56, shape("bpmn_end", "none", "Selesai"));
    const note = await add(140, 260, 180, 70, { ...shape("bpmn_annotation", "", "Dicek tim gudang"), fill: "", align: "left" });
    const link = async (from, to, extra = {}) => (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from, to, ...extra })).body.connector;
    const c1 = await link(start.id, task.id);
    await link(task.id, gw.id, { style: { start: "none", end: "arrow", line: "solid", color: "" } });
    await link(gw.id, end.id, { style: { start: "none", end: "arrow", line: "solid", color: "" } });
    const c4 = await link(task.id, note.id);

    const errors = [];
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: W, height: 800 } });
    const ps = await ctx.newPage();
    ps.on("pageerror", (e) => errors.push(String(e)));
    await ps.goto(BASE + `/whiteboard/${board}`);
    await saved(ps);
    await ps.getByRole("button", { name: "Pas ke layar" }).click();
    await ps.waitForTimeout(300);
    const mctx = await browser.newContext({ storageState: manajer.state, viewport: { width: W, height: 800 } });
    const pm = await mctx.newPage();
    await pm.goto(BASE + `/whiteboard/${board}`);
    await pm.locator(`[data-card="${task.id}"]`).waitFor({ timeout: 10000 });

    expect((await marksOf(ps, task.id)) === 2, where("ikon task"), `tanda task pengguna ${await marksOf(ps, task.id)}`);
    expect((await marksOf(ps, gw.id)) === 1, where("ikon gateway"), `tanda gateway eksklusif ${await marksOf(ps, gw.id)}`);
    expect((await marksOf(ps, start.id)) === 0, where("event polos"), "event mulai polos bertanda");
    const sw = await ps.locator(`[data-card="${end.id}"] svg path`).first().getAttribute("stroke-width");
    expect(Number(sw) === 4, where("event selesai"), `tebal garis event selesai ${sw}`);
    const label = await ps.locator(`[data-card="${gw.id}"]`).getByText("Stok ada?").boundingBox();
    const gb = await ps.locator(`[data-card="${gw.id}"] svg`).first().boundingBox();
    expect(label && gb && label.y >= gb.y + gb.height - 2, where("label di bawah"), `label ${JSON.stringify(label)} bentuk ${JSON.stringify(gb)}`);

    // Preset sequence flow untuk garis pertama dan message flow ke anotasi lewat panel garis.
    for (const [id, name] of [[c1.id, "BPMN: Sequence flow"], [c4.id, "BPMN: Message flow"]]) {
      await ps.locator(`[data-connector="${id}"] path[data-hit]`).dispatchEvent("dblclick");
      const dlg = ps.getByRole("dialog", { name: "Garis penghubung" });
      await dlg.waitFor({ timeout: 5000 });
      await dlg.getByRole("button", { name: /^Preset garis/ }).click();
      await ps.getByTestId("wb-picker").getByRole("button", { name, exact: true }).click();
      await saved(ps);
      await ps.keyboard.press("Escape");
    }
    const conns = (await boardOf(board)).connectors;
    const s1 = conns.find((c) => c.id === c1.id).style;
    const s4 = conns.find((c) => c.id === c4.id).style;
    expect(s1.line === "solid" && s1.end === "arrow" && s1.start === "none", where("sequence flow"), JSON.stringify(s1));
    expect(s4.line === "dashed" && s4.start === "circle" && s4.end === "open_arrow", where("message flow"), JSON.stringify(s4));

    // Ganti varian task lewat Gaya bentuk; Manajer melihat ikonnya berubah.
    const before = await markD(pm, task.id);
    await ps.locator(`[data-card="${task.id}"]`).click();
    await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
    await ps.getByTestId("wb-menu").getByRole("menuitem", { name: /^Gaya bentuk/ }).click();
    const style = ps.getByRole("dialog", { name: "Gaya bentuk" });
    await style.waitFor({ timeout: 5000 });
    await style.getByRole("button", { name: /^Varian:/ }).click();
    await ps.getByTestId("wb-picker").getByRole("button", { name: "Task layanan", exact: true }).click();
    await saved(ps);
    await ps.keyboard.press("Escape");
    const t2 = (await boardOf(board)).cards.find((c) => c.id === task.id);
    expect(t2.content.variant === "service", where("varian"), JSON.stringify(t2.content));
    await pm.waitForFunction(([id, d]) => [...document.querySelectorAll(`[data-card="${id}"] [data-mark]`)].map((e) => e.getAttribute("d")).join("|") !== d, [task.id, before], { timeout: 10000 }).catch(() => {});
    expect((await markD(pm, task.id)) !== before, where("realtime varian"), "ikon task di layar Manajer tidak berubah");

    // Muat ulang: bentuk dan ikon sama.
    const shown = await markD(ps, task.id);
    await ps.reload();
    await saved(ps);
    expect((await markD(ps, task.id)) === shown && (await marksOf(ps, gw.id)) === 1, where("muat ulang"), "ikon berubah sesudah muat ulang");

    // Ekspor PNG.
    await ps.getByRole("button", { name: "Ekspor", exact: true }).first().click();
    const ex = ps.getByRole("dialog", { name: "Ekspor papan" });
    await ex.waitFor({ timeout: 5000 });
    await ex.getByRole("radio", { name: /^PNG/ }).check();
    const [dl] = await Promise.all([ps.waitForEvent("download", { timeout: 20000 }), ex.getByRole("button", { name: "Unduh PNG" }).click()]);
    expect(dl.suggestedFilename().endsWith(".png"), where("ekspor"), dl.suggestedFilename());
    await ex.getByRole("button", { name: "Selesai" }).click();
    expect(errors.length === 0, where("galat"), errors.join(" | "));
    await mctx.close();
    await ctx.close();
  }
} finally {
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
