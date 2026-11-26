// Uji browser revisi 50: menu klik kanan, gaya kartu, grup, kunci posisi, dan alat gambar. Di 1280 px (Staf dan
// Manajer, dua klien): menu di kartu dan area kosong (klik kanan dan Shift+F10, panah, submenu, Escape mengembalikan
// fokus), setiap kelompok aksi minimal sekali, grup dua judul dan satu gambar yang berjauhan lalu digeser bersama,
// dobel klik masuk grup dan Escape keluar, grab grup ditolak dengan nama pemegang, judul tanpa latar terbaca di tema
// terang dan gelap, kunci menolak geser dan hapus lalu dibuka orang lain, rata dan sebar, jadikan papan baru, ganti
// gambar lewat menu dan lewat tempel dengan undo, potong lalu muat ulang, dan klien kedua melihat perubahan. Di 360 px
// (layar sentuh): tekan lama membuka menu di dalam layar dengan item 44 px.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r50.mjs
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
const holders = [];

async function makeBoard(tag) {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r50 ${tag} ${Date.now()}`, x: 9800, y: 9600 + holders.length * 200, w: 200, h: 120 })).body.card;
  holders.push(holder);
  return holder.ref.id;
}
const add = async (board, body) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, body)).body.card;
const heading = (board, text, x, y) => add(board, { kind: "heading", x, y, w: 240, h: 80, content: { text, level: 1 } });
const note = (board, text, x, y) =>
  add(board, { kind: "note", x, y, w: 200, h: 120, content: { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text }] }] } } });
const cardsOf = async (board) => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board;
const byId = async (board, id) => (await cardsOf(board)).cards.find((c) => c.id === id);
const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
const settle = async (p) => {
  await p.waitForTimeout(300);
  await saved(p);
};

// Gambar PNG dibuat di halaman (kanvas) lalu diunggah lewat rute unggahan biasa, atau dikembalikan sebagai berkas.
const pngIn = (p, w, h, color) =>
  p.evaluate(
    async ([w, h, color]) => {
      const c = document.createElement("canvas");
      c.width = w;
      c.height = h;
      const g = c.getContext("2d");
      g.fillStyle = color;
      g.fillRect(0, 0, w, h);
      const blob = await new Promise((r) => c.toBlob(r, "image/png"));
      return Array.from(new Uint8Array(await blob.arrayBuffer()));
    },
    [w, h, color],
  );
async function uploadImage(p, board, name, x, y) {
  return p.evaluate(
    async ([board, name, x, y]) => {
      const c = document.createElement("canvas");
      c.width = 300;
      c.height = 200;
      const g = c.getContext("2d");
      g.fillStyle = "#3b6";
      g.fillRect(0, 0, 300, 200);
      const blob = await new Promise((r) => c.toBlob(r, "image/png"));
      const fd = new FormData();
      fd.append("file", new File([blob], name, { type: "image/png" }));
      const q = new URLSearchParams({ card_id: crypto.randomUUID(), x: String(x), y: String(y), w: "240", h: "160" });
      const res = await fetch(`/api/whiteboard/boards/${board}/files?${q}`, { method: "POST", body: fd });
      return (await res.json()).card;
    },
    [board, name, x, y],
  );
}

const lin = (c) => {
  const f = (v) => {
    v /= 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * f(c[0]) + 0.7152 * f(c[1]) + 0.0722 * f(c[2]);
};
const ratio = (a, b) => (Math.max(lin(a), lin(b)) + 0.05) / (Math.min(lin(a), lin(b)) + 0.05);
const rgb = (s) => (s.match(/[\d.]+/g) ?? []).slice(0, 4).map(Number);

async function selectCards(p, ids) {
  await p.locator(".wb-canvas").focus();
  await p.keyboard.press("Escape");
  await p.locator(`[data-card="${ids[0]}"]`).click({});
  for (const id of ids.slice(1)) await p.locator(`[data-card="${id}"]`).click({ modifiers: ["Shift"] });
}
// Pas ke layar lalu perkecil satu langkah, supaya kartu paling atas tidak tertutup bar pilihan.
async function fitAll(p) {
  await p.getByRole("button", { name: "Pas ke layar" }).click();
  await p.locator(".wb-canvas").focus();
  await p.keyboard.press("Control+Minus");
  await p.waitForTimeout(300);
}
// Pilih satu kartu saja: Escape melepas pilihan dulu (klik pada kartu yang sudah terpilih tidak mengubah pilihan).
async function pick(p, id) {
  await p.locator(".wb-canvas").focus();
  await p.keyboard.press("Escape");
  await p.locator(`[data-card="${id}"]`).click();
}
// Revisi 52: select diganti pemilih popover (WbPicker): pemicu berlabel "Label: nilai", pilihan berupa tombol.
async function choose(scope, page, label, option) {
  await scope.getByRole("button", { name: new RegExp(`^${label}:`) }).click();
  await page.getByTestId("wb-picker").getByRole("button", { name: option, exact: true }).click();
}
const selectedCount = (p) => p.locator(".wb-card-selected").count();
const menuVisible = (p) => p.getByTestId("wb-menu").first().isVisible().catch(() => false);
async function menuItem(p, name) {
  await p.getByRole("menuitem", { name }).first().click();
  await p.waitForTimeout(200);
}
async function rightClick(p, id) {
  await p.locator(`[data-card="${id}"]`).click({ button: "right" });
  const ok = await p.getByTestId("wb-menu").first().waitFor({ timeout: 5000 }).then(() => true, () => false);
  if (!ok) {
    if (process.env.E2E_SHOT) await p.screenshot({ path: process.env.E2E_SHOT });
    throw new Error(`menu tidak terbuka di kartu ${id}`);
  }
}
const inView = (b, vw, vh) => !!b && b.x >= -0.5 && b.y >= -0.5 && b.x + b.width <= vw + 0.5 && b.y + b.height <= vh + 0.5;
async function smallItems(p) {
  return p.locator("[data-testid=wb-menu] [role^=menuitem]").evaluateAll((els) =>
    els.filter((e) => e.getBoundingClientRect().height < 44 || e.getBoundingClientRect().width < 44).length,
  );
}
async function drag(p, id, dx, dy, hold = false) {
  const b = await p.locator(`[data-card="${id}"]`).boundingBox();
  const x = b.x + b.width / 2
  const y = b.y + b.height / 2
  await p.mouse.move(x, y);
  await p.mouse.down();
  await p.mouse.move(x + dx / 2, y + dy / 2, { steps: 4 });
  await p.mouse.move(x + dx, y + dy, { steps: 4 });
  if (!hold) await p.mouse.up();
}

try {
  // ------------------------------------------------------------------------------------------------ 1280 px
  {
    const W = 1280;
    const board = await makeBoard("1280");
    const h1 = await heading(board, "Judul satu", 0, 0);
    const h2 = await heading(board, "Judul dua", 1100, 0);
    const n1 = await note(board, "Catatan A", 0, 700);
    const n2 = await note(board, "Catatan B", 330, 790);
    const n3 = await note(board, "Catatan C", 760, 740);
    const shape = await add(board, {
      kind: "shape", x: 1100, y: 420, w: 160, h: 100,
      content: { shape: "rect", rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text: "Bentuk", align: "center", font_size: 16 },
    });
    const opts = { viewport: { width: W, height: 800 }, permissions: ["clipboard-read", "clipboard-write"] };
    const cs = await browser.newContext({ storageState: staf.state, ...opts });
    const cm = await browser.newContext({ storageState: mgr.state, viewport: { width: W, height: 800 } });
    const ps = await cs.newPage();
    const pm = await cm.newPage();
    const errors = [];
    ps.on("console", (m) => m.type() === "error" && !/status of 4\d\d/.test(m.text()) && errors.push(m.text()));
    await ps.goto(BASE + `/whiteboard/${board}`);
    await saved(ps);
    const img = await uploadImage(ps, board, "poster.png", 0, 320);
    await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: img.id, to: shape.id });
    await ps.reload();
    await pm.goto(BASE + `/whiteboard/${board}`);
    for (const p of [ps, pm]) await saved(p);
    await fitAll(ps);
    await fitAll(pm);
    await ps.waitForTimeout(300);

    // Menu di kartu: klik kanan, di dalam layar, item 44 px, Escape menutup dan fokus kembali ke kartu.
    await rightClick(ps, n1.id);
    const mb = await ps.getByTestId("wb-menu").first().boundingBox();
    expect(inView(mb, W, 800), "menu kartu", `menu keluar layar ${JSON.stringify(mb)}`);
    expect((await smallItems(ps)) === 0, "menu kartu", "item menu di bawah 44 px");
    expect((await ps.getByRole("menu").first().getAttribute("aria-label"))?.includes("terpilih"), "menu kartu", "label menu");
    await ps.keyboard.press("ArrowDown");
    const second = await ps.evaluate(() => document.activeElement?.textContent ?? "");
    expect(/Salin/.test(second), "menu kartu", `panah bawah tidak pindah item (${second})`);
    await ps.keyboard.press("Escape");
    expect(!(await menuVisible(ps)) && (await ps.evaluate((id) => document.activeElement?.getAttribute("data-card") === id, n1.id)), "menu kartu", "Escape tidak menutup menu atau fokus tidak kembali");

    // Shift+F10 di kartu terfokus.
    await ps.keyboard.press("Shift+F10");
    expect(await menuVisible(ps), "Shift+F10", "menu tidak terbuka dengan Shift+F10");
    await ps.keyboard.press("Escape");

    // Menu area kosong di dekat tepi kanan bawah: tetap di dalam layar; submenu Tambah kartu lewat panah kanan.
    const canvasBox = await ps.locator(".wb-canvas").boundingBox();
    await ps.mouse.click(canvasBox.x + canvasBox.width - 20, canvasBox.y + 70, { button: "right" });
    await ps.getByTestId("wb-menu").first().waitFor({ timeout: 5000 });
    const eb = await ps.getByTestId("wb-menu").first().boundingBox();
    expect(inView(eb, W, 800), "menu kosong", `menu keluar layar di tepi kanvas ${JSON.stringify(eb)}`);
    const headingsBefore = (await cardsOf(board)).cards.filter((c) => c.kind === "heading").length;
    await ps.getByRole("menuitem", { name: "Tambah kartu di sini" }).focus();
    await ps.keyboard.press("ArrowRight");
    const sub = await ps.evaluate(() => document.activeElement?.textContent ?? "");
    expect(/Catatan/.test(sub), "submenu", `panah kanan tidak membuka submenu (${sub})`);
    const sb = await ps.getByTestId("wb-menu").nth(1).boundingBox();
    expect(inView(sb, W, 800), "submenu", `submenu keluar layar ${JSON.stringify(sb)}`);
    await ps.keyboard.press("ArrowDown");
    await ps.keyboard.press("Enter");
    await ps.waitForTimeout(600);
    await ps.keyboard.press("Escape");
    await settle(ps);
    const headingsAfter = (await cardsOf(board)).cards.filter((c) => c.kind === "heading");
    expect(headingsAfter.length === headingsBefore + 1, "tambah di sini", "judul tidak dibuat dari submenu");
    const made = headingsAfter.find((c) => ![h1.id, h2.id].includes(c.id));
    if (made) await sApi.call("POST", `/whiteboard/boards/${board}/cards/delete`, { ids: [made.id] });
    await ps.reload();
    await saved(ps);
    await fitAll(ps);

    // Grup dua judul dan satu gambar yang berjauhan (Ctrl+G), klik memilih seluruh grup, geser bersama.
    await selectCards(ps, [h1.id, h2.id, img.id]);
    await ps.keyboard.press("Control+g");
    await settle(ps);
    let cs1 = await cardsOf(board);
    const g = cs1.cards.find((c) => c.id === h1.id)?.group_ids?.[0];
    expect(!!g && [h2.id, img.id].every((id) => cs1.cards.find((c) => c.id === id)?.group_ids?.[0] === g), "grup", "tiga kartu tidak satu grup");
    await ps.locator(".wb-canvas").click({ position: { x: 5, y: canvasBox.height - 5 } });
    await ps.locator(`[data-card="${h1.id}"]`).click({});
    expect((await selectedCount(ps)) === 3, "grup", `klik memilih ${await selectedCount(ps)} kartu, bukan seluruh grup`);
    expect((await ps.getByTestId("wb-group-frame").count()) === 1, "grup", "bingkai grup tidak tampil");
    const before = Object.fromEntries(cs1.cards.map((c) => [c.id, c]));
    await drag(ps, h1.id, 120, 40);
    await settle(ps);
    cs1 = await cardsOf(board);
    const moved = [h1.id, h2.id, img.id].map((id) => {
      const c = cs1.cards.find((x) => x.id === id);
      return [Math.round(c.x - before[id].x), Math.round(c.y - before[id].y)];
    });
    expect(moved[0][0] !== 0 && moved.every((m) => m[0] === moved[0][0] && m[1] === moved[0][1]), "geser grup", `geseran anggota grup berbeda ${JSON.stringify(moved)}`);
    // Klien kedua melihat anggota grup yang berjauhan ikut bergeser.
    const h2After = cs1.cards.find((c) => c.id === h2.id);
    await pm
      .waitForFunction(([id, x]) => document.querySelector(`[data-card="${id}"]`)?.style.transform.includes(`translate(${x}px`), [h2.id, h2After.x], { timeout: 8000 })
      .catch(() => {});
    expect(await pm.evaluate(([id, x]) => document.querySelector(`[data-card="${id}"]`)?.style.transform.includes(`translate(${x}px`), [h2.id, h2After.x]), "realtime", "Manajer tidak melihat grup bergeser");

    // Dobel klik masuk grup (satu anggota terpilih), Escape keluar (seluruh grup terpilih lagi).
    await ps.locator(`[data-card="${h1.id}"]`).dblclick({});
    await ps.waitForTimeout(200);
    expect((await selectedCount(ps)) === 1, "masuk grup", `dobel klik memilih ${await selectedCount(ps)} kartu`);
    expect((await ps.locator(".ProseMirror").count()) === 0, "masuk grup", "dobel klik pertama membuka editor, bukan masuk grup");
    await ps.keyboard.press("Escape");
    expect((await selectedCount(ps)) === 3, "keluar grup", "Escape tidak kembali memilih seluruh grup");

    // Grab grup ditolak bila satu anggota dipegang orang lain, dengan nama pemegang.
    await drag(ps, h1.id, 60, 0, true);
    await pm.waitForTimeout(400);
    await pm.locator(`[data-card="${h2.id}"]`).click({});
    await drag(pm, h2.id, 60, 30);
    const toast = await pm.getByText(/sedang memindahkan/).first().textContent({ timeout: 5000 }).catch(() => "");
    expect(toast.includes(staf.user.name), "grab grup", `penolakan tanpa nama pemegang: "${toast}"`);
    await ps.mouse.up();
    await settle(ps);

    // Gaya: tanpa latar lewat Panel gaya (Ctrl+Alt+S), judul tetap terbaca di tema terang dan gelap.
    // Pilihan grup memuat gambar (tanpa Latar), jadi masuk grup dulu untuk memilih judulnya saja.
    await ps.locator(`[data-card="${h1.id}"]`).click({});
    await ps.locator(`[data-card="${h1.id}"]`).dblclick({});
    await ps.keyboard.press("Control+Alt+s");
    const dlg = ps.getByRole("dialog", { name: "Gaya kartu" });
    await dlg.waitFor({ timeout: 5000 });
    await choose(dlg, ps, "Latar", "Tanpa latar");
    await choose(dlg, ps, "Garis tepi", "Tebal");
    await ps.keyboard.press("Escape");
    await ps.keyboard.press("Escape");
    await settle(ps);
    const st = (await byId(board, h1.id))?.style;
    expect(st?.bg === "transparan" && st?.border === "tebal", "gaya", `gaya tersimpan ${JSON.stringify(st)}`);
    for (const theme of ["terang", "gelap"]) {
      const m = await ps.evaluate(
        ([id, theme]) => {
          if (theme === "gelap") document.documentElement.setAttribute("data-theme", "gelap");
          else document.documentElement.removeAttribute("data-theme");
          const card = document.querySelector(`[data-card="${id}"]`);
          const text = card.querySelector("p") ?? card;
          return {
            bg: getComputedStyle(card).backgroundColor,
            fg: getComputedStyle(text).color,
            canvas: getComputedStyle(document.querySelector(".wb-canvas")).backgroundColor,
          };
        },
        [h1.id, theme],
      );
      const r = ratio(rgb(m.fg), rgb(m.canvas));
      expect(/rgba\(0, 0, 0, 0\)|transparent/.test(m.bg), `tanpa latar ${theme}`, `latar kartu ${m.bg}`);
      expect(r >= 4.5, `tanpa latar ${theme}`, `kontras judul ${r.toFixed(2)}:1`);
    }
    await ps.evaluate(() => document.documentElement.removeAttribute("data-theme"));
    // Layar Manajer menerima gaya baru lewat siaran.
    await pm.waitForFunction((id) => document.querySelector(`[data-card="${id}"]`)?.classList.contains("wb-st-clear"), h1.id, { timeout: 8000 }).catch(() => {});
    expect(await pm.evaluate((id) => document.querySelector(`[data-card="${id}"]`)?.classList.contains("wb-st-clear"), h1.id), "realtime", "gaya tidak tampil di layar Manajer");

    // Kunci posisi grup (Ctrl+Alt+L): geser dan hapus ditolak, gembok tampil; Manajer membuka kunci lewat menu.
    await ps.locator(`[data-card="${h1.id}"]`).click({});
    await ps.keyboard.press("Control+Alt+l");
    await settle(ps);
    cs1 = await cardsOf(board);
    expect([h1.id, h2.id, img.id].every((id) => cs1.cards.find((c) => c.id === id)?.locked), "kunci", "grup tidak terkunci");
    expect((await ps.getByTestId("wb-lock").count()) >= 1, "kunci", "gembok tidak tampil");
    const lockedX = cs1.cards.find((c) => c.id === h1.id).x;
    await drag(ps, h1.id, 100, 0);
    await settle(ps);
    expect((await byId(board, h1.id)).x === lockedX, "kunci", "kartu terkunci tergeser");
    expect(await ps.getByText(/terkunci/).first().isVisible().catch(() => false), "kunci", "tidak ada pesan kartu terkunci");
    await ps.keyboard.press("Delete");
    await settle(ps);
    expect(!!(await byId(board, h1.id)), "kunci", "kartu terkunci terhapus");
    await pm.reload();
    await saved(pm);
    await fitAll(pm);
    await rightClick(pm, h2.id);
    await menuItem(pm, "Buka kunci posisi");
    await settle(pm);
    cs1 = await cardsOf(board);
    expect([h1.id, h2.id, img.id].every((id) => !cs1.cards.find((c) => c.id === id)?.locked), "buka kunci", "Manajer tidak bisa membuka kunci");
    const act = (await sApi.call("GET", `/whiteboard/boards/${board}/activity`)).body;
    expect(JSON.stringify(act).includes("card_unlocked"), "buka kunci", "buka kunci tidak tercatat di riwayat");

    // Rata dan sebar (tiga catatan), samakan lebar, paling depan.
    await ps.reload();
    await saved(ps);
    await fitAll(ps);
    await selectCards(ps, [n1.id, n2.id, n3.id]);
    await ps.keyboard.press("Alt+Shift+KeyW");
    await settle(ps);
    let ns = (await cardsOf(board)).cards.filter((c) => [n1.id, n2.id, n3.id].includes(c.id));
    expect(new Set(ns.map((c) => c.y)).size === 1, "rata atas", `y berbeda ${ns.map((c) => c.y)}`);
    await rightClick(ps, n2.id);
    await ps.getByRole("menuitem", { name: "Rata" }).click();
    await menuItem(ps, "Sebar mendatar");
    await settle(ps);
    ns = (await cardsOf(board)).cards.filter((c) => [n1.id, n2.id, n3.id].includes(c.id)).sort((a, b) => a.x - b.x);
    const gaps = [ns[1].x - (ns[0].x + ns[0].w), ns[2].x - (ns[1].x + ns[1].w)];
    expect(Math.abs(gaps[0] - gaps[1]) <= 1, "sebar", `jarak tidak sama ${gaps}`);
    await ps.keyboard.press("Alt+Shift+Digit1");
    await settle(ps);
    ns = (await cardsOf(board)).cards.filter((c) => [n1.id, n2.id, n3.id].includes(c.id));
    expect(new Set(ns.map((c) => c.w)).size === 1, "ukuran", "lebar tidak disamakan");
    await pick(ps, n1.id);
    await ps.keyboard.press("Control+Shift+BracketRight");
    await settle(ps);
    const all = (await cardsOf(board)).cards;
    expect(all.find((c) => c.id === n1.id).z === Math.max(...all.map((c) => c.z)), "urutan", "paling depan tidak di atas semua");

    // Salin dan tempel gaya, duplikat, potong lalu tempel di sini.
    await ps.keyboard.press("Control+Alt+s");
    await choose(ps.getByRole("dialog", { name: "Gaya kartu" }), ps, "Sudut", "Bulat");
    await ps.keyboard.press("Escape");
    await settle(ps);
    await ps.keyboard.press("Control+Alt+c");
    await pick(ps, n2.id);
    await ps.keyboard.press("Control+Alt+v");
    await settle(ps);
    expect((await byId(board, n2.id))?.style?.radius === "bulat", "tempel gaya", "gaya tidak tertempel");
    const count = async () => (await cardsOf(board)).cards.length;
    const c0 = await count();
    await ps.keyboard.press("Control+d");
    await settle(ps);
    expect((await count()) === c0 + 1, "duplikat", "Ctrl+D tidak menduplikat");
    const dup = (await cardsOf(board)).cards.find((c) => c.kind === "note" && ![n1.id, n2.id, n3.id].includes(c.id));
    expect(dup?.style?.radius === "bulat", "duplikat", "gaya tidak ikut terduplikat");
    await ps.keyboard.press("Control+x");
    await settle(ps);
    expect((await count()) === c0, "potong", "Ctrl+X tidak memindahkan kartu");
    await ps.mouse.click(canvasBox.x + 300, canvasBox.y + 40, { button: "right" });
    await menuItem(ps, "Tempel di sini");
    await settle(ps);
    expect((await count()) === c0 + 1, "tempel di sini", "tempel di sini tidak membuat kartu");

    // Bentuk: putar 90 dan balik mendatar.
    await pick(ps, shape.id);
    await ps.keyboard.press("Shift+KeyR");
    await settle(ps);
    await ps.keyboard.press("Shift+KeyH");
    await settle(ps);
    const sc = (await byId(board, shape.id))?.content;
    expect(sc?.rotation === 90 && sc?.flip_x === true, "bentuk", `putar dan balik ${JSON.stringify(sc)}`);

    // Hubungkan dengan garis (urutan pilih), lalu undo.
    const conns = async () => (await cardsOf(board)).connectors.length;
    const k0 = await conns();
    await selectCards(ps, [n1.id, n3.id]);
    await ps.keyboard.press("Alt+Shift+KeyK");
    await settle(ps);
    const bAfter = await cardsOf(board);
    expect(bAfter.connectors.length === k0 + 1 && bAfter.connectors.some((c) => c.from === n1.id && c.to === n3.id), "hubungkan", "garis tidak dibuat dari kartu pertama ke kedua");
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Control+z");
    await settle(ps);
    expect((await conns()) === k0, "undo garis", "undo tidak menghapus garis baru");

    // Salin tautan kartu, lalu Manajer membuka tautan itu dan kartunya terpilih.
    await pick(ps, shape.id);
    await ps.keyboard.press("Alt+Shift+KeyU");
    await ps.waitForTimeout(300);
    const link = await ps.evaluate(() => navigator.clipboard.readText()).catch(() => "");
    expect(link.includes(`kartu=${shape.id}`), "salin tautan", `tautan ${link}`);
    if (link) {
      await pm.goto(link);
      await pm.waitForFunction((id) => document.querySelector(`[data-card="${id}"]`)?.classList.contains("wb-card-selected"), shape.id, { timeout: 8000 }).catch(() => {});
      expect(await pm.evaluate((id) => document.querySelector(`[data-card="${id}"]`)?.classList.contains("wb-card-selected"), shape.id), "salin tautan", "tautan tidak memilih kartunya");
    }

    // Ekspor pilihan, zoom ke pilihan, pilih semua sejenis, komentar.
    await ps.keyboard.press("Alt+Shift+KeyE");
    expect(await ps.getByRole("dialog", { name: /Ekspor/ }).isVisible().catch(() => false), "ekspor", "dialog ekspor pilihan tidak terbuka");
    await ps.keyboard.press("Escape");
    const zoomBefore = await ps.getByTestId("wb-zoom").textContent();
    await pick(ps, shape.id);
    await ps.keyboard.press("Shift+Digit2");
    await ps.waitForTimeout(200);
    expect((await ps.getByTestId("wb-zoom").textContent()) !== zoomBefore, "zoom pilihan", "zoom tidak berubah");
    await fitAll(ps);
    await pick(ps, n1.id);
    await ps.keyboard.press("Alt+Shift+KeyJ");
    const notes = (await cardsOf(board)).cards.filter((c) => c.kind === "note").length;
    expect((await selectedCount(ps)) === notes, "sejenis", `terpilih ${await selectedCount(ps)} dari ${notes} catatan`);
    await pick(ps, n1.id);
    await ps.keyboard.press("Control+Alt+KeyM");
    expect(await ps.getByRole("dialog", { name: /Komentar/ }).isVisible().catch(() => false), "komentar", "panel komentar tidak terbuka");
    await ps.keyboard.press("Escape");

    // Gambar: ganti lewat menu (posisi dan garis tetap), undo, ganti lewat tempel, klien kedua melihat gambar baru.
    await fitAll(ps);
    await ps.locator(`[data-card="${img.id}"]`).dblclick({});
    await ps.keyboard.press("Escape");
    await ps.locator(`[data-card="${img.id}"]`).dblclick({});
    const imgBefore = await byId(board, img.id);
    await rightClick(ps, img.id);
    const chooser = ps.waitForEvent("filechooser");
    await menuItem(ps, "Ganti gambar");
    const fc = await chooser;
    await fc.setFiles({ name: "baru.png", mimeType: "image/png", buffer: Buffer.from(await pngIn(ps, 120, 120, "#c33")) });
    await settle(ps);
    let imgNow = await byId(board, img.id);
    expect(imgNow.file.id !== imgBefore.file.id && imgNow.file.name === "baru.png" && imgNow.x === imgBefore.x && imgNow.y === imgBefore.y, "ganti gambar", `kartu ${JSON.stringify({ f: imgNow.file, x: imgNow.x })}`);
    expect((await cardsOf(board)).connectors.some((c) => c.from === img.id), "ganti gambar", "garis hilang setelah ganti gambar");
    await pm.waitForFunction(([id, f]) => [...document.querySelectorAll(`[data-card="${id}"] img`)].some((i) => i.src.includes(f)), [img.id, imgNow.file.id], { timeout: 8000 }).catch(() => {});
    expect(await pm.evaluate(([id, f]) => [...document.querySelectorAll(`[data-card="${id}"] img`)].some((i) => i.src.includes(f)), [img.id, imgNow.file.id]), "realtime gambar", "Manajer tidak melihat gambar berganti");
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Control+z");
    await settle(ps);
    expect((await byId(board, img.id)).file.id === imgBefore.file.id, "undo ganti", "undo tidak mengembalikan berkas lama");
    // Tempel (Ctrl+V) berkas gambar saat kartu gambar terpilih: berkasnya diganti.
    await ps.locator(`[data-card="${img.id}"]`).click({});
    const pasted = await pngIn(ps, 80, 80, "#33c");
    await ps.evaluate((bytes) => {
      const dt = new DataTransfer();
      dt.items.add(new File([new Uint8Array(bytes)], "tempel.png", { type: "image/png" }));
      document.dispatchEvent(new ClipboardEvent("paste", { clipboardData: dt, bubbles: true }));
    }, pasted);
    await ps.waitForTimeout(500);
    await settle(ps);
    imgNow = await byId(board, img.id);
    expect(imgNow.file.id !== imgBefore.file.id && imgNow.x === imgBefore.x, "tempel gambar", "tempel tidak mengganti gambar kartu terpilih");

    // Potong lalu muat ulang: tetap terpotong. Lihat ukuran penuh: Escape menutup dan fokus kembali.
    await rightClick(ps, img.id);
    await menuItem(ps, "Potong gambar");
    const crop = ps.getByRole("dialog", { name: "Potong gambar" });
    await crop.waitFor({ timeout: 5000 });
    await crop.getByLabel("Kiri (%)").fill("10");
    await crop.getByLabel("Kiri (%)").press("Tab");
    await crop.getByLabel("Lebar (%)").fill("50");
    await crop.getByLabel("Lebar (%)").press("Tab");
    await crop.getByLabel("Tinggi (%)").fill("50");
    await crop.getByLabel("Tinggi (%)").press("Tab");
    await crop.getByRole("button", { name: "Simpan potongan" }).click();
    await settle(ps);
    await ps.reload();
    await saved(ps);
    expect((await ps.locator(`[data-card="${img.id}"] [data-testid=wb-cropped]`).count()) === 1, "potong", "potongan hilang setelah muat ulang");
    const cropSaved = (await byId(board, img.id)).content.crop;
    expect(cropSaved && Math.abs(cropSaved.x - 0.1) < 0.01 && Math.abs(cropSaved.w - 0.5) < 0.01, "potong", `potongan ${JSON.stringify(cropSaved)}`);
    await fitAll(ps);
    await ps.locator(`[data-card="${img.id}"]`).click({});
    await ps.locator(`[data-card="${img.id}"]`).dblclick({});
    await ps.keyboard.press("Alt+Shift+KeyO");
    const full = ps.getByRole("dialog", { name: "Gambar ukuran penuh" });
    expect(await full.isVisible().catch(() => false), "lihat penuh", "dialog ukuran penuh tidak terbuka");
    const dl = ps.waitForEvent("download", { timeout: 8000 }).catch(() => null);
    await full.getByRole("link", { name: /Unduh berkas asli/ }).click();
    const d = await dl;
    expect(!!d && /\.png$/.test(d.suggestedFilename()), "unduh", `unduhan ${d?.suggestedFilename()}`);
    await ps.keyboard.press("Escape");

    // Jadikan papan baru: dua catatan dipindah ke papan anak baru.
    await fitAll(ps);
    await selectCards(ps, [n2.id, n3.id]);
    await ps.keyboard.press("Alt+Shift+KeyB");
    await ps.waitForTimeout(1200);
    await ps.keyboard.press("Escape");
    await settle(ps);
    const after = await cardsOf(board);
    const nb = after.cards.find((c) => c.kind === "board");
    const child = nb ? await cardsOf(nb.ref.id) : null;
    expect(!!nb && !after.cards.some((c) => [n2.id, n3.id].includes(c.id)) && child?.cards.length === 2, "papan baru", "pilihan tidak dipindah ke papan anak baru");

    const l = await layout(ps);
    expect(!l.overflow, `${W}`, "gulir horizontal");
    expect(errors.length === 0, `${W}`, `console: ${errors.join(" | ").slice(0, 300)}`);
    await cs.close();
    await cm.close();
  }

  // ------------------------------------------------------------------------------------------------ 360 px sentuh
  {
    const board = await makeBoard("360");
    const n = await note(board, "Tekan lama", 0, 0);
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: 360, height: 740 }, hasTouch: true, isMobile: true });
    const p = await ctx.newPage();
    await p.goto(BASE + `/whiteboard/${board}`);
    await saved(p);
    const b = await p.locator(`[data-card="${n.id}"]`).boundingBox();
    const cdp = await ctx.newCDPSession(p);
    const pt = { x: Math.round(b.x + b.width - 20), y: Math.round(b.y + 20) };
    await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [pt] });
    await p.waitForTimeout(700);
    await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
    await p.waitForTimeout(300);
    expect(await menuVisible(p), "360 tekan lama", "tekan lama tidak membuka menu");
    const mb = await p.getByTestId("wb-menu").first().boundingBox();
    expect(inView(mb, 360, 740), "360 menu", `menu keluar layar ${JSON.stringify(mb)}`);
    expect((await smallItems(p)) === 0, "360 menu", "item menu di bawah 44 px");
    const l = await layout(p);
    expect(!l.overflow, "360", "gulir horizontal");
    await ctx.close();
  }
} finally {
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await mApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
