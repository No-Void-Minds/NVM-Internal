// Uji browser revisi 57c di 1280 px (editor juga di 360 px): mind map, garis waktu, corong, dan chevron.
// Mind map: Tab menambah anak (langsung mengetik, Enter selesai), Enter menambah cabang sejajar, anak baru topik pusat
// ke sisi yang lebih sepi, tata letak otomatis (kolom per tingkat), lipat dan buka cabang (tombol 44 px, aria-expanded,
// cabang dan garisnya tersembunyi), Delete cabang beranak dengan konfirmasi, F2 mengedit, undo per aksi, klien kedua.
// Garis waktu: editor daftar (judul, label, tanggal, posisi), tambah dan hapus peristiwa, titik diseret dan digeser
// dengan panah. Corong dan chevron: tahap ditambah, diganti nama, dipindah dengan Alt dan panah; chevron lama tanpa
// tahap memakai teksnya sebagai tahap pertama. Muat ulang; ekspor PNG.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r57c.mjs
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
const settle = async (p) => {
  await p.waitForTimeout(400);
  await saved(p);
};
const shape = (s, text, extra = {}) => ({ shape: s, rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text, align: "center", font_size: 16, ...extra });

try {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r57c ${Date.now()}`, x: 9900, y: 12800, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const C = await add(0, 0, 200, 90, shape("mm_central", "Kampanye", { fill: "kuning", font_size: 20 }));
  const T1 = await add(260, 15, 160, 60, shape("mm_topic", "Konten", { fill: "hijau" }));
  await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: C.id, to: T1.id, style: { start: "none", end: "none", line: "solid", color: "" }, geometry: { from: { port: "auto" }, to: { port: "auto" }, route: "curved", points: [] } });
  const funnel = await add(0, 500, 240, 180, shape("funnel", "", { stages: [{ label: "Kenal" }, { label: "Minat" }, { label: "Beli" }] }));
  const oldChevron = await add(300, 500, 180, 70, shape("chevron", "Rilis"));
  const view = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board;
  const cardOf = async (id) => (await view()).cards.find((c) => c.id === id);
  const until = async (fn, pred) => {
    let v = await fn();
    for (let k = 0; k < 30 && !pred(v); k++) {
      await new Promise((r) => setTimeout(r, 200));
      v = await fn();
    }
    return v;
  };
  const kidsOf = async (id) => {
    const v = await view();
    return v.connectors.filter((l) => l.from === id).map((l) => v.cards.find((c) => c.id === l.to)).filter(Boolean);
  };

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
  await pm.locator(`[data-card="${C.id}"]`).waitFor({ timeout: 10000 });

  const unselect = async () => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
  };
  const pick = async (id) => {
    await unselect();
    const b = await ps.locator(`[data-card="${id}"]`).boundingBox();
    // Tengah atas kartu: tepi luar cabang mind map bisa tertutup tombol lipat (44 px).
    await ps.mouse.click(b.x + b.width / 2, b.y + 6);
    await ps.waitForTimeout(150);
  };
  const press = async (k) => {
    await ps.keyboard.press(k);
    await settle(ps);
  };
  const fit = async () => {
    await ps.getByRole("button", { name: "Pas ke layar" }).click();
    await ps.waitForTimeout(300);
  };

  // 1. Tab di cabang Konten: anak baru langsung bisa diketik, Enter selesai; letaknya di kanan Konten.
  await pick(T1.id);
  await ps.keyboard.press("Tab");
  await ps.locator("[contenteditable=true]").first().waitFor({ timeout: 5000 });
  await ps.keyboard.type("Artikel");
  await ps.keyboard.press("Enter");
  await settle(ps);
  let kids = await until(() => kidsOf(T1.id), (k) => k.length === 1 && k[0].content.text === "Artikel");
  const A1 = kids[0];
  const t1 = await cardOf(T1.id);
  expect(A1?.content.text === "Artikel" && A1.x === t1.x + t1.w + 56, "Tab anak", JSON.stringify(A1 && { x: A1.x, text: A1.content.text, t1x: t1.x }));
  expect(A1?.content.fill === "hijau", "gaya anak ikut induk", A1?.content.fill);
  // 2. Enter: cabang sejajar di bawahnya, kolom yang sama.
  await press("Enter");
  await ps.keyboard.type("Video");
  await ps.keyboard.press("Enter");
  await settle(ps);
  kids = await until(() => kidsOf(T1.id), (k) => k.length === 2 && k.some((c) => c.content.text === "Video"));
  const V1 = kids.find((c) => c.content.text === "Video");
  const a1 = await cardOf(A1.id);
  expect(V1 && V1.x === a1.x && V1.y === a1.y + 60 + 16, "Enter sejajar", JSON.stringify(V1 && { x: V1.x, y: V1.y, ax: a1.x, ay: a1.y }));
  await pm.getByText("Video").first().waitFor({ timeout: 10000 }).catch(() => {});
  expect(await pm.getByText("Video").first().isVisible().catch(() => false), "klien kedua", "Manajer tidak melihat cabang baru");
  // 3. Tab di topik pusat: anak baru ke kiri (kanan sudah satu cabang).
  await pick(C.id);
  await ps.keyboard.press("Tab");
  await ps.locator("[contenteditable=true]").first().waitFor({ timeout: 5000 });
  await ps.keyboard.type("Iklan");
  await ps.keyboard.press("Enter");
  await settle(ps);
  kids = await until(() => kidsOf(C.id), (k) => k.length === 2);
  const L1 = kids.find((c) => c.id !== T1.id);
  expect(L1 && L1.x === C.x - 56 - 160, "anak topik pusat ke kiri", JSON.stringify(L1 && { x: L1.x }));
  // 4. Undo dua kali: teks Iklan, lalu cabangnya; redo mengembalikan cabang.
  await fit();
  await press("Escape");
  await ps.locator(".wb-canvas").focus();
  await press("Control+z");
  await press("Control+z");
  kids = await until(() => kidsOf(C.id), (k) => k.length === 1);
  expect(kids.length === 1, "undo tambah cabang", `${kids.length} anak`);
  await press("Control+Shift+z");
  kids = await until(() => kidsOf(C.id), (k) => k.length === 2);
  expect(kids.length === 2, "redo tambah cabang", `${kids.length} anak`);

  // 5. Lipat Konten: dua anak dan garisnya tersembunyi, tombol menyebut jumlahnya; buka lagi.
  await fit();
  // Tombol lipat tampil saat cabang tidak terpilih (saat terpilih, tempatnya milik pegangan port; lipat lewat menu).
  // Tombol lipat hanya tampil saat cabang di layar minimal 44 px tinggi: perbesar ke cabang Konten dulu.
  await pick(T1.id);
  await ps.keyboard.press("Shift+Digit2");
  await ps.waitForTimeout(400);
  expect((await ps.locator(`[data-mind-fold="${T1.id}"]`).count()) === 0, "lipat tersembunyi saat terpilih", "tombol lipat bertumpuk dengan pegangan port");
  await unselect();
  const fold = ps.locator(`[data-mind-fold="${T1.id}"]`);
  const fb = await fold.boundingBox();
  expect(fb && fb.width >= 43.5 && fb.height >= 43.5, "tombol lipat 44 px", JSON.stringify(fb));
  await fold.click();
  await settle(ps);
  const hiddenNow = (await ps.locator(`[data-card="${A1.id}"], [data-card="${V1.id}"]`).count()) === 0;
  expect(hiddenNow && (await fold.getAttribute("aria-expanded")) === "false" && (await fold.textContent()).includes("+2"), "lipat", `tersembunyi ${hiddenNow}, ${await fold.getAttribute("aria-expanded")}`);
  expect((await cardOf(T1.id)).content.folded === true, "lipat tersimpan", "folded tidak tersimpan");
  // Buka lewat menu klik kanan (jalur keyboard dan saat terpilih).
  await pick(T1.id);
  await ps.locator(`[data-card="${T1.id}"]`).click({ button: "right", position: { x: 150, y: 8 } });
  await ps.getByRole("menuitem", { name: "Buka cabang" }).first().click();
  await settle(ps);
  await unselect();
  expect((await ps.locator(`[data-card="${A1.id}"]`).count()) === 1 && (await fold.getAttribute("aria-expanded")) === "true", "buka", "cabang tidak tampil lagi");

  // 6. F2 mengedit teks cabang.
  await pick(V1.id);
  await ps.keyboard.press("F2");
  const f2 = await ps.locator("[contenteditable=true]").first().waitFor({ timeout: 3000 }).then(() => true, () => false);
  expect(f2, "F2 edit", "editor tidak terbuka");
  await ps.keyboard.press("Escape");
  await settle(ps);

  // 7. Delete Konten (beranak): konfirmasi, tiga kartu ke Tempat sampah; undo mengembalikan.
  await pick(T1.id);
  await ps.keyboard.press("Delete");
  const confirm = ps.getByTestId("wb-mind-delete-confirm");
  await confirm.waitFor({ timeout: 5000 });
  await confirm.click();
  await settle(ps);
  let ids = new Set((await view()).cards.map((c) => c.id));
  expect(![T1.id, A1.id, V1.id].some((id) => ids.has(id)), "hapus cabang", "cabang atau anaknya masih ada");
  await ps.locator(".wb-canvas").focus();
  await press("Control+z");
  ids = await until(async () => new Set((await view()).cards.map((c) => c.id)), (s) => s.has(T1.id));
  expect([T1.id, A1.id, V1.id].every((id) => ids.has(id)), "undo hapus cabang", "cabang tidak kembali");

  // 8. Garis waktu dari palet: empat peristiwa; editor daftar: label dan tanggal, tambah, hapus; undo per aksi.
  await unselect();
  await ps.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: "Bentuk", exact: true }).click();
  const pal = ps.getByRole("dialog", { name: "Pilih bentuk" });
  await pal.waitFor({ timeout: 5000 });
  await pal.getByRole("combobox", { name: "Cari bentuk" }).fill("garis waktu");
  await pal.getByRole("option", { name: "Garis waktu", exact: true }).click();
  await settle(ps);
  const TL = (await view()).cards.find((c) => c.content?.shape === "timeline");
  expect(TL?.content.events?.length === 4, "garis waktu dari palet", JSON.stringify(TL?.content.events));
  await fit();
  await unselect();
  await ps.locator(`[data-card="${TL.id}"]`).dblclick();
  const ed = ps.locator("[data-items-editor]");
  await ed.waitFor({ timeout: 5000 });
  const small = await ed.locator("button, input").evaluateAll((els) => els.filter((e) => { const b = e.getBoundingClientRect(); return b.width < 43.5 || b.height < 43.5; }).length);
  expect(small === 0, "editor 44 px", `${small} elemen kecil`);
  await ed.locator('[data-item="0"] [data-field="label"]').fill("Riset");
  await ed.locator('[data-item="0"] [data-field="label"]').press("Tab");
  await ps.keyboard.type("Jan 2027");
  await ps.keyboard.press("Tab");
  await settle(ps);
  let tl = await until(() => cardOf(TL.id), (c) => c.content.events[0].label === "Riset" && c.content.events[0].date === "Jan 2027");
  expect(tl.content.events[0].label === "Riset" && tl.content.events[0].date === "Jan 2027", "label dan tanggal", JSON.stringify(tl.content.events[0]));
  await ed.getByRole("button", { name: "Tambah peristiwa" }).click();
  await settle(ps);
  tl = await until(() => cardOf(TL.id), (c) => c.content.events.length === 5);
  expect(tl.content.events.length === 5, "tambah peristiwa", `${tl.content.events.length}`);
  await ed.getByRole("button", { name: "Hapus peristiwa Peristiwa 4" }).click();
  await settle(ps);
  tl = await until(() => cardOf(TL.id), (c) => c.content.events.length === 4);
  expect(!tl.content.events.some((e) => e.label === "Peristiwa 4"), "hapus peristiwa", tl.content.events.map((e) => e.label).join(","));
  await ps.keyboard.press("Escape");
  await settle(ps);
  await ps.locator(".wb-canvas").focus();
  await press("Control+z");
  tl = await until(() => cardOf(TL.id), (c) => c.content.events.length === 5);
  expect(tl.content.events.some((e) => e.label === "Peristiwa 4"), "undo hapus peristiwa", tl.content.events.map((e) => e.label).join(","));
  expect((await ps.locator(`[data-card="${TL.id}"] [data-event]`).count()) === 5, "label peristiwa tampil", "jumlah label tidak 5");

  // 9. Titik peristiwa: seret ke kanan dan panah kanan.
  await pick(TL.id);
  const first = tl.content.events[0];
  const dot = ps.locator(`[data-card="${TL.id}"] [data-event-dot="${first.id}"]`);
  await dot.waitFor({ timeout: 5000 });
  const db = await dot.boundingBox();
  const cb = await ps.locator(`[data-card="${TL.id}"]`).boundingBox();
  await ps.mouse.move(db.x + db.width / 2, db.y + db.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(db.x + db.width / 2 + cb.width * 0.1, db.y + db.height / 2, { steps: 6 });
  await ps.mouse.up();
  await settle(ps);
  tl = await until(() => cardOf(TL.id), (c) => c.content.events.find((e) => e.id === first.id).at > first.at + 5);
  const moved = tl.content.events.find((e) => e.id === first.id).at;
  expect(moved > first.at + 5, "seret titik", `${first.at} ke ${moved}`);
  await dot.focus();
  await ps.keyboard.press("ArrowRight");
  await settle(ps);
  tl = await until(() => cardOf(TL.id), (c) => c.content.events.find((e) => e.id === first.id).at === Math.round((moved + 1) * 10) / 10);
  expect(tl.content.events.find((e) => e.id === first.id).at === Math.round((moved + 1) * 10) / 10, "panah titik", `${tl.content.events.find((e) => e.id === first.id).at}`);

  // 10. Corong: ganti nama tahap, tambah dengan Enter di baris terakhir, Alt+panah atas; undo.
  await unselect();
  await ps.locator(`[data-card="${funnel.id}"]`).dblclick();
  await ed.waitFor({ timeout: 5000 });
  await ed.locator('[data-item="0"] [data-field="label"]').fill("Sadar");
  await ed.locator('[data-item="2"] [data-field="label"]').click();
  await ps.keyboard.press("Enter");
  await settle(ps);
  const focused = await ps
    .waitForFunction(() => document.activeElement?.closest("[data-item]")?.getAttribute("data-item") === "3", null, { timeout: 3000 })
    .then(() => true, () => false);
  expect(focused, "fokus tahap baru", "fokus tidak pindah ke tahap baru sesudah Enter");
  await ps.keyboard.type("Setia");
  await ps.keyboard.press("Alt+ArrowUp");
  await settle(ps);
  let fu = await until(() => cardOf(funnel.id), (c) => c.content.stages.map((s) => s.label).join("|") === "Sadar|Minat|Setia|Beli");
  expect(fu.content.stages.map((s) => s.label).join("|") === "Sadar|Minat|Setia|Beli", "tahap corong", fu.content.stages.map((s) => s.label).join("|"));
  await ps.keyboard.press("Escape");
  await settle(ps);
  expect((await ps.locator(`[data-card="${funnel.id}"] [data-stage]`).count()) === 4, "label tahap tampil", "jumlah label tahap tidak 4");
  await ps.locator(".wb-canvas").focus();
  await press("Control+z");
  fu = await until(() => cardOf(funnel.id), (c) => c.content.stages.map((s) => s.label).join("|") === "Sadar|Minat|Beli|Setia");
  expect(fu.content.stages.map((s) => s.label).join("|") === "Sadar|Minat|Beli|Setia", "undo pindah tahap", fu.content.stages.map((s) => s.label).join("|"));

  // 11. Chevron lama tanpa tahap: teksnya menjadi tahap pertama; tambah tahap menjadi deret dua chevron.
  await unselect();
  await ps.locator(`[data-card="${oldChevron.id}"]`).dblclick();
  await ed.waitFor({ timeout: 5000 });
  expect((await ed.locator('[data-item="0"] [data-field="label"]').inputValue()) === "Rilis", "chevron lama", "teks lama tidak menjadi tahap pertama");
  await ed.getByRole("button", { name: "Tambah tahap" }).click();
  await settle(ps);
  await ps.keyboard.type("Evaluasi");
  await ps.keyboard.press("Escape");
  await settle(ps);
  const ch = await until(() => cardOf(oldChevron.id), (c) => c.content.stages?.length === 2 && c.content.stages[1].label === "Evaluasi");
  expect(ch.content.stages?.map((s) => s.label).join("|") === "Rilis|Evaluasi" && ch.content.text === "", "chevron bertahap", JSON.stringify({ s: ch.content.stages, t: ch.content.text }));

  // 12. 360 px: editor daftar di dalam layar, tanpa gulir horizontal.
  const nctx = await browser.newContext({ storageState: staf.state, viewport: { width: 360, height: 740 } });
  const pn = await nctx.newPage();
  pn.on("pageerror", (e) => errors.push(String(e)));
  await pn.goto(BASE + `/whiteboard/${board}`);
  await saved(pn);
  await pn.locator(`[data-card="${TL.id}"]`).dblclick();
  const ned = pn.locator("[data-items-editor]");
  await ned.waitFor({ timeout: 5000 });
  const fitN = await ned.evaluate((el) => {
    const b = el.getBoundingClientRect();
    return { inside: b.left >= 0 && b.right <= innerWidth && b.top >= 0, scroll: document.documentElement.scrollWidth > innerWidth };
  });
  expect(fitN.inside && !fitN.scroll, "360 editor", JSON.stringify(fitN));
  await pn.keyboard.press("Escape");
  await nctx.close();

  // 13. Muat ulang dan ekspor PNG.
  await ps.reload();
  await saved(ps);
  await fit();
  expect((await ps.locator(`[data-card="${funnel.id}"] [data-stage]`).count()) === 4, "muat ulang", "label tahap hilang");
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
