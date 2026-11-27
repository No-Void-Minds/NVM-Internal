// Uji browser revisi 57a di 1280 px (dan editor di 360 px): tabel ERD.
// Tabel dari palet (kolom awal, tinggi ikut baris); editor di tempat lewat dobel klik: ketik nama dan tipe, Tab,
// Enter di baris terakhir menambah baris (tinggi tabel ikut), PK menonjol dengan ikon dan teks, Alt+panah dan seret
// pegangan mengurutkan ulang, hapus baris; tombol editor minimal 44 px di layar; undo dan redo per aksi; garis ke
// baris tertentu (dari API, dari port kartu ke baris tabel lain, dan dari port baris di editor) ikut baris saat
// diurutkan ulang dan tabel digeser, jatuh ke kotak tabel saat barisnya dihapus dan kembali saat dibatalkan; pilihan
// baris di panel garis; tabel terkunci menolak tambah kolom; kunci lunak orang lain; klien kedua; salin tempel;
// template; muat ulang; ekspor PNG.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r57a.mjs
import { randomUUID } from "node:crypto";
import { chromium } from "playwright-core";
import { api, realLogins, session } from "./sesi.mjs";

const BASE = process.env.E2E_BASE || "http://localhost:5173";
const browser = await chromium.launch({ channel: "msedge", headless: true });
let checks = 0;
const fails = [];
const expect = (ok, where, msg) => {
  checks++;
  if (!ok) fails.push(`${where}: ${msg}`);
};
const HEAD = 36;
const ROW = 28;
const nums = (d) => (d.match(/-?\d+(\.\d+)?/g) ?? []).map(Number);
const near = (a, b) => Math.abs(a - b) < 0.6;

const staf = await session(browser, BASE, "staf");
const manajer = await session(browser, BASE, "manajer");
const sApi = await api(browser, BASE, staf.state);
const holders = [];
const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
const table = (name, rows) => ({ shape: "erd_table", rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text: "", align: "center", font_size: 14, name, rows });
const col = (id, name, type, keys = [], nullable = false) => ({ id, name, type, keys, nullable });

