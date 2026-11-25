// Uji browser revisi 47: papan task realtime dengan dua context browser (Manajer dan Staf), di 360 dan 1280 px.
//   1. Manajer memindahkan kartu lewat tombol Pindahkan; papan Staf ikut berubah tanpa memuat ulang halaman.
//   2. Detail task yang sedang dibuka Staf menampilkan penanda dengan tombol Muat ulang, isian yang sedang diketik
//      tetap ada.
//   3. Dialog Pindahkan Staf menampilkan penanda; Muat ulang mengganti status di dialog.
//   4. (1280, drag hanya untuk mouse) Staf sedang menyeret kartu yang dipindah Manajer: papan menahan pembaruan
//      sampai kartu dilepas, lalu toast "<ID> sudah dipindah <nama> ke <status>" dan kartu tidak ikut dipindah.
//   5. Keputusan tambahan: klik angka di kartu profil Manajer membuka Task saya (scope=mine di URL, tombol Task saya
//      aktif); kontrol To Do tidak tampil di luar mode Atur.
// Penanda terputus tidak boleh tampil selama tersambung. Task uji dibuat lewat API dan dihapus lagi.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r47.mjs
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
const statusLabel = { requested: "Requested", on_progress: "On Progress", ready: "Ready", completed: "Completed" };

const staf = await session(browser, BASE, "staf");
const mgr = await session(browser, BASE, "manajer");
const sApi = await api(browser, BASE, staf.state);
const mApi = await api(browser, BASE, mgr.state);
const made = [];

async function makeTask(title) {
  const r = await mApi.call("POST", "/tasks", { title, urgency: "medium", pic_ids: [staf.user.id] });
  if (r.status !== 201) throw new Error(`membuat task: ${r.status} ${JSON.stringify(r.body)}`);
  made.push(r.body.id);
  return r.body.id;
}
const taskOf = async (id) => (await mApi.call("GET", `/tasks/${id}`)).body?.task;
// Perubahan nyata (versi naik): deliverable dengan URL unik.
let touches = 0;
async function touch(id) {
  const t = await taskOf(id);
  return mApi.call("PUT", `/tasks/${id}/deliverable`, { version: t.version, deliverable_url: `https://contoh.test/r47-${++touches}` });
}
async function moveByApi(id, to) {
  const t = await taskOf(id);
  return mApi.call("POST", `/tasks/${id}/move`, { version: t.version, to });
}

// Kolom tempat kartu berada, dibaca dari id judul kolom (kolom-<status>). Kolom yang disembunyikan di 360 px tetap
// ada di DOM, jadi ini bekerja di semua lebar.
const columnOf = (page, id) =>
  page.evaluate((id) => {
    const tag = [...document.querySelectorAll(".specimen-tag")].find((e) => e.textContent.trim() === id);
    const sec = tag?.closest("section[aria-labelledby]");
    return sec ? sec.getAttribute("aria-labelledby").replace("kolom-", "") : null;
  }, id);
const waitColumn = async (page, id, status, ms = 5000) => {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    if ((await columnOf(page, id)) === status) return true;
    await page.waitForTimeout(100);
  }
  return false;
};
const cardOf = (page, id) =>
  page.locator("article", { has: page.locator(".specimen-tag", { hasText: new RegExp(`^${id}$`) }) });

async function newPage(state, width) {
  const ctx = await browser.newContext({ storageState: state, viewport: { width, height: 900 } });
  const page = await ctx.newPage();
  const errors = [];
  page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
  return { ctx, page, errors };
}

