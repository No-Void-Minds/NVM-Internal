// Uji browser revisi 49: toolbar format teks melayang. Di 1280 px: di catatan, butir
// checklist, sel tabel, dan teks bentuk, teks diblok lalu ditebalkan (Ctrl+B), diberi warna Merah lewat toolbar, dan
// rata tengah (Ctrl+Shift+E). Format tersimpan di server, tetap ada setelah muat ulang, dan tampil di layar Manajer.
// Alt+F10 memindahkan fokus ke toolbar, panah berpindah tombol, Escape kembali ke teks. Toolbar tidak menutupi baris
// yang dipilih dan tetap di dalam layar, juga untuk kartu di pojok kanan atas kanvas (pindah ke bawah teks). Di layar
// sentuh (360 px, hanya catatan) toolbar menempel di bawah layar dan digulir menyamping. Satu editor TipTap terpasang
// sekaligus.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r49.mjs
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
const para = (text) => ({ type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text }] }] });
const holders = [];

async function makeBoard(width) {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r49 ${width} ${Date.now()}`, x: 9900, y: 9900 + width, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (body) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, body)).body.card;
  const cards = {
    note: await add({ kind: "note", x: 0, y: 0, w: 260, h: 120, content: { doc: para("Catatan format") } }),
    checklist: await add({ kind: "checklist", x: 0, y: 180, w: 260, h: 120, content: { title: "Daftar", items: [{ id: crypto.randomUUID(), text: "Butir format", done: false }] } }),
    table: await add({ kind: "table", x: 0, y: 360, w: 260, h: 100, content: { title: "", header: false, rows: [["Sel format", "b"]] } }),
    shape: await add({
      kind: "shape", x: 0, y: 520, w: 220, h: 120,
      content: { shape: "rect", rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text: "Bentuk format", align: "left", font_size: 16 },
    }),
    // Pojok kanan atas kanvas: toolbar harus pindah ke bawah teks dan tetap di dalam layar.
    edge: await add({ kind: "note", x: 700, y: -200, w: 260, h: 120, content: { doc: para("Tepi kanvas") } }),
  };
  return { board, cards };
}

const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
const editorCount = (p) => p.locator(".ProseMirror").count();

// Membuka field lewat dobel klik pada teksnya, lalu memblok seluruh isi field itu.
async function openAndSelect(p, card, sel, word) {
  const box = await p.locator(`[data-card="${card.id}"] ${sel}`).getByText(word).first().boundingBox();
  await p.mouse.dblclick(box.x + 8, box.y + box.height / 2);
  const ok = await p.waitForFunction(() => document.activeElement?.classList.contains("ProseMirror"), null, { timeout: 10000 }).then(() => true, () => false);
  if (!ok) {
    const info = await p.evaluate(() => ({ active: document.activeElement?.outerHTML.slice(0, 160), pm: document.querySelectorAll(".ProseMirror").length }));
    if (process.env.E2E_SHOT) await p.screenshot({ path: process.env.E2E_SHOT });
    expect(false, word, `editor tidak terbuka: ${JSON.stringify(info)}`);
    return false;
  }
  await p.keyboard.press("Control+A");
  return true;
}

const inView = (b, vw, vh) => b && b.x >= -0.5 && b.y >= -0.5 && b.x + b.width <= vw + 0.5 && b.y + b.height <= vh + 0.5;
const overlap = (a, b) => a && b && a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height;
const selectionBox = (p) =>
  p.evaluate(() => {
    const r = getSelection()?.getRangeAt(0).getClientRects()[0];
    return r ? { x: r.x, y: r.y, width: r.width, height: r.height } : null;
  });

try {
  for (const width of [360, 1280]) {
    const touch = width === 360;
    const { board, cards } = await makeBoard(width);
    const opts = { viewport: { width, height: 800 }, hasTouch: touch, isMobile: touch };
    const cs = await browser.newContext({ storageState: staf.state, ...opts });
    const cm = await browser.newContext({ storageState: mgr.state, viewport: { width, height: 800 } });
    const ps = await cs.newPage();
    const pm = await cm.newPage();
    const errors = [];
    ps.on("console", (m) => m.type() === "error" && errors.push(m.text()));
    for (const p of [ps, pm]) {
      await p.goto(BASE + `/whiteboard/${board}`);
      await saved(p);
    }

    const targets = [
      ["catatan", cards.note, "", "Catatan", (c) => c.content.doc],
      ["butir checklist", cards.checklist, "[data-item-text]", "Butir", (c) => c.content.items[0]?.doc],
      ["sel tabel", cards.table, '[data-cell-text="0,0"]', "Sel", (c) => c.content.docs?.[0]?.[0]],
      ["teks bentuk", cards.shape, "", "Bentuk", (c) => c.content.doc],
    ];
    // 360 px hanya bagian yang berbeda di ponsel: toolbar menempel di bawah dan digulir menyamping (satu field).
    for (const [name, card, sel, word, docOf] of touch ? targets.slice(0, 1) : targets) {
      const where = `${width} ${name}`;
      await ps.getByRole("button", { name: "Pas ke layar" }).click();
      await ps.waitForTimeout(300);
      if (!(await openAndSelect(ps, card, sel, word))) continue;
      expect((await editorCount(ps)) === 1, where, `editor TipTap terpasang ${await editorCount(ps)}`);
      const bar = ps.getByTestId("wb-formatbar");
      expect(await bar.isVisible(), where, "toolbar format tidak muncul");
      await ps.keyboard.press("Control+B");
      await bar.getByRole("button", { name: "Warna teks" }).click();
      await bar.getByRole("button", { name: "Merah", exact: true }).click();
      await ps.keyboard.press("Control+Shift+E");
      await ps.waitForTimeout(150);

      const bb = await bar.boundingBox();
      expect(inView(bb, width, 800), where, `toolbar keluar layar ${JSON.stringify(bb)}`);
      if (!touch) expect(!overlap(bb, await selectionBox(ps)), where, "toolbar menutupi baris yang dipilih");
      expect((await bar.getByRole("button", { name: "Tebal" }).getAttribute("aria-pressed")) === "true", where, "Tebal tidak aria-pressed");

      // Alt+F10 ke toolbar, panah pindah tombol, Escape kembali ke teks (pilihan tetap).
      await ps.keyboard.press("Alt+F10");
      const inBar = await ps.evaluate(() => !!document.activeElement?.closest("[role=toolbar]"));
      expect(inBar, where, "Alt+F10 tidak memindahkan fokus ke toolbar");
      const first = await ps.evaluate(() => document.activeElement?.getAttribute("aria-label"));
      await ps.keyboard.press("ArrowRight");
      const second = await ps.evaluate(() => document.activeElement?.getAttribute("aria-label"));
      expect(first !== second && !!second, where, `panah tidak pindah tombol (${first} lalu ${second})`);
      await ps.keyboard.press("Escape");
      const back = await ps.evaluate(() => ({ ok: !!document.activeElement?.classList.contains("ProseMirror"), el: document.activeElement?.getAttribute("aria-label") + " panel " + [...document.querySelectorAll("[data-tb][aria-expanded=true]")].map((b) => b.getAttribute("aria-label")).join(",") }));
      expect(back.ok, where, `Escape tidak kembali ke teks (${first} > ${second}, fokus ${back.el})`);

      if (touch) {
        const dock = await bar.evaluate((el) => {
          const r = el.getBoundingClientRect();
          const row = el.querySelector("[role=toolbar]");
          return { bottom: Math.round(r.bottom), vh: Math.round(visualViewport.height), w: Math.round(r.width), scroll: row.scrollWidth > row.clientWidth };
        });
        expect(Math.abs(dock.bottom - dock.vh) <= 1 && dock.w === width && dock.scroll, where, `toolbar sentuh tidak menempel di bawah atau tidak digulir menyamping ${JSON.stringify(dock)}`);
        const small = await bar.locator("button:visible").evaluateAll((bs) => bs.filter((b) => { const r = b.getBoundingClientRect(); return r.width < 44 || r.height < 44; }).length);
        expect(small === 0, where, `${small} tombol toolbar di bawah 44 px`);
      }

      await ps.keyboard.press("Escape");
      await saved(ps);
      expect((await editorCount(ps)) === 0, where, "editor masih terpasang setelah ditutup");
      expect((await ps.locator("[data-field]").count()) === 0, where, "kartu masih dalam mode edit setelah Escape");
      const c = (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.find((x) => x.id === card.id);
      const doc = JSON.stringify(docOf(c) ?? null);
      expect(doc.includes('"bold"') && doc.includes('"color":"merah"') && doc.includes('"textAlign":"center"'), where, `format tidak tersimpan: ${doc.slice(0, 200)}`);
    }

    if (touch) {
      expect(errors.length === 0, `${width}`, `console: ${errors.join(" | ").slice(0, 200)}`);
      await cs.close();
      await cm.close();
      continue;
    }
    // Muat ulang: format dirender dari JSON. Layar Manajer menerima perubahan yang sama lewat siaran.
    await ps.reload();
    await saved(ps);
    for (const [where, p] of [[`${width} muat ulang`, ps], [`${width} Manajer`, pm]]) {
      for (const [name, card] of [["catatan", cards.note], ["checklist", cards.checklist], ["tabel", cards.table], ["bentuk", cards.shape]]) {
        const sel = `[data-card="${card.id}"]`;
        await p.waitForFunction((s) => !!document.querySelector(`${s} .wb-al-center strong .wb-c-merah, ${s} .wb-al-center .wb-c-merah strong`), sel, { timeout: 10000 }).catch(() => {});
        const ok = await p.evaluate((s) => !!document.querySelector(`${s} .wb-al-center strong .wb-c-merah, ${s} .wb-al-center .wb-c-merah strong`), sel);
        expect(ok, `${where} ${name}`, "tebal, merah, atau rata tengah tidak tampil");
      }
      expect((await editorCount(p)) === 0, where, "TipTap terpasang tanpa ada field yang diedit");
    }
    // Teks merah terhitung terhadap latar kartu (putih) lolos 4,5:1: dibaca dari halaman, bukan ditebak.
    const red = await ps.evaluate((id) => {
      const el = document.querySelector(`[data-card="${id}"] .wb-c-merah`);
      return el ? getComputedStyle(el).color : "";
    }, cards.note.id);
    expect(/rgb\(140, 47, 34\)/.test(red), `${width} warna`, `warna merah terhitung ${red}`);

    // Kartu di pojok kanan atas: toolbar tetap di dalam layar dan tidak menutupi teks.
    await ps.getByRole("button", { name: "Pas ke layar" }).click();
    await ps.waitForTimeout(300);
    await openAndSelect(ps, cards.edge, "", "Tepi");
    await ps.waitForTimeout(150);
    const eb = await ps.getByTestId("wb-formatbar").boundingBox();
    expect(inView(eb, width, 800), `${width} tepi`, `toolbar keluar layar ${JSON.stringify(eb)}`);
    if (!touch) expect(!overlap(eb, await selectionBox(ps)), `${width} tepi`, "toolbar menutupi teks di tepi kanvas");
    // Ctrl+K membuka isian tautan; href berbahaya ditolak dengan pesan.
    await ps.keyboard.press("Control+K");
    const url = ps.getByLabel("Alamat tautan");
    expect(await url.isVisible(), `${width} tautan`, "Ctrl+K tidak membuka isian tautan");
    await url.fill("javascript:alert(1)");
    await url.press("Enter");
    expect(await ps.getByRole("alert").filter({ hasText: "Tautan harus diawali" }).isVisible(), `${width} tautan`, "href berbahaya tidak ditolak");
    await url.press("Escape");
    await ps.keyboard.press("Escape");
    await saved(ps);

    const l = await layout(ps);
    expect(!l.overflow, `${width}`, "gulir horizontal");
    expect(errors.length === 0, `${width}`, `console: ${errors.join(" | ").slice(0, 200)}`);
    await cs.close();
    await cm.close();
  }
} finally {
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
