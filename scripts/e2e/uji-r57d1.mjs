// Uji browser revisi 57d1 di 1280 px: tambah bentuk tersambung, plus petunjuk pintasan dan garis tidak wajar di mind map.
// Empat tombol 44 px di sisi bentuk terpilih (tidak bertumpuk dengan pegangan port); bentuk baru memakai jenis, varian,
// dan gaya bentuk asal, disambung garis dengan port yang menghadap, tanpa menimpa kartu lain; satu undo menghapus
// bentuk dan garisnya, redo mengembalikan; Ctrl+Enter searah garis keluar terakhir; tidak ada tombol untuk wadah,
// mind map, lifeline, dan pilihan ganda; Ctrl+Enter di editor teks tidak membuat bentuk. Mind map: petunjuk pintasan di
// bar pilihan (kontras AA, aria-describedby kartu); lingkaran garis tidak membuat halaman macet, lipat dan rapikan
// tetap jalan.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r57d1.mjs
import { chromium } from "playwright-core";
import { api, contrastOf, realLogins, session } from "./sesi.mjs";

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
const holders = [];
const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
const settle = async (p) => {
  await p.waitForTimeout(400);
  await saved(p);
};
const shape = (s, text, extra = {}) => ({ shape: s, rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text, align: "center", font_size: 16, ...extra });
const overlap = (a, b) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;

