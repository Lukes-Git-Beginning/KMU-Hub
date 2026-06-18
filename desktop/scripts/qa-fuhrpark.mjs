/**
 * QA-Script fuer das Fuhrpark-Modul (Playwright, Demo-Mode/MSW).
 *
 * Prueft: Fahrzeuge-Tab, Wartung-Tab, Tankprotokoll-Tab, Fahrtenbuch-Tab,
 * GPS-Tracking-Tab, Detail/Dialog, Raw-i18n-Keys, Emojis, leere Zustaende.
 *
 * Voraussetzung: Dev-Server laeuft (`npm run dev`, :5173, --mode demo).
 *   node desktop/scripts/qa-fuhrpark.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots/qa-fuhrpark')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()

const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

const shot = async (name) => {
  await page.screenshot({ path: resolve(outDir, `${name}.png`), fullPage: false })
  console.log(`  screenshot: ${name}.png`)
}

const clickTabByText = async (needle) => {
  const tabs = await page.$$('button[role="tab"], button')
  for (const t of tabs) {
    const txt = (await page.evaluate((el) => el.textContent, t)) || ''
    if (txt.includes(needle)) {
      await t.click()
      await page.waitForTimeout(700)
      return true
    }
  }
  return false
}

console.log('\n=== Fuhrpark QA ===\n')

await page.goto(`${BASE}/#/fuhrpark`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500) // MSW + React Query

// 1. Fahrzeuge-Tab (default)
await shot('01_fahrzeuge_tab')
try {
  await page.waitForSelector('table tbody tr', { timeout: 6000 })
  await shot('02_fahrzeuge_table')
} catch {
  console.warn('  no table rows — skeleton or empty state')
  await shot('02_fahrzeuge_empty')
}

// 2. Wartung-Tab
if (await clickTabByText('artung')) await shot('03_wartung_tab')

// 3. Tankprotokoll-Tab
if (await clickTabByText('ankprotokoll')) await shot('04_tankprotokoll_tab')

// 4. Fahrtenbuch-Tab
if (await clickTabByText('ahrtenbuch')) await shot('05_fahrtenbuch_tab')

// 5. GPS-Tracking-Tab
if (await clickTabByText('racking')) await shot('06_tracking_tab')

// 6. Raw-i18n-Key-Scan
const bodyText = await page.evaluate(() => document.body.innerText)
const rawKeys = [...new Set(bodyText.match(/fuhrpark\.[a-zA-Z_.]+/g) || [])]
if (rawKeys.length) console.error(`  RAW KEYS FOUND: ${rawKeys.slice(0, 8).join(', ')}`)
else console.log('  no raw i18n keys')

const emojis = bodyText.match(/[\u{1F300}-\u{1FAFF}]/gu) || []
if (emojis.length) console.warn(`  emojis found: ${[...new Set(emojis)].slice(0, 8).join('')}`)
else console.log('  no emojis')

if (errors.length) console.error(`  pageerrors: ${errors.slice(0, 3).join(' | ')}`)
else console.log('  no pageerrors')

await browser.close()
console.log(`\nDone. Screenshots: ${outDir}\n`)
