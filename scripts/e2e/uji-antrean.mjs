// Uji antrean perubahan Whiteboard saat jaringan gagal sesaat tanpa WebSocket ikut putus (perbaikan setelah W5).
// Satu permintaan simpan posisi digagalkan (fetch gagal seperti jaringan putus), sementara WebSocket tetap tersambung
// dan browser tidak mengirim event offline atau online. Perubahan harus tetap tersimpan sendiri dalam beberapa detik
// dan penanda kembali Tersimpan, tanpa muat ulang dan tanpa aksi lain. Di kode lama antrean hanya dikirim ulang
// saat event online atau saat WebSocket bergabung ulang, jadi seretan itu menggantung selamanya.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-antrean.mjs
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
const sApi = await api(browser, BASE, staf.state);
const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E antrean ${Date.now()}`, x: 9500, y: 9500, w: 200, h: 120 })).body.card;
const board = holder.ref.id;
const doc = { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Kartu antrean" }] }] };
const note = (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "note", x: 0, y: 0, w: 240, h: 120, content: { doc } })).body.card;

try {
  const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: 1280, height: 800 } });
  const p = await ctx.newPage();
  await p.goto(BASE + `/whiteboard/${board}`);
  const label = () => p.locator("[data-testid=wb-save]").textContent().then((t) => (t ?? "").trim());
  await p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 });

  // Hanya permintaan simpan posisi berikutnya yang gagal; WebSocket tidak tersentuh.
  let failed = 0;
  await p.route("**/cards/patch", (route) => {
    if (failed === 0) {
      failed++;
      return route.abort("internetdisconnected");
    }
    return route.continue();
  });
  const b = await p.locator(`[data-card="${note.id}"]`).boundingBox();
  await p.mouse.move(b.x + 6, b.y + 6);
  await p.mouse.down();
  await p.mouse.move(b.x + 90, b.y + 6, { steps: 6 });
  await p.mouse.up();
  await p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Terputus", null, { timeout: 5000 }).catch(() => {});
  expect(failed === 1, "awal", "permintaan simpan posisi tidak tergagalkan");
  expect((await label()) === "Terputus", "awal", `penanda saat gagal: ${await label()}`);

  // Tanpa event online dan tanpa sambung ulang WebSocket: antrean harus terkirim sendiri.
  const ok = await p
    .waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 20000 })
    .then(() => true, () => false);
  expect(ok, "pulih", `penanda tidak kembali Tersimpan dalam 20 detik (${await label()})`);
  const saved = (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards.find((c) => c.id === note.id);
  expect(saved && saved.x !== note.x, "antrean", `seret yang gagal terkirim tidak tersimpan (x ${saved?.x})`);
  await ctx.close();
} finally {
  await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: [holder.id] });
  await sApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
