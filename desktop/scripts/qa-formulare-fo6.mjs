/**
 * QA — formulare FO-6: conditional logic applied live in the interactive preview.
 *  Kundenfeedback has a follow-up field "Was genau lief schief?" shown only when
 *  Gesamtbewertung === 'Unzufrieden'.
 *  1) Editor → Vorschau: field hidden initially, appears on 'Unzufrieden', hides again.
 *  2) Eingänge → submission detail: dissatisfied entry shows the follow-up answer;
 *     a satisfied entry hides the field entirely.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-fo6')
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

const browser = await chromium.launch()
const out = []
const errs = []
const FOLLOWUP = 'Was genau lief schief?'

// ── 1) Editor → interactive preview: conditional show/hide ──
{
  const ctx = await browser.newContext({ viewport: { width: 1400, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('preview: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED/.test(m.text())) errs.push('preview console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('[role="button"]:has-text("Kundenfeedback")').first().locator('button').first().click({ timeout: 6000 })
  await page.waitForTimeout(400)
  await page.locator('[role="menuitem"]:has-text("Bearbeiten")').first().click({ timeout: 5000 })
  await page.waitForTimeout(1000)
  await page.locator('button:has-text("Vorschau")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  const dlg = page.locator('[role="dialog"]').last()
  const sel = dlg.locator('select').first()
  const hiddenInitially = await dlg.locator(`label:has-text("${FOLLOWUP}")`).count()
  await page.screenshot({ path: resolve(outDir, '1-preview-hidden.png') })
  await sel.selectOption({ label: 'Unzufrieden' })
  await page.waitForTimeout(600)
  const shownOnUnzufrieden = await dlg.locator(`label:has-text("${FOLLOWUP}")`).count()
  await page.screenshot({ path: resolve(outDir, '2-preview-shown.png') })
  await sel.selectOption({ label: 'Sehr zufrieden' })
  await page.waitForTimeout(600)
  const hiddenAgain = await dlg.locator(`label:has-text("${FOLLOWUP}")`).count()
  out.push({
    check: 'preview-conditional',
    hiddenInitially,
    shownOnUnzufrieden,
    hiddenAgain,
    ok: hiddenInitially === 0 && shownOnUnzufrieden === 1 && hiddenAgain === 0,
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Submission detail: conditional answer shown only for dissatisfied entry ──
{
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NODRAFT)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('detail: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  await page.locator('button:has-text("Kundenfeedback")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  // Thomas Weber = Unzufrieden → follow-up answer visible
  await page.locator('tr[role="button"]:has-text("Thomas Weber")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  let dlg = page.locator('[role="dialog"]').last()
  const dissatText = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '3-detail-dissatisfied.png') })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)
  // Max Mustermann = Sehr zufrieden → follow-up hidden
  await page.locator('tr[role="button"]:has-text("Max Mustermann")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
  dlg = page.locator('[role="dialog"]').last()
  const satText = await dlg.textContent()
  await page.screenshot({ path: resolve(outDir, '4-detail-satisfied.png') })
  out.push({
    check: 'detail-conditional',
    dissatisfiedShowsFollowup: (dissatText || '').includes(FOLLOWUP),
    satisfiedHidesFollowup: !(satText || '').includes(FOLLOWUP),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
