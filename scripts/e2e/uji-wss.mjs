// Uji wss:// Whiteboard di Chrome, Edge, dan Firefox (W2, docs/whiteboard.md bagian 3), tanpa screenshot.
//
// Server dev memakai http dan ws biasa, jadi skrip ini menjalankan proxy HTTPS lokal (sertifikat self-signed dari
// openssl, berlaku satu hari) di https://localhost:8443 yang:
//   - menyajikan build produksi frontend (frontend/dist) dengan header keamanan yang sama persis dengan
//     frontend/nginx-security-headers.conf, termasuk CSP connect-src 'self';
//   - meneruskan /api dan WebSocket /api/whiteboard/ws ke backend dev di 127.0.0.1:8080 dengan Host asli.
// Lolos berarti: halaman Whiteboard tersambung lewat wss:// (penanda "Tersimpan"), WebSocket buatan sendiri di
// halaman itu menerima "joined", dan tidak ada galat console (termasuk pelanggaran CSP).
//
// Cara menjalankan (backend dev di :8080 sudah jalan, openssl ada di PATH, frontend sudah di-build):
//   cd frontend && npm run build && cd ../scripts/e2e
//   DEV_STAFF_USERNAME=... DEV_STAFF_PASSWORD=... node uji-wss.mjs
// Chrome dan Edge lewat Playwright (channel), Firefox lewat WebDriver BiDi bawaan Firefox (--remote-debugging-port),
// karena Playwright tidak bisa mengendalikan Firefox biasa. Safari tidak tersedia di Windows.
import { execFileSync, spawn } from "node:child_process";
import fs from "node:fs";
import http from "node:http";
import https from "node:https";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { chromium } from "playwright-core";

const ROOT = path.resolve(import.meta.dirname, "../..");
const DIST = path.join(ROOT, "frontend/dist");
const PORT = 8443;
// Backend tujuan; bawaan backend dev di 8080. WSS_BACKEND_PORT dipakai bila backend kode terbaru jalan di port lain.
const BACKEND_PORT = Number(process.env.WSS_BACKEND_PORT || 8080);
const ORIGIN = `https://localhost:${PORT}`;
const USER = process.env.DEV_STAFF_USERNAME;
const PASS = process.env.DEV_STAFF_PASSWORD;
if (!USER || !PASS) {
  console.error("DEV_STAFF_USERNAME dan DEV_STAFF_PASSWORD wajib diisi.");
  process.exit(2);
}
if (!fs.existsSync(path.join(DIST, "index.html"))) {
  console.error("frontend/dist belum ada. Jalankan npm run build di folder frontend dulu.");
  process.exit(2);
}

// Header keamanan dibaca dari berkas nginx yang sama dengan produksi, supaya CSP yang diuji tidak menyimpang.
const conf = fs.readFileSync(path.join(ROOT, "frontend/nginx-security-headers.conf"), "utf8");
const secHeaders = {};
for (const m of conf.matchAll(/add_header\s+(\S+)\s+"([^"]*)"\s+always;/g)) secHeaders[m[1].toLowerCase()] = m[2];
if (!secHeaders["content-security-policy"]) {
  console.error("CSP tidak terbaca dari nginx-security-headers.conf.");
  process.exit(2);
}

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "nvm-wss-"));
const key = path.join(tmp, "key.pem");
const cert = path.join(tmp, "cert.pem");
execFileSync("openssl", ["req", "-x509", "-newkey", "rsa:2048", "-nodes", "-keyout", key, "-out", cert, "-days", "1",
  "-subj", "/CN=localhost", "-addext", "subjectAltName=DNS:localhost"], { stdio: "ignore" });

const types = { ".html": "text/html", ".js": "application/javascript", ".css": "text/css", ".svg": "image/svg+xml",
  ".woff2": "font/woff2", ".png": "image/png", ".ico": "image/x-icon", ".json": "application/json", ".txt": "text/plain" };

const server = https.createServer({ key: fs.readFileSync(key), cert: fs.readFileSync(cert) }, (req, res) => {
  if (req.url.startsWith("/api/")) {
    const up = http.request({ host: "127.0.0.1", port: BACKEND_PORT, method: req.method, path: req.url, headers: req.headers }, (ur) => {
      res.writeHead(ur.statusCode, { ...ur.headers, ...secHeaders });
      ur.pipe(res);
    });
    up.on("error", () => {
      res.writeHead(502);
      res.end();
    });
    req.pipe(up);
    return;
  }
  // SPA: berkas yang ada disajikan, sisanya index.html (sama dengan try_files di nginx.conf).
  const clean = path.normalize(decodeURIComponent(req.url.split("?")[0])).replace(/^([/\\])+/, "");
  let file = path.join(DIST, clean);
  if (!file.startsWith(DIST) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) file = path.join(DIST, "index.html");
  res.writeHead(200, { "content-type": types[path.extname(file)] ?? "application/octet-stream", ...secHeaders });
  fs.createReadStream(file).pipe(res);
});
server.on("upgrade", (req, socket, head) => {
  const up = net.connect(BACKEND_PORT, "127.0.0.1", () => {
    let raw = `${req.method} ${req.url} HTTP/1.1\r\n`;
    for (let i = 0; i < req.rawHeaders.length; i += 2) raw += `${req.rawHeaders[i]}: ${req.rawHeaders[i + 1]}\r\n`;
    up.write(raw + "\r\n");
    if (head?.length) up.write(head);
    up.pipe(socket);
    socket.pipe(up);
  });
  up.on("error", () => socket.destroy());
  socket.on("error", () => up.destroy());
});
await new Promise((r) => server.listen(PORT, "127.0.0.1", r));

