// QA work Phase 4 — Portfolio-View, abgeleitetes Budget, Auslastung-Vorschau.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUP = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(work|common|moduleSettings|settings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function shot(page, name) { await page.screenshot({ path: resolve(outDir, name), fullPage: true }) }

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const errors = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(SUP)
const page = await ctx.newPage()
page.on('pageerror', (e) => errors.push(String(e).split('\n')[0]))

try {
  // 1. Projects → portfolio toggle
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2800)
  await page.getByRole('button', { name: /^Portfolio$/ }).first().click({ timeout: 4000 }).catch((e) => { out.portfolioToggleErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.portfolioRows = await page.locator('text=/Cosmi v2.0|Website Relaunch|Mobile App/').count().catch(() => -1)
  out.healthSignals = await page.locator('text=/Im Plan|In Verzug|Überfällig|Fertig/').count().catch(() => -1)
  await shot(page, 'portfolio-table.png')
  out.portfolioRawKeys = await scanRawKeys(page)

  // 2. Open a project → budget section derived + "Geschätzt" badge
  await page.goto(`${BASE}/#/work/projects/prj-001`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  // ensure list/kanban (budget shows there)
  await page.getByRole('button', { name: /Liste/i }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(1000)
  out.budgetEstimatedBadge = await page.getByText(/^Geschätzt$/).first().isVisible().catch(() => false)
  out.budgetVisible = await page.getByText(/Geplantes Budget|Budget/).first().isVisible().catch(() => false)
  // breakdown should list real assignee names from tasks
  out.budgetHasRealNames = await page.locator('text=/Julia Weber|Stefan Müller|Thomas Braun/').count().catch(() => -1)
  await shot(page, 'portfolio-budget.png')
  out.budgetRawKeys = await scanRawKeys(page)

  // 3. Auslastung tab → preview banner
  await page.getByRole('button', { name: /Auslastung/i }).first().click({ timeout: 3000 }).catch((e) => { out.auslastungErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.previewBanner = await page.getByText(/Vorschau mit Beispieldaten/).first().isVisible().catch(() => false)
  await shot(page, 'portfolio-auslastung.png')
  out.auslastungRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.pageErrors = [...new Set(errors)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
