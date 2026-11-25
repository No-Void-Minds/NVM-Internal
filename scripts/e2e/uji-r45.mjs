// Uji browser revisi 45: "Task saya" juga memuat task yang diminta pengguna, dengan penanda "Kamu peminta" bila dia
// bukan PIC task atau PIC sub task. Di 360 dan 1280 px, tema terang dan gelap. Membuat dua task uji di database dev
// lewat API lalu menghapusnya lagi.
//
// Cara menjalankan (dari scripts/e2e; backend dan Vite jalan; sesi dari scripts/e2e/.auth, atau variabel DEV_* sama
// dengan uji13.mjs bila sesi belum ada):
//   E2E_BASE=http://localhost:5174 node uji-r45.mjs
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
const mgr = await session(browser, BASE, "manajer");
const sApi = await api(browser, BASE, staf.state);
const mApi = await api(browser, BASE, mgr.state);
const made = [];

try {
  for (const width of [360, 1280]) {
    const tag = `${width}`;
    const countsBefore = (await sApi.call("GET", "/me/task-counts")).body.counts;
    // Staf meminta task yang PIC-nya Manajer; Manajer meminta task yang PIC-nya Staf.
    const asked = await sApi.call("POST", "/tasks", { title: `E2E r45 diminta Staf ${width}`, urgency: "medium", pic_ids: [mgr.user.id] });
    expect(asked.status === 201, tag, `Staf membuat task: ${asked.status} ${JSON.stringify(asked.body)}`);
    if (asked.status !== 201) break;
    made.push(asked.body.id);
    const picked = await mApi.call("POST", "/tasks", { title: `E2E r45 PIC Staf ${width}`, urgency: "medium", pic_ids: [staf.user.id] });
    expect(picked.status === 201, tag, `Manajer membuat task: ${picked.status}`);
    made.push(picked.body.id);

    const countsAfter = (await sApi.call("GET", "/me/task-counts")).body.counts;
    expect(countsAfter.requested === countsBefore.requested + 2, tag, `hitungan Requested Staf ${countsBefore.requested} lalu ${countsAfter.requested}, mau +2`);

    for (const scheme of ["light", "dark"]) {
      const where = `${tag} ${scheme}`;
      const ctx = await browser.newContext({ storageState: staf.state, viewport: { width, height: 800 }, colorScheme: scheme });
      const page = await ctx.newPage();
      const errors = [];
      page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
      await page.goto(BASE + `/task?q=${encodeURIComponent("E2E r45")}`);
      await page.waitForLoadState("networkidle");
      const cardOf = (id) => page.locator("article", { has: page.locator(".specimen-tag", { hasText: new RegExp(`^${id}$`) }) });
      const own = cardOf(asked.body.id);
      const theirs = cardOf(picked.body.id);
      expect((await own.count()) === 1, where, "task yang diminta Staf tidak muncul di Task saya");
      expect((await theirs.count()) === 1, where, "task yang dipegang Staf tidak muncul di Task saya");
      const badge = own.getByTestId("requester-badge");
      expect((await badge.count()) === 1 && (await badge.textContent()).trim() === "Kamu peminta", where, "penanda Kamu peminta tidak tampil");
      expect((await theirs.getByTestId("requester-badge").count()) === 0, where, "penanda tampil di task yang dipegang Staf");
      // Hak tidak berubah: peminta Staf tidak mendapat tombol Pindahkan di task orang lain.
      expect((await own.getByRole("button", { name: /^Pindahkan/ }).count()) === 0, where, "peminta Staf mendapat tombol Pindahkan");
      if (await badge.count()) {
        const ratio = await contrastOf(badge);
        expect(ratio >= 4.5, where, `kontras penanda ${ratio.toFixed(2)}:1`);
      }
      const l = await layout(page);
      expect(!l.overflow, where, "gulir horizontal");
      expect(l.small.length === 0, where, `tombol di bawah 44 px: ${l.small.join(", ")}`);
      expect(errors.length === 0, where, `console: ${errors.join(" | ").slice(0, 200)}`);
      await ctx.close();
    }
  }
} finally {
  // DELETE wajib membawa versi; tanpa itu backend menjawab 400 dan task uji tertinggal di database dev.
  for (const id of made) {
    const t = (await mApi.call("GET", `/tasks/${id}`)).body?.task;
    if (t) await mApi.call("DELETE", `/tasks/${id}?version=${t.version}`);
  }
  await sApi.ctx.close();
  await mApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
