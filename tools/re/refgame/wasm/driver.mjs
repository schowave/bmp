// Driver for the original game in the browser: keeps a headless Chromium
// with the WASM image (js-dos) open and runs commands POSTed as a JSON array
// to http://localhost:$BMP_DRIVER_PORT (default 9333), see docs/re/spec_ui.md. Mouse positions
// are game coordinates (320x240); the game is reached through the js-dos
// command interface, which serve.sh exposes as window.ci.
import { chromium } from 'playwright';
import http from 'node:http';
import fs from 'node:fs';

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1000, height: 800 } });
page.on('console', m => { if (m.type() === 'error') console.log('console:', m.text()); });
await page.goto(process.env.BMP_URL ?? 'http://localhost:8000/');
await page.waitForFunction(() => typeof window.ci?.resume === 'function', null, { timeout: 120000 });
await page.evaluate(() => ci.resume());
// Screenshots while DOSBox still boots and switches the video mode can crash
// the js-dos backend ("memory access out of bounds"): wait until the game
// runs and has drawn its first picture.
await page.waitForFunction(async () => (await ci.getRunningProgram()) === 'BMMAIN', null, { timeout: 120000, polling: 1000 });
await new Promise(r => setTimeout(r, 6000));
console.log('ci ready');

const sleep = ms => new Promise(r => setTimeout(r, ms));

// Screens recognised by a few pixels of the UI that do not depend on the
// game data (measured on remake/testdata/original): x, y, r, g, b.
const STATES = {
    title: [[20, 20, 243, 243, 243], [100, 140, 178, 211, 243]],
    setup: [[40, 229, 0, 0, 0], [300, 230, 0, 0, 0], [100, 140, 65, 65, 97]],
    options: [[40, 229, 0, 0, 0], [100, 140, 0, 0, 113], [160, 120, 0, 0, 113]],
    question: [[100, 140, 97, 0, 16], [160, 120, 97, 0, 16], [40, 229, 97, 97, 130]],
    menu: [[150, 10, 81, 97, 16], [40, 229, 162, 162, 195], [300, 230, 97, 97, 130]],
};

// state returns the name of the screen shown, or "unknown".
async function state() {
    const px = await page.evaluate(async pts => {
        const d = await ci.screenshot();
        return pts.map(([x, y]) => Array.from(d.data.slice((y * d.width + x) * 4, (y * d.width + x) * 4 + 3)));
    }, Object.values(STATES).flat());
    let i = 0;
    for (const [name, pts] of Object.entries(STATES)) {
        const ok = pts.every(([, , r, g, b], k) => px[i + k].join() === [r, g, b].join());
        i += pts.length;
        if (ok) return name;
    }
    return 'unknown';
}

// waitFor polls until the screen is one of names; it throws after ms.
async function waitFor(names, ms = 30000) {
    const until = Date.now() + ms;
    for (;;) {
        const s = await state();
        if (names.includes(s)) return s;
        if (Date.now() > until) throw new Error(`waiting for ${names.join('/')}, the screen is ${s}`);
        await sleep(500);
    }
}

async function click(x, y, btn = 0) {
    await page.evaluate(([x, y]) => { const w = ci.width(), h = ci.height(); ci.sendMouseMotion((x + 0.5) / w, (y + 0.5) / h); }, [x, y]);
    await sleep(120);
    await page.evaluate(b => ci.sendMouseButton(b, true), btn);
    await sleep(120);
    await page.evaluate(b => ci.sendMouseButton(b, false), btn);
    await sleep(150);
}

// newGame goes from the title or the setup to the main menu of a new game
// with one manager and the club in the middle of the strip, declining the
// cup draws.
async function newGame(name) {
    if (await waitFor(['title', 'setup'], 120000) === 'title') {
        await click(160, 120);
        await waitFor(['setup']);
    }
    await click(40, 229);
    await page.keyboard.type(name, { delay: 80 });
    await page.keyboard.press('Enter');
    await sleep(300);
    await click(40, 135);
    await click(160, 160, 1);
    await waitFor(['options']);
    await click(160, 160, 1);
    // The game now builds the season; screenshots meanwhile crash the
    // backend as during the boot.
    await sleep(10000);
    while (await waitFor(['question', 'menu']) === 'question') {
        await click(190, 168); // KEIN GEDANKE
        await sleep(1500);
    }
    return 'menu';
}

async function run(cmds) {
    const out = [];
    for (const c of cmds) {
        const [op, ...a] = c;
        if (op === 'move') {
            await page.evaluate(([x, y]) => { const w = ci.width(), h = ci.height(); ci.sendMouseMotion((x + 0.5) / w, (y + 0.5) / h); }, a);
        } else if (op === 'click' || op === 'rclick') {
            const btn = op === 'click' ? 0 : 1;
            await page.evaluate(([x, y]) => { const w = ci.width(), h = ci.height(); ci.sendMouseMotion((x + 0.5) / w, (y + 0.5) / h); }, a);
            await sleep(120);
            await page.evaluate(b => ci.sendMouseButton(b, true), btn);
            await sleep(120);
            await page.evaluate(b => ci.sendMouseButton(b, false), btn);
        } else if (op === 'key') {
            await page.evaluate(k => { ci.sendKeyEvent(k, true); }, a[0]);
            await sleep(80);
            await page.evaluate(k => { ci.sendKeyEvent(k, false); }, a[0]);
        } else if (op === 'type') {
            await page.focus('canvas').catch(() => {});
            await page.keyboard.type(a[0], { delay: 80 });
        } else if (op === 'press') {
            await page.keyboard.press(a[0]);
        } else if (op === 'wait') {
            await sleep(a[0]);
        } else if (op === 'shot') {
            const img = await page.evaluate(async () => {
                const d = await ci.screenshot();
                const t = document.createElement('canvas'); t.width = d.width; t.height = d.height;
                t.getContext('2d').putImageData(d, 0, 0);
                const c = document.createElement('canvas'); c.width = d.width * 2; c.height = d.height * 2;
                const x = c.getContext('2d'); x.imageSmoothingEnabled = false; x.drawImage(t, 0, 0, c.width, c.height);
                return { w: d.width, h: d.height, url: c.toDataURL('image/png') };
            });
            fs.writeFileSync(a[0], Buffer.from(img.url.split(',')[1], 'base64'));
            out.push(`${a[0]} ${img.w}x${img.h}`);
        } else if (op === 'state') {
            out.push(await state());
        } else if (op === 'waitfor') {
            out.push(await waitFor(a[0].split('/'), a[1]));
        } else if (op === 'newgame') {
            out.push(await newGame(a[0] ?? 'TEST'));
        } else if (op === 'quit') {
            out.push('bye');
            setTimeout(async () => { await browser.close(); process.exit(0); }, 1000);
        } else if (op === 'size') {
            out.push(await page.evaluate(() => `${ci.width()}x${ci.height()}`));
        } else if (op === 'eval') {
            out.push(String(await page.evaluate(a[0])));
        }
        await sleep(op === 'wait' ? 0 : 150);
    }
    return out;
}

http.createServer((req, res) => {
    let body = '';
    req.on('data', d => body += d);
    req.on('end', async () => {
        try {
            const r = await run(JSON.parse(body));
            res.end(r.join('\n') + '\n');
        } catch (e) {
            res.statusCode = 500;
            res.end(String(e.stack) + '\n');
        }
    });
}).listen(Number(process.env.BMP_DRIVER_PORT ?? 9333), () => console.log('listening', process.env.BMP_DRIVER_PORT ?? 9333));
