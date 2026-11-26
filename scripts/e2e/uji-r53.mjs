// Uji browser revisi 53 di 1280 dan 360 px: garis ikut urutan tumpuk dan garis elastis.
// - Diagram use case: batas sistem yang dibuat terakhir menutupi panah aktor ke use case; sesudah dikirim ke paling
//   belakang lewat menu klik kanan, panah terlihat di atasnya (elemen teratas di titik garis lewat elementFromPoint).
//   Kartu yang sengaja di depan menutupi garis. Urutan garis manual lewat menu klik kanan garis (Paling depan),
//   undo, lalu lewat menu Aksi bar garis (Paling depan, Otomatis); layar Manajer melihat urutannya berubah.
// - Batas sistem dari palet Bentuk langsung di paling belakang; preset include (putus-putus, panah terbuka, label).
// - Garis bebas dengan dua titik belok miring dari pegangan sisip; geser kartu ujung lalu titik belok bergeser
//   proporsional (pemetaan dari anchor); ganti ke melengkung lalu siku tanpa kehilangan titik; Manajer melihatnya.
// - Ekspor PNG tidak error.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r53.mjs
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
const shape = (s, text) => ({ shape: s, rotation: 0, fill: s === "uml_usecase" ? "putih" : "", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text, align: "center", font_size: 16 });

// Titik garis di layar pada pecahan t dari panjangnya (path area klik, koordinat dunia lewat getScreenCTM).
const linePoint = (page, id, t) =>
  page.evaluate(([id, t]) => {
    const p = document.querySelector(`[data-connector="${id}"] path[data-hit]`);
    if (!p) return null;
    const q = p.getPointAtLength(p.getTotalLength() * t);
    const m = p.getScreenCTM();
    return { x: m.a * q.x + m.c * q.y + m.e, y: m.b * q.x + m.d * q.y + m.f };
  }, [id, t]);
const topAt = (page, pt) =>
  page.evaluate(({ x, y }) => {
    const el = document.elementFromPoint(x, y);
    return { line: el?.closest("[data-connector]")?.getAttribute("data-connector") ?? null, card: el?.closest("[data-card]")?.getAttribute("data-card") ?? null };
  }, pt);
const zIndexOf = (page, sel) => page.locator(sel).first().evaluate((el) => Number(getComputedStyle(el).zIndex) || 0);
const conn = async (board, id) => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.connectors.find((c) => c.id === id);
const pickMenu = async (page, ...names) => {
  const menu = page.getByTestId("wb-menu");
  await menu.first().waitFor({ timeout: 5000 });
  let scope = menu.first();
  for (const n of names) {
    await scope.getByRole(/Otomatis/.test(n) ? "menuitemcheckbox" : "menuitem", { name: new RegExp(`^${n}`) }).first().press("Enter");
    scope = page.getByRole("menu", { name: n });
  }
};
// Pemetaan kemiripan dari anchor ke pusat sekarang (rumus sama dengan utils/wbShapes.ts elastic, ditulis ulang).
const mapPts = (pts, [a0, b0], a1, b1) => {
  const vx = b0[0] - a0[0], vy = b0[1] - a0[1], len = vx * vx + vy * vy;
  const ux = b1[0] - a1[0], uy = b1[1] - a1[1];
  const sr = (ux * vx + uy * vy) / len, si = (uy * vx - ux * vy) / len;
  return pts.map(([x, y]) => [a1[0] + sr * (x - a0[0]) - si * (y - a0[1]), a1[1] + si * (x - a0[0]) + sr * (y - a0[1])]);
};
const pathPts = async (page, id) => {
  const d = await page.locator(`[data-connector="${id}"] path`).first().getAttribute("d");
  const n = (d.match(/-?\d+(\.\d+)?/g) ?? []).map(Number);
  const out = [];
  for (let i = 0; i + 1 < n.length; i += 2) out.push([n[i], n[i + 1]]);
  return { d, pts: out };
};

