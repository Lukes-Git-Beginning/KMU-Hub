/**
 * QA — formulare FD-2: distribution overview (Verteilung tab).
 *  1) Form with links → table: channel, created, expires, views, responses
 *     (+conversion %), status, actions.
 *  2) Closed form's link shows "Abgelaufen".
 *  3) Form without links → empty state with a share action.
 *  4) Deactivate action flips a link to "Deaktiviert" (MSW-stateful).
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-fd2')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|moduleSettings\.[a-z]|\{\{|, plural,)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return [...new Set(
      Array.from(document.querySelectorAll('body *'))
        .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
        .map((n) => n.textContent.trim()),
    )].slice(0, 12)
  }, RAW_RE.source)
}

async function openVerteilung(page, title) {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator(`[role="button"]:has-text("${title}")`).first().click({ timeout: 6000 })
  await page.waitForTimeout(800)
  const dlg = page.locator('[role="dialog"]').last()
  await dlg.locator('button:has-text("Verteilung")').first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  return dlg
}

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1) Kundenfeedback: links table ─────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('table: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('table console: ' + m.text()) })
  const dlg = await openVerteilung(page, 'Kundenfeedback')
  const rows = await dlg.locator('table tbody tr').count()
  const text = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '1-verteilung-table.png') })
  out.push({
    check: 'links-table',
    rows,
    hasViews: /Aufrufe/.test(text || ''),
    hasConversion: /%\)/.test(text || ''),
    hasChannels: /Link/.test(text || '') && /QR-Code/.test(text || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) closed form: expired link ───────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('expired: ' + String(e)))
  const dlg = await openVerteilung(page, 'Eventanmeldung')
  const text = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '2-expired-link.png') })
  out.push({ check: 'expired', hasExpired: /Abgelaufen/.test(text || '') })
  await ctx.close()
}

// ── 3) form without links: empty state ─────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('empty: ' + String(e)))
  const dlg = await openVerteilung(page, 'Bewerbungsformular')
  const text = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '3-empty-state.png') })
  out.push({ check: 'empty', hasEmpty: /Noch nicht geteilt/.test(text || '') })
  await ctx.close()
}

// ── 4) deactivate action ───────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('toggle: ' + String(e)))
  const dlg = await openVerteilung(page, 'Kundenfeedback')
  // open the first row's actions menu and click Deaktivieren
  await dlg.locator('table tbody tr').first().locator('button').last().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(500)
  await page.locator('[role="menuitem"]:has-text("Deaktivieren")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(900)
  const text = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '4-after-deactivate.png') })
  out.push({ check: 'deactivate', hasDisabled: /Deaktiviert/.test(text || '') })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
