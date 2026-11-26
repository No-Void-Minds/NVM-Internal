// Uji browser revisi 55 di 1280 px: setiap operasi garis satu langkah undo dan redo. Untuk tiap operasi: lakukan,
// Ctrl+Z (kembali ke semula di server), Ctrl+Y (maju lagi), muat ulang (tetap), dan klien kedua (Manajer) melihat
// keadaan yang sama. Operasi: buat garis dari port, hapus garis, ganti titik sambung, geser, tambah, dan hapus titik
// belok, ubah jalur, label, gaya, preset, jenis ujung, dan urutan tumpuk. Gabungan: geser kartu lalu satu Ctrl+Z
// mengembalikan kartu dan titik belok garisnya; hapus kartu lalu Ctrl+Z mengembalikan garisnya lengkap dengan titik
// belok dan label. Kunci: undo garis yang menempel di kartu terkunci ditolak dengan pesan kartu terkunci.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r55.mjs
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
const note = (text) => ({ doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text }] }] } });

try {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r55 ${Date.now()}`, x: 9900, y: 9600, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, text) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "note", x, y, w: 160, h: 90, content: note(text) })).body.card;
  const a = await add(0, 0, "Aktor");
  const b = await add(500, 0, "Use case");
  const c = await add(500, 300, "Lain");
  const lines = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.connectors;
  const cardsNow = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards;
  // Keadaan garis yang dibandingkan: tanpa versi dan seq.
  const pick = (l) => (l ? JSON.stringify({ from: l.from, to: l.to, style: l.style, geometry: l.geometry, label: l.label, z: l.z }) : "tidak ada");
  const stateOf = async (id) => pick((await lines()).find((l) => l.id === id));

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
  await pm.locator(`[data-card="${a.id}"]`).waitFor({ timeout: 10000 });

  const canvasKey = async (key) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press(key);
    await saved(ps);
    await ps.waitForTimeout(400);
  };
  const pathOf = (page, id) => page.locator(`[data-connector="${id}"] path`).first().getAttribute("d").catch(() => null);
  // Tunggu sampai API menunjukkan keadaan yang diharapkan (siaran dan simpan asinkron).
  const until = async (id, want) => {
    for (let i = 0; i < 40; i++) {
      if ((await stateOf(id)) === want) return true;
      await ps.waitForTimeout(150);
    }
    return false;
  };
  // Satu siklus: lakukan, undo, redo, muat ulang, klien kedua.
  const cycle = async (name, id, run) => {
    const s0 = await stateOf(id);
    await run();
    await saved(ps);
    await ps.waitForTimeout(300);
    const s1 = await stateOf(id);
    expect(s1 !== s0, name, "operasi tidak mengubah garis");
    await canvasKey("Control+z");
    expect(await until(id, s0), `${name} undo`, `sesudah Ctrl+Z ${await stateOf(id)} harapan ${s0}`);
    await canvasKey("Control+y");
    expect(await until(id, s1), `${name} redo`, `sesudah Ctrl+Y ${await stateOf(id)} harapan ${s1}`);
    await ps.reload();
    await saved(ps);
    expect((await stateOf(id)) === s1, `${name} muat ulang`, "keadaan berubah sesudah muat ulang");
    const mine = await pathOf(ps, id);
    await pm.waitForFunction(([lid, d]) => (document.querySelector(`[data-connector="${lid}"] path`)?.getAttribute("d") ?? null) === d, [id, mine], { timeout: 10000 }).catch(() => {});
    expect((await pathOf(pm, id)) === mine, `${name} klien kedua`, "layar Manajer berbeda");
  };
  const openLine = async (id) => {
    await ps.locator(`[data-connector="${id}"] path[data-hit]`).dispatchEvent("dblclick");
    const dlg = ps.getByRole("dialog", { name: "Garis penghubung" });
    await dlg.waitFor({ timeout: 5000 });
    return dlg;
  };
  const pickIn = async (dlg, label, option) => {
    await dlg.getByRole("button", { name: new RegExp(`^${label}:`) }).first().click();
    await ps.getByTestId("wb-picker").getByRole("button", { name: option, exact: true }).click();
    await saved(ps);
    await ps.keyboard.press("Escape");
  };

  // 1. Buat garis dari port: undo menghapusnya, redo memulihkannya.
  await ps.locator(`[data-card="${a.id}"]`).click();
  const port = ps.locator('[data-testid=wb-diagram-handles] [data-port="right"] circle').last();
  const pb = await port.boundingBox();
  const tb = await ps.locator(`[data-card="${b.id}"]`).boundingBox();
  await ps.mouse.move(pb.x + pb.width / 2, pb.y + pb.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(tb.x + tb.width / 2, tb.y + tb.height / 2, { steps: 8 });
  await ps.mouse.up();
  await saved(ps);
  const made = (await lines()).find((l) => l.from === a.id && l.to === b.id);
  expect(!!made, "buat", "garis dari port tidak dibuat");
  const L = made.id;
  const sMade = await stateOf(L);
  await canvasKey("Control+z");
  expect(await until(L, "tidak ada"), "buat undo", "garis baru tidak terhapus oleh Ctrl+Z");
  await canvasKey("Control+y");
  expect(await until(L, sMade), "buat redo", "garis tidak kembali oleh Ctrl+Y");
  await ps.reload();
  await saved(ps);
  await pm.locator(`[data-connector="${L}"]`).waitFor({ timeout: 10000 }).catch(() => {});
  expect((await pm.locator(`[data-connector="${L}"]`).count()) === 1, "buat klien kedua", "Manajer tidak melihat garis");

  // 2. Titik belok: tambah (pegangan sisip), geser, hapus (dobel klik).
  const select = async () => {
    await ps.locator(`[data-connector="${L}"] path[data-hit]`).dispatchEvent("pointerdown");
    await ps.waitForTimeout(100);
  };
  const dragHandle = async (sel, i, dx, dy) => {
    await select();
    const h = ps.locator(`[data-testid=wb-diagram-handles] ${sel}`).nth(i).locator("circle").last();
    const bb = await h.boundingBox();
    await ps.mouse.move(bb.x + bb.width / 2, bb.y + bb.height / 2);
    await ps.mouse.down();
    await ps.mouse.move(bb.x + bb.width / 2 + dx / 2, bb.y + bb.height / 2 + dy / 2, { steps: 6 });
    await ps.mouse.move(bb.x + bb.width / 2 + dx, bb.y + bb.height / 2 + dy, { steps: 6 });
    await ps.mouse.up();
  };
  await cycle("tambah titik belok", L, () => dragHandle("[data-insert]", 0, 0, 70));
  await cycle("geser titik belok", L, () => dragHandle("[data-bend]", 0, 30, 25));
  await cycle("hapus titik belok", L, async () => {
    await select();
    await ps.locator("[data-testid=wb-diagram-handles] [data-bend]").nth(0).locator("circle").last().dispatchEvent("dblclick");
  });
  // Titik belok lagi untuk uji gabungan nanti.
  await dragHandle("[data-insert]", 0, 0, 80);
  await saved(ps);

  // 3. Panel garis: jalur, titik sambung, label, gaya, preset, jenis ujung.
  await cycle("jalur", L, async () => pickIn(await openLine(L), "Jalur", "Melengkung"));
  await cycle("titik sambung", L, async () => pickIn(await openLine(L), "Menempel di awal", "Bawah"));
  await cycle("label", L, async () => {
    const dlg = await openLine(L);
    await dlg.getByLabel("Label tengah").fill("lalu");
    await dlg.getByLabel("Label tengah").press("Tab");
    await saved(ps);
    await ps.keyboard.press("Escape");
  });
  await cycle("gaya", L, async () => pickIn(await openLine(L), "Garis", "Putus-putus"));
  await cycle("preset", L, async () => pickIn(await openLine(L), "Preset garis", "BPMN: Message flow"));
  await cycle("jenis ujung", L, async () => pickIn(await openLine(L), "Ujung akhir", "Belah ketupat"));
  await cycle("urutan", L, async () => {
    await select();
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Shift+F10");
    await ps.getByTestId("wb-menu").first().getByRole("menuitem", { name: /^Urutan/ }).press("Enter");
    await ps.getByRole("menu", { name: "Urutan" }).getByRole("menuitem", { name: /^Paling depan/ }).press("Enter");
  });

  // 4. Hapus garis (Delete): undo memulihkan dengan titik belok dan label, redo menghapus lagi.
  const sFull = await stateOf(L);
  await select();
  await canvasKey("Delete");
  expect(await until(L, "tidak ada"), "hapus", "garis tidak terhapus");
  await canvasKey("Control+z");
  expect(await until(L, sFull), "hapus undo", `garis tidak kembali lengkap: ${await stateOf(L)}`);
  await canvasKey("Control+y");
  expect(await until(L, "tidak ada"), "hapus redo", "Ctrl+Y tidak menghapus lagi");
  await canvasKey("Control+z");
  expect(await until(L, sFull), "hapus undo kedua", "garis tidak kembali");
  await ps.reload();
  await saved(ps);

  // 5. Gabungan: geser kartu ujung, satu Ctrl+Z mengembalikan kartu dan bentuk garis.
  const pathBefore = await pathOf(ps, L);
  const aBefore = (await cardsNow()).find((x) => x.id === a.id);
  const ab = await ps.locator(`[data-card="${a.id}"]`).boundingBox();
  await ps.mouse.move(ab.x + 20, ab.y + 10);
  await ps.mouse.down();
  await ps.mouse.move(ab.x + 20, ab.y + 160, { steps: 10 });
  await ps.mouse.up();
  await saved(ps);
  const moved = (await cardsNow()).find((x) => x.id === a.id);
  expect(moved.y !== aBefore.y && (await pathOf(ps, L)) !== pathBefore, "geser kartu", "kartu atau garis tidak bergeser");
  const lineAfterMove = await stateOf(L);
  await canvasKey("Control+z");
  const back = (await cardsNow()).find((x) => x.id === a.id);
  expect(back.y === aBefore.y && (await pathOf(ps, L)) === pathBefore, "geser kartu undo", `kartu ${back.y} harapan ${aBefore.y}, garis sama ${(await pathOf(ps, L)) === pathBefore}`);
  expect((await stateOf(L)) === lineAfterMove, "geser kartu undo", "data garis ikut berubah (harusnya hanya kartu yang kembali)");

  // 5b. Kedua ujung digeser bersama: path bergeser persis sebesar geseran (bukan dua kali), data garis tidak ditulis
  // ulang, dan satu Ctrl+Z mengembalikan kartu dan garis.
  const nums = (d) => (d.match(/-?\d+(\.\d+)?/g) ?? []).map(Number);
  const both0 = await pathOf(ps, L);
  const line0 = await stateOf(L);
  const y0 = (await cardsNow()).filter((x) => x.id === a.id || x.id === b.id).map((x) => x.y);
  const x0 = (await cardsNow()).find((x) => x.id === a.id).x;
  await ps.locator(".wb-canvas").focus();
  await ps.keyboard.press("Escape");
  await ps.locator(`[data-card="${a.id}"]`).click();
  await ps.locator(`[data-card="${b.id}"]`).click({ modifiers: ["Shift"] });
  const bb2 = await ps.locator(`[data-card="${b.id}"]`).boundingBox();
  await ps.mouse.move(bb2.x + 20, bb2.y + 10);
  await ps.mouse.down();
  await ps.mouse.move(bb2.x + 20, bb2.y + 130, { steps: 10 });
  await ps.mouse.up();
  await saved(ps);
  await ps.waitForTimeout(500);
  const y1 = (await cardsNow()).filter((x) => x.id === a.id || x.id === b.id).map((x) => x.y);
  const dyWorld = y1[0] - y0[0];
  // Snap grid bisa ikut merapikan x kartu; geseran yang diharapkan memakai posisi kartu sebenarnya.
  const dxWorld = (await cardsNow()).find((x) => x.id === a.id).x - x0;
  const p0 = nums(both0);
  const p1 = nums(await pathOf(ps, L));
  const shiftOk = p0.length === p1.length && p0.every((v, i) => Math.abs(p1[i] - v - (i % 2 ? dyWorld : dxWorld)) < 1.01);
  expect(dyWorld !== 0 && y1[1] - y0[1] === dyWorld, "geser dua ujung", `geseran kartu ${JSON.stringify({ y0, y1 })}`);
  expect(shiftOk, "geser dua ujung", `path tidak bergeser tepat ${dxWorld},${dyWorld}: ${both0} -> ${await pathOf(ps, L)}`);
  expect((await stateOf(L)) === line0, "geser dua ujung", "data garis ditulis ulang saat kedua ujung digeser");
  await canvasKey("Control+z");
  expect((await pathOf(ps, L)) === both0, "geser dua ujung undo", "satu Ctrl+Z tidak mengembalikan kartu dan garis");

  // 6. Hapus kartu beserta garisnya, Ctrl+Z mengembalikan garis lengkap.
  const sBeforeCardDelete = await stateOf(L);
  await ps.locator(`[data-card="${a.id}"]`).click();
  await canvasKey("Delete");
  expect(await until(L, "tidak ada"), "hapus kartu", "garis masih tampil sesudah kartu dihapus");
  await canvasKey("Control+z");
  expect(await until(L, sBeforeCardDelete), "hapus kartu undo", `garis tidak kembali lengkap: ${await stateOf(L)}`);
  await pm.locator(`[data-connector="${L}"]`).waitFor({ timeout: 10000 }).catch(() => {});
  expect((await pm.locator(`[data-connector="${L}"]`).count()) === 1, "hapus kartu klien kedua", "Manajer tidak melihat garis kembali");

  // 7. Kunci: label diubah, kartu ujung dikunci, Ctrl+Z ditolak dengan pesan kartu terkunci; buka kunci, Ctrl+Z jalan.
  await ps.reload();
  await saved(ps);
  const dlg = await openLine(L);
  await dlg.getByLabel("Label tengah").fill("dikunci");
  await dlg.getByLabel("Label tengah").press("Tab");
  await saved(ps);
  await ps.keyboard.press("Escape");
  const sLabel = await stateOf(L);
  await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: a.id, locked: true }] });
  await ps.waitForFunction((id) => document.querySelector(`[data-card="${id}"] [data-testid=wb-lock-badge], [data-card="${id}"][data-locked="true"]`) !== null, a.id, { timeout: 5000 }).catch(() => {});
  await ps.waitForTimeout(500);
  await canvasKey("Control+z");
  const toast = await ps.getByText(/terkunci\. Buka kunci dulu/).first().isVisible().catch(() => false);
  expect(toast, "kunci", "pesan kartu terkunci tidak tampil");
  expect((await stateOf(L)) === sLabel, "kunci", "garis berubah lewat undo padahal kartu ujung terkunci");
  await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: a.id, locked: false }] });
  await ps.waitForTimeout(800);
  await canvasKey("Control+z");
  expect(JSON.parse(await stateOf(L)).label !== "dikunci", "kunci dibuka", "undo tidak jalan sesudah kunci dibuka");

  expect(errors.length === 0, "galat halaman", errors.join(" | "));
  await mctx.close();
  await ctx.close();
} finally {
  // Dicetak juga saat skrip berhenti di tengah, supaya kegagalan sebelumnya tetap terlihat.
  for (const f of fails) console.log("GAGAL", f);
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
