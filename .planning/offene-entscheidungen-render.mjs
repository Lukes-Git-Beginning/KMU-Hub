// Rendert offene-entscheidungen-<datum>.html nach PDF, mit den Cosmi-Fonts eingebettet.
// Aufruf aus dem Repo-Wurzelverzeichnis:  node .planning/offene-entscheidungen-render.mjs
import { createRequire } from 'node:module'
import { readFileSync } from 'node:fs'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { dirname, join, resolve } from 'node:path'

const HERE = dirname(fileURLToPath(import.meta.url))
const REPO = resolve(HERE, '..')
const require = createRequire(join(REPO, 'desktop', 'package.json'))
const { chromium } = require('playwright')

const DOC = process.argv[2] ?? 'offene-entscheidungen-2026-08-20'
const FONT_DIR = join(REPO, 'desktop', 'src', 'renderer', 'public', 'fonts')

const face = (family, file, weight = '200 900', style = 'normal') => {
  const b64 = readFileSync(join(FONT_DIR, file)).toString('base64')
  return `@font-face{font-family:'${family}';font-style:${style};font-weight:${weight};font-display:block;src:url(data:font/woff2;base64,${b64}) format('woff2');}`
}

const fontCss = [
  face('Plus Jakarta Sans', 'plus-jakarta-sans-latin.woff2'),
  face('Plus Jakarta Sans', 'plus-jakarta-sans-latin-ext.woff2'),
  face('Plus Jakarta Sans', 'plus-jakarta-sans-italic-latin.woff2', '200 900', 'italic'),
  face('Playfair Display', 'playfair-display-latin.woff2', '400 900'),
  face('Playfair Display', 'playfair-display-latin-ext.woff2', '400 900'),
  face('JetBrains Mono', 'jetbrains-mono-latin.woff2', '100 800'),
  face('JetBrains Mono', 'jetbrains-mono-latin-ext.woff2', '100 800'),
].join('\n')

const browser = await chromium.launch()
const page = await browser.newPage()

await page.goto(pathToFileURL(join(HERE, `${DOC}.html`)).href, { waitUntil: 'load' })
await page.addStyleTag({ content: fontCss })
await page.evaluate(() => document.fonts.ready)
await page.emulateMedia({ media: 'print' })

const footer = `
<div style="width:100%;font-size:6.5pt;font-family:'JetBrains Mono',monospace;color:#9aa5a3;
            padding:0 18mm;display:flex;justify-content:space-between;letter-spacing:.08em;">
  <span>ZENTRIA · COSMI 1.0 · OFFENE ENTSCHEIDUNGEN</span>
  <span class="pageNumber"></span>
</div>`

await page.pdf({
  path: join(HERE, `${DOC}.pdf`),
  format: 'A4',
  printBackground: true,
  displayHeaderFooter: true,
  headerTemplate: '<div></div>',
  footerTemplate: footer,
  margin: { top: '16mm', bottom: '16mm', left: '18mm', right: '18mm' },
})

console.log(`${DOC}.pdf geschrieben (${await page.evaluate(() => document.querySelectorAll('.q').length)} Fragen).`)
await browser.close()
