// Uji performa revisi 54 (renderer bentuk berbasis data): papan 500 bentuk campuran dari kelompok baru (BPMN, wadah,
// activity dan state, sequence, ERD dan class, flowchart lanjutan, arsitektur, wireframe, mind map, lainnya) dengan
// varian, lane, dan ikon, plus 100 garis (pesan sequence, crow's foot, melengkung, siku). Diukur di 1280 px: waktu muat
// sampai kartu tampil, p95 durasi frame selama geser kanvas 3 detik, dan p95 selama menyeret satu bentuk 2 detik.
// Batas frame 20 ms hanya berlaku untuk build produksi (vite build lalu vite preview), seperti wbperforma di uji13.
//
// Cara menjalankan (dari scripts/e2e; sesi dari .auth atau variabel DEV_* seperti uji13.mjs):
//   E2E_BASE=http://localhost:5175 node uji-perf-bentuk.mjs
import { chromium } from "playwright-core";
import { api, realLogins, session } from "./sesi.mjs";

const BASE = process.env.E2E_BASE || "http://localhost:5173";
const browser = await chromium.launch({ channel: "msedge", headless: process.env.E2E_HEADED !== "1" });
let checks = 0;
const fails = [];
const expect = (ok, where, msg) => {
  checks++;
  if (!ok) fails.push(`${where}: ${msg}`);
};

const staf = await session(browser, BASE, "staf");
const sApi = await api(browser, BASE, staf.state);
const holders = [];
const shape = (s, text, extra = {}) => ({ shape: s, rotation: 0, fill: "putih", stroke: "tinta", stroke_width: 2, stroke_style: "solid", text, align: "center", font_size: 16, ...extra });
// Bentuk kelompok baru beserta ukuran dan isi tambahan (varian, lane, bagian kelas).
const KINDS = [
  ["bpmn_start", 56, 56, { variant: "message" }], ["bpmn_intermediate", 56, 56, { variant: "timer" }], ["bpmn_end", 56, 56, { variant: "error" }],
  ["bpmn_task", 160, 90, { variant: "service" }], ["bpmn_gateway", 64, 64, { variant: "parallel" }], ["bpmn_data_store", 72, 64, {}],
  ["bpmn_pool", 400, 200, { fill: "", lanes: [{ label: "A", size: 100 }, { label: "B", size: 100 }] }],
  ["uml_swimlane", 300, 200, { fill: "", lanes: [{ label: "A", size: 100 }, { label: "B", size: 100 }] }],
  ["uml_initial", 40, 40, {}], ["uml_fork", 160, 40, {}], ["uml_action", 160, 70, {}], ["uml_state_composite", 300, 200, { fill: "" }],
  ["uml_lifeline", 140, 300, { variant: "destroyed" }], ["uml_activation", 40, 120, {}], ["uml_fragment", 300, 200, { variant: "alt", fill: "", lanes: [{ label: "ya", size: 100 }, { label: "tidak", size: 100 }] }],
  ["erd_entity", 160, 70, {}], ["erd_weak_entity", 160, 70, {}], ["erd_key_attribute", 130, 60, {}],
  ["uml_class", 220, 150, { variant: "interface", name: "Antarmuka", attributes: "", methods: "+ jalan()" }],
  ["fc_predefined", 180, 80, {}], ["fc_delay", 160, 80, {}], ["arch_server", 70, 90, {}], ["arch_cloud", 150, 100, {}], ["arch_node", 160, 110, {}],
  ["wf_browser", 360, 240, {}], ["wf_checkbox", 180, 40, { variant: "checked" }], ["wf_dropdown", 200, 44, {}],
  ["mm_central", 200, 90, { fill: "kuning" }], ["mm_topic", 160, 60, {}], ["callout", 200, 120, {}], ["funnel", 160, 140, {}], ["timeline", 420, 80, { fill: "" }],
];

