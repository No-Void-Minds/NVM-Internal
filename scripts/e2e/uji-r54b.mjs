// Uji browser revisi 54b: wadah (pool dan lane, swimlane, batas sistem).
// 1280 px: proses BPMN di pool dua lane di dalam batas sistem. Seret pool membawa event, task, gateway, dan garisnya
// (satu Ctrl+Z mengembalikan semuanya); kartu di luar dan kartu di bawah pool tidak ikut; task yang diseret keluar
// tidak lagi ikut; kartu yang dilepas di dalam pool naik ke atasnya dan ikut; anggota terkunci membatalkan seret;
// batas sistem membawa pool beserta isinya. Pembatas lane diseret dan digeser dengan panah, Tambah lane satu langkah
// undo, label lane lewat Gaya bentuk; muat ulang tetap; klien kedua melihat lane dan posisi; ekspor PNG tanpa galat.
// 360 px: palet menemukan pool dan swimlane, tanpa gulir horizontal.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r54b.mjs
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

try {
  // 360: palet.
  {
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: 360, height: 800 } });
    const ps = await ctx.newPage();
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54b palet ${Date.now()}`, x: 9900, y: 9800, w: 200, h: 120 })).body.card;
    holders.push(holder);
    await ps.goto(BASE + `/whiteboard/${holder.ref.id}`);
    await saved(ps);
    await ps.getByRole("button", { name: "Tambah", exact: true }).click();
    await ps.locator("dialog[open]").getByRole("button", { name: "Bentuk", exact: true }).click();
    const dlg = ps.getByRole("dialog", { name: "Pilih bentuk" });
    await dlg.waitFor({ timeout: 5000 });
    await ps.waitForTimeout(300);
    await dlg.getByRole("combobox", { name: "Cari bentuk" }).fill("lane");
    const opts = await dlg.getByRole("option").allTextContents();
    expect(opts.some((t) => /Pool dan lane/.test(t)) && opts.some((t) => /Swimlane/.test(t)), "360 palet", opts.join(", "));
    const lay = await layout(ps);
    expect(!lay.overflow && lay.small.length === 0, "360 palet tata letak", JSON.stringify(lay));
    await ctx.close();
  }

  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54b ${Date.now()}`, x: 9900, y: 10000, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content, kind = "shape") => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind, x, y, w, h, content })).body.card;
  const noteC = (t) => ({ doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: t }] }] } });
  // Kartu di bawah pool: dibuat sebelum pool, jadi z-nya lebih rendah.
  const bawah = await add(600, 240, 120, 60, noteC("Di bawah"), "note");
  const sys = await add(0, 0, 1200, 700, shape("uml_system", "Toko daring", { fill: "" }));
  await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: bawah.id, z: -5 }, { id: sys.id, z: -10 }] });
  const pool = await add(40, 60, 840, 340, shape("bpmn_pool", "Pemesanan", { fill: "", lanes: [{ label: "Penjual", size: 150 }, { label: "Gudang", size: 150 }] }));
  const start = await add(110, 110, 56, 56, shape("bpmn_start", "Pesanan", { variant: "none" }));
  const task = await add(220, 93, 160, 90, shape("bpmn_task", "Periksa stok", { variant: "user" }));
  const gw = await add(460, 270, 64, 64, shape("bpmn_gateway", "Ada?", { variant: "exclusive" }));
  const end = await add(640, 274, 56, 56, shape("bpmn_end", "Selesai", { variant: "none" }));
  const luar = await add(950, 100, 160, 90, noteC("Di luar pool"), "note");
  const masuk = await add(950, 450, 120, 60, noteC("Masuk"), "note");
  const link = async (from, to) => (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from, to })).body.connector;
  const l1 = await link(start.id, task.id);
  await link(task.id, gw.id);
  await link(gw.id, end.id);
  const cards = async () => new Map((await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.map((c) => [c.id, c]));

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
  await pm.locator(`[data-card="${pool.id}"]`).waitFor({ timeout: 10000 });

  const key = async (k) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press(k);
    await saved(ps);
    await ps.waitForTimeout(400);
  };
  // Seret dari titik layar (fx, fy relatif kotak kartu) sejauh dx, dy px layar.
  const dragCard = async (id, fx, fy, dx, dy) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
    const b = await ps.locator(`[data-card="${id}"]`).boundingBox();
    const x = b.x + b.width * fx;
    const y = b.y + b.height * fy;
    await ps.mouse.move(x, y);
    await ps.mouse.down();
    await ps.mouse.move(x + dx / 2, y + dy / 2, { steps: 5 });
    await ps.mouse.move(x + dx, y + dy, { steps: 5 });
    await ps.mouse.up();
    await saved(ps);
    await ps.waitForTimeout(400);
  };
  const delta = (a, b, id) => [b.get(id).x - a.get(id).x, b.get(id).y - a.get(id).y];
  const same = (p, q) => p[0] === q[0] && p[1] === q[1];

  // 1. Seret pool dari pita namanya: anggota ikut, luar dan bawah tidak; satu Ctrl+Z.
  const c0 = await cards();
  const line0 = await ps.locator(`[data-connector="${l1.id}"] path`).first().getAttribute("d");
  await dragCard(pool.id, 0.02, 0.5, 80, 40);
  const c1 = await cards();
  const dPool = delta(c0, c1, pool.id);
  expect(dPool[0] !== 0 || dPool[1] !== 0, "seret pool", "pool tidak bergeser");
  for (const [name, id] of [["event mulai", start.id], ["task", task.id], ["gateway", gw.id], ["event selesai", end.id]])
    expect(same(delta(c0, c1, id), dPool), "seret pool", `${name} bergeser ${delta(c0, c1, id)} bukan ${dPool}`);
  expect(same(delta(c0, c1, luar.id), [0, 0]) && same(delta(c0, c1, bawah.id), [0, 0]), "seret pool", "kartu di luar atau di bawah pool ikut bergeser");
  expect((await ps.locator(`[data-connector="${l1.id}"] path`).first().getAttribute("d")) !== line0, "seret pool", "garis di dalam pool tidak ikut");
  await pm.waitForFunction(([id, x]) => {
    const el = document.querySelector(`[data-card="${id}"]`);
    return el && getComputedStyle(el).transform.includes(String(x));
  }, [task.id, c1.get(task.id).x], { timeout: 10000 }).catch(() => {});
  await key("Control+z");
  const c2 = await cards();
  expect([pool.id, start.id, task.id, gw.id, end.id].every((id) => same(delta(c0, c2, id), [0, 0])), "seret pool undo", "satu Ctrl+Z tidak mengembalikan pool dan isinya");

  // 2. Task diseret keluar pool: seret pool berikutnya tidak membawanya.
  await dragCard(task.id, 0.5, 0.5, 0, 400);
  const c3 = await cards();
  await dragCard(pool.id, 0.02, 0.5, 60, 0);
  const c4 = await cards();
  expect(same(delta(c3, c4, task.id), [0, 0]) && !same(delta(c3, c4, start.id), [0, 0]), "keluar pool", "task yang sudah di luar masih ikut atau isi lain tidak ikut");
  await key("Control+z");
  await key("Control+z");

  // 3. Kartu yang dilepas di dalam pool naik ke atasnya dan menjadi anggota.
  const pb = await ps.locator(`[data-card="${pool.id}"]`).boundingBox();
  const mb = await ps.locator(`[data-card="${masuk.id}"]`).boundingBox();
  await dragCard(masuk.id, 0.5, 0.5, pb.x + pb.width * 0.75 - (mb.x + mb.width / 2), pb.y + pb.height * 0.3 - (mb.y + mb.height / 2));
  const c5 = await cards();
  expect(c5.get(masuk.id).z > c5.get(pool.id).z, "masuk pool", `z ${c5.get(masuk.id).z} tidak di atas pool ${c5.get(pool.id).z}`);
  await dragCard(pool.id, 0.02, 0.5, 0, 40);
  const c6 = await cards();
  expect(same(delta(c5, c6, masuk.id), delta(c5, c6, pool.id)), "masuk pool", "kartu yang dilepas di dalam pool tidak ikut terseret");
  await key("Control+z");

  // 4. Anggota terkunci: seret pool dibatalkan utuh dengan pesan kunci.
  await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: gw.id, locked: true }] });
  await ps.waitForTimeout(800);
  const c7 = await cards();
  await dragCard(pool.id, 0.02, 0.5, 60, 60);
  const c8 = await cards();
  expect(same(delta(c7, c8, pool.id), [0, 0]) && same(delta(c7, c8, start.id), [0, 0]), "anggota terkunci", "pool atau isinya bergeser padahal ada anggota terkunci");
  expect(await ps.getByText(/terkunci\. Buka kunci dulu/).first().isVisible().catch(() => false), "anggota terkunci", "pesan kunci tidak tampil");
  await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: gw.id, locked: false }] });
  await ps.waitForTimeout(800);

  // 5. Batas sistem membawa pool beserta isinya.
  const c9 = await cards();
  await dragCard(sys.id, 0.5, 0.03, 0, 50);
  const c10 = await cards();
  const dSys = delta(c9, c10, sys.id);
  expect(!same(dSys, [0, 0]) && [pool.id, start.id, gw.id, luar.id].every((id) => same(delta(c9, c10, id), dSys)), "wadah bersarang", `sistem ${dSys}, pool ${delta(c9, c10, pool.id)}, gateway ${delta(c9, c10, gw.id)}`);
  await key("Control+z");

  // 6. Pembatas lane: seret dan panah; Ctrl+Z mengembalikan.
  await ps.locator(".wb-canvas").focus();
  await ps.keyboard.press("Escape");
  const pb2 = await ps.locator(`[data-card="${pool.id}"]`).boundingBox();
  await ps.mouse.click(pb2.x + 8, pb2.y + pb2.height / 2);
  const div = ps.locator(`[data-card="${pool.id}"] [data-lane-divider]`).first();
  await div.waitFor({ timeout: 5000 });
  const db = await div.boundingBox();
  expect(db.height >= 43.5, "pembatas", `area pembatas ${db.height} px`);
  const lanes0 = (await cards()).get(pool.id).content.lanes;
  await ps.mouse.move(db.x + db.width / 2, db.y + db.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(db.x + db.width / 2, db.y + db.height / 2 + 30, { steps: 6 });
  await ps.mouse.up();
  await saved(ps);
  const lanes1 = (await cards()).get(pool.id).content.lanes;
  expect(lanes1[0].size > lanes0[0].size && lanes1[1].size < lanes0[1].size, "seret pembatas", JSON.stringify(lanes1));
  await div.focus();
  await ps.keyboard.press("ArrowUp");
  await saved(ps);
  const lanes2 = (await cards()).get(pool.id).content.lanes;
  expect(lanes2[0].size < lanes1[0].size, "panah pembatas", JSON.stringify(lanes2));
  await key("Control+z");
  await key("Control+z");
  expect(JSON.stringify((await cards()).get(pool.id).content.lanes) === JSON.stringify(lanes0), "pembatas undo", "ukuran lane tidak kembali");

  // 7. Tambah lane lewat menu Aksi: tiga lane dan pool lebih tinggi; satu Ctrl+Z mengembalikan keduanya.
  const h0 = (await cards()).get(pool.id).h;
  await ps.mouse.click(pb2.x + 8, pb2.y + pb2.height / 2);
  await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
  await ps.getByTestId("wb-menu").getByRole("menuitem", { name: "Tambah lane" }).click();
  await saved(ps);
  await ps.waitForTimeout(400);
  const p3 = (await cards()).get(pool.id);
  expect(p3.content.lanes.length === 3 && p3.h > h0, "tambah lane", `lane ${p3.content.lanes.length}, tinggi ${h0} -> ${p3.h}`);
  await pm.waitForFunction((id) => document.querySelectorAll(`[data-card="${id}"] [data-lane]`).length === 3, pool.id, { timeout: 10000 }).catch(() => {});
  expect((await pm.locator(`[data-card="${pool.id}"] [data-lane]`).count()) === 3, "tambah lane klien kedua", "Manajer tidak melihat lane ketiga");
  await key("Control+z");
  const p4 = (await cards()).get(pool.id);
  expect(p4.content.lanes.length === 2 && p4.h === h0, "tambah lane undo", `lane ${p4.content.lanes.length}, tinggi ${p4.h}`);
  await key("Control+y");
  expect((await cards()).get(pool.id).content.lanes.length === 3, "tambah lane redo", "Ctrl+Y tidak menambah lane lagi");

  // 8. Label lane lewat Gaya bentuk.
  await ps.mouse.click(pb2.x + 8, pb2.y + pb2.height / 2);
  await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
  await ps.getByTestId("wb-menu").getByRole("menuitem", { name: /^Gaya bentuk/ }).click();
  const style = ps.getByRole("dialog", { name: "Gaya bentuk" });
  await style.waitFor({ timeout: 5000 });
  await style.getByLabel("Label lane 1").fill("Pembeli");
  await style.getByLabel("Label lane 1").press("Tab");
  await saved(ps);
  await ps.keyboard.press("Escape");
  expect((await cards()).get(pool.id).content.lanes[0].label === "Pembeli", "label lane", "label tidak tersimpan");
  await pm.getByText("Pembeli").first().waitFor({ timeout: 10000 }).catch(() => {});
  expect(await pm.getByText("Pembeli").first().isVisible().catch(() => false), "label lane klien kedua", "Manajer tidak melihat label lane");

  // 9. Muat ulang dan ekspor.
  await ps.reload();
  await saved(ps);
  expect((await ps.locator(`[data-card="${pool.id}"] [data-lane]`).count()) === 3, "muat ulang", "lane berubah sesudah muat ulang");
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
