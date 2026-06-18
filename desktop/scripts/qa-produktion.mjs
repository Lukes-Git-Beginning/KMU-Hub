/**
 * QA-Script fuer das Produktion-Modul (Playwright, Demo-Mode/MSW).
 *
 * Prueft: Auftraege-Tab, Stuecklisten-Tab (aufgeklappt), Qualitaet-Tab,
 * Maschinen/Chart-Tab, Detail-Panel, Neuer-Auftrag-Dialog, Raw-i18n-Keys,
 * Emojis, leere Zustaende.
 *
 * Voraussetzung: Dev-Server laeuft (`npm run dev`, :5173, --mode demo).
 *   node desktop/scripts/qa-produktion.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots/qa-produktion')
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

console.log('\n=== Produktion QA ===\n')

await page.goto(`${BASE}/#/produktion`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500) // MSW + React Query

// 1. Auftraege-Tab (default)
await shot('01_auftraege_tab')

// 2. Table rows
try {
  await page.waitForSelector('table tbody tr', { timeout: 6000 })
  await shot('02_auftraege_table')
} catch {
  console.warn('  no table rows — skeleton or empty state')
  await shot('02_auftraege_table_empty')
}

// 3. Detail panel (click first row)
const firstRow = await page.$('table tbody tr')
if (firstRow) {
  await firstRow.click()
  await page.waitForTimeout(700)
  await shot('03_detail_panel')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
}

// 4. Stuecklisten-Tab
const stueckFound = await clickTabByText('ckliste')
if (stueckFound) {
  await shot('04_stuecklisten_tab')
  // Expand first BOM row
  const expandBtn = await page.$('button[aria-label], table tbody tr button')
  if (expandBtn) {
    await expandBtn.click()
    await page.waitForTimeout(500)
    await shot('05_stueckliste_expanded')
  }
}

// 5. Qualitaet-Tab
const qualFound = await clickTabByText('alit')
if (qualFound) {
  await shot('06_qualitaet_tab')
}

// 6. Maschinen-Tab
const maschinFound = await clickTabByText('aschinen')
if (maschinFound) {
  await shot('07_maschinen_tab')
}

// 7. Neuer Auftrag Dialog
// Navigate back to auftraege tab first
await clickTabByText('ftrag')
await page.waitForTimeout(500)
const buttons = await page.$$('button')
for (const btn of buttons) {
  const txt = (await page.evaluate((el) => el.textContent, btn)) || ''
  if (txt.includes('Neuer Auftrag') || txt.includes('uftrag erstellen')) {
    await btn.click()
    await page.waitForTimeout(600)
    await shot('08_new_order_dialog')
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
    break
  }
}

// 8. Raw-i18n-Key-Scan
const bodyText = await page.evaluate(() => document.body.innerText)
const rawKeys = [...new Set(bodyText.match(/produktion\.[a-zA-Z_.]+/g) || [])]
if (rawKeys.length) console.error(`  RAW KEYS FOUND: ${rawKeys.slice(0, 8).join(', ')}`)
else console.log('  no raw i18n keys')

const emojis = bodyText.match(/[\u{1F300}-\u{1FAFF}]/gu) || []
if (emojis.length) console.warn(`  emojis found: ${[...new Set(emojis)].slice(0, 8).join('')}`)
else console.log('  no emojis')

if (errors.length) console.error(`  pageerrors: ${errors.slice(0, 3).join(' | ')}`)
else console.log('  no pageerrors')

await browser.close()
console.log(`\nDone. Screenshots: ${outDir}\n`)