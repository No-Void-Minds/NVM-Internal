// Uji kontras kursor dan bingkai kolaborasi Whiteboard, untuk kedelapan indeks warna di tema terang dan gelap, di 360
// dan 1280 px. Dua konteks: Manajer menggerakkan kursor dan memilih kartu, Staf melihat kursor, label nama, bingkai,
// dan label "sedang memilih" di kartu itu. Warna index lain diterapkan ke elemen asli itu (data-collab dan
// --wb-remote seperti yang dipasang aplikasi untuk orang dengan indeks itu), lalu warna terhitung dibaca dengan
// getComputedStyle, termasuk opacity efektif semua induk, dan kontrasnya dihitung di sini:
//   - teks label minimal 4,5:1 terhadap latar labelnya, latar label tanpa alpha;
//   - panah kursor dan bingkai kartu minimal 3:1 terhadap kanvas.
// Juga: label kursor di dekat tepi kanan tidak terpotong kanvas.
//
// Cara menjalankan (dari scripts/e2e; sesi dari scripts/e2e/.auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5174 node uji-kursor.mjs
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
const mgr = await session(browser, BASE, "manajer");
const sApi = await api(browser, BASE, staf.state);
const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E kursor ${Date.now()}`, x: 9700, y: 9700, w: 200, h: 120 })).body.card;
const board = holder.ref.id;
const note = (await sApi.call("POST", `/whiteboard/boards/${board}/cards`, {
  kind: "note", x: 0, y: 0, w: 240, h: 120,
  content: { doc: { type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Kartu uji kursor" }] }] } },
})).body.card;

// Diukur di halaman: warna terhitung, alpha, dan opacity efektif (hasil kali opacity elemen dan semua induknya).
const measure = () => {
  const rgba = (s) => {
    const m = s.match(/rgba?\(([^)]+)\)/);
    if (!m) return null;
    const p = m[1].split(/[ ,/]+/).filter(Boolean).map(Number);
    return { r: p[0], g: p[1], b: p[2], a: p[3] ?? 1 };
  };
  const opacity = (el) => {
    let o = 1;
    for (let e = el; e; e = e.parentElement) o *= Number(getComputedStyle(e).opacity);
    return o;
  };
  const canvas = rgba(getComputedStyle(document.querySelector(".wb-canvas")).backgroundColor);
  const cursor = document.querySelector("[data-testid=wb-cursor]");
  const label = cursor?.querySelector(".wb-remote-label");
  const arrow = cursor?.querySelector(".wb-cursor-arrow");
  const card = document.querySelector(".wb-card-remote, .wb-card-remote-strong");
  const cardLabel = card?.querySelector(".wb-remote-label");
  const shadow = card ? getComputedStyle(card).boxShadow : "none";
  const canvasBox = document.querySelector(".wb-canvas").getBoundingClientRect();
  const lb = label?.getBoundingClientRect();
  return {
    canvas,
    label: label && { bg: rgba(getComputedStyle(label).backgroundColor), fg: rgba(getComputedStyle(label).color), op: opacity(label) },
    arrow: arrow && { fill: rgba(getComputedStyle(arrow).fill), op: opacity(arrow) },
    frame: card ? { color: rgba(shadow), op: opacity(card), raw: shadow } : null,
    cardLabel: cardLabel && { bg: rgba(getComputedStyle(cardLabel).backgroundColor), fg: rgba(getComputedStyle(cardLabel).color), op: opacity(cardLabel) },
    clipped: lb ? lb.right > canvasBox.right + 0.5 || lb.left < canvasBox.left - 0.5 || lb.bottom > canvasBox.bottom + 0.5 : null,
  };
};

const lin = (c) => {
  const f = (v) => {
    v /= 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * f(c.r) + 0.7152 * f(c.g) + 0.0722 * f(c.b);
};
const ratio = (a, b) => {
  const [x, y] = [lin(a), lin(b)];
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05);
};
// Campur warna dengan alpha (dan opacity efektif) ke latar di bawahnya.
const over = (top, under, extra = 1) => {
  const a = (top?.a ?? 0) * extra;
  return { r: top.r * a + under.r * (1 - a), g: top.g * a + under.g * (1 - a), b: top.b * a + under.b * (1 - a), a: 1 };
};

try {
  for (const width of [360, 1280]) {
    const cs = await browser.newContext({ storageState: staf.state, viewport: { width, height: 800 } });
    const cm = await browser.newContext({ storageState: mgr.state, viewport: { width, height: 800 } });
    const ps = await cs.newPage();
    const pm = await cm.newPage();
    const errors = [];
    ps.on("console", (m) => m.type() === "error" && errors.push(m.text()));
    for (const p of [ps, pm]) {
      await p.goto(BASE + `/whiteboard/${board}`);
      await p.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});
    }
    // Manajer memilih kartu (bingkai dan label di kartu itu) lalu menggerakkan kursor ke tengah kanvas.
    await pm.locator(`[data-card="${note.id}"]`).click();
    const box = await pm.locator(".wb-canvas").boundingBox();
    await pm.mouse.move(box.x + box.width / 2, box.y + box.height * 0.75, { steps: 4 });
    const seen = await ps.locator("[data-testid=wb-cursor]").waitFor({ timeout: 10000 }).then(() => true, () => false);
    expect(seen, `${width}`, "kursor Manajer tidak tampil di layar Staf");
    await ps.locator(".wb-card-remote, .wb-card-remote-strong").waitFor({ timeout: 10000 }).catch(() => {});
    if (!seen) continue;

    for (const theme of ["terang", "gelap"]) {
      await ps.evaluate((t) => {
        if (t === "gelap") document.documentElement.setAttribute("data-theme", "gelap");
        else document.documentElement.removeAttribute("data-theme");
      }, theme);
      for (let n = 1; n <= 8; n++) {
        const where = `${width} ${theme} warna ${n}`;
        await ps.evaluate((n) => {
          for (const el of document.querySelectorAll("[data-testid=wb-cursor], .wb-card-remote, .wb-card-remote-strong")) {
            el.dataset.collab = String(n);
            el.style.setProperty("--wb-remote", `var(--color-collab-${n})`);
          }
        }, n);
        const m = await ps.evaluate(measure);
        const canvas = m.canvas;
        for (const [name, l] of [["label kursor", m.label], ["label kartu", m.cardLabel]]) {
          if (!l) {
            expect(false, where, `${name} tidak ada`);
            continue;
          }
          expect(l.bg && l.bg.a === 1 && l.op === 1, where, `${name} tidak solid: latar ${JSON.stringify(l.bg)}, opacity ${l.op}`);
          const bg = over(l.bg ?? { r: 0, g: 0, b: 0, a: 0 }, canvas, l.op);
          const fg = over(l.fg, bg, l.op);
          const r = ratio(fg, bg);
          expect(r >= 4.5, where, `${name} ${r.toFixed(2)}:1`);
        }
        if (m.arrow?.fill) {
          const r = ratio(over(m.arrow.fill, canvas, m.arrow.op), canvas);
          expect(r >= 3, where, `panah kursor ${r.toFixed(2)}:1 terhadap kanvas`);
        } else expect(false, where, "panah kursor tanpa isian");
        if (m.frame?.color) {
          const r = ratio(over(m.frame.color, canvas, m.frame.op), canvas);
          expect(r >= 3, where, `bingkai kartu ${r.toFixed(2)}:1 terhadap kanvas`);
        } else expect(false, where, `bingkai kartu tidak terbaca: ${m.frame?.raw ?? "tidak ada kartu dengan bingkai orang lain"}`);
      }
    }
    await ps.evaluate(() => document.documentElement.removeAttribute("data-theme"));

    // Tepi kanan: kursor Manajer di 6 px dari tepi kanan kanvas (kedua layar memakai papan dan ukuran yang sama).
    await pm.mouse.move(box.x + box.width - 6, box.y + box.height * 0.5, { steps: 4 });
    await ps.waitForTimeout(600);
    const edge = await ps.evaluate(measure);
    expect(edge.clipped === false, `${width} tepi`, "label kursor terpotong di tepi kanvas");
    await pm.mouse.move(box.x + box.width * 0.5, box.y + box.height - 4, { steps: 4 });
    await ps.waitForTimeout(600);
    const bottom = await ps.evaluate(measure);
    expect(bottom.clipped === false, `${width} tepi bawah`, "label kursor terpotong di tepi bawah kanvas");
    expect(errors.length === 0, `${width}`, `console: ${errors.join(" | ").slice(0, 200)}`);
    await cs.close();
    await cm.close();
  }
} finally {
  await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: [holder.id] });
  await sApi.ctx.close();
  await browser.close();
}

for (const f of fails.slice(0, 40)) console.log("GAGAL", f);
if (fails.length > 40) console.log(`... dan ${fails.length - 40} lagi`);
console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
