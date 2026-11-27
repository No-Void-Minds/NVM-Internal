// Uji browser revisi 54f: flowchart lanjutan, arsitektur, wireframe, mind map, dan bentuk lain, plus keterangan enum.
// 1280 px: satu contoh kecil per kelompok (alur flowchart, arsitektur pengguna ke database, halaman wireframe di
// jendela browser, mind map, dan slide dengan callout, chevron, corong, garis waktu). Semua bentuk tergambar dengan
// tandanya; kotak centang diganti ke tercentang lewat Gaya bentuk; garis dari port topik pusat ke cabang dan
// Hubungkan dengan garis antar cabang otomatis melengkung tanpa panah. Gaya bentuk kelas enum yang bagian metodenya
// berisi menampilkan keterangan (kontras diukur minimal 4,5:1), hilang saat varian diganti ke Kelas dan bagian metode
// tampil lagi. Muat ulang tetap; klien kedua melihat bentuk dan garisnya; ekspor PNG tanpa galat.
// 360 px: palet menemukan bentuk tiap kelompok baru, tanpa gulir horizontal dan tombol minimal 44 px.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r54f.mjs
import { chromium } from "playwright-core";
import { api, contrastOf, layout, realLogins, session } from "./sesi.mjs";

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
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54f palet ${Date.now()}`, x: 9900, y: 11400, w: 200, h: 120 })).body.card;
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
    const box = dlg.getByRole("combobox", { name: "Cari bentuk" });
    for (const [q, want] of [["input manual", "Input manual"], ["server", "Server"], ["dropdown", "Dropdown"], ["topik", "Topik pusat"], ["corong", "Corong"], ["garis waktu", "Garis waktu"]]) {
      await box.fill(q);
      expect((await dlg.getByRole("option").allTextContents()).some((t) => t.trim() === want), `360 palet ${q}`, "tidak ditemukan");
    }
    await box.fill("");
    for (const g of ["Arsitektur", "Wireframe", "Mind map", "Lainnya"])
      expect((await dlg.getByTestId("wb-shape-group").filter({ hasText: new RegExp(`${g} \\(`) }).count()) === 1, `360 palet kelompok ${g}`, "kelompok tidak ada");
    const lay = await layout(ps);
    expect(!lay.overflow && lay.small.length === 0, "360 palet tata letak", JSON.stringify(lay));
    await ctx.close();
  }

  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54f ${Date.now()}`, x: 9900, y: 11600, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const link = (from, to) => sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from, to });
  const made = {};
  // Flowchart.
  const fcIds = [];
  for (const [i, s] of ["fc_manual_input", "fc_predefined", "fc_sort", "fc_delay", "fc_preparation", "fc_merge", "fc_offpage"].entries())
    fcIds.push((made[s] = await add(i * 200, 0, s === "fc_offpage" ? 80 : 160, 80, shape(s, s === "fc_offpage" ? "A" : s.slice(3)))).id);
  for (let i = 0; i + 1 < fcIds.length; i++) await link(fcIds[i], fcIds[i + 1]);
  // Arsitektur.
  const arch = ["arch_user", "arch_laptop", "arch_cloud", "arch_firewall", "arch_server", "arch_database", "arch_component", "arch_node", "arch_phone"];
  for (const [i, s] of arch.entries()) made[s] = await add(i * 180, 200, s === "arch_cloud" || s === "arch_component" || s === "arch_node" ? 150 : 80, 90, shape(s, s.slice(5)));
  for (let i = 0; i + 1 < 6; i++) await link(made[arch[i]].id, made[arch[i + 1]].id);
  // Wireframe di jendela browser.
  made.wf_browser = await add(0, 420, 420, 300, shape("wf_browser", "Masuk"));
  made.wf_input = await add(30, 480, 220, 44, shape("wf_input", "Email", { align: "left" }));
  made.wf_checkbox = await add(30, 540, 180, 40, shape("wf_checkbox", "Ingat saya", { variant: "unchecked", align: "left" }));
  made.wf_dropdown = await add(30, 590, 200, 44, shape("wf_dropdown", "Peran", { align: "left" }));
  made.wf_button = await add(30, 650, 120, 44, shape("wf_button", "Masuk"));
  made.wf_image = await add(270, 480, 120, 100, shape("wf_image", ""));
  made.wf_phone = await add(460, 420, 180, 300, shape("wf_phone", "Layar"));
  // Mind map.
  made.mm_central = await add(800, 500, 200, 90, shape("mm_central", "Kampanye", { fill: "kuning", font_size: 20 }));
  const t1 = (made.mm_topic = await add(1100, 420, 160, 60, shape("mm_topic", "Konten")));
  const t2 = await add(1100, 640, 160, 60, shape("mm_topic", "Iklan"));
  // Lainnya.
  made.callout = await add(0, 800, 200, 120, shape("callout", "Catatan"));
  made.chevron = await add(240, 820, 180, 70, shape("chevron", "Langkah 1"));
  made.funnel = await add(460, 780, 160, 140, shape("funnel", "Pengunjung"));
  made.timeline = await add(660, 820, 420, 80, shape("timeline", "Rencana"));
  const enumCls = await add(1200, 0, 220, 160, { ...shape("uml_class", ""), variant: "enum", name: "Status", attributes: "AKTIF\nNONAKTIF", methods: "+ label()" });
  const lines = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.connectors;
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
  await pm.locator(`[data-card="${made.funnel.id}"]`).waitFor({ timeout: 10000 });

  // Semua bentuk tergambar, dengan tanda bila bentuknya punya.
  const withMarks = ["fc_predefined", "fc_sort", "arch_server", "arch_phone", "arch_firewall", "arch_component", "arch_node", "wf_browser", "wf_phone", "wf_dropdown", "wf_image", "funnel", "timeline"];
  for (const [s, c] of Object.entries(made)) {
    const n = await ps.locator(`[data-card="${c.id}"] svg path`).count();
    expect(n >= 1, `bentuk ${s}`, "tidak tergambar");
    if (withMarks.includes(s)) expect((await ps.locator(`[data-card="${c.id}"] [data-mark]`).count()) >= 1, `tanda ${s}`, "tanda tidak ada");
  }

  // Kotak centang ke tercentang lewat Gaya bentuk.
  const openStyle = async (id) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
    await ps.locator(`[data-card="${id}"]`).click();
    await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
    await ps.getByTestId("wb-menu").getByRole("menuitem", { name: /^Gaya bentuk/ }).click();
    const d = ps.getByRole("dialog", { name: "Gaya bentuk" });
    await d.waitFor({ timeout: 5000 });
    return d;
  };
  let style = await openStyle(made.wf_checkbox.id);
  await style.getByRole("button", { name: /^Varian:/ }).click();
  await ps.getByTestId("wb-picker").getByRole("button", { name: "Kotak centang tercentang", exact: true }).click();
  await saved(ps);
  await ps.keyboard.press("Escape");
  expect((await ps.locator(`[data-card="${made.wf_checkbox.id}"] [data-mark]`).count()) === 1, "kotak centang", "tanda centang tidak muncul");

  // Mind map: garis dari port topik pusat ke cabang melengkung tanpa panah.
  await ps.locator(".wb-canvas").focus();
  await ps.keyboard.press("Escape");
  await ps.locator(`[data-card="${made.mm_central.id}"]`).click();
  const port = ps.locator('[data-testid=wb-diagram-handles] [data-port="right"] circle').last();
  const pb = await port.boundingBox();
  const tb = await ps.locator(`[data-card="${t1.id}"]`).boundingBox();
  await ps.mouse.move(pb.x + pb.width / 2, pb.y + pb.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(tb.x + tb.width / 2, tb.y + tb.height / 2, { steps: 8 });
  await ps.mouse.up();
  await saved(ps);
  const mm = (await lines()).find((l) => l.from === made.mm_central.id && l.to === t1.id);
  expect(mm?.geometry.route === "curved" && mm?.style.end === "none", "mind map dari port", JSON.stringify(mm && { r: mm.geometry.route, e: mm.style.end }));
  // Hubungkan dengan garis antar cabang.
  await ps.locator(".wb-canvas").focus();
  await ps.keyboard.press("Escape");
  await ps.locator(`[data-card="${t1.id}"]`).click();
  await ps.locator(`[data-card="${t2.id}"]`).click({ modifiers: ["Shift"] });
  await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
  await ps.getByTestId("wb-menu").getByRole("menuitem", { name: /^Hubungkan dengan garis/ }).click();
  await saved(ps);
  await ps.keyboard.press("Escape");
  const mm2 = (await lines()).find((l) => l.from === t1.id && l.to === t2.id);
  expect(mm2?.geometry.route === "curved" && mm2?.style.end === "none", "mind map hubungkan", JSON.stringify(mm2 && { r: mm2.geometry.route, e: mm2.style.end }));
  await pm.locator(`[data-connector="${mm.id}"]`).waitFor({ timeout: 10000 }).catch(() => {});
  expect((await pm.locator(`[data-connector="${mm.id}"] path`).first().getAttribute("d"))?.includes("C"), "mind map klien kedua", "Manajer tidak melihat garis melengkung");

  // Keterangan enum di Gaya bentuk.
  style = await openStyle(enumCls.id);
  const note = style.getByTestId("wb-enum-note");
  expect(await note.isVisible().catch(() => false), "keterangan enum", "keterangan tidak tampil");
  const ratio = await contrastOf(note);
  expect(ratio >= 4.5, "keterangan enum kontras", `rasio ${ratio.toFixed(2)}`);
  await style.getByRole("button", { name: /^Varian:/ }).click();
  await ps.getByTestId("wb-picker").getByRole("button", { name: "Kelas", exact: true }).click();
  await saved(ps);
  expect(!(await note.isVisible().catch(() => false)), "keterangan enum", "keterangan masih tampil sesudah varian Kelas");
  expect(await ps.locator(`[data-card="${enumCls.id}"]`).getByText("+ label()").isVisible().catch(() => false), "metode kembali", "bagian metode tidak muncul lagi");
  await style.getByRole("button", { name: /^Varian:/ }).click();
  await ps.getByTestId("wb-picker").getByRole("button", { name: "Enum", exact: true }).click();
  await saved(ps);
  expect(await note.isVisible().catch(() => false), "keterangan enum lagi", "keterangan tidak tampil lagi");
  expect((await cards()).get(enumCls.id).content.methods === "+ label()", "isi metode", "isi metode berubah");
  await ps.keyboard.press("Escape");

  // Muat ulang, klien kedua, ekspor.
  await ps.reload();
  await saved(ps);
  expect((await ps.locator(`[data-card="${made.wf_checkbox.id}"] [data-mark]`).count()) === 1, "muat ulang", "kotak centang kembali kosong");
  expect((await pm.locator(`[data-card="${made.timeline.id}"] [data-mark]`).count()) === 4, "klien kedua", "Manajer tidak melihat garis waktu");
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
