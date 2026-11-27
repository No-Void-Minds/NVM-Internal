// Uji browser revisi 54e: ERD dan class.
// 1280 px: ERD dua entitas (Pelanggan dan Pesanan) dengan relasi, entitas lemah, atribut, dan atribut kunci bergaris
// bawah; garis Pelanggan ke Pesanan diberi preset "Satu ke banyak" lewat panel garis (ujung crow's foot), lalu ujung
// akhir diganti "nol atau banyak" lewat pemilih Ujung akhir. Class: interface dan enum bertanda stereotip, kelas
// abstrak bernama miring, enum tanpa bagian metode. Muat ulang tetap; klien kedua melihat ujung crow's foot dan
// stereotip; ekspor PNG tanpa galat. 360 px: palet menemukan bentuk ERD dan varian kelas.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r54e.mjs
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
const cls = (variant, name, attributes, methods) => ({ ...shape("uml_class", ""), variant, name, attributes, methods });

try {
  // 360: palet.
  {
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54e palet ${Date.now()}`, x: 9900, y: 11000, w: 200, h: 120 })).body.card;
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
    await box.fill("entitas");
    expect((await dlg.getByRole("option").count()) === 2, "360 palet entitas", (await dlg.getByRole("option").allTextContents()).join(", "));
    await box.fill("atribut");
    expect((await dlg.getByRole("option").count()) === 2, "360 palet atribut", (await dlg.getByRole("option").allTextContents()).join(", "));
    for (const n of ["Interface", "Enum", "Kelas abstrak"]) {
      await box.fill(n.toLowerCase());
      expect((await dlg.getByRole("option").allTextContents()).some((t) => t.trim() === n), `360 palet ${n}`, "tidak ditemukan");
    }
    const lay = await layout(ps);
    expect(!lay.overflow && lay.small.length === 0, "360 palet tata letak", JSON.stringify(lay));
    await ctx.close();
  }

  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r54e ${Date.now()}`, x: 9900, y: 11200, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const cust = await add(0, 100, 160, 70, shape("erd_entity", "Pelanggan"));
  const order = await add(460, 100, 160, 70, shape("erd_entity", "Pesanan"));
  await add(230, 90, 140, 90, shape("erd_relationship", "membuat"));
  const detail = await add(460, 300, 160, 70, shape("erd_weak_entity", "Detail pesanan"));
  const key = await add(-160, 0, 130, 60, shape("erd_key_attribute", "id_pelanggan"));
  await add(-160, 200, 130, 60, shape("erd_attribute", "nama"));
  const rel = (await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, { from: cust.id, to: order.id })).body.connector;
  const iface = await add(0, 500, 220, 150, cls("interface", "Pembayaran", "", "+ bayar()"));
  const abs = await add(260, 500, 220, 150, cls("abstract", "Akun", "# id: int", "+ masuk()"));
  const en = await add(520, 500, 220, 150, cls("enum", "Metode", "KARTU\nTUNAI", "tidak tampil"));
  const lines = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.connectors;

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
  await pm.locator(`[data-card="${cust.id}"]`).waitFor({ timeout: 10000 });

  // Bentuk ERD dan class.
  const deco = await ps.locator(`[data-card="${key.id}"]`).getByText("id_pelanggan").evaluate((el) => getComputedStyle(el.closest(".underline") ?? el).textDecorationLine);
  expect(deco.includes("underline"), "atribut kunci", `dekorasi ${deco}`);
  expect((await ps.locator(`[data-card="${detail.id}"] [data-mark]`).count()) === 1, "entitas lemah", "garis dalam tidak ada");
  expect((await ps.locator(`[data-card="${iface.id}"] [data-stereotype]`).textContent())?.trim() === "«interface»", "interface", "stereotip tidak ada");
  expect((await ps.locator(`[data-card="${en.id}"] [data-stereotype]`).textContent())?.trim() === "«enumeration»", "enum", "stereotip tidak ada");
  expect(!(await ps.locator(`[data-card="${en.id}"]`).getByText("tidak tampil").isVisible().catch(() => false)), "enum", "bagian metode masih tampil");
  const style = await ps.locator(`[data-card="${abs.id}"] [data-class-name]`).evaluate((el) => getComputedStyle(el).fontStyle);
  expect(style === "italic", "kelas abstrak", `gaya nama ${style}`);

  // Preset Satu ke banyak, lalu ujung akhir nol atau banyak.
  const openLine = async () => {
    await ps.locator(`[data-connector="${rel.id}"] path[data-hit]`).dispatchEvent("dblclick");
    const d = ps.getByRole("dialog", { name: "Garis penghubung" });
    await d.waitFor({ timeout: 5000 });
    return d;
  };
  let dlg = await openLine();
  await dlg.getByRole("button", { name: /^Preset garis/ }).click();
  await ps.getByTestId("wb-picker").getByRole("button", { name: "ERD: Satu ke banyak", exact: true }).click();
  await saved(ps);
  await ps.keyboard.press("Escape");
  let st = (await lines()).find((l) => l.id === rel.id).style;
  expect(st.start === "cf_one" && st.end === "cf_many", "preset satu ke banyak", JSON.stringify(st));
  // Kaki tiga cabang: path ujung akhir punya tiga sub-path.
  const heads = await ps.locator(`[data-connector="${rel.id}"] path`).evaluateAll((els) => els.slice(1, -1).map((e) => (e.getAttribute("d").match(/M/g) ?? []).length));
  expect(heads.includes(3), "crow's foot", `sub-path ujung ${JSON.stringify(heads)}`);
  await pm.waitForFunction((id) => [...document.querySelectorAll(`[data-connector="${id}"] path`)].some((e) => (e.getAttribute("d").match(/M/g) ?? []).length === 3), rel.id, { timeout: 10000 }).catch(() => {});
  expect(await pm.locator(`[data-connector="${rel.id}"] path`).evaluateAll((els) => els.some((e) => (e.getAttribute("d").match(/M/g) ?? []).length === 3)), "crow's foot klien kedua", "Manajer tidak melihat kaki tiga cabang");
  dlg = await openLine();
  await dlg.getByRole("button", { name: /^Ujung akhir:/ }).click();
  await ps.getByTestId("wb-picker").getByRole("button", { name: "Crow's foot: nol atau banyak", exact: true }).click();
  await saved(ps);
  await ps.keyboard.press("Escape");
  st = (await lines()).find((l) => l.id === rel.id).style;
  expect(st.end === "cf_zero_many", "ujung nol atau banyak", JSON.stringify(st));

  // Muat ulang dan ekspor.
  await ps.reload();
  await saved(ps);
  expect((await ps.locator(`[data-card="${iface.id}"] [data-stereotype]`).count()) === 1 && (await ps.locator(`[data-connector="${rel.id}"]`).count()) === 1, "muat ulang", "stereotip atau garis hilang");
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
