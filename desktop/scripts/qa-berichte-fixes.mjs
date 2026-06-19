// QA — berichte live-fixes: schedule row → detail modal, KPI sparkline tooltip.
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4000)

  // --- F2: KPI sparkline tooltip ---
  const spark = page.locator('button:has-text("Umsatz (MTD)") .recharts-surface').first()
  const sb = await spark.boundingBox()
  out.sparkBox = sb ? `${Math.round(sb.width)}x${Math.round(sb.height)}` : null
  if (sb) {
    for (const fx of [0.3, 0.5, 0.7]) {
      await page.mouse.move(sb.x + sb.width * fx, sb.y + sb.height * 0.5)
      await page.waitForTimeout(250)
    }
    await page.waitForTimeout(400)
  }
  out.tooltipVisible = await page.locator('.recharts-tooltip-wrapper').count()
  out.tooltipText = await page.locator('.recharts-tooltip-wrapper').first().innerText().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '6-sparkline-tooltip.png') })

  // --- F1: schedule row → detail modal ---
  await page.getByRole('button', { name: /^Geplant/ }).first().click()
  await page.waitForTimeout(1000)
  // click the name cell of the first row (inner toggle/delete stopPropagation)
  await page.locator('table tbody tr').first().locator('td').first().click()
  await page.waitForTimeout(800)
  const dialog = page.locator('[role="dialog"]')
  out.detailOpen = await dialog.isVisible().catch(() => false)
  const dText = await dialog.innerText().catch(() => '')
  out.detailHasReport = /Bericht|Report/.test(dText)
  out.detailHasHistory = /Lauf-Historie|Run history/.test(dText)
  out.detailHasRecipient = /@zentria\.tech/.test(dText)
  out.detailHistoryRows = await dialog.locator('ul li').count().catch(() => 0)
  out.detailRawKeys = [...new Set([...dText.matchAll(/\bberichte\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
  await page.screenshot({ path: resolve(outDir, '7-schedule-detail.png') })
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errs.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
