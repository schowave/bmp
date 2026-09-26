// Opens the game page and saves a screenshot, optionally after pressing keys.
//   node shot.mjs <out.png> [key ...]
import { chromium } from 'playwright';

const url = process.env.BMP_URL ?? 'http://localhost:18080/';
const [out = 'shot.png', ...keys] = process.argv.slice(2);

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 640, height: 480 } });
await page.goto(url);
await page.waitForSelector('#screen canvas');
await page.waitForTimeout(3000);
for (const k of keys) {
    if (k.startsWith('click:')) {
        const [x, y] = k.slice(6).split(',').map(Number);
        await page.mouse.click(x, y);
    } else if (k.startsWith('wait:')) {
        await page.waitForTimeout(Number(k.slice(5)));
    } else {
        await page.keyboard.press(k);
    }
    await page.waitForTimeout(700);
}
await page.screenshot({ path: out });
await browser.close();
