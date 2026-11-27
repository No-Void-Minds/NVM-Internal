// Uji browser revisi 57b di 1280 px: gambar di bingkai ponsel, jendela browser, dan placeholder gambar.
// Isi lewat klik kanan (Isi gambar..., unggah), ganti, Muat gambar utuh dan Isi penuh bingkai, Geser posisi gambar
// (panah dan seret, Escape selesai), Lepas gambar, undo dan redo; tempel Ctrl+V saat bingkai terpilih; seret kartu
// gambar ke bingkai kosong (bingkai terisi, kartu gambar ke Tempat sampah, satu langkah undo); bingkai ponsel sebagai
// wadah (tombol di dalamnya ikut terseret); undo "isi lagi" sesudah berkasnya dibersihkan (jawaban 400 ditiru)
// dilewati dengan pesan; klien kedua; muat ulang; ekspor PNG. Ketahanan berkas terhadap job pembersih, template, dan
// penghapusan kartu asal diuji di TestBingkaiBergambar (test DB).
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r57b.mjs
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

try {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r57b ${Date.now()}`, x: 9900, y: 12600, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const add = async (x, y, w, h, content) => (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content })).body.card;
  const phone = await add(0, 0, 180, 340, shape("wf_phone", "Layar"));
  const button = await add(30, 240, 120, 44, shape("wf_button", "Masuk"));
  const browserFrame = await add(260, 0, 360, 240, shape("wf_browser", "Halaman"));
  const place = await add(660, 0, 160, 120, shape("wf_image", ""));
  const view = async () => (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board;
  const cardOf = async (id) => (await view()).cards.find((c) => c.id === id);
  const until = async (id, pred) => {
    let c = await cardOf(id);
    for (let k = 0; k < 30 && !pred(c); k++) {
      await new Promise((r) => setTimeout(r, 200));
      c = await cardOf(id);
    }
    return c;
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
  await pm.locator(`[data-card="${phone.id}"]`).waitFor({ timeout: 10000 });

  const unselect = async () => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
  };
  const key = async (k) => {
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press(k);
    await settle(ps);
  };
  const rightClick = async (id) => {
    await unselect();
    const b = await ps.locator(`[data-card="${id}"]`).boundingBox();
    await ps.mouse.click(b.x + b.width - 6, b.y + 6);
    await ps.mouse.click(b.x + b.width - 6, b.y + 6, { button: "right" });
    await ps.getByTestId("wb-menu").first().waitFor({ timeout: 5000 });
  };
  const menuItem = async (name) => {
    await ps.getByRole("menuitem", { name }).first().click();
    await ps.waitForTimeout(200);
  };
  const shown = (p, id) => p.locator(`[data-card="${id}"] [data-frame-image] img`);

  // 1. Isi gambar lewat klik kanan: gambar tampil di layar ponsel, teks bawaan "Layar" disembunyikan.
  await rightClick(phone.id);
  let chooser = ps.waitForEvent("filechooser");
  await menuItem("Isi gambar...");
  await (await chooser).setFiles({ name: "layar.png", mimeType: "image/png", buffer: Buffer.from(await pngIn(ps, 300, 600, "#2a6")) });
  await settle(ps);
  let p = await until(phone.id, (c) => !!c.file);
  const first = p.file?.id;
  expect(!!first && (await shown(ps, phone.id).count()) === 1, "isi gambar", JSON.stringify(p.file));
  expect(!(await ps.locator(`[data-card="${phone.id}"]`).textContent()).includes("Layar"), "teks bawaan", "teks Layar masih tampil di atas gambar");
  await pm.locator(`[data-card="${phone.id}"] [data-frame-image] img`).waitFor({ timeout: 10000 }).catch(() => {});
  expect((await shown(pm, phone.id).count()) === 1, "klien kedua", "Manajer tidak melihat gambar bingkai");
  // Gambar ada di dalam kotak kartu (layar ponsel), bukan di luarnya.
  const cb = await ps.locator(`[data-card="${phone.id}"]`).boundingBox();
  const ib = await ps.locator(`[data-card="${phone.id}"] [data-frame-image]`).boundingBox();
  expect(ib.x > cb.x && ib.y > cb.y && ib.x + ib.width < cb.x + cb.width && ib.y + ib.height < cb.y + cb.height, "layar ponsel", `${JSON.stringify(ib)} di ${JSON.stringify(cb)}`);

  // 2. Muat gambar utuh, lalu Isi penuh bingkai; undo.
  await rightClick(phone.id);
  await menuItem("Muat gambar utuh");
  await settle(ps);
  p = await until(phone.id, (c) => c.content.image?.fit === "contain");
  expect(p.content.image?.fit === "contain" && (await shown(ps, phone.id).evaluate((i) => getComputedStyle(i).objectFit)) === "contain", "muat utuh", JSON.stringify(p.content.image));
  await key("Control+z");
  p = await until(phone.id, (c) => (c.content.image?.fit ?? "cover") === "cover");
  expect((p.content.image?.fit ?? "cover") === "cover", "undo muat utuh", JSON.stringify(p.content.image));

  // 3. Geser posisi gambar: panah kiri dua kali (posisi x naik 10), Escape selesai; seret ke kanan menurunkan x.
  await rightClick(phone.id);
  await menuItem("Geser posisi gambar");
  const pan = ps.locator(`[data-card="${phone.id}"] [data-frame-pan]`);
  await pan.waitFor({ timeout: 5000 });
  expect(await pan.evaluate((el) => el === document.activeElement), "fokus geser", "area geser tidak terfokus");
  await ps.keyboard.press("ArrowLeft");
  await settle(ps);
  await ps.keyboard.press("ArrowLeft");
  await settle(ps);
  p = await until(phone.id, (c) => c.content.image?.x === 60);
  expect(p.content.image?.x === 60 && p.content.image?.y === 50, "geser panah", JSON.stringify(p.content.image));
  const pb = await pan.boundingBox();
  await ps.mouse.move(pb.x + pb.width / 2, pb.y + pb.height / 2);
  await ps.mouse.down();
  await ps.mouse.move(pb.x + pb.width / 2 + pb.width * 0.2, pb.y + pb.height / 2, { steps: 6 });
  await ps.mouse.up();
  await settle(ps);
  p = await until(phone.id, (c) => c.content.image?.x < 60);
  expect(p.content.image?.x < 60, "geser seret", JSON.stringify(p.content.image));
  await ps.keyboard.press("Escape");
  await ps.waitForTimeout(200);
  expect((await pan.count()) === 0, "selesai geser", "area geser masih ada sesudah Escape");
  await key("Control+z");
  p = await until(phone.id, (c) => c.content.image?.x === 60);
  expect(p.content.image?.x === 60, "undo seret posisi", JSON.stringify(p.content.image));

  // 4. Bingkai ponsel sebagai wadah: tombol di dalamnya ikut terseret.
  await unselect();
  const hb = await ps.locator(`[data-card="${phone.id}"]`).boundingBox();
  await ps.mouse.move(hb.x + hb.width / 2, hb.y + 12);
  await ps.mouse.down();
  await ps.mouse.move(hb.x + hb.width / 2, hb.y + 12 + 60, { steps: 6 });
  await ps.mouse.up();
  await settle(ps);
  const ph1 = await cardOf(phone.id);
  const bt1 = await cardOf(button.id);
  expect(ph1.y > phone.y && bt1.y - button.y === ph1.y - phone.y, "wadah ponsel", `ponsel ${ph1.y}, tombol ${bt1.y}`);
  await key("Control+z");

  // 5. Lepas gambar, undo, redo.
  await rightClick(phone.id);
  await menuItem("Lepas gambar");
  await settle(ps);
  p = await until(phone.id, (c) => !c.file);
  expect(!p.file && (await shown(ps, phone.id).count()) === 0, "lepas gambar", JSON.stringify(p.file));
  await key("Control+z");
  p = await until(phone.id, (c) => c.file?.id === first);
  expect(p.file?.id === first && (await shown(ps, phone.id).count()) === 1, "undo lepas", JSON.stringify(p.file));
  await key("Control+Shift+z");
  p = await until(phone.id, (c) => !c.file);
  expect(!p.file, "redo lepas", JSON.stringify(p.file));

  // 6. Berkasnya sudah dibersihkan job harian (jawaban 400 server ditiru): undo dilewati dengan pesan, papan tetap.
  await ps.route("**/cards/patch", async (route) => {
    const body = route.request().postData() ?? "";
    if (body.includes(first)) await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: { code: "invalid", message: "Berkas itu sudah tidak ada. Unggah ulang berkasnya." } }) });
    else await route.continue();
  });
  await key("Control+z");
  await ps.waitForTimeout(400);
  const toast = await ps.getByText("Gambar sudah tidak tersedia (berkasnya sudah dibersihkan). Langkah ini dilewati.").count();
  const indicator = (await ps.getByTestId("wb-save").textContent())?.trim();
  expect(toast > 0 && indicator === "Tersimpan", "undo berkas hilang", `pesan ${toast}, penanda ${indicator}`);
  await ps.unroute("**/cards/patch");
  expect(!(await cardOf(phone.id)).file, "undo berkas hilang, bingkai tetap kosong", "bingkai terisi");

  // 7. Tempel Ctrl+V saat placeholder gambar terpilih: placeholder terisi, tanda silang hilang.
  await unselect();
  const plb = await ps.locator(`[data-card="${place.id}"]`).boundingBox();
  await ps.mouse.click(plb.x + plb.width / 2, plb.y + plb.height / 2);
  const bytes = await pngIn(ps, 200, 150, "#36c");
  await ps.evaluate((b) => {
    const dt = new DataTransfer();
    dt.items.add(new File([new Uint8Array(b)], "tempel.png", { type: "image/png" }));
    document.dispatchEvent(new ClipboardEvent("paste", { clipboardData: dt, bubbles: true }));
  }, bytes);
  await settle(ps);
  const pl = await until(place.id, (c) => !!c.file);
  expect(!!pl.file && (await shown(ps, place.id).count()) === 1 && (await ps.locator(`[data-card="${place.id}"] [data-mark]`).count()) === 0, "tempel ke placeholder", JSON.stringify(pl.file));

  // 8. Seret kartu gambar ke jendela browser kosong: bingkai terisi, kartu gambar ke Tempat sampah; satu undo
  // mengembalikan keduanya.
  const img = await ps.evaluate(async ([board]) => {
    const c = document.createElement("canvas");
    c.width = 240;
    c.height = 160;
    const g = c.getContext("2d");
    g.fillStyle = "#c63";
    g.fillRect(0, 0, 240, 160);
    const blob = await new Promise((r) => c.toBlob(r, "image/png"));
    const fd = new FormData();
    fd.append("file", new File([blob], "poster.png", { type: "image/png" }));
    const q = new URLSearchParams({ card_id: crypto.randomUUID(), x: "300", y: "400", w: "160", h: "110" });
    const res = await fetch(`/api/whiteboard/boards/${board}/files?${q}`, { method: "POST", body: fd });
    return (await res.json()).card;
  }, [board]);
  await ps.waitForTimeout(800);
  // Toast di bawah layar (langkah 6) bisa menutupi kartu gambar yang diseret; tunggu hilang dulu.
  await ps.locator("div[role=status].shadow-dialog").waitFor({ state: "detached", timeout: 10000 }).catch(() => {});
  await ps.getByRole("button", { name: "Pas ke layar" }).click();
  await ps.waitForTimeout(400);
  await unselect();
  const gb = await ps.locator(`[data-card="${img.id}"]`).boundingBox();
  const bb = await ps.locator(`[data-card="${browserFrame.id}"]`).boundingBox();
  await ps.mouse.move(gb.x + gb.width / 2, gb.y + 10);
  await ps.mouse.down();
  await ps.mouse.move(bb.x + bb.width / 2, bb.y + bb.height / 2, { steps: 10 });
  await ps.mouse.up();
  await settle(ps);
  const br = await until(browserFrame.id, (c) => !!c.file);
  const gone = !(await view()).cards.some((c) => c.id === img.id);
  expect(br.file?.id === img.file.id && gone, "seret ke bingkai", `bingkai ${br.file?.id}, kartu gambar hilang ${gone}`);
  await key("Control+z");
  const back = await until(browserFrame.id, (c) => !c.file);
  const restored = (await view()).cards.some((c) => c.id === img.id);
  expect(!back.file && restored, "undo seret ke bingkai", `bingkai ${back.file?.id}, kartu gambar kembali ${restored}`);

  // 9. Muat ulang dan ekspor PNG.
  await ps.reload();
  await saved(ps);
  await ps.getByRole("button", { name: "Pas ke layar" }).click();
  await ps.waitForTimeout(400);
  expect((await shown(ps, place.id).count()) === 1, "muat ulang", "gambar placeholder hilang");
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
