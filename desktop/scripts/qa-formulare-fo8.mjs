/**
 * QA — formulare FO-8: submissions-over-time chart in the Auswertung tab.
 *  Open Kundenfeedback → Auswertung: the "Einreichungen über Zeit" area chart
 *  renders above the conversion funnel and per-field cards.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-fo8')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NODRAFT = `try{localStorage.removeItem('cosmi-formulare-draft')}catch(e){}`
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|moduleSettings\.[a-z]|\{\{|, plural,)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 12)
  }, RAW_RE.source)
}

const browser = await chromium.launch()
const out = []
const errs = []

{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('eval: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED/.test(m.text())) errs.push('eval console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  // open Kundenfeedback detail (whole card click)
  await page.locator('[role="button"]:has-text("Kundenfeedback")').first().click({ timeout: 6000 })
  await page.waitForTimeout(800)
  const dlg = page.locator('[role="dialog"]').last()
  await dlg.locator('button:has-text("Auswertung")').first().click({ timeout: 5000 })
  await page.waitForTimeout(1200)
  const dlgText = (await dlg.textContent()) || ''
  const svgCount = await dlg.locator('svg.recharts-surface').count()
  await page.screenshot({ path: resolve(outDir, '1-auswertung-timechart.png'), fullPage: true })
  out.push({
    check: 'time-chart',
    hasTimeTitle: dlgText.includes('Einreichungen über Zeit'),
    chartCount: svgCount,
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
