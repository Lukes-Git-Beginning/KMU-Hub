/**
 * QA-Script für das Rapporte-Modul (Playwright, Demo-Mode/MSW).
 *
 * Prüft: Tagesberichte-Liste, Aufmaß/Measurements, Vorlagen/Templates,
 * Detail-Panel, Raw-i18n-Keys, Emojis, leere Zustände.
 *
 * Voraussetzung: Dev-Server läuft (`npm run dev`, :5173, --mode demo).
 *   node desktop/scripts/qa-rapporte.mjs
 *
 * Muster: scripts/qa-welle0-absences.mjs (electronAPI-Stub + cosmi-ui Onboarding).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots/rapporte')
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

console.log('\n=== Rapporte QA ===\n')

await page.goto(`${BASE}/#/rapporte`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500) // MSW + React Query
await shot('01_initial_load')

// Berichte-Liste / Karten
try {
  await page.waitForSelector('table tbody tr, [class*="card"], [class*="Card"]', { timeout: 6000 })
  await shot('02_berichte_list')
} catch {
  console.warn('  ⚠️  keine Berichte sichtbar — evtl. Skeleton/leer')
  await shot('02_berichte_empty')
}

// Detail (erste Zeile/Karte)
const firstRow = await page.$('table tbody tr')
if (firstRow) {
  await firstRow.click()
  await page.waitForTimeout(600)
  await shot('03_detail_panel')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
}

// Tabs: Aufmaß / Vorlagen
if (await clickTabByText('Aufmaß') || await clickTabByText('Aufmass') || await clickTabByText('Measurement')) await shot('04_aufmass')
if (await clickTabByText('Vorlage') || await clickTabByText('Template')) await shot('05_vorlagen')

// Raw-i18n-Key-Scan
const bodyText = await page.evaluate(() => document.body.innerText)
const rawKeys = [...new Set(bodyText.match(/rapporte\.[a-zA-Z.]+/g) || [])]
if (rawKeys.length) console.error(`  ❌ Raw-Keys: ${rawKeys.slice(0, 8).join(', ')}`)
else console.log('  ✅ keine Raw-Keys')

const emojis = bodyText.match(/[\u{1F300}-\u{1FAFF}]/gu) || []
if (emojis.length) console.warn(`  ⚠️  Emojis: ${[...new Set(emojis)].slice(0, 8).join('')}`)
else console.log('  ✅ keine Emojis')

if (errors.length) console.error(`  ❌ pageerror: ${errors.slice(0, 3).join(' | ')}`)
else console.log('  ✅ keine pageerrors')

await browser.close()
console.log(`\n✅ Screenshots: ${outDir}\n`)
