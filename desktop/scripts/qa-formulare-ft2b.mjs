/**
 * QA — formulare FT-2b: rating field, page titles, DACH templates, thank-you.
 *  1) Vorlagen tab shows the 5 new DACH templates; Lieferantenbewertung detail
 *     renders rating fields.
 *  2) Builder: add a Bewertung field → stars in preview; config offers 5/10
 *     scale; NPS scale renders number buttons.
 *  3) Page title input on a page break; thank-you config → custom message in
 *     the fill preview.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-ft2b')
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

async function openEditor(page, formName) {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator(`[role="button"]:has-text("${formName}")`).first().click({ timeout: 6000 })
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"]').last().locator('button:has-text("Bearbeiten")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
}

async function addField(page, label) {
  await page.locator('button:has-text("Feld hinzufügen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(400)
  await page.locator(`button:has-text("${label}")`).last().click({ timeout: 5000 })
  await page.waitForTimeout(500)
}

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1) DACH templates + rating template detail ──
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('templates: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('templates console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('button:has-text("Vorlagen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  const tabText = await page.locator('body').textContent()
  await page.screenshot({ path: resolve(outDir, '1-dach-templates.png'), fullPage: true })
  await page.locator('[role="button"]:has-text("Lieferantenbewertung")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  const detailText = await page.locator('[role="dialog"]').last().textContent()
  await page.screenshot({ path: resolve(outDir, '2-lieferanten-template-detail.png') })
  out.push({
    check: 'dach-templates',
    hasReklamation: /Reklamationserfassung/.test(tabText || ''),
    hasUrlaub: /Urlaubsantrag/.test(tabText || ''),
    hasAufmass: /Aufmaßerfassung/.test(tabText || ''),
    hasSchaden: /Schadensanzeige/.test(tabText || ''),
    detailHasRating: /Liefertreue|Weiterempfehlung/.test(detailText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Rating field in builder + config scale + NPS ──
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('rating: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('rating console: ' + m.text()) })
  await openEditor(page, 'Kundenfeedback')
  await addField(page, 'Bewertung')
  await page.screenshot({ path: resolve(outDir, '3-rating-stars-preview.png') })
  // edit the new rating field (last edit button) → scale selector
  await page.locator('button[title="Bearbeiten"]').last().click({ force: true, timeout: 5000 })
  await page.waitForTimeout(600)
  const cfgDlg = page.locator('[role="dialog"]').last()
  const cfgText = await cfgDlg.textContent()
  await page.screenshot({ path: resolve(outDir, '4-rating-scale-config.png') })
  await cfgDlg.locator('button:has-text("10 (NPS)")').first().click({ timeout: 5000 })
  await cfgDlg.locator('button:has-text("Speichern")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '5-rating-nps-preview.png') })
  out.push({
    check: 'rating-builder',
    configHasScale: /Skala/.test(cfgText || ''),
    configHasStars: /5 Sterne/.test(cfgText || ''),
    configHasNps: /NPS/.test(cfgText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 3) Page title + thank-you config + preview ──
{
  const ctx = await browser.newContext({ viewport: { width: 980, height: 1200 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('thankyou: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('thankyou console: ' + m.text()) })
  await openEditor(page, 'Kundenfeedback')
  // page break → title input
  await addField(page, 'Seitenumbruch')
  await page.locator('input[placeholder*="Seitentitel"]').first().fill('Ihre Kontaktdaten')
  await page.waitForTimeout(300)
  // thank-you message
  await page.locator('textarea[placeholder*="Vielen Dank"]').first().fill('Danke für Ihr Feedback! Wir melden uns in Kürze.')
  await page.waitForTimeout(300)
  await page.screenshot({ path: resolve(outDir, '6-pagetitle-thankyou-config.png'), fullPage: true })
  // open preview, fill required, submit → custom thank-you
  await page.locator('button[title="Vorschau"]').first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  const dlg = page.locator('[role="dialog"]').last()
  const previewText = await dlg.textContent()
  await dlg.locator('input[type="email"]').first().fill('kunde@example.com')
  await dlg.locator('select').first().selectOption({ index: 1 }).catch(() => {})
  await dlg.locator('input[type="checkbox"]').first().check().catch(() => {})
  await dlg.locator('button:has-text("Absenden")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  const doneText = await page.locator('[role="dialog"]').last().textContent()
  await page.screenshot({ path: resolve(outDir, '7-custom-thankyou.png') })
  out.push({
    check: 'pagetitle-thankyou',
    pageTitleInPreview: /Ihre Kontaktdaten/.test(previewText || ''),
    customThankYou: /Danke für Ihr Feedback/.test(doneText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