try {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r57a ${Date.now()}`, x: 9900, y: 12400, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const A = await add(0, 0, 260, HEAD + ROW * 3, table("pesanan", [col("p1", "id", "bigint", ["pk"]), col("p2", "pelanggan_id", "bigint", ["fk"]), col("p3", "total", "numeric", [], true)]));
  const B = await add(520, 0, 260, HEAD + ROW * 2, table("pelanggan", [col("c1", "id", "bigint", ["pk"]), col("c2", "nama", "text")]));
  const C = await add(520, 300, 260, HEAD + ROW * 2, table("produk", [col("r1", "id", "bigint", ["pk"]), col("r2", "nama", "text")]));
  const fk = (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: A.id, to: B.id, style: { start: "cf_many", end: "cf_one", line: "solid", color: "" }, geometry: { from: { port: "right", row: "p2" }, to: { port: "left", row: "c1" }, route: "orthogonal", points: [] } })).body.connector;
  expect(fk?.geometry?.from?.row === "p2", "garis ke baris lewat API", JSON.stringify(fk?.geometry));
  const bad = await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: A.id, to: B.id, geometry: { from: { port: "top", row: "p2" }, to: { port: "auto" }, route: "straight", points: [] } });
  expect(bad.status === 400, "baris di port atas ditolak", `${bad.status}`);
  const view = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board;
  const cardOf = async (id) => (await view()).cards.find((c) => c.id === id);
  const names = async (id) => (await cardOf(id)).content.rows.map((r) => r.name).join("|");

  const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: 1280, height: 800 } });
  const ps = await ctx.newPage();
  const errors = [];
  ps.on("pageerror", (e) => errors.push(String(e)));
  await ps.goto(BASE + `/whiteboard/${board}`);
  await saved(ps);
  await ps.getByRole("button", { name: "Pas ke layar" }).click();
  await ps.waitForTimeout(300);
  const mctx = await browser.newContext({ storageState: manajer.state, viewport: { width: 1280, height: 800 } });
  const pm = await mctx.newPage();
  await pm.goto(BASE + `/whiteboard/${board}`);
  await pm.locator(`[data-card="${A.id}"]`).waitFor({ timeout: 10000 });

  const pathOf = async (id) => nums((await ps.locator(`[data-connector="${id}"] path`).first().getAttribute("d")) ?? "");
  const unselect = async () => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
  };
  const key = async (k) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press(k);
    await saved(ps);
    await ps.waitForTimeout(300);
  };
  const editor = ps.locator("[data-erd-editor]");
  const openEditor = async (id) => {
    await unselect();
    await ps.locator(`[data-card="${id}"] [data-table-name]`).dblclick();
    await editor.waitFor({ timeout: 5000 });
  };
  const closeEditor = async () => {
    await ps.keyboard.press("Escape");
    await editor.waitFor({ state: "detached", timeout: 5000 });
    await saved(ps);
  };

  // 0. Tampilan: PK dengan ikon kunci dan teks, kolom null, garis di tengah baris p2 dan c1.
  const pkRow = ps.locator(`[data-card="${A.id}"] [data-row="p1"]`);
  expect((await pkRow.locator("[data-key-icon]").count()) === 1 && (await pkRow.locator("[data-keys]").textContent()).includes("PK"), "PK menonjol", "ikon kunci atau teks PK tidak ada");
  expect((await ps.locator(`[data-card="${A.id}"] [data-row="p3"]`).textContent()).includes("null"), "nullable tampil", "kolom null tidak ditandai");
  let pf = await pathOf(fk.id);
  expect(near(pf[0], A.x + 260) && near(pf[1], A.y + HEAD + ROW * 1.5) && near(pf.at(-2), B.x) && near(pf.at(-1), B.y + HEAD + ROW / 2), "garis di baris", `path ${pf.slice(0, 2)} .. ${pf.slice(-2)}`);

  // 1. Tabel dari palet: kolom awal id (PK) dan nama, tinggi dua baris.
  await ps.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Bentuk", exact: true }).click();
  const pal = ps.getByRole("dialog", { name: "Pilih bentuk" });
  await pal.waitFor({ timeout: 5000 });
  await pal.getByRole("combobox", { name: "Cari bentuk" }).fill("tabel");
  await pal.getByRole("option", { name: "Tabel", exact: true }).click();
  await saved(ps);
  const made = (await view()).cards.find((c) => c.content?.shape === "erd_table" && ![A.id, B.id, C.id].includes(c.id));
  expect(made && made.h === HEAD + ROW * 2 && made.content.rows.length === 2 && made.content.rows[0].keys[0] === "pk", "tabel dari palet", JSON.stringify(made && { h: made.h, rows: made.content.rows }));
  await key("Delete");
  await ps.getByRole("button", { name: "Pas ke layar" }).click();
  await ps.waitForTimeout(300);

  // 2. Editor: tombol 44 px, ketik nama dan tipe dengan Tab, Enter di baris terakhir menambah baris.
  await openEditor(A.id);
  const small = await editor.locator("button, input").evaluateAll((els) => els.filter((e) => { const b = e.getBoundingClientRect(); return b.width < 43.5 || b.height < 43.5; }).map((e) => e.getAttribute("aria-label") || e.textContent.trim()));
  expect(small.length === 0, "tap 44 px", small.slice(0, 5).join(", "));
  const focused = await ps.evaluate(() => document.activeElement?.getAttribute("aria-label"));
  expect(focused === "Nama kolom 1", "fokus awal", `${focused}`);
  await editor.locator('[data-edit-row="p3"] [data-field="name"]').click();
  await ps.keyboard.press("Enter");
  await saved(ps);
  let a = await cardOf(A.id);
  expect(a.content.rows.length === 4 && a.h === HEAD + ROW * 4, "Enter menambah baris", `${a.content.rows.length} baris, tinggi ${a.h}`);
  const fresh = a.content.rows[3].id;
  expect((await ps.evaluate(() => document.activeElement?.getAttribute("data-field"))) === "name", "fokus baris baru", "fokus tidak di nama kolom baru");
  await ps.keyboard.type("dibuat_pada");
  await ps.keyboard.press("Tab");
  await ps.keyboard.type("timestamptz");
  await ps.keyboard.press("Tab");
  await saved(ps);
  a = await cardOf(A.id);
  expect(a.content.rows[3].name === "dibuat_pada" && a.content.rows[3].type === "timestamptz", "ketik dan Tab", JSON.stringify(a.content.rows[3]));
  // Tombol PK baris baru: tekan dengan Enter (fokus sudah di sana sesudah Tab).
  expect((await ps.evaluate(() => document.activeElement?.getAttribute("data-key-toggle"))) === "pk", "Tab ke tombol PK", "fokus bukan di PK");
  await ps.keyboard.press("Space");
  await saved(ps);
  a = await cardOf(A.id);
  expect(a.content.rows[3].keys.join() === "pk", "PK lewat keyboard", JSON.stringify(a.content.rows[3].keys));
  // Klien kedua melihat baris baru.
  await pm.locator(`[data-card="${A.id}"] [data-row="${fresh}"]`).waitFor({ timeout: 10000 }).catch(() => {});
  expect((await pm.locator(`[data-card="${A.id}"] [data-row="${fresh}"]`).count()) === 1, "klien kedua", "Manajer tidak melihat baris baru");
  // Kunci lunak: Manajer membuka tabel yang sedang diedit hanya baca.
  await pm.locator(`[data-card="${A.id}"] [data-table-name]`).dblclick();
  await pm.waitForTimeout(400);
  expect((await pm.locator("[data-erd-editor]").count()) === 0, "kunci lunak", "Manajer bisa membuka editor tabel yang sedang diedit");

  // 3. Urutan: Alt+panah atas di baris p2 memindahnya ke atas p1; garis ikut ke baris pertama.
  await editor.locator('[data-edit-row="p2"] [data-field="name"]').click();
  await ps.keyboard.press("Alt+ArrowUp");
  await saved(ps);
  expect((await names(A.id)) === "pelanggan_id|id|total|dibuat_pada", "Alt+panah atas", await names(A.id));
  pf = await pathOf(fk.id);
  expect(near(pf[1], A.y + HEAD + ROW / 2), "garis ikut urutan", `pangkal y ${pf[1]}`);
  // Seret pegangan baris terakhir ke atas baris kedua.
  const grip = await editor.locator(`[data-edit-row="${fresh}"] [data-grip]`).boundingBox();
  const target = await editor.locator('[data-edit-row="p1"]').boundingBox();
  await ps.mouse.move(grip.x + grip.width / 2, grip.y + grip.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(grip.x + grip.width / 2, target.y + 4, { steps: 8 });
  await ps.mouse.up();
  await saved(ps);
  expect((await names(A.id)) === "pelanggan_id|dibuat_pada|id|total", "seret pegangan", await names(A.id));
  // Hapus baris dibuat_pada.
  await editor.getByRole("button", { name: "Hapus kolom dibuat_pada" }).click();
  await saved(ps);
  a = await cardOf(A.id);
  expect((await names(A.id)) === "pelanggan_id|id|total" && a.h === HEAD + ROW * 3, "hapus baris", `${await names(A.id)} tinggi ${a.h}`);
  await closeEditor();

  // 4. Undo per aksi: hapus, seret, Alt+panah, PK, ketik tipe, ketik nama, tambah baris. Undo isi berjalan async,
  // jadi tiap pemeriksaan menunggu sampai keadaan server sesuai (paling lama 6 detik).
  const until = async (pred) => {
    let c = await cardOf(A.id);
    for (let k = 0; k < 30 && !pred(c); k++) {
      await ps.waitForTimeout(200);
      c = await cardOf(A.id);
    }
    return c;
  };
  const rowNames = (c) => c.content.rows.map((r) => r.name).join("|");
  const step = async (k, pred, where) => {
    await key(k);
    const c = await until(pred);
    expect(pred(c), where, `${rowNames(c)} tinggi ${c.h} ${JSON.stringify(c.content.rows[3] ?? null)}`);
  };
  await step("Control+z", (c) => rowNames(c) === "pelanggan_id|dibuat_pada|id|total" && c.h === HEAD + ROW * 4, "undo hapus");
  await step("Control+z", (c) => rowNames(c) === "pelanggan_id|id|total|dibuat_pada", "undo seret");
  await step("Control+z", (c) => rowNames(c) === "id|pelanggan_id|total|dibuat_pada", "undo urutan");
  await ps.waitForTimeout(300);
  pf = await pathOf(fk.id);
  expect(near(pf[1], A.y + HEAD + ROW * 1.5), "garis kembali", `pangkal y ${pf[1]}`);
  await step("Control+z", (c) => c.content.rows[3]?.keys.length === 0, "undo PK");
  await step("Control+z", (c) => c.content.rows[3]?.type === "", "undo tipe");
  await step("Control+z", (c) => c.content.rows[3]?.name === "", "undo nama");
  await step("Control+z", (c) => c.content.rows.length === 3 && c.h === HEAD + ROW * 3, "undo tambah baris");
  await step("Control+Shift+z", (c) => c.content.rows.length === 4 && c.h === HEAD + ROW * 4, "redo tambah baris");
  await step("Control+z", (c) => c.content.rows.length === 3 && c.h === HEAD + ROW * 3, "undo lagi");

  // 5. Baris yang ditempeli garis dihapus: garis tetap ada, ujungnya di kotak tabel; undo mengembalikan ke baris.
  await openEditor(A.id);
  await editor.getByRole("button", { name: "Hapus kolom pelanggan_id" }).click();
  await saved(ps);
  await closeEditor();
  const still = (await view()).connectors.some((c) => c.id === fk.id);
  pf = await pathOf(fk.id);
  a = await cardOf(A.id);
  expect(still && near(pf[0], A.x + 260) && near(pf[1], A.y + a.h / 2), "baris dihapus, garis ke kotak", `ada ${still}, pangkal ${pf.slice(0, 2)}`);
  await key("Control+z");
  pf = await pathOf(fk.id);
  expect(near(pf[1], A.y + HEAD + ROW * 1.5), "undo hapus baris, garis kembali", `pangkal y ${pf[1]}`);

  // 6. Geser dan lebarkan tabel: ujung tetap di barisnya.
  await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: A.id, x: 40, y: 80, w: 300 }] });
  await ps.waitForTimeout(800);
  pf = await pathOf(fk.id);
  expect(near(pf[0], 340) && near(pf[1], 80 + HEAD + ROW * 1.5), "geser dan lebarkan", `pangkal ${pf.slice(0, 2)}`);

  // 7. Garis baru dari port kanan tabel B ke baris kedua tabel C: ujung akhir menempel di baris itu.
  await unselect();
  await ps.locator(`[data-card="${B.id}"] [data-table-name]`).click();
  const port = ps.locator('[data-testid=wb-diagram-handles] [data-port="right"] circle').last();
  const pb = await port.boundingBox();
  const rowC = await ps.locator(`[data-card="${C.id}"] [data-row="r2"]`).boundingBox();
  await ps.mouse.move(pb.x + pb.width / 2, pb.y + pb.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(rowC.x + rowC.width - 10, rowC.y + rowC.height / 2, { steps: 8 });
  await ps.mouse.up();
  await saved(ps);
  const toRow = (await view()).connectors.find((c) => c.from === B.id && c.to === C.id);
  expect(toRow?.geometry.to.row === "r2" && toRow.geometry.to.port === "right", "lepas di baris", JSON.stringify(toRow?.geometry.to));
  expect(toRow?.style.start === "cf_many" && toRow?.style.end === "cf_one", "ujung crow's foot awal", JSON.stringify(toRow?.style));

  // 8. Garis dari port baris di editor tabel C (baris id) ke baris nama tabel B.
  // Toast di bawah layar bisa menutupi pegangan port baris kiri; tunggu hilang dulu.
  await ps.locator("div[role=status].shadow-dialog").waitFor({ state: "detached", timeout: 10000 }).catch(() => {});
  await openEditor(C.id);
  const rp = await editor.locator('[data-edit-row="r1"] [data-row-port="left"]').boundingBox();
  const rowB = await ps.locator(`[data-card="${B.id}"] [data-row="c2"]`).boundingBox();
  await ps.mouse.move(rp.x + rp.width / 2, rp.y + rp.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(rowB.x + 10, rowB.y + rowB.height / 2, { steps: 10 });
  await ps.mouse.up();
  await saved(ps);
  const fromRow = (await view()).connectors.find((c) => c.from === C.id && c.to === B.id);
  expect(fromRow?.geometry.from.row === "r1" && fromRow.geometry.from.port === "left" && fromRow.geometry.to.row === "c2", "port baris di editor", JSON.stringify(fromRow?.geometry));
  expect(fromRow?.style.start === "cf_one" && fromRow?.style.end === "cf_many", "PK ke bukan PK dibalik", JSON.stringify(fromRow?.style));
  expect((await editor.count()) === 0, "editor tertutup", "editor masih terbuka sesudah menarik garis");

  // 9. Panel garis: pilihan baris untuk ujung awal.
  await unselect();
  await ps.locator(`[data-connector="${fk.id}"] path[data-hit]`).dispatchEvent("dblclick");
  const dlg = ps.getByRole("dialog", { name: "Garis penghubung" });
  await dlg.waitFor({ timeout: 5000 });
  await dlg.getByRole("button", { name: /^Menempel di awal/ }).click();
  await ps.getByTestId("wb-picker").getByRole("button", { name: "Kolom total, kanan", exact: true }).click();
  await saved(ps);
  await ps.keyboard.press("Escape");
  const moved = (await view()).connectors.find((c) => c.id === fk.id);
  expect(moved.geometry.from.row === "p3" && moved.geometry.from.port === "right", "panel pilih baris", JSON.stringify(moved.geometry.from));

  // 10. Tabel terkunci: tambah kolom ditolak dengan pesan, ganti nama kolom tetap bisa.
  await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: B.id, locked: true }] });
  await ps.waitForTimeout(800);
  await openEditor(B.id);
  await editor.getByRole("button", { name: "Tambah kolom" }).click();
  await ps.waitForTimeout(500);
  const toast = await ps.getByText(/Tabel terkunci/).count();
  expect(toast > 0 && (await cardOf(B.id)).content.rows.length === 2, "terkunci menolak tambah", `pesan ${toast}`);
  await editor.locator('[data-edit-row="c2"] [data-field="name"]').fill("nama_lengkap");
  await editor.locator('[data-edit-row="c2"] [data-field="name"]').press("Tab");
  await saved(ps);
  expect((await names(B.id)) === "id|nama_lengkap", "terkunci tetap bisa ganti nama", await names(B.id));
  await closeEditor();
  await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: B.id, locked: false }] });

  // 11. Salin tempel: tabel baru dengan kolom sama.
  await unselect();
  await ps.locator(`[data-card="${C.id}"] [data-table-name]`).click();
  await ps.keyboard.press("Control+c");
  await ps.keyboard.press("Control+v");
  await saved(ps);
  const copies = (await view()).cards.filter((c) => c.content?.name === "produk");
  expect(copies.length === 2 && copies.every((c) => c.content.rows.map((r) => r.name).join() === "id,nama"), "salin tempel", `${copies.length} tabel produk`);

  // 12. Template: tabel dan garis ke baris ikut tersalin.
  await sApi.call("PUT", `/whiteboard/boards/${board}/template`, { is_template: true });
  const holder2 = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r57a tujuan ${Date.now()}`, x: 10200, y: 12400, w: 200, h: 120 })).body.card;
  holders.push(holder2);
  const fromT = await sApi.call("POST", `/whiteboard/boards/${holder2.ref.id}/from-template`, { template_id: board, id: randomUUID(), x: 0, y: 0, w: 200, h: 120 });
  const copyBoard = fromT.body?.card?.ref?.id;
  const cv = copyBoard ? (await sApi.call("GET", `/whiteboard/boards/${copyBoard}`)).body.board : null;
  const tA = cv?.cards.find((c) => c.content?.name === "pesanan");
  const tFk = cv?.connectors.find((c) => c.from === tA?.id);
  expect(tA?.content.rows.length === 3 && tFk?.geometry.from.row === "p3", "template", JSON.stringify(tFk?.geometry.from));
  await sApi.call("PUT", `/whiteboard/boards/${board}/template`, { is_template: false });

  // 13. Muat ulang dan ekspor PNG.
  await ps.reload();
  await saved(ps);
  expect((await ps.locator(`[data-card="${A.id}"] [data-row]`).count()) === 3, "muat ulang", "baris hilang");
  await ps.getByRole("button", { name: "Ekspor", exact: true }).first().click();
  const ex = ps.getByRole("dialog", { name: "Ekspor papan" });
  await ex.waitFor({ timeout: 5000 });
  await ex.getByRole("radio", { name: /^PNG/ }).check();
  const [dl] = await Promise.all([ps.waitForEvent("download", { timeout: 20000 }), ex.getByRole("button", { name: "Unduh PNG" }).click()]);
  expect(dl.suggestedFilename().endsWith(".png"), "ekspor", dl.suggestedFilename());
  await ex.getByRole("button", { name: "Selesai" }).click();
  expect(errors.length === 0, "galat halaman", errors.join(" | "));

  // 14. 360 px: editor seluruhnya di dalam layar, tombol dan isian 44 px, tanpa gulir horizontal, Escape menutup.
  const nctx = await browser.newContext({ storageState: staf.state, viewport: { width: 360, height: 740 } });
  const pn = await nctx.newPage();
  pn.on("pageerror", (e) => errors.push(String(e)));
  await pn.goto(BASE + `/whiteboard/${board}`);
  await saved(pn);
  await pn.locator(`[data-card="${A.id}"] [data-table-name]`).dblclick();
  const ned = pn.locator("[data-erd-editor]");
  await ned.waitFor({ timeout: 5000 });
  const fit = await ned.evaluate((el) => {
    const b = el.getBoundingClientRect();
    const items = [...el.querySelectorAll("button, input")].map((x) => x.getBoundingClientRect());
    return {
      inside: b.left >= 0 && b.right <= innerWidth && b.top >= 0,
      small: items.filter((r) => r.width < 43.5 || r.height < 43.5).length,
      out: items.filter((r) => r.left < 0 || r.right > innerWidth).length,
      scroll: document.documentElement.scrollWidth > innerWidth,
      box: [Math.round(b.left), Math.round(b.right), Math.round(b.top)],
    };
  });
  expect(fit.inside && fit.out === 0, "360 di dalam layar", JSON.stringify(fit));
  expect(fit.small === 0, "360 tap 44 px", `${fit.small} elemen di bawah 44 px`);
  expect(!fit.scroll, "360 gulir horizontal", "halaman bergulir mendatar");
  await pn.keyboard.press("Escape");
  await ned.waitFor({ state: "detached", timeout: 5000 }).catch(() => {});
  expect((await ned.count()) === 0, "360 Escape", "editor tidak tertutup");
  await nctx.close();
  expect(errors.length === 0, "galat halaman 360", errors.join(" | "));
  await mctx.close();
  await ctx.close();
} finally {
  for (const f of fails) console.log("GAGAL", f);
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
