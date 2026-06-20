/**
 * QA — formulare FT-3a: evaluation dashboard tab.
 *  1) Kundenfeedback → Auswertung tab: KPI row + per-field cards (bar chart for
 *     the select field, top-answers list for free text, consent distribution).
 *  2) Eventanmeldung → number avg/min/max + radio/checkbox bar charts.
 *  Charts must actually render (recharts). No raw keys, 0 pageErrors. :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-ft3a')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
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

async function openEvaluation(page, formName) {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator(`[role="button"]:has-text("${formName}")`).first().click({ timeout: 6000 })
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"]').last().locator('button:has-text("Auswertung")').first().click({ timeout: 5000 })
  await page.waitForTimeout(2200) // let recharts mount + animate
}

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1) Kundenfeedback evaluation ──
{
  const ctx = await browser.newContext({ viewport: { width: 1100, height: 1200 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('eval-feedback: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('eval console: ' + m.text()) })
  await openEvaluation(page, 'Kundenfeedback')
  const dlg = page.locator('[role="dialog"]').last()
  const text = await dlg.textContent()
  const svgCount = await dlg.locator('svg.recharts-surface').count()
  await dlg.screenshot({ path: resolve(outDir, '1-feedback-evaluation.png') })
  out.push({
    check: 'eval-feedback',
    hasKpiRow: /Gesamt/.test(text || '') && /Abschlussrate/.test(text || ''),
    chartsRendered: svgCount,
    hasConsentDist: /Zugestimmt/.test(text || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Eventanmeldung evaluation (number + radio + checkbox) ──
{
  const ctx = await browser.newContext({ viewport: { width: 1100, height: 1300 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('eval-event: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('eval2 console: ' + m.text()) })
  await openEvaluation(page, 'Eventanmeldung')
  const dlg = page.locator('[role="dialog"]').last()
  const text = await dlg.textContent()
  const svgCount = await dlg.locator('svg.recharts-surface').count()
  await dlg.screenshot({ path: resolve(outDir, '2-event-evaluation.png') })
  out.push({
    check: 'eval-event',
    chartsRendered: svgCount,
    hasNumberStats: /Ø|Min|Max/.test(text || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 3) Empty state — Support-Ticket (Entwurf) has no submissions ──
{
  const ctx = await browser.newContext({ viewport: { width: 900, height: 900 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('eval-empty: ' + String(e)))
  await openEvaluation(page, 'Support-Ticket')
  const dlg = page.locator('[role="dialog"]').last()
  const text = await dlg.textContent()
  await dlg.screenshot({ path: resolve(outDir, '3-empty-evaluation.png') })
  out.push({
    check: 'eval-empty',
    emptyShown: /Noch keine Eingänge/.test(text || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
