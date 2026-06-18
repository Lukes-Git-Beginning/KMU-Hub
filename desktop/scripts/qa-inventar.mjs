/**
 * QA-Script für das Inventar-Modul (Playwright, Demo-Mode/MSW).
 *
 * Prüft: Artikel-Liste, Lagerorte, Bewegungen, Inventur-Tab, Detail-Panel,
 * Raw-i18n-Keys, Emojis, leere Zustände.
 *
 * Voraussetzung: Dev-Server läuft (`npm run dev`, :5173, --mode demo).
 *   node desktop/scripts/qa-inventar.mjs
 *
 * Muster: scripts/qa-welle0-absences.mjs (electronAPI-Stub + cosmi-ui Onboarding).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots/inventar')
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
  console.log(`  📸 ${name}.png`)
}

const clickTabByText = async (needle) => {
  const tabs = await page.$$('button')
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

console.log('\n=== Inventar QA ===\n')

await page.goto(`${BASE}/#/inventar`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500) // MSW + React Query
await shot('01_initial_load')

// Artikel-Tabelle
try {
  await page.waitForSelector('table tbody tr', { timeout: 6000 })
  await shot('02_artikel_table')
} catch {
  console.warn('  ⚠️  keine Tabellenzeilen — evtl. Skeleton/leer')
  await shot('02_artikel_table_empty')
}

// Detail-Panel
const firstRow = await page.$('table tbody tr')
if (firstRow) {
  await firstRow.click()
  await page.waitForTimeout(600)
  await shot('03_detail_panel')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
}

// Tabs
if (await clickTabByText('Lagerort')) await shot('04_lagerorte')
if (await clickTabByText('Bewegung')) await shot('05_bewegungen')
if (await clickTabByText('Inventur')) await shot('06_inventur')

// Raw-i18n-Key-Scan
const bodyText = await page.evaluate(() => document.body.innerText)
const rawKeys = [...new Set(bodyText.match(/inventar\.[a-zA-Z.]+/g) || [])]
if (rawKeys.length) console.error(`  ❌ Raw-Keys: ${rawKeys.slice(0, 8).join(', ')}`)
else console.log('  ✅ keine Raw-Keys')

const emojis = bodyText.match(/[\u{1F300}-\u{1FAFF}]/gu) || []
if (emojis.length) console.warn(`  ⚠️  Emojis: ${[...new Set(emojis)].slice(0, 8).join('')}`)
else console.log('  ✅ keine Emojis')

if (errors.length) console.error(`  ❌ pageerror: ${errors.slice(0, 3).join(' | ')}`)
else console.log('  ✅ keine pageerrors')

await browser.close()
console.log(`\n✅ Screenshots: ${outDir}\n`)
