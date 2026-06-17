/**
 * QA — helpdesk Demo-tief-Pass (H-1 … H-8), parallel-batch sub-terminal.
 *
 * Screenshot harness against the :5174 QA dev server (vite.qa.config.mjs).
 * Usage: node scripts/qa-helpdesk.mjs [tag]   — tag names the screenshot subdir.
 *
 * Stubs window.electronAPI + marks onboarding complete (same pattern as the
 * vertraege/dashboard QA scripts), then drives the helpdesk routes and dumps a
 * JSON report of structural checks + raw-key / double-brace scans.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const TAG = process.argv[2] || 'baseline'
const outDir = resolve('.qa-screenshots', `helpdesk-${TAG}`)
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
// Wipe persisted helpdesk store so each run starts from fresh seed data.
const WIPE = `try{localStorage.removeItem('cosmi-helpdesk')}catch(e){}`

// Raw-key / double-brace scanners over visible text.
function scanners(page) {
  return page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
      .filter(Boolean)
    const rawKeys = all.filter((t) => /^(helpdesk|common|shared|moduleSettings)\.[a-zA-Z]/.test(t)).slice(0, 12)
    const doubleBrace = all.filter((t) => /\{\{|\}\}/.test(t)).slice(0, 12)
    return { rawKeys, doubleBrace }
  })
}

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(WIPE)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('[console] ' + m.text()) })

const out = []
try {
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.screenshot({ path: resolve(outDir, '0-tickets.png'), fullPage: true })
  const rows = await page.locator('tbody tr').count()
  out.push({ check: 'tickets-list', rows, ...(await scanners(page)) })

  // open first ticket → detail modal
  const row = page.locator('tbody tr').first()
  await row.evaluate((el) => el.click()).catch(async () => { await row.click({ timeout: 5000 }) })
  await page.waitForTimeout(1200)
  const dialogOpen = await page.locator('[role="dialog"]').isVisible().catch(() => false)
  const gradientStripe = await page.locator('[role="dialog"] .bg-gradient-to-r').count()
  await page.screenshot({ path: resolve(outDir, '1-ticket-detail.png') })
  out.push({ check: 'ticket-detail', dialogOpen, gradientStripe, ...(await scanners(page)) })

  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)

  // KB tab
  await page.locator('button:has-text("Wissensdatenbank")').first().click().catch(() => {})
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '2-kb.png'), fullPage: true })
  out.push({ check: 'kb-tab', ...(await scanners(page)) })

  // Stats tab
  await page.locator('button:has-text("Statistik")').first().click().catch(() => {})
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '3-stats.png'), fullPage: true })
  out.push({ check: 'stats-tab', ...(await scanners(page)) })

  out.push({ pageErrors: errs.slice(0, 6) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 6) })
} finally {
  await ctx.close()
  await browser.close()
}

console.log(JSON.stringify(out, null, 2))