try {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r57d1 ${Date.now()}`, x: 9900, y: 13000, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const link = (from, to) => sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from, to, style: { start: "none", end: "none", line: "solid", color: "" }, geometry: { from: { port: "auto" }, to: { port: "auto" }, route: "curved", points: [] } });
  const P = await add(0, 0, 180, 80, shape("process", "Terima pesanan", { fill: "biru", stroke: "biru", stroke_width: 3 }));
  const G = await add(0, 300, 64, 64, shape("bpmn_gateway", "", { variant: "parallel" }));
  const pool = await add(600, 300, 500, 200, shape("bpmn_pool", "Pool", { fill: "", lanes: [{ label: "A", size: 100 }, { label: "B", size: 100 }] }));
  const C = await add(0, 700, 200, 90, shape("mm_central", "Pusat", { fill: "kuning", font_size: 20 }));
  const A = await add(300, 650, 160, 60, shape("mm_topic", "Satu"));
  const B = await add(300, 780, 160, 60, shape("mm_topic", "Dua"));
  // Lingkaran C ke A ke B ke C.
  await link(C.id, A.id);
  await link(A.id, B.id);
  await link(B.id, C.id);
  const view = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board;
  const until = async (pred) => {
    let v = await view();
    for (let k = 0; k < 30 && !pred(v); k++) {
      await new Promise((r) => setTimeout(r, 200));
      v = await view();
    }
    return v;
  };

  const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: 1280, height: 800 } });
  const ps = await ctx.newPage();
  const errors = [];
  ps.on("pageerror", (e) => errors.push(String(e)));
  await ps.goto(BASE + `/whiteboard/${board}`);
  await saved(ps);
  const fit = async () => {
    await ps.getByRole("button", { name: "Pas ke layar" }).click();
    await ps.waitForTimeout(300);
  };
  await fit();
  const unselect = async () => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
  };
  const pick = async (id) => {
    await unselect();
    const b = await ps.locator(`[data-card="${id}"]`).boundingBox();
    await ps.mouse.click(b.x + b.width / 2, b.y + 6);
    await ps.waitForTimeout(200);
  };
  const press = async (k) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press(k);
    await settle(ps);
  };

  // 1. Tombol tambah: 44 px, tidak bertumpuk dengan pegangan port, berlabel.
  await pick(P.id);
  const right = ps.locator('[data-quick-add="right"]');
  await right.waitFor({ timeout: 5000 });
  expect((await ps.locator("[data-quick-add]").count()) === 4, "empat tombol", `${await ps.locator("[data-quick-add]").count()}`);
  const rb = await right.boundingBox();
  expect(rb.width >= 43.5 && rb.height >= 43.5, "tombol 44 px", JSON.stringify(rb));
  const hit = await ps.evaluate(([x, y]) => !!document.elementFromPoint(x, y)?.closest("[data-quick-add]"), [rb.x + rb.width / 2, rb.y + rb.height / 2]);
  expect(hit, "tidak tertutup", "titik tengah tombol tertutup elemen lain");
  const port = await ps.locator('[data-testid=wb-diagram-handles] [data-port="right"] circle').last().boundingBox();
  expect(port && port.x + port.width <= rb.x, "tidak bertumpuk port", `${JSON.stringify(port)} ${JSON.stringify(rb)}`);
  expect((await right.getAttribute("aria-label")) === "Tambah bentuk tersambung di kanan", "label", await right.getAttribute("aria-label"));
  // Tombol atas di luar pegangan putar (100 px di atas bentuk).
  const ub = await ps.locator('[data-quick-add="up"]').boundingBox();
  const rot = await ps.locator(`[data-card="${P.id}"] [data-handle="rotate"]`).boundingBox();
  expect(rot && ub.y + ub.height <= rot.y, "tidak bertumpuk putar", `${JSON.stringify(ub)} ${JSON.stringify(rot)}`);

  // 2. Klik kanan: proses baru bergaya sama, garis kanan ke kiri, satu undo, redo.
  await right.click();
  await settle(ps);
  let v = await until((x) => x.cards.length === 7 && x.connectors.length === 4);
  const P2 = v.cards.find((c) => c.content?.shape === "process" && c.id !== P.id);
  const line = v.connectors.find((l) => l.from === P.id && l.to === P2?.id);
  expect(P2 && P2.x === 260 && P2.y === 0 && P2.content.fill === "biru" && P2.content.stroke_width === 3 && P2.content.text === "", "bentuk baru", JSON.stringify(P2 && { x: P2.x, y: P2.y, c: P2.content }));
  expect(line?.geometry.from.port === "right" && line.geometry.to.port === "left" && line.style.end === "arrow", "garis baru", JSON.stringify(line?.geometry));
  expect((await ps.locator(`[data-card="${P2.id}"]`).getAttribute("class")).includes("wb-card-selected"), "bentuk baru terpilih", "tidak terpilih");
  await press("Control+z");
  v = await until((x) => !x.cards.some((c) => c.id === P2.id));
  expect(!v.cards.some((c) => c.id === P2.id) && !v.connectors.some((l) => l.id === line.id), "undo satu langkah", "bentuk atau garis masih ada");
  await press("Control+Shift+z");
  v = await until((x) => x.cards.some((c) => c.id === P2.id));
  expect(v.cards.some((c) => c.id === P2.id) && v.connectors.some((l) => l.from === P.id && l.to === P2.id), "redo", "bentuk atau garis tidak kembali");

  // 3. Ctrl+Enter di bentuk baru: searah alur (kanan), tidak menimpa kartu; varian ikut (gateway paralel ke bawah).
  await fit();
  await pick(P2.id);
  await ps.keyboard.press("Control+Enter");
  await settle(ps);
  v = await until((x) => x.cards.filter((c) => c.content?.shape === "process").length === 3);
  const P3 = v.cards.find((c) => c.content?.shape === "process" && ![P.id, P2.id].includes(c.id));
  expect(P3 && P3.x > P2.x + P2.w && P3.y === P2.y, "Ctrl+Enter ke kanan", JSON.stringify(P3 && { x: P3.x, y: P3.y }));
  await fit();
  await pick(G.id);
  await ps.locator('[data-quick-add="down"]').click();
  await settle(ps);
  v = await until((x) => x.cards.filter((c) => c.content?.shape === "bpmn_gateway").length === 2);
  const G2 = v.cards.find((c) => c.content?.shape === "bpmn_gateway" && c.id !== G.id);
  expect(G2?.content.variant === "parallel" && G2.y > G.y + G.h && G2.x === G.x, "varian dan arah bawah", JSON.stringify(G2 && { v: G2.content.variant, x: G2.x, y: G2.y }));
  // Tidak menimpa kartu: penghalang tepat di kanan G2.
  const block = await add(G2.x + G2.w + 80, G2.y - 10, 120, 84, shape("rect", "Penghalang"));
  await ps.waitForTimeout(600);
  await fit();
  await pick(G2.id);
  // Tombol kanan menimpa penghalang, jadi tidak ditampilkan (tidak menghalangi klik ke kartu itu); Ctrl+Enter tetap.
  expect((await ps.locator('[data-quick-add="right"]').count()) === 0 && (await ps.locator('[data-quick-add="left"]').count()) === 1, "tombol menimpa kartu disembunyikan", `${await ps.locator("[data-quick-add]").count()} tombol`);
  await ps.keyboard.press("Control+Enter");
  await settle(ps);
  v = await until((x) => x.cards.filter((c) => c.content?.shape === "bpmn_gateway").length === 3);
  const G3 = v.cards.find((c) => c.content?.shape === "bpmn_gateway" && ![G.id, G2.id].includes(c.id));
  const others = v.cards.filter((c) => c.id !== G3?.id && c.kind === "shape" && c.content.shape !== "bpmn_pool");
  expect(G3 && !others.some((o) => overlap(G3, o)), "tanpa menimpa", JSON.stringify(G3 && { x: G3.x, y: G3.y, blok: [block.x, block.y] }));

  // 4. Tidak ada tombol untuk wadah, mind map, dan pilihan ganda.
  await fit();
  await pick(pool.id);
  expect((await ps.locator("[data-quick-add]").count()) === 0, "wadah", "tombol tampil di pool");
  await pick(A.id);
  expect((await ps.locator("[data-quick-add]").count()) === 0, "mind map", "tombol tampil di cabang mind map");
  await pick(P.id);
  await ps.keyboard.down("Shift");
  const p2b = await ps.locator(`[data-card="${P2.id}"]`).boundingBox();
  await ps.mouse.click(p2b.x + p2b.width / 2, p2b.y + 6);
  await ps.keyboard.up("Shift");
  await ps.waitForTimeout(200);
  expect((await ps.locator("[data-quick-add]").count()) === 0, "pilihan ganda", "tombol tampil untuk dua kartu");

  // 5. Ctrl+Enter di editor teks: baris baru, bukan bentuk baru.
  const before = (await view()).cards.length;
  await unselect();
  await ps.locator(`[data-card="${P.id}"]`).dblclick();
  await ps.locator("[contenteditable=true]").first().waitFor({ timeout: 5000 });
  await ps.keyboard.press("Control+Enter");
  await ps.keyboard.press("Escape");
  await settle(ps);
  expect((await view()).cards.length === before, "Ctrl+Enter di editor", "bentuk baru dibuat dari dalam editor teks");

  // 6. Mind map: petunjuk pintasan, kontras AA, dibaca pembaca layar.
  await pick(A.id);
  const hint = ps.getByTestId("wb-mind-hint");
  expect((await hint.textContent()).trim() === "Tab: anak, Enter: sejajar, F2: ubah teks", "petunjuk", await hint.textContent());
  const ratio = await contrastOf(hint);
  expect(ratio >= 4.5, "kontras petunjuk", ratio.toFixed(2));
  expect((await ps.locator(`[data-card="${A.id}"]`).getAttribute("aria-describedby")) === "wb-mind-hint", "aria-describedby", "kartu tidak merujuk petunjuk");
  const hb = await hint.boundingBox();
  const ab = await ps.locator(`[data-card="${A.id}"]`).boundingBox();
  expect(!(hb.x < ab.x + ab.width && ab.x < hb.x + hb.width && hb.y < ab.y + ab.height && ab.y < hb.y + hb.height), "tidak menutupi kartu", `${JSON.stringify(hb)} ${JSON.stringify(ab)}`);
  await pick(P.id);
  expect((await hint.count()) === 0, "petunjuk hanya mind map", "petunjuk tampil di bentuk biasa");

  // 7. Lingkaran garis: rapikan dan lipat tetap jalan, halaman tidak macet.
  await pick(C.id);
  await ps.locator(`[data-card="${C.id}"]`).click({ button: "right", position: { x: 100, y: 8 } });
  await ps.getByRole("menuitem", { name: "Rapikan mind map" }).first().click();
  await settle(ps);
  v = await view();
  const a1 = v.cards.find((c) => c.id === A.id);
  const b1 = v.cards.find((c) => c.id === B.id);
  expect(a1.x === C.x + 200 + 56 && b1.x === a1.x && b1.y === a1.y + 60 + 16, "rapikan lingkaran", JSON.stringify({ a: [a1.x, a1.y], b: [b1.x, b1.y] }));
  await ps.locator(`[data-card="${C.id}"]`).click({ button: "right", position: { x: 100, y: 8 } });
  await ps.getByRole("menuitem", { name: "Lipat cabang" }).first().click();
  await settle(ps);
  expect((await ps.locator(`[data-card="${A.id}"], [data-card="${B.id}"]`).count()) === 0, "lipat lingkaran", "cabang masih tampil");
  expect(await ps.evaluate(() => document.readyState === "complete"), "tidak macet", "halaman tidak menjawab");
  expect(errors.length === 0, "galat halaman", errors.join(" | "));
  await ctx.close();
} finally {
  for (const f of fails) console.log("GAGAL", f);
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
