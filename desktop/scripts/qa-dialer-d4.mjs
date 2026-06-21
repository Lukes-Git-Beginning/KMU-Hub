// QA — dialer D-4 i18n + dead-string cleanup (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/dialer-d4')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 940 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(9000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const step = async (name, fn) => {
  try { await fn() } catch (err) { out[`ERR_${name}`] = String(err).split('\n')[0] }
}
const rawKeysOf = () => page.evaluate(() => {
  const all = Array.from(document.querySelectorAll('body *'))
    .filter((n) => n.children.length === 0)
    .map((n) => (n.textContent || '').trim())
  return [...new Set(all.filter((t) => /^(dialer|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 12)
})

// ── Agent dashboard (StatCard string values must not be NaN) ──
await step('dashboard', async () => {
  await page.goto(`${BASE}/#/dialer/dashboard`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  const body = await bodyText()
  out.dashHasNaN = /NaN/.test(body)
  out.dashRawKeys = await rawKeysOf()
  await page.screenshot({ path: resolve(outDir, 'd4-1-dashboard.png'), fullPage: false })
})

// ── Campaign detail (stats labels, table headers) ──
await step('campaignDetail', async () => {
  await page.goto(`${BASE}/#/dialer/campaigns/dlr-camp-001`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  const body = await bodyText()
  out.detailRawKeys = await rawKeysOf()
  out.hasGesamt = /Gesamt/.test(body)
  out.hasPerformance = /Performance/.test(body)
  out.hasStatusHeader = /Status/.test(body)
  await page.screenshot({ path: resolve(outDir, 'd4-2-campaign-detail.png'), fullPage: false })
})

// ── New campaign dialog (form labels) ──
await step('campaignForm', async () => {
  await page.goto(`${BASE}/#/dialer/campaigns`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2500)
  await page.getByRole('button', { name: /Neue Kampagne/ }).first().click()
  await page.waitForTimeout(900)
  const body = await bodyText()
  out.formRawKeys = await rawKeysOf()
  out.hasBeschreibung = /Beschreibung/.test(body)
  out.hasModus = /Modus/.test(body)
  await page.screenshot({ path: resolve(outDir, 'd4-3-campaign-form.png'), fullPage: false })
})

out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
