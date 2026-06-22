/**
 * QA — formulare FO-1/FO-2: field duplicate + home-stat real value + a11y.
 *  1) Home stats: third card shows real "Neue Eingänge" count (no hardcoded 87%).
 *  2) Editor: per-field "duplizieren" button clones a field in place (+1 row).
 *  3) Eingänge: submission rows are keyboard-focusable (role=button + tabindex).
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-fo1')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
// Reset the persisted draft so the editor opens clean each run.
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

// ── 1) Home stats (no 87%, has Neue Eingänge) ──
{
  const ctx = await browser.newContext({ viewport: { width: 1400, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('home: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('home console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  const bodyText = await page.locator('body').textContent()
  await page.screenshot({ path: resolve(outDir, '1-home-stats.png'), fullPage: true })
  out.push({
    check: 'home-stats',
    hasNeueEingaenge: /Neue Eingänge/.test(bodyText || ''),
    hasHardcoded87: /\b87%/.test(bodyText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Editor: duplicate a field (+1 duplicate-buttons) ──
{
  const ctx = await browser.newContext({ viewport: { width: 1400, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('editor: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('editor console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  // open the first card's ItemActions → Bearbeiten
  const card = page.locator('[role="button"]:has-text("Kundenfeedback")').first()
  await card.locator('button').first().click({ timeout: 6000 })
  await page.waitForTimeout(400)
  await page.locator('[role="menuitem"]:has-text("Bearbeiten")').first().click({ timeout: 5000 })
  await page.waitForTimeout(1000)
  const dupSel = 'button[title="Feld duplizieren"]'
  const before = await page.locator(dupSel).count()
  // hover the first field row to reveal the inline controls, then duplicate
  const firstField = page.locator(dupSel).first()
  await firstField.scrollIntoViewIfNeeded()
  await firstField.click({ force: true, timeout: 5000 })
  await page.waitForTimeout(800)
  const after = await page.locator(dupSel).count()
  await page.screenshot({ path: resolve(outDir, '2-editor-duplicated.png'), fullPage: true })
  out.push({
    check: 'field-duplicate',
    before,
    after,
    increased: after === before + 1,
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 3) Eingänge: submission rows keyboard-focusable ──
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('eingaenge: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  // expand the first form group to reveal the submissions table
  await page.locator('button:has-text("Kundenfeedback")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  const focusableRows = await page.locator('tr[role="button"][tabindex="0"]').count()
  await page.screenshot({ path: resolve(outDir, '3-eingaenge-list.png'), fullPage: true })
  out.push({
    check: 'submission-a11y',
    focusableRows,
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