try {
  const holder = (await sApi.call("POST", "/whiteboard/boards/root/cards", { kind: "board", title: `E2E perf bentuk ${Date.now()}`, x: 9900, y: 12000, w: 200, h: 120 })).body.card;
  holders.push(holder);
  const board = holder.ref.id;
  const made = [];
  const jobs = [];
  for (let i = 0; i < 500; i++) {
    const [s, w, h, extra] = KINDS[i % KINDS.length];
    const x = (i % 20) * 460;
    const y = Math.floor(i / 20) * 340;
    jobs.push(() => sApi.call("POST", `/whiteboard/boards/${board}/cards`, { kind: "shape", x, y, w, h, content: shape(s, `B${i}`, extra) }).then((r) => r.body?.card && made.push(r.body.card)));
  }
  for (let i = 0; i < jobs.length; i += 8) await Promise.all(jobs.slice(i, i + 8).map((j) => j()));
  made.sort((a, b) => a.y - b.y || a.x - b.x);
  const byShape = (s) => made.filter((c) => c.content.shape === s);
  const lifes = byShape("uml_lifeline");
  const ents = byShape("erd_entity");
  const minds = byShape("mm_topic");
  let n = 0;
  const link = async (body) => {
    const r = await sApi.call("POST", `/whiteboard/boards/${board}/connectors`, body);
    if (r.status === 201) n++;
  };
  for (let i = 0; i + 1 < lifes.length && n < 25; i++) await link({ from: lifes[i].id, to: lifes[i + 1].id, geometry: { from: { port: "auto" }, to: { port: "auto" }, route: "message", points: [[0, 120]] }, style: { start: "none", end: "arrow", line: "solid", color: "" } });
  for (let i = 0; i + 1 < ents.length && n < 50; i++) await link({ from: ents[i].id, to: ents[i + 1].id, style: { start: "cf_one", end: "cf_many", line: "solid", color: "" } });
  for (let i = 0; i + 1 < minds.length && n < 70; i++) await link({ from: minds[i].id, to: minds[i + 1].id, geometry: { from: { port: "auto" }, to: { port: "auto" }, route: "curved", points: [] }, style: { start: "none", end: "none", line: "solid", color: "" } });
  for (let i = 0; i + 1 < made.length && n < 100; i += 3) await link({ from: made[i].id, to: made[i + 1].id, geometry: { from: { port: "auto" }, to: { port: "auto" }, route: "orthogonal", points: [] } });
  const view = (await sApi.call("GET", `/whiteboard/boards/${board}`)).body.board;
  expect(view.cards.length >= 500 && view.connectors.length >= 100, "isi", `bentuk ${view.cards.length}, garis ${view.connectors.length}`);

  const ctx = await browser.newContext({ storageState: staf.state, viewport: { width: 1280, height: 800 } });
  const ps = await ctx.newPage();
  const errors = [];
  ps.on("pageerror", (e) => errors.push(String(e)));
  await ps.goto(BASE + "/whiteboard");
  await ps.waitForLoadState("networkidle");
  const t0 = Date.now();
  await ps.goto(BASE + `/whiteboard/${board}`);
  await ps.waitForFunction(() => document.querySelectorAll("[data-card]").length > 20, null, { timeout: 15000 });
  const loadMs = Date.now() - t0;
  expect(loadMs < 2000, "muat", `papan 500 bentuk dimuat ${loadMs} ms`);
  await ps.waitForFunction(() => document.querySelector("[data-testid=wb-save]")?.textContent?.trim() === "Tersimpan", null, { timeout: 15000 }).catch(() => {});

  const record = () =>
    ps.evaluate(() => {
      window.__frames = [];
      let last = performance.now();
      const tick = (t) => {
        window.__frames.push(t - last);
        last = t;
        if (window.__frames.length < 1000) window.__raf = requestAnimationFrame(tick);
      };
      window.__raf = requestAnimationFrame(tick);
    });
  const stop = async () => {
    const f = await ps.evaluate(() => {
      cancelAnimationFrame(window.__raf);
      return window.__frames.slice(2);
    });
    f.sort((a, b) => a - b);
    return { p95: f[Math.floor(f.length * 0.95)] ?? 999, n: f.length };
  };
  const dev = await ps.evaluate(() => !!document.querySelector('script[src*="/@vite/client"]'));

  // Seret satu bentuk 2 detik dari titik yang kartunya sendiri paling atas.
  // Bentuk pertama yang terlihat dan punya titik tekan yang kartunya sendiri paling atas (bukan wadah, supaya seret
  // tidak membawa isi wadah).
  const cand = made.filter((c) => !["bpmn_pool", "uml_swimlane", "uml_state_composite", "uml_fragment", "uml_lifeline"].includes(c.content.shape)).map((c) => c.id);
  const grip = await ps.evaluate(
    ([ids]) => {
      // Kartu pertama yang terlihat di layar dan punya titik tekan yang kartunya sendiri paling atas.
      for (const id of ids) {
        const el0 = document.querySelector(`[data-card="${id}"]`);
        if (!el0) continue;
        const b = el0.getBoundingClientRect();
        if (b.left < 0 || b.top < 60 || b.right > innerWidth || b.bottom > innerHeight) continue;
        for (const fy of [0.5, 0.3, 0.7, 0.15, 0.85])
          for (const fx of [0.5, 0.25, 0.75, 0.1, 0.9]) {
            const x = b.left + b.width * fx;
            const y = b.top + b.height * fy;
            const el = document.elementFromPoint(x, y);
            if (el?.closest("[data-card]")?.getAttribute("data-card") === id && !el.closest("[data-connector], [data-handle]")) return { id, x, y };
          }
      }
      return null;
    },
    [cand],
  );
  expect(!!grip, "seret", "tidak ada bentuk yang terlihat dan tidak tertutup");
  if (!grip) throw new Error("tidak ada titik seret");
  await record();
  await ps.mouse.move(grip.x, grip.y);
  await ps.mouse.down();
  const ds = Date.now();
  let k = 0;
  while (Date.now() - ds < 2000) {
    k++;
    await ps.mouse.move(grip.x + (k % 60) * 3, grip.y + (k % 40) * 2);
  }
  const drag = await stop();
  await ps.mouse.up();

  // Geser kanvas 3 detik (Spasi ditahan).
  const canvas = ps.locator(".wb-canvas");
  const cb = await canvas.boundingBox();
  await record();
  await canvas.focus();
  await ps.keyboard.down("Space");
  await ps.mouse.move(cb.x + cb.width / 2, cb.y + cb.height / 2);
  await ps.mouse.down();
  const ps0 = Date.now();
  k = 0;
  while (Date.now() - ps0 < 3000) {
    k++;
    await ps.mouse.move(cb.x + cb.width / 2 - (k % 120) * 4, cb.y + cb.height / 2 - (k % 80) * 3);
  }
  await ps.mouse.up();
  await ps.keyboard.up("Space");
  const pan = await stop();
  if (!dev) {
    expect(pan.p95 < 20, "frame geser", `p95 ${pan.p95.toFixed(1)} ms dari ${pan.n} frame`);
    expect(drag.p95 < 20, "frame seret", `p95 ${drag.p95.toFixed(1)} ms dari ${drag.n} frame`);
  }
  console.log(`catatan: perf bentuk${dev ? " (Vite dev, batas tidak dinilai)" : " (build produksi)"} ${view.cards.length} bentuk, ${view.connectors.length} garis, muat ${loadMs} ms, geser p95 ${pan.p95.toFixed(1)} ms (${pan.n} frame), seret p95 ${drag.p95.toFixed(1)} ms (${drag.n} frame)`);
  expect(errors.length === 0, "galat halaman", errors.join(" | "));
  await ctx.close();
} finally {
  for (const f of fails) console.log("GAGAL", f);
  if (holders.length) await sApi.call("POST", "/whiteboard/boards/root/cards/delete", { ids: holders.map((h) => h.id) });
  await sApi.ctx.close();
  await browser.close();
}

console.log(`pemeriksaan: ${checks}, gagal: ${fails.length}, login sungguhan: ${realLogins}`);
process.exit(fails.length ? 1 : 0);
