// Uji browser revisi 54c: activity diagram dan state machine.
// 1280 px: activity (node awal, aksi, keputusan, fork yang diputar 90 derajat lewat menu, dua aksi, join, node akhir,
// flow final, kirim dan terima sinyal, object node) di swimlane dua lane, dan state machine (state awal, state,
// choice, state akhir) di dalam state komposit. Node awal dan fork pekat, node akhir bertanda; seret state komposit
// membawa isinya; muat ulang tetap; klien kedua melihat putaran fork; ekspor PNG tanpa galat.
// 360 px: palet menemukan bentuk activity dan state, tanpa gulir horizontal dan tombol minimal 44 px.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r54c.mjs
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
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54c palet ${Date.now()}`, x: 9900, y: 10200, w: 200, h: 120 })).body.card;
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
    await box.fill("fork");
    expect((await dlg.getByRole("option").allTextContents()).some((t) => /Fork atau join/.test(t)), "360 palet fork", "Fork atau join tidak ditemukan");
    await box.fill("state");
    const st = await dlg.getByRole("option").allTextContents();
    expect(st.filter((t) => /state/i.test(t)).length >= 4, "360 palet state", st.join(", "));
    const lay = await layout(ps);
    expect(!lay.overflow && lay.small.length === 0, "360 palet tata letak", JSON.stringify(lay));
    await ctx.close();
  }

  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54c ${Date.now()}`, x: 9900, y: 10400, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const link = (from, to) => sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from, to });
  // Activity di swimlane dua lane.
  await add(0, 0, 900, 560, shape("uml_swimlane", "Pesanan", { fill: "", lanes: [{ label: "Pelanggan", size: 150 }, { label: "Toko", size: 150 }] }));
  const init = await add(200, 90, 40, 40, shape("uml_initial", ""));
  const act = await add(140, 170, 160, 70, shape("uml_action", "Kirim pesanan"));
  const dec = await add(196, 270, 48, 48, shape("uml_decision", "Stok?"));
  const fork = await add(540, 120, 160, 40, shape("uml_fork", ""));
  const a1 = await add(480, 200, 140, 60, shape("uml_action", "Kemas"));
  const a2 = await add(640, 200, 140, 60, shape("uml_action", "Tagih"));
  const final = await add(600, 470, 40, 40, shape("uml_final", ""));
  const ff = await add(200, 470, 40, 40, shape("uml_flow_final", ""));
  await add(470, 300, 160, 60, shape("uml_send_signal", "Kabari"));
  await add(660, 300, 160, 60, shape("uml_receive_signal", "Bayar masuk"));
  await add(480, 390, 140, 50, shape("uml_object", "Faktur"));
  await link(init.id, act.id);
  await link(act.id, dec.id);
  await link(dec.id, fork.id);
  await link(fork.id, a1.id);
  await link(fork.id, a2.id);
  await link(dec.id, ff.id);
  // State machine di state komposit.
  const comp = await add(1000, 0, 460, 320, shape("uml_state_composite", "Pembayaran", { fill: "" }));
  const si = await add(1040, 80, 40, 40, shape("uml_initial", ""));
  const s1 = await add(1120, 70, 150, 70, shape("uml_state", "Menunggu"));
  const ch = await add(1320, 85, 40, 40, shape("uml_choice", ""));
  const sf = await add(1320, 220, 40, 40, shape("uml_final", ""));
  await link(si.id, s1.id);
  await link(s1.id, ch.id);
  await link(ch.id, sf.id);
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
  await pm.locator(`[data-card="${fork.id}"]`).waitFor({ timeout: 10000 });

  const fillOf = (page, id) => page.locator(`[data-card="${id}"] svg path`).first().getAttribute("fill");
  expect(!["none", null, "var(--color-surface)"].includes(await fillOf(ps, init.id)), "node awal", `isian ${await fillOf(ps, init.id)}`);
  expect(!["none", null, "var(--color-surface)"].includes(await fillOf(ps, fork.id)), "fork", `isian ${await fillOf(ps, fork.id)}`);
  expect((await ps.locator(`[data-card="${final.id}"] [data-mark]`).count()) === 1, "node akhir", "lingkaran dalam tidak ada");
  expect((await ps.locator(`[data-card="${ff.id}"] [data-mark]`).count()) === 1, "flow final", "silang tidak ada");

  // Putar fork 90 derajat lewat menu Aksi; Manajer melihat putarannya.
  await ps.locator(`[data-card="${fork.id}"]`).click();
  await ps.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
  await ps.getByTestId("wb-menu").getByRole("menuitem", { name: /^Putar 90 derajat/ }).click();
  await saved(ps);
  expect((await cards()).get(fork.id).content.rotation === 90, "putar fork", "rotasi fork tidak 90");
  await pm.waitForFunction((id) => getComputedStyle(document.querySelector(`[data-card="${id}"]`)).transform !== "none" && document.querySelector(`[data-card="${id}"]`).style.transform.includes("rotate(90deg)"), fork.id, { timeout: 10000 }).catch(() => {});
  expect(await pm.locator(`[data-card="${fork.id}"]`).evaluate((el) => el.style.transform.includes("rotate(90deg)")), "putar fork klien kedua", "Manajer tidak melihat fork diputar");

  // Seret state komposit membawa isinya.
  await ps.locator(".wb-canvas").focus();
  await ps.keyboard.press("Escape");
  const c0 = await cards();
  const cb = await ps.locator(`[data-card="${comp.id}"]`).boundingBox();
  await ps.mouse.move(cb.x + cb.width - 15, cb.y + cb.height - 15);
  await ps.mouse.down();
  await ps.mouse.move(cb.x + cb.width - 15, cb.y + cb.height + 25, { steps: 8 });
  await ps.mouse.up();
  await saved(ps);
  await ps.waitForTimeout(400);
  const c1 = await cards();
  const d = (id) => `${c1.get(id).x - c0.get(id).x},${c1.get(id).y - c0.get(id).y}`;
  expect(d(comp.id) !== "0,0" && [si.id, s1.id, ch.id, sf.id].every((id) => d(id) === d(comp.id)), "state komposit", `komposit ${d(comp.id)}, isi ${[si.id, s1.id, ch.id, sf.id].map(d).join(" ")}`);
  expect(d(act.id) === "0,0", "state komposit", "kartu di luar ikut bergeser");

  // Muat ulang dan ekspor.
  await ps.reload();
  await saved(ps);
  expect((await ps.locator(`[data-card="${fork.id}"]`).evaluate((el) => el.style.transform.includes("rotate(90deg)"))) && (await ps.locator(`[data-card="${final.id}"] [data-mark]`).count()) === 1, "muat ulang", "bentuk berubah sesudah muat ulang");
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
