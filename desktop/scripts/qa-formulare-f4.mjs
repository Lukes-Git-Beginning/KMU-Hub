/**
 * QA — formulare F-4: DSGVO consent + submission depth + real export.
 * Verifies: consent field shows in form detail + preview, submission detail
 * surfaces the consent confirmation (date) + IP, the per-form export triggers a
 * real file download, and adding a consent field reveals the consent config.
 * 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-f4')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
const failedApi = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })
page.on('requestfailed', (r) => { if (r.url().includes('/api/v1/formulare')) failedApi.push(`FAILED ${r.url()}`) })
page.on('response', (r) => { if (r.url().includes('/api/v1/formulare') && r.status() >= 400) failedApi.push(`HTTP ${r.status()} ${r.url()}`) })

const out = []
try {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4000)

  // Form detail → consent field present
  await page.locator('.grid [role="button"]:has-text("Kontaktanfrage")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  const detailText = await page.locator('[role="dialog"]').first().textContent().catch(() => '')
  const hasConsentField = /Datenschutz-Einwilligung/.test(detailText || '') && /Einwilligung/.test(detailText || '')
  await page.screenshot({ path: resolve(outDir, '0-form-detail-consent.png') })
  out.push({ check: 'consent-in-detail', hasConsentField })
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)

  // Eingänge → expand → submission detail consent block
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const groupHeaders = page.locator('.space-y-4 > .rounded-lg button:has(svg)')
  await groupHeaders.first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.locator('table tbody tr').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  const subText = await page.locator('[role="dialog"]').first().textContent().catch(() => '')
  const hasConsentBlock = /Eingewilligt am/.test(subText || '')
  const hasIp = /IP-Adresse/.test(subText || '')
  await page.screenshot({ path: resolve(outDir, '1-submission-consent.png') })
  out.push({ check: 'submission-consent', hasConsentBlock, hasIp })
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)

  // Real export download from the group header
  let downloadName = null
  const downloadPromise = page.waitForEvent('download', { timeout: 6000 }).catch(() => null)
  await page.locator('.space-y-4 > .rounded-lg').first().locator('button:has-text("Exportieren")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(400)
  await page.locator('button:has-text("CSV")').first().click({ timeout: 4000 }).catch(() => {})
  const dl = await downloadPromise
  if (dl) downloadName = dl.suggestedFilename()
  await page.screenshot({ path: resolve(outDir, '2-after-export.png') })
  out.push({ check: 'export-download', downloadName, downloaded: !!dl })

  // Builder: add a consent field → config shows consent inputs
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 3000 }).catch(() => {})
  await page.locator('button:has-text("Meine Formulare")').first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(500)
  await page.locator('.grid [role="button"]:has-text("Newsletter")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"] button:has-text("Bearbeiten")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  await page.locator('button:has-text("Feld hinzufügen")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(400)
  await page.locator('button:has-text("Einwilligung")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '3-builder-consent.png'), fullPage: true })
  const editorText = await page.locator('body').textContent().catch(() => '')
  out.push({ check: 'builder-add-consent', hasConsentRow: /Datenschutz-Einwilligung/.test(editorText || '') })
} catch (e) {
  out.push({ fatal: String(e) })
} finally {
  console.log(JSON.stringify({ results: out, pageErrors: errs, failedApi }, null, 2))
  await browser.close()
}
