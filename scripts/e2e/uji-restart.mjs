// Uji sambung ulang Whiteboard saat backend di-restart, seperti rilis.sh (W2, docs/whiteboard.md bagian 3), tanpa
// screenshot. Supaya backend dev yang sedang dipakai tidak ikut mati, skrip ini mem-build dan menjalankan backend
// kedua di 127.0.0.1:8081 (database dev yang sama, .env dibaca backend sendiri) dan Vite kedua di :5174 yang
// meneruskan /api ke sana (VITE_API_TARGET). Lalu:
//   1. tiga orang (Staf, Manajer, Admin) membuka papan uji yang sama;
//   2. backend dimatikan; Staf menyeret kartu selama backend mati (perubahan masuk antrean, penanda Terputus);
//   3. setelah 6 detik backend dinyalakan lagi;
//   4. lolos bila ketiganya kembali "Tersimpan", seret Staf tersimpan di server dan terlihat di layar Manajer, dan
//      waktu sambung ulang ketiganya tidak serentak (jeda acak).
//
// Cara menjalankan (dari scripts/e2e; variabel sama dengan uji13.mjs):
//   DEV_ADMIN_PASSWORD=... DEV_STAFF_USERNAME=... DEV_STAFF_PASSWORD=... DEV_MANAGER_USERNAME=... DEV_MANAGER_PASSWORD=... \
//     node uji-restart.mjs
import { execFileSync, spawn } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { chromium } from "playwright-core";

const ROOT = path.resolve(import.meta.dirname, "../..");
const BACKEND = "127.0.0.1:8081";
const BASE = "http://localhost:5174";
const roles = {
  staf: [process.env.DEV_STAFF_USERNAME, process.env.DEV_STAFF_PASSWORD],
  manajer: [process.env.DEV_MANAGER_USERNAME, process.env.DEV_MANAGER_PASSWORD],
  admin: [process.env.DEV_ADMIN_USERNAME || "admin", process.env.DEV_ADMIN_PASSWORD],
};
for (const [r, [u, p]] of Object.entries(roles)) {
  if (!u || !p) {
    console.error(`Akun ${r} belum diisi lewat variabel DEV_*.`);
    process.exit(2);
  }
}

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "nvm-restart-"));
const exe = path.join(tmp, process.platform === "win32" ? "nvm-server.exe" : "nvm-server");
execFileSync("go", ["build", "-o", exe, "./cmd/server"], { cwd: path.join(ROOT, "backend"), stdio: "inherit" });

