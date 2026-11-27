// Uji browser revisi 54d: sequence diagram, plus port aktor di ujung tangan.
// 1280 px: dua lifeline dan satu lifeline aktor di dalam fragmen alt. Pesan dari port lifeline ke lifeline lurus
// mendatar; preset balasan; seret pesan ke bawah tetap mendatar (Ctrl+Z kembali); pesan ke diri sendiri dari menu;
// lifeline digeser membawa pesannya; activation bar yang dilepas di dekat lifeline menempel di garis tengahnya dan ikut
// saat lifeline digeser; lifeline hanya punya pegangan bawah dan memanjang ke bawah; preset destroy memberi X di
// lifeline aktor dalam satu langkah undo; fragmen alt dengan dua operand dan Tambah operand. Port kanan aktor UML dan
// lifeline aktor di ujung tangan, juga sesudah aktor diputar 90 derajat. Muat ulang tetap; klien kedua melihat pesan ke
// diri sendiri dan X; ekspor PNG tanpa galat. 360 px: palet menemukan bentuk sequence.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r54d.mjs
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
const shape = (s, text, extra = {}) => ({ shape: s, rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text, align: "center", font_size: 16, ...extra });
const nums = (d) => (d.match(/-?\d+(\.\d+)?/g) ?? []).map(Number);

