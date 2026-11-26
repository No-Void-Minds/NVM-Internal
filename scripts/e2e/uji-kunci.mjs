// Uji browser gerbang akhir revisi 48 sampai 52: jalur klik kartu terkunci dan grid serta snap. Di 1280 dan 360 px:
// untuk kartu terkunci saja dan pilihan campuran (terkunci dan tidak), "Pindahkan ke papan lain" dan "Jadikan papan
// baru" tidak muncul di menu klik kanan maupun menu Aksi, dan pintasannya (Alt+Shift+M, Alt+Shift+B) menampilkan
// "Buka kunci dulu" tanpa memindah kartu; kartu tidak terkunci tetap menawarkan keduanya. G hanya mengatur grid dan
// Shift+G hanya snap (revisi 50 memisahkan keduanya), terbaca dari kelas kanvas dan saklar di menu area kosong.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-kunci.mjs
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
const holders = [];
const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
const MOVE = ["Pindahkan ke papan lain", "Jadikan papan baru"];

try {
  for (const W of [1280, 360]) {
    const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E kunci ${W} ${Date.now()}`, x: 9900, y: 8400 + holders.length * 200, w: 200, h: 120 })).body.card;
    holders.push(holder);
    const board = holder.ref.id;
    const note = async (text, x, locked) =>
      (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, {
        kind: "note", x, y: 300, w: 160, h: 100, locked,
        content: { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text }] }] } },
      })).body.card;
    const a = await note("Terkunci", 0, true);
    const b = await note("Bebas", 220, false);
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: W, height: 800 } });
    const p = await ctx.newPage();
    await p.goto(BASE + `/whiteboard/${board}`);
    await saved(p);
    await p.getByRole("button", { name: "Pas ke layar" }).click();
    await p.locator(".wb-canvas").focus();
    await p.keyboard.press("Control+Minus");
    await p.waitForTimeout(300);

    const pick = async (ids) => {
      await p.locator(".wb-canvas").focus();
      await p.keyboard.press("Escape");
      await p.locator(`[data-card="${ids[0]}"]`).click();
      for (const id of ids.slice(1)) await p.locator(`[data-card="${id}"]`).click({ modifiers: ["Shift"] });
    };
    const menuItems = async () => {
      await p.getByTestId("wb-menu").first().waitFor({ timeout: 5000 });
      const t = await p.getByTestId("wb-menu").first().getByRole("menuitem").allTextContents();
      await p.keyboard.press("Escape");
      return t.join(" | ");
    };

    for (const [name, ids, locked] of [
      ["terkunci", [a.id], true],
      ["campuran", [a.id, b.id], true],
      ["bebas", [b.id], false],
    ]) {
      const where = `${W} ${name}`;
      await pick(ids);
      await p.locator(`[data-card="${ids[0]}"]`).click({ button: "right" });
      const ctxMenu = await menuItems();
      await p.getByTestId("wb-selbar").getByRole("button", { name: "Aksi" }).click();
      const barMenu = await menuItems();
      for (const m of MOVE) {
        expect(ctxMenu.includes(m) === !locked, where, `klik kanan ${locked ? "menampilkan" : "tidak menampilkan"} "${m}"`);
        expect(barMenu.includes(m) === !locked, where, `menu Aksi ${locked ? "menampilkan" : "tidak menampilkan"} "${m}"`);
      }
      if (!locked) continue;
      for (const key of ["Alt+Shift+KeyM", "Alt+Shift+KeyB"]) {
        await p.locator(".wb-canvas").focus();
        await pick(ids);
        await p.keyboard.press(key);
        const toast = await p.getByText(/Buka kunci dulu/).first().isVisible().catch(() => false);
        expect(toast, where, `${key} tidak menampilkan "Buka kunci dulu"`);
        expect(!(await p.getByRole("dialog", { name: /Pindahkan/ }).isVisible().catch(() => false)), where, `${key} membuka dialog pindah`);
        await p.waitForTimeout(200);
      }
      const after = (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board.cards;
      expect(after.length === 2 && !after.some((c) => c.kind === "board"), where, "kartu terkunci terpindah atau papan baru dibuat");
    }

    // G hanya grid, Shift+G hanya snap.
    const states = async () => {
      const grid = await p.locator(".wb-canvas").evaluate((el) => el.classList.contains("wb-grid"));
      const cb = await p.locator(".wb-canvas").boundingBox();
      await p.mouse.click(cb.x + cb.width - 30, cb.y + 80, { button: "right" });
      await p.getByTestId("wb-menu").first().waitFor({ timeout: 5000 });
      const snap = await p.getByRole("menuitemcheckbox", { name: /Snap ke grid/ }).getAttribute("aria-checked");
      const gridItem = await p.getByRole("menuitemcheckbox", { name: /^Grid/ }).getAttribute("aria-checked");
      await p.keyboard.press("Escape");
      return { grid, gridItem, snap };
    };
    await p.locator(".wb-canvas").focus();
    await p.keyboard.press("Escape");
    const s0 = await states();
    await p.locator(".wb-canvas").focus();
    await p.keyboard.press("g");
    const s1 = await states();
    expect(s1.grid !== s0.grid && s1.gridItem !== s0.gridItem && s1.snap === s0.snap, `${W} G`, `G mengubah ${JSON.stringify({ s0, s1 })}`);
    await p.locator(".wb-canvas").focus();
    await p.keyboard.press("Shift+G");
    const s2 = await states();
    expect(s2.grid === s1.grid && s2.snap !== s1.snap, `${W} Shift+G`, `Shift+G mengubah ${JSON.stringify({ s1, s2 })}`);
    await ctx.close();
  }
} finally {
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