let backend = null;
function startBackend() {
  backend = spawn(exe, [], { cwd: path.join(ROOT, "backend"), env: { ...process.env, HTTP_ADDR: BACKEND }, stdio: "ignore" });
}
async function waitHttp(url, ms) {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    try {
      const r = await fetch(url);
      if (r.status < 500) return true;
    } catch {
      // Belum siap.
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  return false;
}

startBackend();
// Satu string perintah (bukan daftar argumen) karena npx di Windows butuh shell; isinya tetap, tanpa masukan luar.
const vite = spawn("npx vite --port 5174 --strictPort", {
  cwd: path.join(ROOT, "frontend"),
  env: { ...process.env, VITE_API_TARGET: `http://${BACKEND}` },
  stdio: "ignore",
  shell: true,
});
const fails = [];
let checks = 0;
const expect = (cond, msg) => {
  checks++;
  if (!cond) fails.push(msg);
};

const browser = await chromium.launch({ channel: "msedge", headless: true });
try {
  if (!(await waitHttp(`http://${BACKEND}/api/health`, 30000))) throw new Error("backend kedua tidak menyala");
  if (!(await waitHttp(BASE + "/", 60000))) throw new Error("Vite kedua tidak menyala");

  const ctxs = {};
  for (const [r, [u, p]] of Object.entries(roles)) {
    const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
    const res = await ctx.request.post(BASE + "/api/auth/login", { headers: { Origin: BASE }, data: { username: u, password: p } });
    if (res.status() !== 200) throw new Error(`login ${r} ${res.status()}`);
    ctxs[r] = ctx;
  }
  const api = async (method, p, data) => {
    const r = await ctxs.staf.request.fetch(BASE + "/api" + p, { method, headers: { Origin: BASE }, data });
    return r.json().catch(() => null);
  };
  const holder = (await api("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E restart ${Date.now()}`, x: 9000, y: 9000, w: 200, h: 120 })).card;
  const board = holder.ref.id;
  const note = (await api("POST", `/whiteboard/boards/${board}/cards`, { kind: "note", x: 0, y: 0, w: 240, h: 120, content: { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Restart" }] }] } } })).card;
  await api("POST", `/whiteboard/boards/${board}/cards`, { kind: "note", x: 500, y: 300, w: 240, h: 120 });

  const pages = {};
  const opens = {};
  for (const r of Object.keys(roles)) {
    const page = await ctxs[r].newPage();
    opens[r] = [];
    page.on("websocket", (ws) => ws.url().includes("/api/whiteboard/ws") && opens[r].push(Date.now()));
    await page.goto(BASE + `/whiteboard/${board}`);
    await page.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 20000 });
    pages[r] = page;
  }

  // Backend mati, seperti saat rilis.sh mem-build ulang container.
  const killedAt = Date.now();
  backend.kill();
  await new Promise((r) => setTimeout(r, 1500));
  const ps = pages.staf;
  const box = await ps.locator(`[data-card="${note.id}"]`).boundingBox();
  await ps.mouse.move(box.x + 6, box.y + 6);
  await ps.mouse.down();
  await ps.mouse.move(box.x + 120, box.y + 6, { steps: 8 });
  await ps.mouse.up();
  await ps.locator("[data-testid=wb-offline]").waitFor({ timeout: 10000 }).catch(() => {});
  expect(await ps.locator("[data-testid=wb-offline]").isVisible(), "penanda Terputus tidak tampil saat backend mati");
  for (const [r, page] of Object.entries(pages)) {
    const label = (await page.locator("[data-testid=wb-save]").textContent()).trim();
    expect(label.includes("Terputus"), `${r}: penanda saat backend mati "${label}"`);
  }

  await new Promise((r) => setTimeout(r, 6000));
  startBackend();
  for (const [r, page] of Object.entries(pages)) {
    const ok = await page
      .waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 60000 })
      .then(() => true, () => false);
    expect(ok, `${r}: tidak kembali Tersimpan setelah backend menyala`);
  }
  const after = (await api("GET", `/whiteboard/boards/${board}`)).board.cards.find((c) => c.id === note.id);
  expect(after.x !== note.x, `seret selama backend mati hilang (x ${after.x})`);
  // Manajer melihat posisi baru tanpa muat ulang.
  const pm = pages.manajer;
  await pm.waitForTimeout(500);
  const z = parseInt((await pm.locator("text=/^\\d+%$/").first().textContent()) ?? "100", 10) / 100;
  const mb = await pm.locator(`[data-card="${note.id}"]`).boundingBox();
  const sb = (await pm.evaluate(() => document.querySelector(".wb-world")?.getBoundingClientRect().x)) ?? 0;
  // Kartu di layar ada di (posisi elemen dunia) + x * zoom.
  expect(mb && Math.abs(mb.x - sb - after.x * z) < 4, `Manajer tidak melihat posisi baru (${mb?.x}, ${sb}, ${after.x})`);
  // Jeda acak: waktu sambung ulang pertama setelah backend mati berbeda antar klien.
  const firstAfter = Object.fromEntries(Object.entries(opens).map(([r, ts]) => [r, ts.filter((t) => t > killedAt)]));
  const reconnectTimes = Object.values(firstAfter).map((ts) => ts.at(-1) ?? 0);
  const spread = Math.max(...reconnectTimes) - Math.min(...reconnectTimes);
  expect(reconnectTimes.every((t) => t > 0), `ada klien yang tidak mencoba menyambung ulang: ${JSON.stringify(firstAfter)}`);
  expect(spread > 20, `sambung ulang serentak (selisih ${spread} ms)`);
  console.log(`percobaan sambung ulang per klien: ${Object.entries(firstAfter).map(([r, ts]) => `${r} ${ts.length}`).join(", ")}; selisih sambungan terakhir ${spread} ms`);
  await api("POST", "/whiteboard/boards/root/cards/delete", { ids: [holder.id] });
} catch (e) {
  fails.push(String(e.message ?? e));
} finally {
  await browser.close();
  backend?.kill();
  // npx di Windows berjalan lewat shell; taskkill mematikan Vite beserta anaknya.
  if (process.platform === "win32") {
    try {
      execFileSync("taskkill", ["/pid", String(vite.pid), "/T", "/F"], { stdio: "ignore" });
    } catch {
      // Sudah berhenti.
    }
  } else vite.kill();
}
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}`);
for (const f of fails) console.log("FAIL " + f);
process.exitCode = fails.length ? 1 : 0;