// Pemeriksaan di dalam halaman: WebSocket buatan sendiri ke origin yang sama, di bawah CSP produksi.
const probe = `new Promise((res) => {
  const ws = new WebSocket("wss://" + location.host + "/api/whiteboard/ws");
  const t = setTimeout(() => res("timeout"), 8000);
  ws.onopen = () => ws.send(JSON.stringify({ t: "join", board: "root", tab_id: "uji-wss" }));
  ws.onmessage = (e) => { if (String(e.data).includes('"joined"')) { clearTimeout(t); ws.close(); res("joined"); } };
  ws.onerror = () => { clearTimeout(t); res("error"); };
})`;
const saved = `new Promise((res) => {
  const end = Date.now() + 15000;
  const tick = () => {
    const s = document.querySelector("[data-testid=wb-save]")?.textContent?.trim();
    if (s === "Tersimpan") return res(s);
    if (Date.now() > end) return res(s ?? "tidak ada penanda");
    setTimeout(tick, 200);
  };
  tick();
})`;
const login = `fetch("/api/auth/login", { method: "POST", headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ username: ${JSON.stringify(USER)}, password: ${JSON.stringify(PASS)} }) }).then((r) => r.status)`;

const results = [];

async function viaPlaywright(name, channel) {
  const browser = await chromium.launch({ channel, headless: true });
  try {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    const errors = [];
    page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
    page.on("pageerror", (e) => errors.push(e.message));
    const sockets = [];
    page.on("websocket", (ws) => sockets.push(ws.url()));
    await page.goto(ORIGIN + "/");
    const status = await page.evaluate(login);
    // 401 dari /api/auth/me sebelum login memang diharapkan; galat dihitung sejak halaman Whiteboard dibuka.
    errors.length = 0;
    await page.goto(ORIGIN + "/whiteboard");
    const s = await page.evaluate(saved);
    const p = await page.evaluate(probe);
    const wss = sockets.some((u) => u.startsWith(`wss://localhost:${PORT}/api/whiteboard/ws`));
    results.push({ name, ok: status === 200 && s === "Tersimpan" && p === "joined" && wss && errors.length === 0,
      detail: `login ${status}, penanda "${s}", probe ${p}, wss ${wss}, galat ${errors.length}${errors.length ? ": " + errors.join(" | ").slice(0, 200) : ""}` });
  } finally {
    await browser.close();
  }
}

async function viaFirefoxBidi() {
  const exe = ["C:/Program Files/Mozilla Firefox/firefox.exe", "C:/Program Files (x86)/Mozilla Firefox/firefox.exe"].find((f) => fs.existsSync(f));
  if (!exe) {
    results.push({ name: "Firefox", ok: false, detail: "Firefox tidak terpasang" });
    return;
  }
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), "nvm-ff-"));
  const bidiPort = 9333;
  const ff = spawn(exe, ["-headless", "-no-remote", "-profile", profile, `--remote-debugging-port=${bidiPort}`], { stdio: "ignore" });
  try {
    let ws;
    for (let i = 0; i < 60 && !ws; i++) {
      await new Promise((r) => setTimeout(r, 500));
      ws = await new Promise((res) => {
        const s = new WebSocket(`ws://127.0.0.1:${bidiPort}/session`);
        s.onopen = () => res(s);
        s.onerror = () => res(undefined);
      });
    }
    if (!ws) throw new Error("BiDi Firefox tidak bisa dihubungi");
    let id = 0;
    const waiting = new Map();
    const errors = [];
    ws.onmessage = (e) => {
      const m = JSON.parse(e.data);
      if (m.id && waiting.has(m.id)) {
        waiting.get(m.id)(m);
        waiting.delete(m.id);
      } else if (m.method === "log.entryAdded" && m.params.level === "error") errors.push(m.params.text);
    };
    const cmd = (method, params) =>
      new Promise((res, rej) => {
        const n = ++id;
        waiting.set(n, (m) => (m.type === "error" ? rej(new Error(`${method}: ${m.error} ${m.message}`)) : res(m.result)));
        ws.send(JSON.stringify({ id: n, method, params }));
      });
    await cmd("session.new", { capabilities: { alwaysMatch: { acceptInsecureCerts: true } } });
    await cmd("session.subscribe", { events: ["log.entryAdded"] });
    const tree = await cmd("browsingContext.getTree", {});
    const context = tree.contexts[0].context;
    const evaluate = async (expression) =>
      (await cmd("script.evaluate", { expression, target: { context }, awaitPromise: true })).result?.value;
    await cmd("browsingContext.navigate", { context, url: ORIGIN + "/", wait: "complete" });
    const status = await evaluate(login);
    await cmd("browsingContext.navigate", { context, url: ORIGIN + "/whiteboard", wait: "complete" });
    const s = await evaluate(saved);
    const p = await evaluate(probe);
    results.push({ name: "Firefox", ok: status === 200 && s === "Tersimpan" && p === "joined" && errors.length === 0,
      detail: `login ${status}, penanda "${s}", probe ${p}, galat ${errors.length}${errors.length ? ": " + errors.join(" | ").slice(0, 200) : ""}` });
    ws.close();
  } catch (e) {
    results.push({ name: "Firefox", ok: false, detail: String(e.message ?? e) });
  } finally {
    ff.kill();
  }
}

try {
  await viaPlaywright("Chrome", "chrome");
  await viaPlaywright("Edge", "msedge");
  await viaFirefoxBidi();
} finally {
  server.close();
}
for (const r of results) console.log(`${r.ok ? "LOLOS" : "GAGAL"} ${r.name}: ${r.detail}`);
console.log("Safari: belum teruji (tidak tersedia di Windows).");
process.exitCode = results.every((r) => r.ok) && results.length === 3 ? 0 : 1;
