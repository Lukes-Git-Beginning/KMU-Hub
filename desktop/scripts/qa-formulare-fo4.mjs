/**
 * QA — formulare FO-4: per-form submission notifications + submitter confirmation.
 *  1) Editor: "Benachrichtigungen" section shows seeded recipient + confirm toggle;
 *     adding a recipient renders a new chip.
 *  2) Interactive preview submit shows the mocked dispatch summary.
 *  3) Eingänge → submission detail shows the "Versendete Benachrichtigungen" block.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-fo4')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
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

async function openEditor(page, formTitle) {
  await page.locator(`[role="button"]:has-text("${formTitle}")`).first().locator('button').first().click({ timeout: 6000 })
  await page.waitForTimeout(400)
  await page.locator('[role="menuitem"]:has-text("Bearbeiten")').first().click({ timeout: 5000 })
  await page.waitForTimeout(1000)
}

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1) Editor notifications section ──
{
  const ctx = await browser.newContext({ viewport: { width: 1400, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('editor: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED/.test(m.text())) errs.push('editor console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await openEditor(page, 'Kontaktanfrage')
  await page.locator('h3:has-text("Benachrichtigungen")').scrollIntoViewIfNeeded()
  await page.waitForTimeout(400)
  const seededChip = await page.locator('span:has-text("vertrieb@zentria.tech")').count()
  await page.screenshot({ path: resolve(outDir, '1-editor-notify.png'), fullPage: true })
  // add a recipient (Enter in the recipient input — avoids matching "Feld hinzufügen")
  const recInput = page.getByPlaceholder('E-Mail-Adresse hinzufügen')
  await recInput.fill('neu@example.com')
  await recInput.press('Enter')
  await page.waitForTimeout(500)
  const addedChip = await page.locator('span:has-text("neu@example.com")').count()
  await page.locator('h3:has-text("Benachrichtigungen")').scrollIntoViewIfNeeded()
  await page.screenshot({ path: resolve(outDir, '2-editor-notify-added.png'), fullPage: true })
  out.push({ check: 'editor-notify', seededChip, addedChip, rawKeys: await rawKeys(page) })
  await ctx.close()
}

// ── 2) Preview submit → dispatch summary ──
{
  const ctx = await browser.newContext({ viewport: { width: 1400, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('preview: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await openEditor(page, 'Kontaktanfrage')
  await page.locator('button:has-text("Vorschau")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  const dlg = page.locator('[role="dialog"]').last()
  // fill all text/email inputs + textareas + check consent
  const textInputs = dlg.locator('input[type="text"], input[type="email"]')
  const ti = await textInputs.count()
  for (let i = 0; i < ti; i++) {
    const inp = textInputs.nth(i)
    const type = await inp.getAttribute('type')
    await inp.fill(type === 'email' ? 'kunde@example.com' : 'Testeingabe')
  }
  const tas = dlg.locator('textarea')
  const ta = await tas.count()
  for (let i = 0; i < ta; i++) await tas.nth(i).fill('Eine Test-Nachricht für die Demo.')
  const cbs = dlg.locator('input[type="checkbox"]')
  const cb = await cbs.count()
  for (let i = 0; i < cb; i++) await cbs.nth(i).check().catch(() => {})
  await dlg.locator('button:has-text("Absenden")').first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  const thankText = (await dlg.textContent()) || ''
  await page.screenshot({ path: resolve(outDir, '3-preview-dispatch.png') })
  out.push({
    check: 'preview-dispatch',
    hasNotifyTitle: thankText.includes('Benachrichtigungen versendet'),
    hasRecipient: thankText.includes('vertrieb@zentria.tech'),
    hasConfirmation: thankText.includes('kunde@example.com'),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 3) Submission detail dispatch record ──
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('detail: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  await page.locator('button:has-text("Kontaktanfrage")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  await page.locator('tr[role="button"]:has-text("Sandra Fischer")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  const dlg = page.locator('[role="dialog"]').last()
  const detailText = (await dlg.textContent()) || ''
  await page.screenshot({ path: resolve(outDir, '4-detail-dispatch.png') })
  out.push({
    check: 'detail-dispatch',
    hasBlock: detailText.includes('Versendete Benachrichtigungen'),
    hasRecipient: detailText.includes('vertrieb@zentria.tech'),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
