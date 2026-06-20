/**
 * QA — formulare FD-0: lifecycle guards (draft → active → closed → archived).
 *  1) Formulare tab renders all four status badges (closed=warning, archived=muted).
 *  2) Draft detail shows "Veröffentlichen" and a DISABLED "Teilen".
 *  3) Closed detail (Eventanmeldung) shows "Wieder öffnen".
 *  4) Closed form preview shows the close notice with the configured message.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-fd0')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|moduleSettings\.[a-z]|\{\{)/

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

// ── 1) Formulare tab — status badges ───────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('tab: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('tab console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  const bodyText = await page.locator('body').textContent()
  await page.screenshot({ path: resolve(outDir, '1-tab-status-badges.png'), fullPage: true })
  out.push({
    check: 'tab-badges',
    hasGeschlossen: /Geschlossen/.test(bodyText || ''),
    hasArchiviert: /Archiviert/.test(bodyText || ''),
    hasEntwurf: /Entwurf/.test(bodyText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Draft detail: publish + disabled share ──────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('draft: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.locator('[role="button"]:has-text("Support-Ticket")').first().click({ timeout: 6000 }).catch(() => {})
  await page.waitForTimeout(900)
  const dlg = page.locator('[role="dialog"]').last()
  const dlgText = await dlg.textContent().catch(() => '')
  const shareBtn = dlg.locator('button:has-text("Teilen")').first()
  const shareDisabled = await shareBtn.isDisabled().catch(() => null)
  await page.screenshot({ path: resolve(outDir, '2-draft-detail.png') })
  out.push({
    check: 'draft-detail',
    hasPublish: /Veröffentlichen/.test(dlgText || ''),
    shareDisabled,
  })
  await ctx.close()
}

// ── 3) Closed detail: reopen + 4) closed preview notice ────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('closed: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.locator('[role="button"]:has-text("Eventanmeldung")').first().click({ timeout: 6000 }).catch(() => {})
  await page.waitForTimeout(900)
  const dlg = page.locator('[role="dialog"]').last()
  const dlgText = await dlg.textContent().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '3-closed-detail.png') })
  const hasReopen = /Wieder öffnen/.test(dlgText || '')
  // Open the preview from the detail footer
  await dlg.locator('button:has-text("Vorschau")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(900)
  const prevText = await page.locator('[role="dialog"]').last().textContent().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '4-closed-preview.png') })
  out.push({
    check: 'closed',
    hasReopen,
    previewClosedNotice: /Dieses Formular ist geschlossen/.test(prevText || ''),
    previewHasMessage: /Sommerfest ist abgeschlossen/.test(prevText || ''),
  })
  await ctx.close()
}

// ── 5) Width responsiveness ────────────────────────────────────
for (const w of [1280, 1024]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: 900 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push(`w${w}: ` + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await page.screenshot({ path: resolve(outDir, `5-width-${w}.png`), fullPage: true })
  out.push({ check: `width-${w}`, rawKeys: await rawKeys(page) })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
