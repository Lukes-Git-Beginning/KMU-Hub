// Screenshot the rendered snapshot HTML for visual QA of the PDF content.
import { dirname, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from 'playwright';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '..', '..');
const buildDir = resolve(repoRoot, '.planning', 'pdf-build');
const htmlPath = resolve(buildDir, 'snapshot.html');

const browser = await chromium.launch();
try {
  const page = await browser.newPage({ viewport: { width: 820, height: 1160 }, deviceScaleFactor: 2 });
  await page.goto(pathToFileURL(htmlPath).href, { waitUntil: 'networkidle', timeout: 60000 });
  await page.waitForFunction(() => window.__renderDone === true, { timeout: 60000 });
  await page.evaluate(() => document.fonts && document.fonts.ready);

  const fullHeight = await page.evaluate(() => document.body.scrollHeight);
  console.log('body scrollHeight:', fullHeight);

  const parts = 4;
  const chunk = Math.ceil(fullHeight / parts);
  await page.setViewportSize({ width: 820, height: chunk + 30 });
  const names = ['top', 'mid1', 'mid2', 'bot'];
  for (let i = 0; i < parts; i++) {
    const y = i * chunk;
    if (y >= fullHeight) break;
    await page.evaluate((yy) => window.scrollTo(0, yy), y);
    await page.waitForTimeout(150);
    await page.screenshot({ path: resolve(buildDir, `preview-${names[i]}.png`) });
  }
} finally {
  await browser.close();
}
