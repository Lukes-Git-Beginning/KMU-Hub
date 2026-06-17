/**
 * QA — vertraege Darien-Fixes F1-F3
 *  F1: no "Unterschrift" footer button + no signer section in detail modal
 *  F2: ContractDialog (new/edit) is a centered DetailModal (gradient stripe)
 *  F3: clicking "Bearbeiten" closes the detail modal, opens edit dialog (no stacking)
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/vertraege-fixes')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(vertraege\.[a-z]|common\.[a-z])/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('[role="dialog"] *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim()).slice(0, 8)
  }, RAW_RE.source)
}

const browser = await chromium.launch()
const out = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage(); const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

try {
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.screenshot({ path: resolve(outDir, '0-page.png') })

  // open first contract row → detail modal (JS-click fallback for flaky table rows)
  const rowCount = await page.locator('tbody tr').count()
  const row = page.locator('tbody tr').first()
  await row.evaluate((el) => el.click()).catch(async () => { await row.click({ timeout: 5000 }) })
  await page.waitForTimeout(1000)
  out.push({ check: 'row-count', rowCount })
  const dialogOpen = await page.locator('[role="dialog"]').isVisible().catch(() => false)
  const detailText = await page.evaluate(() => document.querySelector('[role="dialog"]')?.textContent || '')
  await page.screenshot({ path: resolve(outDir, '1-detail-no-signature.png') })

  // F1: footer should NOT contain a "Unterschrift" button; body no signer section
  const hasSignatureBtn = await page.locator('[role="dialog"] button:has-text("Unterschrift")').count()
  const hasSignerSection = /Unterschriften|Signaturen/.test(detailText)
  const hasEditBtn = await page.locator('[role="dialog"] button:has-text("Bearbeiten")').count()
  out.push({ check: 'F1-no-signature', dialogOpen, signatureButtons: hasSignatureBtn, signerSectionInBody: hasSignerSection, editButtons: hasEditBtn, rawKeys: await rawKeys(page) })

  // F3: click Bearbeiten → detail closes, edit dialog opens
  await page.locator('[role="dialog"] button:has-text("Bearbeiten")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  const editTitle = await page.evaluate(() => document.querySelector('[role="dialog"] h3')?.textContent || '')
  const dialogCount = await page.locator('[role="dialog"]').count()
  const hasGradientStripe = await page.locator('[role="dialog"] .bg-gradient-to-r').count()
  await page.screenshot({ path: resolve(outDir, '2-edit-modal.png') })
  out.push({ check: 'F2F3-edit-modal', dialogCount, editTitle, gradientStripe: hasGradientStripe > 0, rawKeys: await rawKeys(page) })

  // close, then F2: "Vertrag anlegen" → centered modal
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)
  const newBtn = page.locator('button:has-text("Vertrag anlegen")').first()
  if (await newBtn.isVisible().catch(() => false)) {
    await newBtn.click({ timeout: 5000 })
    await page.waitForTimeout(900)
    const newTitle = await page.evaluate(() => document.querySelector('[role="dialog"] h3')?.textContent || '')
    const stripe = await page.locator('[role="dialog"] .bg-gradient-to-r').count()
    const numberPrefilled = await page.evaluate(() => {
      const inputs = Array.from(document.querySelectorAll('[role="dialog"] input'))
      return inputs.some((i) => /^V-\d{4}-\d{3}$/.test(i.value || ''))
    })
    await page.screenshot({ path: resolve(outDir, '3-new-modal.png') })
    out.push({ check: 'F2-new-modal', newTitle, gradientStripe: stripe > 0, numberPrefilled })
  }
  out.push({ pageErrors: errs.slice(0, 3) })
} catch (e) { out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 3) }) }
finally { await ctx.close(); await browser.close() }

console.log(JSON.stringify(out, null, 2))
