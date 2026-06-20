/**
 * QA — formulare FT-5: settings completion + template management.
 *  1) Module settings (opened on /formulare → formulare panel): personal view
 *     preference + tenant notification email + default thank-you message.
 *  2) "Als Vorlage speichern" action moves a form to the Vorlagen tab.
 *  3) Template detail modal footer: Bearbeiten + Löschen; delete confirm.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-ft5')
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

// ── 1) Settings panel (view pref + notify email + thank-you) ──
{
  const ctx = await browser.newContext({ viewport: { width: 1400, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('settings: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('settings console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('button:has-text("Modul-Einstellungen"), a:has-text("Modul-Einstellungen")').first().click({ timeout: 6000 })
  await page.waitForTimeout(1200)
  // expand both sections if collapsed (click section headers)
  const dlg = page.locator('[role="dialog"]').last()
  const text = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '1-settings-panel.png'), fullPage: true })
  // scroll the tenant section into view for a second shot
  await dlg.locator('text=Standard-Danke-Nachricht').scrollIntoViewIfNeeded().catch(() => {})
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '1b-settings-tenant.png'), fullPage: true })
  out.push({
    check: 'settings',
    hasFormView: /Standard-Ansicht/.test(text || ''),
    hasThankYou: /Standard-Danke-Nachricht/.test(text || ''),
    hasNotifyEmail: /Benachrichtigungs-E-Mail/.test(text || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Save-as-template action moves a form to Vorlagen ──
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('save-template: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  // open the first card's ItemActions menu (the only real <button> in the card)
  const card = page.locator('[role="button"]:has-text("Kundenfeedback")').first()
  await card.locator('button').first().click({ timeout: 6000 })
  await page.waitForTimeout(500)
  await page.locator('[role="menuitem"]:has-text("Als Vorlage speichern")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  // switch to Vorlagen tab and count
  await page.locator('button:has-text("Vorlagen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  const tabText = await page.locator('body').textContent()
  const cardCount = await page.locator('.grid > [role="button"]').count()
  await page.screenshot({ path: resolve(outDir, '2-saved-as-template.png'), fullPage: true })
  out.push({
    check: 'save-as-template',
    templateCount: cardCount,
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 3) Template detail footer (edit + delete) + delete confirm ──
{
  const ctx = await browser.newContext({ viewport: { width: 1100, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('tmpl-delete: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('button:has-text("Vorlagen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  await page.locator('[role="button"]:has-text("Reklamationserfassung")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  const dlg = page.locator('[role="dialog"]').last()
  const footerText = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '3-template-detail-footer.png') })
  // click delete → confirm dialog
  await dlg.locator('button:has-text("Löschen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(600)
  const alert = page.locator('[role="alertdialog"]')
  const alertText = await alert.textContent().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '4-template-delete-confirm.png') })
  out.push({
    check: 'template-delete',
    hasEdit: /Bearbeiten/.test(footerText || ''),
    hasDelete: /Löschen/.test(footerText || ''),
    confirmShown: /Vorlage löschen/.test(alertText || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
