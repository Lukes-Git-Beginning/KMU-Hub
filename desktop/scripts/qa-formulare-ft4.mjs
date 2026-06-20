/**
 * QA — formulare FT-4: view + filter, plus the batch i18n sweep.
 *  1) Formulare tab: status quick-filter chips + grid/list toggle.
 *  2) Status filter narrows the list; list view renders the compact table.
 *  3) i18n ×4 (de/en/fr/it): formulare tab, Eingänge tab, submission detail,
 *     builder — screenshots + raw-key / unparsed-ICU scan.
 *  0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-ft4')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const LOC = (l) => `try{localStorage.setItem('cosmi-locale', JSON.stringify({state:{locale:'${l}'},version:0}))}catch(e){}`
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|moduleSettings\.[a-z]|\{\{|, plural,|, select,)/

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

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1+2) DE: filter + view toggle ──────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(LOC('de'))
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('de: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('de console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  const gridCards = await page.locator('.grid [role="button"]').count()
  await page.screenshot({ path: resolve(outDir, '1-grid-chips.png'), fullPage: true })
  // filter: Aktiv
  await page.locator('button:has-text("Aktiv")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(500)
  const afterFilter = await page.locator('.grid [role="button"]').count()
  // list view
  await page.locator('button[aria-label="Listenansicht"]').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(500)
  const listRows = await page.locator('table tbody tr').count()
  await page.screenshot({ path: resolve(outDir, '2-list-view.png'), fullPage: true })
  out.push({ check: 'de-filter-view', gridCards, afterActiveFilter: afterFilter, listRows })
  await ctx.close()
}

// ── 3) i18n sweep across the four locales ──────────────────────
for (const loc of ['de', 'en', 'fr', 'it']) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(LOC(loc))
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push(`${loc}: ` + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push(`${loc} console: ` + m.text()) })

  // Formulare tab
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await page.screenshot({ path: resolve(outDir, `loc-${loc}-1-tab.png`), fullPage: true })
  const tabRaw = await rawKeys(page)

  // Eingänge tab + first group expanded
  const tabs = await page.locator('.flex.items-center.gap-4.border-b button').allTextContents().catch(() => [])
  await page.locator('.flex.items-center.gap-4.border-b button').nth(1).click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(900)
  await page.locator('.space-y-4 > .rounded-lg button').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, `loc-${loc}-2-eingaenge.png`), fullPage: true })
  const eingRaw = await rawKeys(page)

  // Submission detail (click first table row)
  await page.locator('table tbody tr').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, `loc-${loc}-3-submission.png`) })
  const subRaw = await rawKeys(page)

  out.push({ locale: loc, tabsSeen: tabs, tabRaw, eingRaw, subRaw })
  await ctx.close()
}

// ── 4) Builder i18n (FR) ───────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(LOC('fr'))
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('fr-builder: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await page.locator('[role="button"]:has-text("Kundenfeedback")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"]').last().locator('button:has-text("Modifier"), button:has-text("Bearbeiten")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, 'loc-fr-4-builder.png'), fullPage: true })
  out.push({ check: 'fr-builder', rawKeys: await rawKeys(page) })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
