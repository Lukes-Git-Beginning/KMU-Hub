// QA P3 zeiterfassung — Auswertungen view (charts, KPIs, week/month toggle).
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/ze-p3')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = ['zeiterfassung.', 'profil.zeiterfassung.']

async function dismiss(page) {
  for (const name of [/überspringen/i, /skip/i]) {
    const b = page.getByRole('button', { name }).first()
    try { if (await b.isVisible({ timeout: 500 })) { await b.click(); await page.waitForTimeout(300) } } catch (e) {}
  }
}
async function openAnalytics(page) {
  await page.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2200)
  await dismiss(page)
  await page.getByRole('tab', { name: /auswertungen/i }).first().click()
  await page.waitForTimeout(1400) // chart render
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

for (const size of [{ name: 'full', width: 1440, height: 950 }, { name: 'small', width: 500, height: 900 }]) {
  const ctx = await browser.newContext({ viewport: { width: size.width, height: size.height } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    await openAnalytics(page)
    await page.screenshot({ path: resolve(outDir, `analytics-week__${size.name}.png`), fullPage: size.name === 'small' })
    const body = (await page.locator('body').innerText()).toLowerCase()
    results.push({ shot: `week ${size.name}`, errors: errors.length, rawKeys: RAW.filter((p) => body.includes(p)) })

    if (size.name === 'full') {
      await page.getByRole('button', { name: /^monat$/i }).first().click()
      await page.waitForTimeout(1200)
      await page.screenshot({ path: resolve(outDir, 'analytics-month__full.png') })
      results.push({ shot: 'month full', errors: errors.length })
    }
  } catch (err) {
    results.push({ shot: size.name, error: String(err).slice(0, 120) })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
