// Uji browser revisi 52: panel format menempel di bar pilihan dan bar pilihan tanpa select bawaan. Di 1280 dan 360 px:
// saat mengedit judul checklist, panel format ada di slot bar pilihan (di kanan Hapus, atau di baris kedua bila
// tidak muat) dan tidak beririsan dengan kotak kartu yang diedit; Alt+F10, panah, Escape; tidak ada elemen select di
// halaman Whiteboard; palet Warna (kotak berlabel, centang di pilihan aktif, Tanpa latar, Gaya lainnya) dan menu Aksi
// (isi sama dengan klik kanan) bekerja dengan mouse dan keyboard; tombol 44 px dan tanpa gulir horizontal.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r52.mjs
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
const sApi = await api(browser, BASE, staf.state);
const holders = [];
const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
const box = (l) => l.boundingBox();
const overlap = (a, b) => !!a && !!b && a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height;

try {
  for (const W of [1280, 360]) {
    const where = `${W}`;
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r52 ${W} ${Date.now()}`, x: 9900, y: 8800 + holders.length * 200, w: 200, h: 120 })).body.card;
    holders.push(holder);
    const board = holder.ref.id;
    const add = async (body) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, body)).body.card;
    const list = await add({ kind: "checklist", x: 0, y: 260, w: 280, h: 140, content: { title: "Judul checklist uji", items: [{ id: crypto.randomUUID(), text: "Butir", done: false }] } });
    const note = await add({ kind: "note", x: 0, y: 0, w: 220, h: 100, content: { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Catatan" }] }] } } });
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: W, height: 800 } });
    const p = await ctx.newPage();
    const errors = [];
    p.on("console", (m) => m.type() === "error" && errors.push(m.text()));
    await p.goto(BASE + `/whiteboard/${board}`);
    await saved(p);
    await p.getByRole("button", { name: "Pas ke layar" }).click();
    await p.waitForTimeout(300);

    expect((await p.locator("select").count()) === 0, where, `${await p.locator("select").count()} elemen select di halaman Whiteboard`);

    // Mengedit judul checklist: panel format di slot bar pilihan, tidak beririsan dengan kartu.
    const titleText = p.locator(`[data-card="${list.id}"]`).getByText("Judul checklist uji");
    const tb = await box(titleText);
    await p.mouse.dblclick(tb.x + 10, tb.y + tb.height / 2);
    await p.locator(`[data-card="${list.id}"] [role=textbox][aria-label="Judul checklist"]`).click();
    await p.waitForFunction(() => document.activeElement?.classList.contains("ProseMirror"), null, { timeout: 10000 });
    const panel = p.getByTestId("wb-formatbar");
    await panel.waitFor({ timeout: 5000 });
    expect(await p.evaluate(() => !!document.querySelector("#wb-format-slot [data-testid=wb-formatbar]")), where, "panel format tidak di slot bar pilihan");
    const pb = await box(panel);
    const cb = await box(p.locator(`[data-card="${list.id}"]`));
    expect(!overlap(pb, cb), where, `panel format beririsan dengan kartu ${JSON.stringify({ pb, cb })}`);
    const sel = await box(p.getByTestId("wb-selbar"));
    const del = await box(p.getByTestId("wb-selbar").getByRole("button", { name: /^Hapus/ }));
    const right = pb.x >= del.x + del.width - 1 && Math.abs(pb.y - sel.y) < 2;
    const below = pb.y >= sel.y + sel.height - 1;
    expect(right || below, where, `panel bukan di kanan Hapus atau di baris kedua ${JSON.stringify({ pb, del, sel })}`);
    if (W === 1280) expect(right, where, "di 1280 panel seharusnya di kanan Hapus");
    expect((await panel.getByRole("button", { name: "Lainnya (kode, superskrip, subskrip)" }).count()) === 1, where, "tombol Lainnya tidak ada");
    // Alt+F10 ke panel, panah pindah tombol, Escape kembali ke teks.
    await p.keyboard.press("Alt+F10");
    const a1 = await p.evaluate(() => document.activeElement?.getAttribute("aria-label"));
    await p.keyboard.press("ArrowRight");
    const a2 = await p.evaluate(() => document.activeElement?.getAttribute("aria-label"));
    expect(!!a1 && !!a2 && a1 !== a2, where, `panah di panel (${a1} lalu ${a2})`);
    // Palet warna teks lewat keyboard: buka dengan Enter, panah bawah masuk palet, Enter memilih.
    await p.keyboard.press("Escape");
    await p.keyboard.press("Control+A");
    await p.keyboard.press("Alt+F10");
    for (let i = 0; i < 12; i++) {
      if ((await p.evaluate(() => document.activeElement?.getAttribute("aria-label"))) === "Warna teks") break;
      await p.keyboard.press("ArrowRight");
    }
    await p.keyboard.press("Enter");
    await p.keyboard.press("ArrowDown");
    await p.keyboard.press("ArrowRight");
    const swatch = await p.evaluate(() => document.activeElement?.getAttribute("aria-label"));
    await p.keyboard.press("Enter");
    expect(swatch === "Warna abu", where, `panah di palet warna teks berhenti di ${swatch}`);
    expect(await p.evaluate(() => document.activeElement?.classList.contains("ProseMirror")), where, "fokus tidak kembali ke teks setelah memilih warna");
    // Tombol di dalam kartu ikut zoom kanvas (perilaku kanvas sejak W1); yang diukur bar pilihan dan panel format.
    const small = await p.evaluate(() =>
      [...document.querySelectorAll("[data-testid=wb-selbar] button, [data-testid=wb-formatbar] button")]
        .filter((b) => b.offsetParent !== null)
        .filter((b) => b.getBoundingClientRect().width < 43.5 || b.getBoundingClientRect().height < 43.5)
        .map((b) => b.getAttribute("aria-label") || b.textContent.trim()),
    );
    const l = await layout(p);
    expect(small.length === 0, where, `tombol bar atau panel di bawah 44 px: ${small.join(", ")}`);
    expect(!l.overflow, where, "gulir horizontal saat mengedit");
    await p.keyboard.press("Escape");
    await saved(p);
    expect((await p.getByTestId("wb-formatbar").count()) === 0, where, "panel format tetap ada setelah keluar editor");
    const lc = (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.find((c) => c.id === list.id);
    expect(JSON.stringify(lc.content.title_doc ?? null).includes('"abu"'), where, "warna teks lewat palet keyboard tidak tersimpan");

    // Kartu di tepi atas kanvas (di bawah bar pilihan): panel pindah menempel di tepi bawah kanvas, tidak menutupi
    // kartu, dan kanvas tidak bergeser.
    const view0 = await p.locator(".wb-world").getAttribute("style");
    const nb = await box(p.locator(`[data-card="${note.id}"]`).getByText("Catatan"));
    await p.mouse.dblclick(nb.x + 8, nb.y + nb.height / 2);
    await p.waitForFunction(() => document.activeElement?.classList.contains("ProseMirror"), null, { timeout: 10000 });
    await p.waitForTimeout(200);
    const pn = await box(p.getByTestId("wb-formatbar"));
    const nc = await box(p.locator(`[data-card="${note.id}"]`));
    expect(!overlap(pn, nc), where, `panel menutupi kartu di bawah bar ${JSON.stringify({ pn, nc })}`);
    expect(inView(pn, W, 800), where, "panel keluar layar saat menghindari kartu");
    expect((await p.locator(".wb-world").getAttribute("style")) === view0, where, "kanvas bergeser saat mengedit kartu di bawah bar");
    await p.keyboard.press("Escape");
    await saved(p);

    // Palet Warna kartu dengan mouse: kotak berlabel, centang di pilihan aktif.
    await p.locator(`[data-card="${note.id}"]`).click();
    await p.getByRole("button", { name: /^Warna kartu terpilih/ }).click();
    const pal = p.getByTestId("wb-picker");
    expect(inView(await box(pal), W, 800), where, "palet keluar layar");
    await pal.getByRole("button", { name: "Warna biru" }).click();
    await saved(p);
    const color = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.find((c) => c.id === note.id);
    expect((await color()).color === "biru", where, "palet mouse tidak mengganti warna");
    await p.getByRole("button", { name: /^Warna kartu terpilih/ }).click();
    expect((await pal.getByRole("button", { name: "Warna biru" }).getAttribute("aria-pressed")) === "true" && (await pal.getByRole("button", { name: "Warna biru" }).locator("svg").count()) === 1, where, "warna aktif tidak ditandai centang");
    // Keyboard: panah kanan ke kotak berikut, Enter memilih, fokus kembali ke pemicu.
    await p.keyboard.press("ArrowRight");
    const next = await p.evaluate(() => document.activeElement?.getAttribute("aria-label"));
    await p.keyboard.press("Enter");
    await saved(p);
    expect(next === "Warna krem" && (await color()).color === "krem", where, `palet keyboard: ${next}, ${(await color()).color}`);
    expect(await p.evaluate(() => document.activeElement?.getAttribute("aria-label")?.startsWith("Warna kartu terpilih")), where, "fokus tidak kembali ke pemicu palet");
    // Tanpa latar dan Gaya lainnya.
    await p.keyboard.press("Enter");
    await pal.getByRole("button", { name: "Tanpa latar" }).click();
    await saved(p);
    expect((await color()).style?.bg === "transparan", where, "Tanpa latar tidak tersimpan");
    await p.getByRole("button", { name: /^Warna kartu terpilih/ }).click();
    await pal.getByRole("button", { name: "Gaya lainnya..." }).click();
    expect(await p.getByRole("dialog", { name: "Gaya kartu" }).isVisible(), where, "Gaya lainnya tidak membuka panel gaya");
    await p.keyboard.press("Escape");

    // Menu Aksi: isi sama dengan klik kanan; mouse dan keyboard.
    const aksi = p.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" });
    await aksi.click();
    const items = await p.getByTestId("wb-menu").first().getByRole("menuitem").allTextContents();
    await p.keyboard.press("Escape");
    expect(await p.evaluate(() => document.activeElement?.getAttribute("data-menu-button") === "bar"), where, "Escape menu Aksi tidak mengembalikan fokus");
    await p.locator(`[data-card="${note.id}"]`).click({ button: "right" });
    const ctxItems = await p.getByTestId("wb-menu").first().getByRole("menuitem").allTextContents();
    await p.keyboard.press("Escape");
    expect(JSON.stringify(items) === JSON.stringify(ctxItems), where, "isi menu Aksi berbeda dengan klik kanan");
    const n0 = (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.length;
    await aksi.focus();
    await p.keyboard.press("Enter");
    await p.getByTestId("wb-menu").first().waitFor({ timeout: 3000 });
    for (let i = 0; i < 8; i++) {
      if ((await p.evaluate(() => document.activeElement?.textContent ?? "")).startsWith("Duplikat")) break;
      await p.keyboard.press("ArrowDown");
    }
    await p.keyboard.press("Enter");
    await saved(p);
    expect((await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.length === n0 + 1, where, "Duplikat lewat menu Aksi (keyboard) tidak berjalan");

    const l2 = await layout(p);
    expect(!l2.overflow, where, "gulir horizontal");
    expect(errors.length === 0, where, `console: ${errors.join(" | ").slice(0, 300)}`);
    await ctx.close();
  }
} finally {
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

function inView(b, vw, vh) {
  return !!b && b.x >= -0.5 && b.y >= -0.5 && b.x + b.width <= vw + 0.5 && b.y + b.height <= vh + 0.5;
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
