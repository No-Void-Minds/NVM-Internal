// Uji browser revisi 51: kartu anggota tim. Di 1280 px: alat Tim menambah lima orang sekaligus (berjajar, tidak
// bertumpuk), menambah satu divisi lewat Pilih satu divisi, mengganti tampilan banyak kartu sekaligus lewat menu klik
// kanan, jabatan yang diubah lewat profil ikut berubah di layar orang lain tanpa muat ulang, dua kartu orang
// disambung dengan garis siku, kontras teks kartu dan tap target dialog. Kartu tidak membawa email, username, atau
// peran. Di 360 px: alat Tim lewat lembar Tambah, dialog tanpa gulir horizontal dan tombol 44 px.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-r51.mjs
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
const holders = [];
const me = (await sApi.call("GET", `/people/${staf.user.id}`)).body;
const oldTitle = me?.job_title ?? "";

async function makeBoard(tag) {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E r51 ${tag} ${Date.now()}`, x: 9900, y: 9300 + holders.length * 200, w: 200, h: 120 })).body.card;
  holders.push(holder);
  return holder.ref.id;
}
const boardOf = async (b) => (await sApi.call("GET", `/whiteboard/boards/${b}`)).body.board;
const persons = async (b) => (await boardOf(b)).cards.filter((c) => c.kind === "person");
const saved = (p) =>
  p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
async function tool(p, width, label) {
  if (width >= 768) await p.getByRole("complementary", { name: "Alat papan" }).getByRole("button", { name: label, exact: true }).click();
  else {
    await p.getByRole("button", { name: "Tambah", exact: true }).click();
    await p.locator("dialog[open]").getByRole("button", { name: label, exact: true }).click();
  }
}
const overlaps = (a, b) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;

try {
  {
    const W = 1280;
    const board = await makeBoard("1280");
    const cs = await browser.newContext({ storageState: staf.state, viewport: { width: W, height: 800 } });
    const cm = await browser.newContext({ storageState: mgr.state, viewport: { width: W, height: 800 } });
    const ps = await cs.newPage();
    const pm = await cm.newPage();
    const errors = [];
    ps.on("console", (m) => m.type() === "error" && errors.push(m.text()));
    for (const p of [ps, pm]) {
      await p.goto(BASE + `/whiteboard/${board}`);
      await saved(p);
    }

    // Lima orang sekaligus, termasuk Staf sendiri (dipakai uji profil di bawah).
    await tool(ps, W, "Tim");
    const dlg = ps.getByRole("dialog", { name: "Tambah anggota tim" });
    await dlg.getByRole("list", { name: "Anggota tim" }).waitFor({ timeout: 10000 });
    await dlg.getByLabel("Cari nama").fill(staf.user.name);
    await dlg.getByRole("checkbox").first().check();
    await dlg.getByLabel("Cari nama").fill("");
    const boxes = dlg.getByRole("checkbox");
    for (let i = 0, n = 0; n < 4 && i < (await boxes.count()); i++) {
      if (!(await boxes.nth(i).isChecked())) {
        await boxes.nth(i).check();
        n++;
      }
    }
    const lt = await layout(ps);
    expect(lt.small.length === 0, "dialog Tim", `tombol di bawah 44 px: ${lt.small.join(", ")}`);
    await dlg.getByRole("button", { name: /Tambahkan 5 orang/ }).click();
    await saved(ps);
    await ps.waitForTimeout(500);
    let list = await persons(board);
    expect(list.length === 5, "lima orang", `${list.length} kartu orang`);
    expect(list.every((a, i) => list.every((b, j) => i === j || !overlaps(a, b))), "lima orang", "kartu orang bertumpuk");
    const raw = JSON.stringify(list);
    expect(!/username|email|"role"/.test(raw), "privasi", "kartu orang membawa username, email, atau peran");
    expect(list.every((c) => Object.keys(c.content).sort().join() === "layout,size,user_id,view"), "privasi", "isi kartu menyalin data profil");

    // Satu divisi lewat Pilih satu divisi.
    await tool(ps, W, "Tim");
    await dlg.getByRole("list", { name: "Anggota tim" }).waitFor({ timeout: 10000 });
    const divChip = dlg.getByRole("group", { name: "Filter divisi" }).getByRole("button").nth(1);
    const divName = (await divChip.textContent()).trim();
    await divChip.click();
    const inDiv = await dlg.getByRole("checkbox").count();
    await dlg.getByRole("button", { name: /Pilih satu divisi/ }).click();
    expect((await dlg.getByText(`${inDiv} dipilih`).count()) === 1, "satu divisi", `Pilih satu divisi tidak memilih ${inDiv} orang`);
    await dlg.getByRole("button", { name: /Tambahkan \d+ orang/ }).click();
    await saved(ps);
    await ps.waitForTimeout(500);
    list = await persons(board);
    expect(list.length === 5 + inDiv, "satu divisi", `${list.length} kartu, harusnya ${5 + inDiv} (${divName})`);

    // Ganti tampilan banyak kartu sekaligus lewat menu klik kanan.
    await ps.getByRole("button", { name: "Pas ke layar" }).click();
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Control+Minus");
    await ps.keyboard.press("Control+a");
    const first = list[0];
    await ps.locator(`[data-card="${first.id}"]`).click({ button: "right" });
    await ps.getByRole("menuitem", { name: "Tampilan kartu orang" }).click();
    await ps.getByRole("menuitemcheckbox", { name: "Lengkap (dengan divisi)" }).click();
    await saved(ps);
    await ps.waitForTimeout(500);
    list = await persons(board);
    expect(list.every((c) => c.content.view === "lengkap"), "tampilan banyak", "tidak semua kartu orang berganti tampilan");
    expect((await ps.locator(`[data-card="${first.id}"]`).textContent()).includes(first.person.division), "tampilan banyak", "divisi tidak tampil di tampilan lengkap");

    // Kontras nama dan jabatan di kartu orang.
    const nameC = await contrastOf(ps.locator(`[data-card="${first.id}"] [data-testid=wb-person] p`).first());
    expect(nameC >= 4.5, "kontras", `nama ${nameC.toFixed(2)}:1`);

    // Jabatan diubah lewat profil: kartu Staf di layar Manajer berubah tanpa muat ulang.
    const mine = list.find((c) => c.content.user_id === staf.user.id);
    const title = `Jabatan uji ${Date.now() % 10000}`;
    const put = await sApi.call("PUT", "/me/profile", { name: staf.user.name, job_title: title });
    expect(put.status === 204, "profil", `ubah profil ${put.status}`);
    await pm.waitForFunction(([id, t]) => document.querySelector(`[data-card="${id}"]`)?.textContent?.includes(t), [mine?.id, title], { timeout: 10000 }).catch(() => {});
    expect(await pm.evaluate(([id, t]) => !!document.querySelector(`[data-card="${id}"]`)?.textContent?.includes(t), [mine?.id, title]), "realtime profil", "jabatan baru tidak tampil di layar Manajer");
    expect(!JSON.stringify((await persons(board)).find((c) => c.id === mine.id).content).includes(title), "realtime profil", "jabatan tersalin ke isi kartu");

    // Dua kartu orang disambung dengan garis siku.
    await ps.locator(".wb-canvas").focus();
    await ps.keyboard.press("Escape");
    await ps.locator(`[data-card="${list[0].id}"]`).click();
    await ps.locator(`[data-card="${list[1].id}"]`).click({ modifiers: ["Shift"] });
    await ps.keyboard.press("Alt+Shift+KeyK");
    await saved(ps);
    const conn = (await boardOf(board)).connectors.find((c) => c.from === list[0].id && c.to === list[1].id);
    expect(conn?.geometry?.route === "orthogonal", "garis siku", `garis ${JSON.stringify(conn?.geometry)}`);
    await ps.waitForTimeout(300);
    expect((await ps.locator("[data-testid=wb-connectors] path, svg path").count()) > 0, "garis siku", "garis tidak tergambar");

    const l = await layout(ps);
    expect(!l.overflow, `${W}`, "gulir horizontal");
    expect(errors.length === 0, `${W}`, `console: ${errors.join(" | ").slice(0, 300)}`);
    await cs.close();
    await cm.close();
  }

  {
    const board = await makeBoard("360");
    const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: 360, height: 740 }, hasTouch: true });
    const p = await ctx.newPage();
    await p.goto(BASE + `/whiteboard/${board}`);
    await saved(p);
    await tool(p, 360, "Tim");
    const dlg = p.getByRole("dialog", { name: "Tambah anggota tim" });
    await dlg.getByRole("list", { name: "Anggota tim" }).waitFor({ timeout: 10000 });
    // Animasi buka dialog (modal-in, skala 0,98 selama 200 ms) selesai dulu sebelum ukuran tombol diukur.
    await p.waitForTimeout(300);
    const l = await layout(p);
    expect(!l.overflow, "360 dialog", "gulir horizontal di dialog Tim");
    expect(l.small.length === 0, "360 dialog", `tombol di bawah 44 px: ${l.small.join(", ")}`);
    const rowH = await dlg.locator("li label").first().evaluate((e) => e.getBoundingClientRect().height);
    expect(rowH >= 44, "360 dialog", `baris pilih ${rowH} px`);
    await dlg.getByRole("checkbox").nth(0).check();
    await dlg.getByRole("checkbox").nth(1).check();
    await dlg.getByRole("button", { name: /Tambahkan 2 orang/ }).click();
    await saved(p);
    await p.waitForTimeout(500);
    expect((await persons(board)).length === 2, "360", "kartu orang tidak bertambah");
    expect((await p.getByTestId("wb-person").count()) === 2, "360", "kartu orang tidak tampil");
    await ctx.close();
  }
} finally {
  await sApi.call("PUT", "/me/profile", { name: staf.user.name, job_title: oldTitle });
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

for (const f of fails) console.log("GAGAL", f);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
