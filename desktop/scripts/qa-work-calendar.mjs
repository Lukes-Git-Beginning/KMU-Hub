// QA work Phase 5 — Kalender-Sicht: Monatsgrid, Navigation, Undated-Tray, Drag.
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
const failedApi = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(SUP)
const page = await ctx.newPage()
page.on('pageerror', (e) => errors.push(String(e).split('\n')[0]))
page.on('requestfailed', (r) => { const u = r.url(); if (u.includes('/api/')) failedApi.push(u) })

try {
  // Open a project, switch to calendar view
  await page.goto(`${BASE}/#/work/projects/prj-001`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2800)
  await page.getByRole('button', { name: /Kalender/i }).first().click({ timeout: 4000 }).catch((e) => { out.calendarToggleErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1500)

  // Grid present (7 weekday headers + day cells)
  out.monthLabel = await page.locator('text=/Januar|Februar|März|April|Mai|Juni|Juli|August|September|Oktober|November|Dezember/').first().textContent().catch(() => null)
  out.undatedTray = await page.getByText(/Ohne Fälligkeit/).first().isVisible().catch(() => false)
  out.todayButton = await page.getByRole('button', { name: /^Heute$/ }).first().isVisible().catch(() => false)
  await shot(page, 'calendar-month.png')
  out.calendarRawKeys = await scanRawKeys(page)

  // Navigate next month
  const nav = page.getByRole('button', { name: /Nächster Monat/ }).first()
  await nav.click({ timeout: 3000 }).catch((e) => { out.navErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  out.monthLabelAfterNext = await page.locator('text=/Januar|Februar|März|April|Mai|Juni|Juli|August|September|Oktober|November|Dezember/').first().textContent().catch(() => null)
  await shot(page, 'calendar-next-month.png')

  // Back to today
  await page.getByRole('button', { name: /^Heute$/ }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(600)

  // Count task chips in grid (draggable buttons with truncate title)
  out.chipCount = await page.locator('div.grid button[type="button"]').count().catch(() => -1)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.pageErrors = [...new Set(errors)].slice(0, 12)
out.failedApiRequests = [...new Set(failedApi)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
