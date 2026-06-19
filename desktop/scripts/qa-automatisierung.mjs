import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/automatisierung')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errors = []
page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}
try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  await page.evaluate(() => { window.location.hash = '#/automatisierung' })
  await page.waitForTimeout(2600)
  // Default tab (automations table + StatsBar)
  await page.screenshot({ path: resolve(outDir, '1-automations.png') })
  out.rows = await page.locator('table tbody tr').count().catch(() => -1)
  out.emptyState = await page.getByText(/keine|empty|noch keine/i).first().isVisible().catch(() => false)
  // Tabs (Radix role=tab): 0=automations, 1=templates, 2=log
  const tabs = page.getByRole('tab')
  out.tabCount = await tabs.count().catch(() => -1)
  if (out.tabCount >= 2) {
    await tabs.nth(1).click({ force: true })
    await page.waitForTimeout(1400)
    out.templatesActive = await tabs.nth(1).getAttribute('data-state').catch(() => null)
    await page.screenshot({ path: resolve(outDir, '2-templates.png') })
  }
  if (out.tabCount >= 3) {
    await tabs.nth(2).click({ force: true })
    await page.waitForTimeout(1400)
    await page.screenshot({ path: resolve(outDir, '3-log.png') })
  }
  // Raw-key sniff
  const body = await page.evaluate(() => document.body.innerText)
  out.rawKeys = (body.match(/automatisierung\.[a-zA-Z.]+/g) || []).slice(0, 8)
  out.doubleBraces = (body.match(/\{\{[^}]+\}\}/g) || []).slice(0, 5)
} catch (e) { out.error = String(e).split('\n')[0] } finally {
  out.consoleErrors = errors.slice(0, 8)
  await ctx.close(); await browser.close()
}
console.log(JSON.stringify(out, null, 2))
