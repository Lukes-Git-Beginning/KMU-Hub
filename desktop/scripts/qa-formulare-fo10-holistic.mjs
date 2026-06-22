/**
 * QA — formulare FO-10: holistic raw-key + layout sweep across the module.
 *  Part A (de): forms grid/list, Eingänge, Vorlagen, editor (incl. notifications),
 *    form-detail tabs (Details/Eingänge/Auswertung/Verteilung), submission detail,
 *    settings panel.
 *  Part B (en): the new surfaces (editor notifications + Auswertung time chart)
 *    to confirm the added keys exist in English too.
 *  Pass = every surface rawKeys:[] and 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-fo10')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NODRAFT = `try{localStorage.removeItem('cosmi-formulare-draft')}catch(e){}`
const setLocale = (lng) => `try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'${lng}'},version:0}))}catch(e){}`
// Raw-key detector: formulare.*/moduleSettings.* leftovers, {{ }}, or ICU plural literals.
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|moduleSettings\.[a-z]|common\.[a-z][a-zA-Z.]+|\{\{|, plural,)/

async function rawKeys(scope) {
  return scope.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 12)
  }, RAW_RE.source)
}

async function newPage(browser, lng, w = 1400, h = 1100) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(NODRAFT)
  if (lng) await ctx.addInitScript(setLocale(lng))
  const page = await ctx.newPage()
  return { ctx, page }
}

async function openEditor(page, formTitle) {
  await page.locator(`[role="button"]:has-text("${formTitle}")`).first().locator('button').first().click({ timeout: 6000 })
  await page.waitForTimeout(400)
  await page.locator('[role="menuitem"]').filter({ hasText: /Bearbeiten|Edit|Modifier|Modifica/ }).first().click({ timeout: 5000 })
  await page.waitForTimeout(1000)
}

const browser = await chromium.launch()
const out = []
const errs = []

// ── Part A — German sweep ──
{
  const { ctx, page } = await newPage(browser, 'de')
  page.on('pageerror', (e) => errs.push('de: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED/.test(m.text())) errs.push('de console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)

  // 1) grid
  await page.screenshot({ path: resolve(outDir, 'a1-grid.png'), fullPage: true })
  out.push({ surface: 'de/grid', rawKeys: await rawKeys(page) })

  // 3) Eingänge
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  out.push({ surface: 'de/eingaenge', rawKeys: await rawKeys(page) })
  await page.screenshot({ path: resolve(outDir, 'a3-eingaenge.png'), fullPage: true })

  // 4) Vorlagen
  await page.locator('button:has-text("Vorlagen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  out.push({ surface: 'de/vorlagen', rawKeys: await rawKeys(page) })
  await page.screenshot({ path: resolve(outDir, 'a4-vorlagen.png'), fullPage: true })

  // 5) editor (with notifications section)
  await page.locator('button:has-text("Formulare")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(600)
  await openEditor(page, 'Bewerbungsformular')
  out.push({ surface: 'de/editor', rawKeys: await rawKeys(page) })
  await page.screenshot({ path: resolve(outDir, 'a5-editor.png'), fullPage: true })
  await ctx.close()
}

// ── Part A2 — form detail tabs + submission detail + settings ──
{
  const { ctx, page } = await newPage(browser, 'de')
  page.on('pageerror', (e) => errs.push('de2: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  // form detail modal — all 4 tabs
  await page.locator('[role="button"]:has-text("Kontaktanfrage")').first().click({ timeout: 6000 })
  await page.waitForTimeout(700)
  const dlg = page.locator('[role="dialog"]').last()
  for (const [tab, file] of [['Details', 'a6-detail-details'], ['Eingänge', 'a6-detail-eingaenge'], ['Auswertung', 'a6-detail-auswertung'], ['Verteilung', 'a6-detail-verteilung']]) {
    await dlg.locator(`button:has-text("${tab}")`).first().click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(900)
    out.push({ surface: `de/detail-${tab}`, rawKeys: await rawKeys(page) })
    await page.screenshot({ path: resolve(outDir, `${file}.png`) })
  }
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  // submission detail
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  await page.locator('button:has-text("Kontaktanfrage")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.locator('tr[role="button"]').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  out.push({ surface: 'de/submission-detail', rawKeys: await rawKeys(page) })
  await page.screenshot({ path: resolve(outDir, 'a7-submission-detail.png') })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  // settings panel
  await page.locator('button:has-text("Modul-Einstellungen"), a:has-text("Modul-Einstellungen")').first().click({ timeout: 6000 }).catch(() => {})
  await page.waitForTimeout(1000)
  out.push({ surface: 'de/settings', rawKeys: await rawKeys(page) })
  await page.screenshot({ path: resolve(outDir, 'a8-settings.png'), fullPage: true })
  await ctx.close()
}

// ── Part B — English spot-check of the new surfaces ──
{
  const { ctx, page } = await newPage(browser, 'en')
  page.on('pageerror', (e) => errs.push('en: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  // editor notifications (English)
  await openEditor(page, 'Kontaktanfrage')
  out.push({ surface: 'en/editor', rawKeys: await rawKeys(page) })
  await page.screenshot({ path: resolve(outDir, 'b1-editor-en.png'), fullPage: true })
  await page.locator('button:has-text("Preview")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.keyboard.press('Escape').catch(() => {})
  await ctx.close()
}

// ── Part B2 — English Auswertung time chart ──
{
  const { ctx, page } = await newPage(browser, 'en', 1280, 1100)
  page.on('pageerror', (e) => errs.push('en2: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('[role="button"]:has-text("Kundenfeedback")').first().click({ timeout: 6000 })
  await page.waitForTimeout(700)
  const dlg = page.locator('[role="dialog"]').last()
  await dlg.locator('button:has-text("Evaluation"), button:has-text("Analysis"), button:has-text("Auswertung")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  out.push({ surface: 'en/auswertung', rawKeys: await rawKeys(page), chartCount: await dlg.locator('svg.recharts-surface').count() })
  await page.screenshot({ path: resolve(outDir, 'b2-auswertung-en.png'), fullPage: true })
  await ctx.close()
}

const anyRaw = out.filter((o) => o.rawKeys.length > 0)
console.log(JSON.stringify({ results: out, rawKeySurfaces: anyRaw, pageErrors: errs }, null, 2))
await browser.close()