try {
  for (const W of [1280, 360]) {
    const where = (s) => `${W} ${s}`;
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r53 ${W} ${Date.now()}`, x: 9900, y: 9000 + holders.length * 200, w: 200, h: 120 })).body.card;
    holders.push(holder);
    const board = holder.ref.id;
    const add = async (body) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, body)).body.card;
    const aktor = await add({ kind: "shape", x: -250, y: 150, w: 80, h: 160, content: shape("uml_actor", "Pelanggan") });
    const uc = await add({ kind: "shape", x: 400, y: 185, w: 180, h: 90, content: shape("uml_usecase", "Pesan") });
    const panah = (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: aktor.id, to: uc.id, style: { start: "none", end: "arrow", line: "solid", color: "" }, geometry: { from: { port: "auto" }, to: { port: "auto" }, route: "straight", points: [] } })).body.connector;
    const tutup = await add({ kind: "note", x: 70, y: 195, w: 90, h: 70, content: { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Di depan" }] }] } } });
    // Dibuat terakhir: di atas semuanya sampai dikirim ke belakang.
    const sys = await add({ kind: "shape", x: 0, y: 0, w: 700, h: 420, content: shape("uml_system", "Sistem toko") });
    const n1 = await add({ kind: "note", x: -250, y: 700, w: 160, h: 90, content: { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Awal" }] }] } } });
    const n2 = await add({ kind: "note", x: 500, y: 700, w: 160, h: 90, content: { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Akhir" }] }] } } });
    const bebas = (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: n1.id, to: n2.id, style: { start: "none", end: "arrow", line: "solid", color: "" }, geometry: { from: { port: "right" }, to: { port: "left" }, route: "straight", points: [] } })).body.connector;

    const errors = [];
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: W, height: 800 } });
    const ps = await ctx.newPage();
    ps.on("pageerror", (e) => errors.push(String(e)));
    await ps.goto(BASE + `/whiteboard/${board}`);
    await saved(ps);
    await ps.getByRole("button", { name: "Pas ke layar" }).click();
    await ps.waitForTimeout(400);

    const mctx = await browser.newContext({ storageState: manajer.state, viewport: { width: 1280, height: 800 } });
    const pm = await mctx.newPage();
    await pm.goto(BASE + `/whiteboard/${board}`);
    await pm.locator(`[data-connector="${panah.id}"]`).waitFor({ timeout: 10000 });

    // Batas sistem di depan menutupi panah; kirim ke paling belakang lewat menu klik kanan.
    const p75 = await linePoint(ps, panah.id, 0.75);
    expect((await topAt(ps, p75)).card === sys.id, where("awal"), `batas sistem yang baru dibuat tidak di atas panah: ${JSON.stringify(await topAt(ps, p75))}`);
    const sb = await ps.locator(`[data-card="${sys.id}"]`).boundingBox();
    await ps.mouse.click(sb.x + sb.width - 12, sb.y + sb.height - 12, { button: "right" });
    await pickMenu(ps, "Urutan", "Paling belakang");
    await saved(ps);
    await ps.waitForTimeout(200);
    const above = await topAt(ps, await linePoint(ps, panah.id, 0.75));
    expect(above.line === panah.id, where("di atas batas sistem"), `elemen teratas di titik panah: ${JSON.stringify(above)}`);
    const covered = await topAt(ps, await linePoint(ps, panah.id, 0.5));
    expect(covered.card === tutup.id, where("kartu di depan"), `kartu yang di depan tidak menutupi garis: ${JSON.stringify(covered)}`);

    // Urutan manual lewat menu klik kanan garis, lalu undo.
    const p90 = await linePoint(ps, panah.id, 0.9);
    await ps.mouse.click(p90.x, p90.y, { button: "right" });
    await pickMenu(ps, "Urutan", "Paling depan");
    await saved(ps);
    // Lepas pilihan garis: pegangan sisip titik belok di tengah garis terpilih selalu paling atas.
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
    await ps.waitForTimeout(200);
    expect((await topAt(ps, await linePoint(ps, panah.id, 0.5))).line === panah.id, where("paling depan"), "garis tidak di atas kartu penutup");
    expect((await conn(board, panah.id)).z !== null, where("paling depan"), "z manual tidak tersimpan");
    await pm.waitForFunction(([l, c]) => {
      const a = document.querySelector(`[data-connector="${l}"]`);
      const b = document.querySelector(`[data-card="${c}"]`);
      return a && b && Number(getComputedStyle(a).zIndex) > Number(getComputedStyle(b).zIndex);
    }, [panah.id, tutup.id], { timeout: 10000 }).catch(() => {});
    expect((await zIndexOf(pm, `[data-connector="${panah.id}"]`)) > (await zIndexOf(pm, `[data-card="${tutup.id}"]`)), where("realtime urutan"), "layar Manajer tidak melihat garis di depan");
    await ps.keyboard.press("Control+z");
    await saved(ps);
    await ps.waitForTimeout(300);
    expect((await conn(board, panah.id)).z === null, where("undo"), "undo tidak mengembalikan urutan otomatis");
    expect((await topAt(ps, await linePoint(ps, panah.id, 0.5))).card === tutup.id, where("undo"), "sesudah undo garis masih di depan kartu penutup");

    // Menu Aksi bar garis: Paling depan lalu Otomatis.
    const p95 = await linePoint(ps, panah.id, 0.92);
    await ps.mouse.click(p95.x, p95.y);
    await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
    await pickMenu(ps, "Urutan", "Paling depan");
    await saved(ps);
    expect((await conn(board, panah.id)).z !== null, where("menu Aksi"), "Paling depan dari menu Aksi tidak tersimpan");
    await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
    await pickMenu(ps, "Urutan", "Otomatis");
    await saved(ps);
    expect((await conn(board, panah.id)).z === null, where("menu Aksi"), "Otomatis dari menu Aksi tidak tersimpan");

    // Preset include lewat panel garis.
    await ps.getByTestId("wb-selbar").getByRole("button", { name: "Gaya garis" }).click();
    const dlg = ps.getByRole("dialog", { name: "Garis penghubung" });
    await dlg.waitFor({ timeout: 5000 });
    await dlg.getByRole("button", { name: "Include", exact: true }).click();
    await saved(ps);
    const inc = await conn(board, panah.id);
    expect(inc.label === "«include»" && inc.style.line === "dashed" && inc.style.end === "open_arrow", where("include"), JSON.stringify({ l: inc.label, s: inc.style }));
    const lay = await layout(ps);
    expect(!lay.overflow, where("tata letak"), "gulir horizontal dengan panel garis terbuka");
    await ps.keyboard.press("Escape");

    // Garis bebas: dua titik belok miring dari pegangan sisip.
    await ps.keyboard.press("Escape");
    const b50 = await linePoint(ps, bebas.id, 0.5);
    await ps.mouse.click(b50.x, b50.y);
    const drag = async (from, dx, dy) => {
      await ps.mouse.move(from.x, from.y);
      await ps.mouse.down();
      await ps.mouse.move(from.x + dx / 2, from.y + dy / 2, { steps: 4 });
      await ps.mouse.move(from.x + dx, from.y + dy, { steps: 4 });
      await ps.mouse.up();
      await saved(ps);
      await ps.waitForTimeout(200);
    };
    const insertAt = async (i) => {
      const c = ps.locator("[data-testid=wb-diagram-handles] [data-insert]").nth(i).locator("circle").first();
      const bb = await c.boundingBox();
      return { x: bb.x + bb.width / 2, y: bb.y + bb.height / 2 };
    };
    await drag(await insertAt(0), -20, -60);
    await drag(await insertAt(1), 25, 50);
    const g1 = await conn(board, bebas.id);
    expect(g1.geometry.route === "bebas" && g1.geometry.points.length === 2 && !!g1.geometry.anchor, where("bebas"), JSON.stringify(g1.geometry));
    const slanted = g1.geometry.points.length === 2 && g1.geometry.points[0][1] !== g1.geometry.points[1][1];
    expect(slanted, where("bebas miring"), "titik belok tidak membentuk ruas miring");

    // Geser kartu akhir: titik belok bergeser proporsional dari anchor ke pusat baru.
    await sApi.call("POST", `/whiteboard/boards/${board}/cards/patch`, { cards: [{ id: n2.id, x: 500, y: 900 }] });
    const c1 = [n1.x + n1.w / 2, n1.y + n1.h / 2];
    const c2 = [500 + n2.w / 2, 900 + n2.h / 2];
    const want = mapPts(g1.geometry.points, g1.geometry.anchor, c1, c2);
    for (const page of [ps, pm]) {
      await page.waitForTimeout(800);
      const got = (await pathPts(page, bebas.id)).pts.slice(1, 3);
      const off = got.length === 2 ? Math.max(...got.map((p, i) => Math.hypot(p[0] - want[i][0], p[1] - want[i][1]))) : Infinity;
      expect(off < 2, where(`elastis ${page === ps ? "staf" : "manajer"}`), `titik belok ${JSON.stringify(got)} harapan ${JSON.stringify(want)}`);
    }

    // Ganti jalur ke melengkung lalu siku: titik belok tidak hilang, Manajer melihatnya.
    for (const [label, route, mark] of [["Melengkung", "curved", "C"], ["Siku", "orthogonal", "L"]]) {
      await ps.locator(".wb-canvas").focus();
      const bp = await linePoint(ps, bebas.id, 0.3);
      await ps.mouse.click(bp.x, bp.y);
      await ps.getByTestId("wb-selbar").getByRole("button", { name: "Gaya garis" }).click();
      const d2 = ps.getByRole("dialog", { name: "Garis penghubung" });
      await d2.waitFor({ timeout: 5000 });
      await d2.getByRole("button", { name: /^Jalur:/ }).click();
      await ps.getByTestId("wb-picker").getByRole("button", { name: label, exact: true }).click();
      await saved(ps);
      await ps.keyboard.press("Escape");
      const g = await conn(board, bebas.id);
      expect(g.geometry.route === route && g.geometry.points.length === 2, where(label), JSON.stringify(g.geometry));
      await pm.waitForFunction(([id, m]) => (document.querySelector(`[data-connector="${id}"] path`)?.getAttribute("d") ?? "").includes(m), [bebas.id, mark], { timeout: 10000 }).catch(() => {});
      expect((await pathPts(pm, bebas.id)).d.includes(mark), where(`${label} realtime`), "layar Manajer tidak melihat jalur baru");
    }

    // Palet Bentuk: Batas sistem langsung paling belakang.
    if (W >= 768) {
      await ps.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Bentuk", exact: true }).click();
    } else {
      await ps.getByRole("button", { name: "Tambah", exact: true }).click();
      await ps.locator("dialog[open]").getByRole("button", { name: "Bentuk", exact: true }).click();
    }
    const pal = ps.getByRole("dialog", { name: "Pilih bentuk" });
    await pal.waitFor({ timeout: 5000 });
    await pal.getByRole("combobox", { name: "Cari bentuk" }).fill("batas");
    await pal.getByRole("option", { name: "Batas sistem", exact: true }).click();
    await saved(ps);
    await ps.waitForTimeout(400);
    const all = (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards;
    const made = all.filter((c) => c.kind === "shape" && c.content.shape === "uml_system" && c.id !== sys.id);
    expect(made.length === 1 && made[0].content.fill === "" && all.every((c) => c.id === made[0].id || c.z > made[0].z), where("palet batas sistem"), JSON.stringify(made.map((c) => ({ z: c.z, fill: c.content.fill }))));

    // Ekspor PNG.
    if (W >= 768) await ps.getByRole("button", { name: "Ekspor", exact: true }).first().click();
    else {
      await ps.getByRole("button", { name: "Tambah", exact: true }).click();
      await ps.locator("dialog[open]").getByRole("button", { name: "Ekspor", exact: true }).click();
    }
    const ex = ps.getByRole("dialog", { name: "Ekspor papan" });
    await ex.waitFor({ timeout: 5000 });
    await ex.getByRole("radio", { name: /^PNG/ }).check();
    const [dl] = await Promise.all([ps.waitForEvent("download", { timeout: 20000 }), ex.getByRole("button", { name: "Unduh PNG" }).click()]);
    expect(dl.suggestedFilename().endsWith(".png"), where("ekspor"), dl.suggestedFilename());
    await ex.getByRole("button", { name: "Selesai" }).click();
    expect(errors.length === 0, where("galat halaman"), errors.join(" | "));

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
