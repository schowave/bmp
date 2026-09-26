// Measures one configuration of the VNC image: CPU per process with and without a
// client, WebSocket traffic, and the time from a mouse click to the changed picture.
//
//   node bench.mjs --label base [--image bmp-bench:base] [--conf dosbox.conf]
//                  [--cmd "container command"] [--secs 20] [--clicks 20]
//
// Starts its own container (bmp-bench on port 18080), appends one JSON line to
// results.jsonl and saves a screenshot as <label>.png.
import { chromium } from 'playwright';
import { execFileSync } from 'node:child_process';
import { appendFileSync, mkdirSync } from 'node:fs';
import { resolve } from 'node:path';
import { parseArgs } from 'node:util';

const { values: opt } = parseArgs({
    options: {
        label: { type: 'string', default: 'base' },
        image: { type: 'string', default: 'bmp-bench:base' },
        conf: { type: 'string' },
        cmd: { type: 'string' },
        secs: { type: 'string', default: '20' },
        clicks: { type: 'string', default: '20' },
        keep: { type: 'boolean', default: false },
    },
});
const SECS = Number(opt.secs);
const NAME = 'bmp-bench';
const PORT = 18080;
const URL = `http://localhost:${PORT}/`;
const sleep = ms => new Promise(r => setTimeout(r, ms));
const podman = (...args) => execFileSync('podman', args, { encoding: 'utf8' });

// --- container -----------------------------------------------------------------

function start() {
    try { podman('rm', '-f', NAME); } catch {}
    const args = ['run', '-d', '--name', NAME, '-p', `${PORT}:8080`];
    if (opt.conf) args.push('-v', `${resolve(opt.conf)}:/home/bmp/.dosbox/dosbox.conf:ro`);
    args.push(opt.image);
    if (opt.cmd) args.push('sh', '-c', opt.cmd);
    podman(...args);
}

// utime+stime in clock ticks (100/s) per process name, summed over all processes of
// that name, plus the container's total.
function ticks() {
    const out = podman('exec', NAME, 'sh', '-c',
        'for p in /proc/[0-9]*; do [ -r $p/stat ] && echo "$(cat $p/comm) $(cut -d" " -f14,15 $p/stat)"; done 2>/dev/null');
    const t = { total: 0 };
    for (const line of out.trim().split('\n')) {
        const [comm, u, s] = line.split(' ');
        const v = Number(u) + Number(s);
        if (Number.isNaN(v)) continue;
        t[comm] = (t[comm] ?? 0) + v;
        t.total += v;
    }
    return t;
}

// Percent of one core per process over the given interval.
async function cpuDuring(fn) {
    const a = ticks();
    const t0 = Date.now();
    await fn();
    const b = ticks();
    const secs = (Date.now() - t0) / 1000;
    const pct = {};
    for (const k of ['dosbox', 'Xtigervnc', 'websockify', 'total']) {
        pct[k] = Math.round(((b[k] ?? 0) - (a[k] ?? 0)) / secs * 10) / 10;
    }
    return pct;
}

// --- browser -------------------------------------------------------------------

// Cheap fingerprint of the team strip, which scrolls when an arrow is clicked.
const REGION_HASH = () => {
    const c = document.querySelector('#screen canvas');
    const d = c.getContext('2d').getImageData(0, 90, c.width, 110).data;
    let h = 0;
    for (let i = 0; i < d.length; i += 28) h = (h * 31 + d[i]) | 0;
    return h;
};

async function openClient(browser) {
    const page = await browser.newPage({ viewport: { width: 640, height: 480 } });
    const cdp = await page.context().newCDPSession(page);
    await cdp.send('Network.enable');
    const ws = { frames: 0, bytes: 0 };
    cdp.on('Network.webSocketFrameReceived', e => {
        ws.frames++;
        // Binary frames arrive base64 encoded.
        ws.bytes += Math.floor(e.response.payloadData.length * 3 / 4);
    });
    await page.goto(URL);
    await page.waitForSelector('#screen canvas');
    // Wait until the first picture is there, i.e. the canvas is no longer uniform.
    await page.waitForFunction(() => {
        const c = document.querySelector('#screen canvas');
        if (!c || c.width < 100) return false;
        const d = c.getContext('2d').getImageData(0, 0, c.width, c.height).data;
        for (let i = 4; i < d.length; i += 400) if (d[i] !== d[0]) return true;
        return false;
    }, null, { timeout: 60000, polling: 100 });
    return { page, ws };
}

// Clicks the right arrow of the team strip and waits for the strip to change.
async function clickLatencies(page, n) {
    const out = [];
    for (let i = 0; i < n; i++) {
        const before = await page.evaluate(REGION_HASH);
        const t0 = performance.now();
        await page.mouse.click(627, 135);
        try {
            await page.waitForFunction(h => {
                const c = document.querySelector('#screen canvas');
                const d = c.getContext('2d').getImageData(0, 90, c.width, 110).data;
                let x = 0;
                for (let i = 0; i < d.length; i += 28) x = (x * 31 + d[i]) | 0;
                return x !== h;
            }, before, { timeout: 5000, polling: 'raf' });
            out.push(Math.round(performance.now() - t0));
        } catch {
            out.push(null);
        }
        await sleep(700);
    }
    return out;
}

const stats = xs => {
    const v = xs.filter(x => x != null).sort((a, b) => a - b);
    if (!v.length) return { n: 0, missed: xs.length };
    const q = p => v[Math.min(v.length - 1, Math.floor(p * v.length))];
    return { n: v.length, missed: xs.length - v.length, p50: q(0.5), p90: q(0.9), max: v.at(-1) };
};

// --- run -----------------------------------------------------------------------

start();
await sleep(15000); // DOSBox boots and the game reaches the team selection

const result = { label: opt.label, image: opt.image, conf: opt.conf ?? null, cmd: opt.cmd ?? null, at: new Date().toISOString() };

result.cpuNoClient = await cpuDuring(() => sleep(SECS * 1000));

const browser = await chromium.launch();
const { page, ws } = await openClient(browser);
await sleep(3000);

ws.frames = ws.bytes = 0;
result.cpuClientIdle = await cpuDuring(() => sleep(SECS * 1000));
result.wsIdle = { framesPerSec: +(ws.frames / SECS).toFixed(1), kbPerSec: +(ws.bytes / 1024 / SECS).toFixed(1) };

ws.frames = ws.bytes = 0;
let lat;
const t0 = Date.now();
result.cpuClicking = await cpuDuring(async () => { lat = await clickLatencies(page, Number(opt.clicks)); });
const clickSecs = (Date.now() - t0) / 1000;
result.wsClicking = { framesPerSec: +(ws.frames / clickSecs).toFixed(1), kbPerSec: +(ws.bytes / 1024 / clickSecs).toFixed(1) };
result.clickLatencyMs = stats(lat);

mkdirSync('shots', { recursive: true });
await page.screenshot({ path: `shots/${opt.label}.png` });
await browser.close();

// After the client left: does the container go back to idle?
await sleep(3000);
result.cpuAfterDisconnect = await cpuDuring(() => sleep(Math.min(SECS, 10) * 1000));

if (!opt.keep) podman('rm', '-f', NAME);

appendFileSync('results.jsonl', JSON.stringify(result) + '\n');
console.log(JSON.stringify(result, null, 2));
