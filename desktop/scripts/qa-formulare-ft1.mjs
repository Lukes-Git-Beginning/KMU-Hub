/**
 * QA — formulare FT-1: submissions depth.
 *  1) Eingänge group: status quick-filter chips + sort menu, row click.
 *  2) Form detail modal: Details/Eingänge tabs; Eingänge shows submissions.
 *  3) Archive guard: archiving a form with unread submissions confirms first.
 *  4) Vorlagen card click opens a template detail modal with field list.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-ft1')
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

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1) Eingänge group: filter chips + sort ─────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('eingaenge: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('eingaenge console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 6000 })
  await page.waitForTimeout(1200)
  // expand the first group
  await page.locator('.space-y-4 > .rounded-lg button').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  const bodyText = await page.locator('body').textContent()
  const hasSort = await page.locator('button[aria-label="Sortieren nach"], button[title="Sortieren nach"]').count()
  await page.screenshot({ path: resolve(outDir, '1-eingaenge-filter-sort.png'), fullPage: true })
  out.push({
    check: 'eingaenge-panel',
    hasFilterChips: /Alle/.test(bodyText || '') && /Neu/.test(bodyText || ''),
    sortMenus: hasSort,
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Form detail Eingänge tab + row → submission detail ──────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('detail-tab: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('[role="button"]:has-text("Kundenfeedback")').first().click({ timeout: 6000 })
  await page.waitForTimeout(800)
  const dlg = page.locator('[role="dialog"]').last()
  await dlg.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  const tableRows = await dlg.locator('table tbody tr').count()
  await page.screenshot({ path: resolve(outDir, '2-detail-eingaenge-tab.png') })
  // click a row → submission detail modal stacks on top
  await dlg.locator('table tbody tr').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  const topDlgText = await page.locator('[role="dialog"]').last().textContent().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '3-submission-from-detail.png') })
  out.push({
    check: 'detail-eingaenge',
    rowsInModal: tableRows,
    submissionOpened: /Absender|IP-Adresse|Antworten/.test(topDlgText || ''),
  })
  await ctx.close()
}

// ── 4) Archive guard + 5) template detail ──────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('guard: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  // Kundenfeedback is active with unread submissions → archive should confirm
  await page.locator('[role="button"]:has-text("Kundenfeedback")').first().click({ timeout: 6000 })
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"]').last().locator('button:has-text("Archivieren")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  const alert = page.locator('[role="alertdialog"]')
  const alertText = await alert.textContent().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '4-archive-guard.png') })
  // dismiss
  await alert.locator('button:has-text("Abbrechen"), button:has-text("Abbruch")').first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(300)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  // Vorlagen tab → click template card
  await page.locator('button:has-text("Vorlagen")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.locator('[role="button"]:has-text("Zufriedenheitsumfrage")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  const tmplText = await page.locator('[role="dialog"]').last().textContent().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '5-template-detail.png') })
  out.push({
    check: 'guard-and-template',
    archiveConfirmShown: /unbearbeitete|unbearbeiteten/.test(alertText || ''),
    templateModalShown: /Vorlage verwenden/.test(tmplText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
