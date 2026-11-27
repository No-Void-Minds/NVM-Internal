// Uji browser revisi 56 di 1280 px: lane langsung di kanvas.
// Pool dua lane berisi task: ganti nama lane dengan dobel klik (Enter simpan) dan keyboard (fokus, Enter, Escape
// batal, Enter simpan), nama pool dengan dobel klik lalu klik di luar; tombol tambah lane (44 px, berlabel) lalu lane
// baru langsung bisa diberi nama; urutan lane lewat klik kanan (task ikut lane-nya); hapus lane berisi task lewat klik
// kanan dengan konfirmasi (task tetap ada di posisinya); undo dan redo tiap langkah; kunci lunak orang lain menolak
// edit nama. Swimlane: ganti nama dan tombol tambah di tepi kanan. Fragmen: syarat operand dengan dobel klik. Muat
// ulang tetap; klien kedua melihat nama dan urutan lane.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r56.mjs
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

const staf = await session(browser, BASE, "staf");
const manajer = await session(browser, BASE, "manajer");
const sApi = await api(browser, BASE, staf.state);
const holders = [];
const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
const shape = (s, text, extra = {}) => ({ shape: s, rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text, align: "center", font_size: 16, ...extra });

try {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r56 ${Date.now()}`, x: 9900, y: 11800, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const pool = await add(0, 0, 700, 300, shape("bpmn_pool", "Pemesanan", { fill: "", lanes: [{ label: "Penjual", size: 150 }, { label: "Gudang", size: 150 }] }));
  const task = await add(300, 190, 160, 80, shape("bpmn_task", "Kemas", { variant: "none" }));
  const swim = await add(0, 420, 480, 320, shape("uml_swimlane", "Alur", { fill: "", lanes: [{ label: "Pelanggan", size: 150 }, { label: "Toko", size: 150 }] }));
  const frag = await add(560, 420, 420, 240, shape("uml_fragment", "", { variant: "alt", fill: "", lanes: [{ label: "ya", size: 120 }, { label: "tidak", size: 120 }] }));
  const cards = async () => new Map((await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.map((c) => [c.id, c]));
  const lanesOf = async (id) => (await cards()).get(id).content.lanes.map((l) => l.label).join("|");

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
  const label = (id, i) => ps.locator(`[data-card="${id}"] [data-lane-index="${i}"]`);
  const input = ps.locator("[data-lane-input]");
  const selectCard = async (id) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
    const b = await ps.locator(`[data-card="${id}"]`).boundingBox();
    await ps.mouse.click(b.x + b.width - 6, b.y + 6);
  };

  // 1. Dobel klik label lane, Enter menyimpan; undo dan redo; klien kedua.
  await label(pool.id, 0).dblclick();
  await input.waitFor({ timeout: 5000 });
  await input.fill("Pembeli");
  await input.press("Enter");
  await saved(ps);
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang", "ganti nama dobel klik", await lanesOf(pool.id));
  await pm.getByText("Pembeli").first().waitFor({ timeout: 10000 }).catch(() => {});
  expect(await pm.getByText("Pembeli").first().isVisible().catch(() => false), "ganti nama klien kedua", "Manajer tidak melihat nama lane");
  await key("Control+z");
  expect((await lanesOf(pool.id)) === "Penjual|Gudang", "ganti nama undo", await lanesOf(pool.id));
  await key("Control+y");
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang", "ganti nama redo", await lanesOf(pool.id));

  // 2. Keyboard: fokus label, Enter, Escape batal; Enter, ketik, Enter simpan.
  await selectCard(pool.id);
  await label(pool.id, 1).focus();
  await ps.keyboard.press("Enter");
  await input.waitFor({ timeout: 5000 });
  await input.fill("Batal saja");
  await input.press("Escape");
  await ps.waitForTimeout(300);
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang" && !(await input.isVisible().catch(() => false)), "keyboard Escape", await lanesOf(pool.id));
  await label(pool.id, 1).focus();
  await ps.keyboard.press("Enter");
  await input.fill("Gudang utama");
  await input.press("Enter");
  await saved(ps);
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang utama", "keyboard Enter", await lanesOf(pool.id));

  // 3. Nama pool: dobel klik pita nama, klik di luar menyimpan.
  // Bagian atas pita: titik tengahnya tertutup pegangan ubah ukuran kiri saat pool terpilih.
  const tb = await ps.locator(`[data-card="${pool.id}"] [data-lane-title]`).boundingBox();
  await ps.mouse.dblclick(tb.x + tb.width / 2, tb.y + tb.height * 0.3);
  await input.waitFor({ timeout: 5000 });
  await input.fill("Pesanan daring");
  const cv = await ps.locator(".wb-canvas").boundingBox();
  await ps.mouse.click(cv.x + cv.width - 20, cv.y + cv.height - 120);
  await saved(ps);
  expect((await cards()).get(pool.id).content.text === "Pesanan daring", "nama pool", (await cards()).get(pool.id).content.text);

  // 4. Tombol tambah lane: 44 px, berlabel; lane baru langsung diberi nama; dua undo mengembalikan.
  await selectCard(pool.id);
  const addBtn = ps.locator(`[data-card="${pool.id}"] [data-lane-add]`);
  await addBtn.waitFor({ timeout: 5000 });
  const ab = await addBtn.boundingBox();
  expect(ab.width >= 43.5 && ab.height >= 43.5, "tombol tambah", `${ab.width} x ${ab.height}`);
  expect((await addBtn.getAttribute("aria-label")) === "Tambah lane", "tombol tambah label", await addBtn.getAttribute("aria-label"));
  await addBtn.focus();
  await ps.keyboard.press("Enter");
  await input.waitFor({ timeout: 5000 });
  await input.fill("Kurir");
  await input.press("Enter");
  await saved(ps);
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang utama|Kurir", "tambah lane", await lanesOf(pool.id));
  await key("Control+z");
  await key("Control+z");
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang utama" && (await cards()).get(pool.id).h === 300, "tambah lane undo", `${await lanesOf(pool.id)} tinggi ${(await cards()).get(pool.id).h}`);
  await key("Control+y");
  await key("Control+y");
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang utama|Kurir", "tambah lane redo", await lanesOf(pool.id));

  // 5. Urutan lewat klik kanan pada lane 1: task di lane 2 ikut naik; undo mengembalikan.
  const menuOnLane = async (id, fx, fy, ...items) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
    const b = await ps.locator(`[data-card="${id}"]`).boundingBox();
    await ps.mouse.click(b.x + b.width * fx, b.y + b.height * fy, { button: "right" });
    const menu = ps.getByTestId("wb-menu");
    await menu.first().waitFor({ timeout: 5000 });
    let scope = menu.first();
    for (const n of items) {
      await scope.getByRole("menuitem", { name: n }).first().press("Enter");
      scope = ps.getByRole("menu", { name: typeof n === "string" ? n : undefined }).last();
    }
  };
  const t0 = (await cards()).get(task.id);
  const p0 = (await cards()).get(pool.id);
  // Pool 3 lane bobot 150, 150, 150 setinggi 450: lane 1 di y 0 sampai 150.
  await menuOnLane(pool.id, 0.95, 0.1, /^Lane 1/, /^Turunkan lane/);
  await saved(ps);
  await ps.waitForTimeout(300);
  const t1 = (await cards()).get(task.id);
  const laneH = p0.h / 3;
  expect((await lanesOf(pool.id)) === "Gudang utama|Pembeli|Kurir" && Math.abs(t1.y - (t0.y - laneH)) < 1, "turunkan lane", `${await lanesOf(pool.id)}, task ${t0.y} -> ${t1.y}`);
  await pm.waitForFunction((id) => document.querySelector(`[data-card="${id}"] [data-lane-index="0"]`)?.textContent?.trim() === "Gudang utama", pool.id, { timeout: 10000 }).catch(() => {});
  expect((await pm.locator(`[data-card="${pool.id}"] [data-lane-index="0"]`).textContent())?.trim() === "Gudang utama", "urutan klien kedua", "Manajer tidak melihat urutan baru");
  await key("Control+z");
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang utama|Kurir" && (await cards()).get(task.id).y === t0.y, "urutan undo", `${await lanesOf(pool.id)}, task ${(await cards()).get(task.id).y}`);

  // 6. Hapus lane berisi task: konfirmasi, task tetap ada dan tidak bergeser.
  await menuOnLane(pool.id, 0.95, 0.5, /^Lane 2/, /^Hapus lane/);
  const confirm = ps.getByTestId("wb-lane-delete-confirm");
  await confirm.waitFor({ timeout: 5000 });
  expect(await ps.getByRole("dialog", { name: "Hapus lane?" }).getByText(/berisi 1 kartu/).isVisible(), "hapus lane tanya", "dialog tidak menyebut kartu");
  await confirm.click();
  await saved(ps);
  const t2 = (await cards()).get(task.id);
  expect((await lanesOf(pool.id)) === "Pembeli|Kurir" && t2 && t2.x === t0.x && t2.y === t0.y, "hapus lane", `${await lanesOf(pool.id)}, task ${t2 && t2.y}`);
  await key("Control+z");
  expect((await lanesOf(pool.id)) === "Pembeli|Gudang utama|Kurir", "hapus lane undo", await lanesOf(pool.id));

  // 7. Kunci lunak: Manajer mengedit pool, staf tidak bisa mengganti nama lane.
  await ps.locator(".wb-canvas").focus();
  await ps.keyboard.press("Escape");
  await ps.getByRole("button", { name: "Pas ke layar" }).click();
  await ps.waitForTimeout(300);
  await pm.getByRole("button", { name: "Pas ke layar" }).click();
  await pm.waitForTimeout(300);
  const pmb = await pm.locator(`[data-card="${pool.id}"]`).boundingBox();
  // Area lane pertama: pojok kanan bawah pool tertumpuk fragmen sesudah pool memanjang.
  await pm.mouse.dblclick(pmb.x + pmb.width * 0.8, pmb.y + pmb.height * 0.12);
  await ps.waitForTimeout(800);
  const lb0 = await label(pool.id, 0).boundingBox();
  await ps.mouse.dblclick(lb0.x + lb0.width / 2, lb0.y + lb0.height / 2);
  await ps.waitForTimeout(300);
  expect(!(await input.isVisible().catch(() => false)) && (await ps.getByText(/sedang mengedit kartu ini/).first().isVisible().catch(() => false)), "kunci lunak", "edit nama tidak ditolak saat Manajer mengedit");
  await pm.keyboard.press("Escape");
  await pm.waitForTimeout(800);

  // 8. Swimlane: ganti nama dan tombol tambah di tepi kanan.
  await ps.locator(".wb-canvas").focus();
  await ps.keyboard.press("Escape");
  await ps.getByRole("button", { name: "Pas ke layar" }).click();
  await ps.waitForTimeout(300);
  await label(swim.id, 1).dblclick();
  await input.waitFor({ timeout: 5000 });
  await input.fill("Toko daring");
  await input.press("Enter");
  await saved(ps);
  expect((await lanesOf(swim.id)) === "Pelanggan|Toko daring", "swimlane ganti nama", await lanesOf(swim.id));
  await selectCard(swim.id);
  const sb = await ps.locator(`[data-card="${swim.id}"]`).boundingBox();
  const sa = await ps.locator(`[data-card="${swim.id}"] [data-lane-add]`).boundingBox();
  expect(sa && sa.x >= sb.x + sb.width - 1, "swimlane tombol tambah", `tombol ${JSON.stringify(sa)} kartu ${JSON.stringify(sb)}`);

  // 9. Fragmen: syarat operand dengan dobel klik.
  await label(frag.id, 1).dblclick();
  await input.waitFor({ timeout: 5000 });
  expect((await input.inputValue()) === "tidak", "fragmen isi awal", await input.inputValue());
  await input.fill("stok habis");
  await input.press("Enter");
  await saved(ps);
  expect((await lanesOf(frag.id)) === "ya|stok habis" && (await label(frag.id, 1).textContent())?.trim() === "[stok habis]", "fragmen syarat", await lanesOf(frag.id));

  // 10. Muat ulang.
  await ps.reload();
  await saved(ps);
  expect((await label(pool.id, 1).textContent())?.trim() === "Gudang utama" && (await label(swim.id, 1).textContent())?.trim() === "Toko daring", "muat ulang", "nama lane berubah");
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
