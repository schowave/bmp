// Driver for the original game in the browser: keeps a headless Chromium
// with the WASM image (js-dos) open and runs commands POSTed as a JSON array
// to http://localhost:$PORT (default 9333), see README.md. Mouse positions
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
await page.evaluate(() => ci.resume()); console.log('ci ready');

const sleep = ms => new Promise(r => setTimeout(r, ms));

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
}).listen(Number(process.env.PORT ?? 9333), () => console.log('listening', process.env.PORT ?? 9333));