try {
  for (const width of [360, 1280]) {
    const tag = `${width}`;
    const q = `E2E r47 ${width}`;
    const id = await makeTask(`${q} realtime`);

    const B = await newPage(staf.state, width);
    const A = await newPage(mgr.state, width);
    await B.page.goto(BASE + `/task?q=${encodeURIComponent(q)}`);
    await B.page.waitForLoadState("networkidle");
    await A.page.goto(BASE + `/task?q=${encodeURIComponent(q)}`);
    await A.page.waitForLoadState("networkidle");
    // Penanda di window: hilang kalau halaman dimuat ulang.
    await B.page.evaluate(() => (window.__r47 = 1));
    expect((await columnOf(B.page, id)) === "requested", tag, "kartu uji tidak ada di Requested papan Staf");
    expect((await B.page.getByTestId("live-offline").count()) === 0, tag, "penanda terputus tampil saat tersambung");

    // 1. Manajer memindahkan lewat tombol Pindahkan di papannya.
    if (width === 360) await A.page.getByRole("button", { name: /^Requested/ }).first().click();
    await cardOf(A.page, id).getByRole("button", { name: /^Pindahkan/ }).click();
    const dlg = A.page.getByRole("dialog", { name: `Pindahkan ${id}` });
    await dlg.waitFor({ timeout: 5000 });
    await dlg.getByRole("radio", { name: /On Progress/ }).check();
    await A.page.getByRole("button", { name: "Pindahkan", exact: true }).click();
    expect(await waitColumn(B.page, id, "on_progress"), tag, "papan Staf tidak ikut berubah setelah Manajer memindahkan");
    expect((await B.page.evaluate(() => window.__r47)) === 1, tag, "papan Staf dimuat ulang halaman penuh");

    // 2. Detail terbuka di Staf: penanda muncul, isian yang diketik tetap ada.
    if (width === 360) await B.page.getByRole("button", { name: /^On Progress/ }).first().click();
    await cardOf(B.page, id).getByRole("heading").getByRole("button").click();
    const panel = B.page.getByRole("dialog").last();
    await panel.getByText(`${q} realtime`).first().waitFor({ timeout: 5000 });
    const box = panel.locator("input[type=url], input[type=text], textarea").locator("visible=true").first();
    const typed = (await box.count()) > 0;
    if (typed) await box.fill("https://contoh.test/draf-r47");
    const upd = await touch(id);
    expect(upd.status === 200 || upd.status === 204, tag, `Manajer mengubah deliverable: ${upd.status}`);
    const note = panel.getByTestId("live-notice");
    expect(await note.waitFor({ timeout: 5000 }).then(() => true, () => false), tag, "detail tidak menampilkan penanda perubahan");
    if (await note.count()) {
      const txt = await note.textContent();
      expect(txt.includes(id) && txt.includes(mgr.user.name), tag, `teks penanda detail: ${txt}`);
      if (typed) expect((await box.inputValue()) === "https://contoh.test/draf-r47", tag, "isian di detail hilang saat penanda muncul");
      await note.getByRole("button", { name: "Muat ulang" }).click();
      expect(await note.waitFor({ state: "detached", timeout: 5000 }).then(() => true, () => false), tag, "penanda detail tidak hilang setelah Muat ulang");
    }
    await B.page.keyboard.press("Escape");
    await panel.waitFor({ state: "hidden", timeout: 5000 }).catch(() => {});

    // 3. Dialog Pindahkan terbuka di Staf: penanda muncul, Muat ulang mengganti status di dialog.
    await cardOf(B.page, id).getByRole("button", { name: /^Pindahkan/ }).click();
    const sd = B.page.getByRole("dialog", { name: `Pindahkan ${id}` });
    await sd.waitFor({ timeout: 5000 });
    const mv = await touch(id);
    expect(mv.status === 200 || mv.status === 204, tag, `Manajer mengubah deliverable: ${mv.status}`);
    const sn = sd.getByTestId("live-notice");
    expect(await sn.waitFor({ timeout: 5000 }).then(() => true, () => false), tag, "dialog Pindahkan tidak menampilkan penanda");
    if (await sn.count()) {
      await sn.getByRole("button", { name: "Muat ulang" }).click();
      const now = (await taskOf(id)).status;
      expect(
        await sd.getByText(`Status saat ini: ${statusLabel[now]}`).waitFor({ timeout: 5000 }).then(() => true, () => false),
        tag,
        `dialog tidak menampilkan status terbaru ${now}`,
      );
      expect((await sn.count()) === 0, tag, "penanda dialog tidak hilang setelah Muat ulang");
    }
    await B.page.getByRole("button", { name: "Batal" }).click();

    // 4. Drag di desktop: pembaruan ditahan sampai kartu dilepas.
    if (width === 1280) {
      const cur = (await taskOf(id)).status;
      if (cur !== "on_progress") await moveByApi(id, "on_progress");
      expect(await waitColumn(B.page, id, "on_progress"), tag, "kartu tidak kembali ke On Progress sebelum uji drag");
      const card = cardOf(B.page, id);
      const cb = await card.boundingBox();
      const target = await B.page.locator("section[aria-labelledby=kolom-ready]").boundingBox();
      await B.page.mouse.move(cb.x + cb.width / 2, cb.y + 20);
      await B.page.mouse.down();
      await B.page.mouse.move(cb.x + cb.width / 2 + 30, cb.y + 40, { steps: 5 });
      const dragging = await B.page.evaluate(() => !!document.querySelector("article.opacity-40, .opacity-40 article"));
      const r = await moveByApi(id, "completed").then(async (x) => (x.status === 200 ? x : moveByApi(id, "ready")));
      const movedTo = (await taskOf(id)).status;
      expect(r.status === 200, tag, `Manajer memindahkan saat Staf menyeret: ${r.status}`);
      await B.page.waitForTimeout(900);
      expect((await columnOf(B.page, id)) === "on_progress", tag, "papan dimuat ulang di tengah drag");
      await B.page.mouse.move(target.x + target.width / 2, target.y + 80, { steps: 8 });
      await B.page.mouse.up();
      const toast = B.page.getByText(`${id} sudah dipindah ${mgr.user.name} ke ${statusLabel[movedTo]}.`);
      expect(await toast.waitFor({ timeout: 5000 }).then(() => true, () => false), tag, `toast setelah drop tidak muncul (drag aktif: ${dragging})`);
      expect((await taskOf(id)).status === movedTo, tag, "kartu basi ikut dipindah setelah drop");
      expect(await waitColumn(B.page, id, movedTo), tag, `papan Staf tidak menampilkan ${movedTo} setelah drop`);
    }

    // 5a. Manajer: klik angka di kartu profil membuka Task saya.
    let countBtn = A.page.getByRole("button", { name: /task, tampilkan di papan/ }).locator("visible=true");
    if ((await countBtn.count()) === 0) {
      await A.page.getByRole("button", { name: /Profil saya/ }).locator("visible=true").first().click();
      countBtn = A.page.getByRole("button", { name: /task, tampilkan di papan/ }).locator("visible=true");
    }
    await countBtn.first().click();
    await A.page.waitForURL(/scope=mine/, { timeout: 5000 }).catch(() => {});
    expect(A.page.url().includes("scope=mine"), tag, `klik angka Manajer tidak membuka Task saya: ${A.page.url()}`);
    const mine = A.page.getByRole("group", { name: "Lingkup task" }).getByRole("button", { name: "Task saya" });
    expect((await mine.getAttribute("aria-pressed")) === "true", tag, "tombol Task saya Manajer tidak aktif");

    // 5b. To Do: tanpa mode Atur tidak ada tombol Hapus, juga saat baris disorot.
    const todo = await sApi.call("POST", "/todos", { title: `E2E r47 ${width} to do panjang untuk memastikan kontrol tidak menutupi teks` });
    await B.page.goto(BASE + "/task");
    await B.page.waitForLoadState("networkidle");
    const panelBtn = B.page.getByRole("button", { name: /^To Do/ }).locator("visible=true");
    if (await panelBtn.count()) await panelBtn.first().click();
    const ttl = B.page.getByRole("button", { name: `Ubah ${todo.body.todo.title}` }).locator("visible=true").first();
    await ttl.waitFor({ timeout: 5000 });
    const h0 = (await ttl.boundingBox()).height;
    await ttl.hover();
    expect((await B.page.getByRole("button", { name: `Hapus ${todo.body.todo.title}` }).locator("visible=true").count()) === 0, tag, "kontrol To Do tampil di luar mode Atur");
    expect(Math.abs((await ttl.boundingBox()).height - h0) < 1, tag, "baris To Do bergeser saat hover");
    await sApi.call("DELETE", `/todos/${todo.body.todo.id}`);

    for (const [who, P] of [["Staf", B], ["Manajer", A]]) {
      const l = await layout(P.page);
      expect(!l.overflow, `${tag} ${who}`, "gulir horizontal");
      expect(l.small.length === 0, `${tag} ${who}`, `tombol di bawah 44 px: ${l.small.join(", ")}`);
      expect(P.errors.length === 0, `${tag} ${who}`, `console: ${P.errors.join(" | ").slice(0, 200)}`);
      await P.ctx.close();
    }
  }
} finally {
  for (const id of made) {
    const t = await taskOf(id);
    if (t) await mApi.call("DELETE", `/tasks/${id}?version=${t.version}`);
  }
  await sApi.ctx.close();
  await mApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
