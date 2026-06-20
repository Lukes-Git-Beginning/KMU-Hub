/**
 * QA — formulare FT-2a: field validation rules.
 *  1) Field-config dialog (text field) shows the validation section:
 *     min/max length + a pattern-type dropdown (Frei/PLZ/Telefon/IBAN).
 *  2) Number field shows min/max value inputs.
 *  3) Fill preview: submit empty → required errors; pattern (PLZ) rejects a
 *     bad value; a correct fill shows the thank-you state.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-ft2a')
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

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1) Field-config validation section (text) + set PLZ pattern ──
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('config-text: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('config console: ' + m.text()) })
  await openEditor(page, 'Kundenfeedback')
  // first field "Name" (text) → edit
  await page.locator('button[title="Bearbeiten"]').first().click({ force: true, timeout: 5000 })
  await page.waitForTimeout(700)
  const dlg = page.locator('[role="dialog"]').last()
  const dlgText = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '1-fieldconfig-text-validation.png') })
  // choose PLZ pattern
  const sel = dlg.locator('select').last()
  await sel.selectOption('plz').catch(() => {})
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '2-fieldconfig-pattern-plz.png') })
  await dlg.locator('button:has-text("Speichern")').first().click({ timeout: 5000 })
  await page.waitForTimeout(600)
  out.push({
    check: 'fieldconfig-text',
    hasValidationTitle: /Validierung/.test(dlgText || ''),
    hasMinMaxLength: /Min\. Länge/.test(dlgText || '') && /Max\. Länge/.test(dlgText || ''),
    hasPatternFormat: /Format/.test(dlgText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Number field shows min/max value ──
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('config-number: ' + String(e)))
  await openEditor(page, 'Eventanmeldung')
  // "Anzahl Personen" is the number field — edit the 3rd field
  await page.locator('button[title="Bearbeiten"]').nth(2).click({ force: true, timeout: 5000 })
  await page.waitForTimeout(700)
  const dlg = page.locator('[role="dialog"]').last()
  const dlgText = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '3-fieldconfig-number-value.png') })
  out.push({
    check: 'fieldconfig-number',
    hasMinMaxValue: /Min\. Wert/.test(dlgText || '') && /Max\. Wert/.test(dlgText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 3) Preview: required errors + PLZ pattern + thank-you ──
{
  const ctx = await browser.newContext({ viewport: { width: 900, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('preview: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('preview console: ' + m.text()) })
  await openEditor(page, 'Kundenfeedback')
  // set PLZ pattern on the first field so the preview can reject a bad value
  await page.locator('button[title="Bearbeiten"]').first().click({ force: true, timeout: 5000 })
  await page.waitForTimeout(600)
  await page.locator('[role="dialog"]').last().locator('select').last().selectOption('plz').catch(() => {})
  await page.locator('[role="dialog"]').last().locator('button:has-text("Speichern")').first().click({ timeout: 5000 })
  await page.waitForTimeout(500)
  // open preview
  await page.locator('button[title="Vorschau"]').first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  const dlg = page.locator('[role="dialog"]').last()
  // bad PLZ + leave required empty → submit
  await dlg.locator('input[type="text"]').first().fill('abc')
  await dlg.locator('button:has-text("Absenden")').first().click({ timeout: 5000 })
  await page.waitForTimeout(600)
  const errText = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '4-preview-validation-errors.png') })
  // fix everything → thank-you
  await dlg.locator('input[type="text"]').first().fill('12345')
  await dlg.locator('input[type="email"]').first().fill('kunde@example.com')
  await dlg.locator('select').first().selectOption({ index: 1 }).catch(() => {})
  await dlg.locator('input[type="checkbox"]').first().check().catch(() => {})
  await dlg.locator('button:has-text("Absenden")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  const doneText = await page.locator('[role="dialog"]').last().textContent()
  await page.screenshot({ path: resolve(outDir, '5-preview-thankyou.png') })
  out.push({
    check: 'preview-validation',
    plzError: /gültige PLZ/.test(errText || ''),
    requiredError: /Pflichtfeld/.test(errText || ''),
    thankYouShown: /Vielen Dank/.test(doneText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
