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
  await page.waitForTimeout(2200)
  // Open the module-settings overlay from the sidebar button
  await page.getByText(/Modul-Einstellungen|Einstellungen/i).first().click({ force: true })
  await page.waitForTimeout(900)
  out.overlayOpen = await page.getByRole('dialog').isVisible().catch(() => false)
  // Click the automatisierung entry in the overlay nav
  const entry = page.getByRole('dialog').getByText('Automatisierung', { exact: true })
  if ((await entry.count()) > 0) {
    await entry.first().click({ force: true })
    await page.waitForTimeout(700)
  }
  out.hasStartTab = await page.getByText(/Standard-Ansicht/i).first().isVisible().catch(() => false)
  out.hasRetention = await page.getByText(/Protokoll-Aufbewahrung|Aufbewahrungsdauer/i).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '7-settings.png') })
  const body = await page.evaluate(() => document.body.innerText)
  out.rawKeys = (body.match(/automatisierung\.[a-zA-Z.]+|moduleSettings\.[a-zA-Z.]+/g) || []).slice(0, 8)
  out.doubleBraces = (body.match(/\{\{[^}]+\}\}/g) || []).slice(0, 5)
} catch (e) { out.error = String(e).split('\n')[0] } finally {
  out.consoleErrors = errors.slice(0, 6)
  await ctx.close(); await browser.close()
}
console.log(JSON.stringify(out, null, 2))