try {
  // 360: palet.
  {
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54d palet ${Date.now()}`, x: 9900, y: 10600, w: 200, h: 120 })).body.card;
    holders.push(holder);
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: 360, height: 800 } });
    const ps = await ctx.newPage();
    await ps.goto(BASE + `/whiteboard/${holder.ref.id}`);
    await saved(ps);
    await ps.getByRole("button", { name: "Tambah", exact: true }).click();
    await ps.locator("dialog[open]").getByRole("button", { name: "Bentuk", exact: true }).click();
    const dlg = ps.getByRole("dialog", { name: "Pilih bentuk" });
    await dlg.waitFor({ timeout: 5000 });
    await ps.waitForTimeout(300);
    await dlg.getByRole("combobox", { name: "Cari bentuk" }).fill("lifeline");
    const o = await dlg.getByRole("option").allTextContents();
    expect(o.length >= 4, "360 palet", o.join(", "));
    await dlg.getByRole("combobox", { name: "Cari bentuk" }).fill("fragmen");
    expect((await dlg.getByRole("option").count()) === 4, "360 palet fragmen", "fragmen alt, opt, loop, par tidak lengkap");
    const lay = await layout(ps);
    expect(!lay.overflow && lay.small.length === 0, "360 palet tata letak", JSON.stringify(lay));
    await ctx.close();
  }

  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54d ${Date.now()}`, x: 9900, y: 10800, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const frag = await add(-260, 120, 820, 360, shape("uml_fragment", "", { variant: "alt", fill: "", lanes: [{ label: "stok ada", size: 120 }, { label: "habis", size: 120 }] }));
  const C = await add(-220, 0, 100, 520, shape("uml_lifeline_actor", "Pembeli", { variant: "none", fill: "" }));
  const A = await add(0, 0, 140, 520, shape("uml_lifeline", "Toko", { variant: "none" }));
  const B = await add(320, 0, 140, 520, shape("uml_lifeline", "Gudang", { variant: "none" }));
  // Activation bar sedikit di luar garis tengah Gudang; seret pertama menempelkannya.
  const bar = await add(395, 160, 40, 120, shape("uml_activation", ""));
  const actor = await add(700, 0, 80, 160, shape("uml_actor", "Kurir", { fill: "" }));
  const note = await add(900, 20, 140, 60, shape("rect", "Catatan"));
  const D = await add(-500, 20, 120, 60, shape("rect", "Sistem lain"));
  const lines = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.connectors;
  const cards = async () => new Map((await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.map((c) => [c.id, c]));
  const reply = (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: B.id, to: A.id, style: { start: "none", end: "arrow", line: "solid", color: "" }, geometry: { from: { port: "auto" }, to: { port: "auto" }, route: "message", points: [[0, 300]] } })).body.connector;
  const toC = (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: A.id, to: C.id, style: { start: "none", end: "arrow", line: "solid", color: "" }, geometry: { from: { port: "auto" }, to: { port: "auto" }, route: "message", points: [[0, 420]] } })).body.connector;
  // Garis biasa dari tangan kanan aktor UML dan lifeline aktor.
  const handA = (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: actor.id, to: note.id, geometry: { from: { port: "right" }, to: { port: "left" }, route: "straight", points: [] } })).body.connector;
  const handC = (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: C.id, to: D.id, geometry: { from: { port: "left" }, to: { port: "right" }, route: "straight", points: [] } })).body.connector;

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
  const key = async (k) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press(k);
    await saved(ps);
    await ps.waitForTimeout(400);
  };
  const pathOf = async (page, id) => nums((await page.locator(`[data-connector="${id}"] path`).first().getAttribute("d")) ?? "");
  const flat = (n) => n.length >= 4 && n.filter((_, i) => i % 2 === 1).every((y) => Math.abs(y - n[1]) < 0.01);
  const box = (id) => ps.locator(`[data-card="${id}"]`).boundingBox();
  const unselect = async () => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
  };

  // Port aktor: ujung tangan kanan aktor UML (80 x 160) di (72, 56); tangan kiri lifeline aktor (100 x 520) di (10, 24.5).
  const pa = await pathOf(ps, handA.id);
  expect(Math.abs(pa[0] - (actor.x + 72)) < 0.6 && Math.abs(pa[1] - (actor.y + 56)) < 0.6, "port aktor", `pangkal ${pa.slice(0, 2)}`);
  const pc = await pathOf(ps, handC.id);
  expect(Math.abs(pc[0] - (C.x + 10)) < 0.6 && Math.abs(pc[1] - (C.y + 24.5)) < 0.6, "port lifeline aktor", `pangkal ${pc.slice(0, 2)}`);
  await ps.locator(`[data-card="${actor.id}"]`).click();
  await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
  await ps.getByTestId("wb-menu").getByRole("menuitem", { name: /^Putar 90 derajat/ }).click();
  await saved(ps);
  // Diputar terhadap pusat (40, 80): tangan kanan ke (64, 112).
  const pr = await pathOf(ps, handA.id);
  expect(Math.abs(pr[0] - (actor.x + 64)) < 0.6 && Math.abs(pr[1] - (actor.y + 112)) < 0.6, "port aktor diputar", `pangkal ${pr.slice(0, 2)}`);
  await unselect();

  // 1. Pesan dari port Toko ke Gudang: lurus mendatar.
  await ps.locator(`[data-card="${A.id}"]`).click({ position: { x: 20, y: 10 } });
  const port = ps.locator('[data-testid=wb-diagram-handles] [data-port="right"] circle').last();
  const pb = await port.boundingBox();
  const bb = await box(B.id);
  await ps.mouse.move(pb.x + pb.width / 2, pb.y + pb.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(bb.x + bb.width / 2, bb.y + bb.height * 0.25, { steps: 8 });
  await ps.mouse.up();
  await saved(ps);
  const made = (await lines()).find((l) => l.from === A.id && l.to === B.id);
  expect(made?.geometry.route === "message", "pesan dari port", JSON.stringify(made?.geometry));
  const m1 = await pathOf(ps, made.id);
  expect(flat(m1), "pesan mendatar", `path ${m1}`);
  await unselect();

  // 2. Preset balasan.
  await ps.locator(`[data-connector="${reply.id}"] path[data-hit]`).dispatchEvent("dblclick");
  const dlg = ps.getByRole("dialog", { name: "Garis penghubung" });
  await dlg.waitFor({ timeout: 5000 });
  await dlg.getByRole("button", { name: /^Preset garis/ }).click();
  await ps.getByTestId("wb-picker").getByRole("button", { name: "Sequence: Balasan", exact: true }).click();
  await saved(ps);
  await ps.keyboard.press("Escape");
  const rs = (await lines()).find((l) => l.id === reply.id).style;
  expect(rs.line === "dashed" && rs.end === "open_arrow", "preset balasan", JSON.stringify(rs));

  // 3. Seret pesan ke bawah: tetap mendatar; Ctrl+Z kembali.
  const y0 = (await lines()).find((l) => l.id === reply.id).geometry.points[0][1];
  await ps.locator(`[data-connector="${reply.id}"] path[data-hit]`).dispatchEvent("pointerdown");
  await ps.waitForTimeout(150);
  expect((await ps.locator("[data-testid=wb-diagram-handles] [data-insert]").count()) === 0, "pegangan pesan", "titik sisip muncul di pesan");
  const h = ps.locator("[data-testid=wb-diagram-handles] [data-bend] circle").last();
  const hb = await h.boundingBox();
  await ps.mouse.move(hb.x + hb.width / 2, hb.y + hb.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(hb.x + hb.width / 2 + 30, hb.y + hb.height / 2 + 30, { steps: 6 });
  await ps.mouse.up();
  await saved(ps);
  const y1 = (await lines()).find((l) => l.id === reply.id).geometry.points[0][1];
  expect(y1 > y0 && flat(await pathOf(ps, reply.id)), "seret pesan", `tinggi ${y0} -> ${y1}, path ${await pathOf(ps, reply.id)}`);
  await key("Control+z");
  expect((await lines()).find((l) => l.id === reply.id).geometry.points[0][1] === y0, "seret pesan undo", "tinggi pesan tidak kembali");

  // 4. Pesan ke diri sendiri dari menu.
  await unselect();
  await ps.locator(`[data-card="${A.id}"]`).click({ position: { x: 20, y: 10 } });
  await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
  await ps.getByTestId("wb-menu").getByRole("menuitem", { name: "Tambah pesan ke diri sendiri" }).click();
  await saved(ps);
  const self = (await lines()).find((l) => l.from === A.id && l.to === A.id);
  expect(!!self && (await pathOf(ps, self.id)).length === 8, "pesan ke diri sendiri", JSON.stringify(self?.geometry));

  // 5. Activation bar menempel di garis tengah Gudang saat dilepas, lalu ikut saat Gudang digeser.
  await unselect();
  const barB = await box(bar.id);
  await ps.mouse.move(barB.x + barB.width / 2, barB.y + barB.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(barB.x + barB.width / 2 - 2, barB.y + barB.height / 2 + 12, { steps: 4 });
  await ps.mouse.up();
  await saved(ps);
  let c1 = await cards();
  expect(c1.get(bar.id).x + 20 === B.x + 70, "activation menempel", `pusat bar ${c1.get(bar.id).x + 20}, garis ${B.x + 70}`);
  const reply0 = await pathOf(ps, reply.id);
  await unselect();
  const lb = await box(B.id);
  await ps.mouse.move(lb.x + 10, lb.y + 8);
  await ps.mouse.down();
  await ps.mouse.move(lb.x + 10, lb.y + 48, { steps: 6 });
  await ps.mouse.up();
  await saved(ps);
  await ps.waitForTimeout(300);
  const c2 = await cards();
  const dyB = c2.get(B.id).y - c1.get(B.id).y;
  expect(dyB !== 0 && c2.get(bar.id).y - c1.get(bar.id).y === dyB, "activation ikut", `lifeline ${dyB}, bar ${c2.get(bar.id).y - c1.get(bar.id).y}`);
  // Tinggi pesan mengikuti lifeline asal: balasan dari Gudang turun sejauh Gudang digeser.
  const reply1 = await pathOf(ps, reply.id);
  expect(flat(reply1) && Math.abs(reply1[1] - reply0[1] - dyB) < 0.6, "pesan ikut lifeline", `balasan ${reply0[1]} -> ${reply1[1]}, lifeline ${dyB}`);
  await key("Control+z");

  // 6. Lifeline hanya punya pegangan bawah dan memanjang ke bawah.
  await unselect();
  await ps.locator(`[data-card="${A.id}"]`).click({ position: { x: 20, y: 10 } });
  const hs = await ps.locator(`[data-card="${A.id}"] [data-handle]`).evaluateAll((els) => els.map((e) => e.dataset.handle));
  expect(JSON.stringify(hs.filter((x) => x !== "rotate")) === '["s"]', "pegangan lifeline", hs.join(","));
  const sh = ps.locator(`[data-card="${A.id}"] [data-handle="s"]`);
  const sb = await sh.boundingBox();
  const a0 = (await cards()).get(A.id);
  await ps.mouse.move(sb.x + sb.width / 2, sb.y + sb.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(sb.x + sb.width / 2, sb.y + sb.height / 2 + 40, { steps: 6 });
  await ps.mouse.up();
  await saved(ps);
  const a1 = (await cards()).get(A.id);
  expect(a1.h > a0.h && a1.y === a0.y, "lifeline memanjang", `y ${a0.y} -> ${a1.y}, h ${a0.h} -> ${a1.h}`);

  // 7. Preset destroy ke lifeline aktor: label, gaya, dan X dalam satu langkah undo.
  await unselect();
  await ps.locator(`[data-connector="${toC.id}"] path[data-hit]`).dispatchEvent("dblclick");
  const d2 = ps.getByRole("dialog", { name: "Garis penghubung" });
  await d2.waitFor({ timeout: 5000 });
  await d2.getByRole("button", { name: /^Preset garis/ }).click();
  await ps.getByTestId("wb-picker").getByRole("button", { name: "Sequence: Destroy", exact: true }).click();
  await saved(ps);
  await ps.keyboard.press("Escape");
  await ps.waitForTimeout(300);
  const cC = (await cards()).get(C.id);
  expect(cC.content.variant === "destroyed" && (await lines()).find((l) => l.id === toC.id).label === "«destroy»", "destroy", JSON.stringify(cC.content.variant));
  await pm.waitForFunction((id) => document.querySelectorAll(`[data-card="${id}"] [data-mark]`).length === 2, C.id, { timeout: 10000 }).catch(() => {});
  expect((await pm.locator(`[data-card="${C.id}"] [data-mark]`).count()) === 2, "destroy klien kedua", "Manajer tidak melihat X");
  await key("Control+z");
  expect((await cards()).get(C.id).content.variant === "none" && (await lines()).find((l) => l.id === toC.id).label === "", "destroy undo", "satu Ctrl+Z tidak mengembalikan X dan label");
  await key("Control+y");

  // 8. Fragmen alt: operator, syarat, dan Tambah operand.
  await unselect();
  expect((await ps.locator(`[data-card="${frag.id}"] [data-fragment-op]`).textContent())?.trim() === "alt", "fragmen", "tab operator tidak alt");
  const guards = await ps.locator(`[data-card="${frag.id}"] [data-lane]`).allTextContents();
  expect(guards.map((t) => t.trim()).join("|") === "[stok ada]|[habis]", "fragmen syarat", guards.join("|"));
  const fb = await box(frag.id);
  await ps.mouse.click(fb.x + fb.width - 10, fb.y + fb.height - 10);
  await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
  await ps.getByTestId("wb-menu").getByRole("menuitem", { name: "Tambah operand" }).click();
  await saved(ps);
  expect((await cards()).get(frag.id).content.lanes.length === 3, "tambah operand", "operand tidak bertambah");

  // 9. Muat ulang, klien kedua, ekspor.
  await ps.reload();
  await saved(ps);
  expect((await pathOf(ps, self.id)).length === 8 && (await ps.locator(`[data-card="${C.id}"] [data-mark]`).count()) === 2, "muat ulang", "pesan ke diri sendiri atau X hilang");
  await pm.locator(`[data-connector="${self.id}"]`).waitFor({ timeout: 10000 }).catch(() => {});
  expect((await pm.locator(`[data-connector="${self.id}"]`).count()) === 1, "klien kedua", "Manajer tidak melihat pesan ke diri sendiri");
  await ps.getByRole("button", { name: "Ekspor", exact: true }).first().click();
  const ex = ps.getByRole("dialog", { name: "Ekspor papan" });
  await ex.waitFor({ timeout: 5000 });
  await ex.getByRole("radio", { name: /^PNG/ }).check();
  const [dl] = await Promise.all([ps.waitForEvent("download", { timeout: 20000 }), ex.getByRole("button", { name: "Unduh PNG" }).click()]);
  expect(dl.suggestedFilename().endsWith(".png"), "ekspor", dl.suggestedFilename());
  await ex.getByRole("button", { name: "Selesai" }).click();
  expect(errors.length === 0, "galat halaman", errors.join(" | "));
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
