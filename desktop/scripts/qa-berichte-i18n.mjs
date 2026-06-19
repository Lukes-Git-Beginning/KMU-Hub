// QA — berichte B-5: English locale sweep across all tabs (dev :5173).
// Verifies migrated keys render in EN (no German UI leftovers, no raw keys).
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const EN = `try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'en'},version:0}))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(EN)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

const collect = async () => (await page.evaluate(() => document.body.innerText)) || ''
let full = ''

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4000)
  full += await collect()
  // drilldown
  await page.getByText('Umsatz (MTD)').first().click().catch(() => {})
  await page.waitForTimeout(700)
  full += '\n' + (await collect())
  await page.screenshot({ path: resolve(outDir, 'en-1-dashboard-drilldown.png') })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  // Geplant
  await page.getByRole('button', { name: /^Scheduled|^Geplant/ }).first().click().catch(() => {})
  await page.waitForTimeout(1000)
  full += '\n' + (await collect())
  await page.screenshot({ path: resolve(outDir, 'en-2-scheduled.png'), fullPage: true })
  // DATEV
  await page.getByRole('button', { name: /^DATEV$/ }).first().click().catch(() => {})
  await page.waitForTimeout(1800)
  full += '\n' + (await collect())
  // Settings overlay
  await page.getByText('Module settings', { exact: true }).first().click().catch(async () => {
    await page.getByText('Modul-Einstellungen', { exact: true }).first().click().catch(() => {})
  })
  await page.waitForTimeout(1200)
  full += '\n' + (await collect())
  await page.screenshot({ path: resolve(outDir, 'en-3-settings.png'), fullPage: true })

  // EN strings that MUST appear (migrated keys)
  out.enPresent = {
    nextRun: full.includes('Next run'),
    trendDemo: full.includes('Trend (demo)'),
    defaultFormat: full.includes('Default format'),
    allowedFormats: full.includes('Allowed export formats'),
  }
  // German UI strings that MUST NOT appear in EN (would mean missing translation)
  out.germanLeftovers = [
    'Nächster Lauf', 'Verlauf (Demo)', 'Standard-Format', 'Erlaubte Export-Formate',
    'Berichte-Einstellungen', 'Zulässige E-Mail-Domains',
  ].filter((s) => full.includes(s))
  // raw keys
  out.rawKeys = [...new Set([...full.matchAll(/\b(berichte|moduleSettings|common|shared)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
  out.doubleBraces = [...new Set([...full.matchAll(/\{\{[^}]+\}\}/g)].map((m) => m[0]))]
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errs.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
