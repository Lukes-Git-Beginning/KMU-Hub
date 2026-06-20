/**
 * QA — formulare F-2: rows → DetailModal + eager stats.
 * Verifies: header shows new-submission count > 0 on FIRST load (eager fetch),
 * form card opens a centered DetailModal with meta + fields + action buttons,
 * "Vorschau" opens the standalone preview, submission row opens a centered modal.
 * 0 pageErrors / failed /api/v1/formulare. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-f2')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|\{\{)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim()).slice(0, 10)
  }, RAW_RE.source)
}

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
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

  // Header description should already include a non-zero "neue Eingänge" (eager load)
  const headerDesc = await page.locator('h1, h2').first().locator('xpath=following-sibling::p').first().textContent().catch(() => null)
  const headerAll = await page.locator('p').allTextContents().catch(() => [])
  const neueEingang = headerAll.find((t) => /neue Eingänge/.test(t)) || headerDesc
  const weeklyStat = (await page.locator('p.text-2xl').allTextContents().catch(() => []))
  await page.screenshot({ path: resolve(outDir, '0-forms-eager-stats.png') })
  out.push({ check: 'eager-stats', neueEingang, weeklyStat })

  // Open a form card → DetailModal
  await page.locator('.grid [role="button"]').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const dialog = page.locator('[role="dialog"]')
  const modalTitle = await dialog.locator('h3').first().textContent().catch(() => null)
  const actionButtons = await dialog.locator('button').allTextContents().catch(() => [])
  const fieldRows = await dialog.locator('.divide-y > div').count().catch(() => 0)
  await page.screenshot({ path: resolve(outDir, '1-form-detail.png') })
  out.push({ check: 'form-detail', modalTitle, fieldRows, actionButtons, rawKeys: await rawKeys(page) })

  // Click "Vorschau" → standalone preview modal
  await dialog.locator('button:has-text("Vorschau")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(900)
  const previewInputs = await page.locator('[role="dialog"] input, [role="dialog"] textarea, [role="dialog"] select').count().catch(() => 0)
  await page.screenshot({ path: resolve(outDir, '2-form-preview.png') })
  out.push({ check: 'form-preview', previewInputs, rawKeys: await rawKeys(page) })
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)

  // Eingänge tab → expand groups → submission modal
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const groupHeaders = page.locator('.space-y-4 > .rounded-lg > button')
  const gc = await groupHeaders.count()
  for (let i = 0; i < Math.min(gc, 2); i++) {
    await groupHeaders.nth(i).click({ timeout: 3000 }).catch(() => {})
    await page.waitForTimeout(500)
  }
  await page.locator('table tbody tr').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  const subDialogVisible = await page.locator('[role="dialog"]').isVisible().catch(() => false)
  const subDialogCentered = await page.locator('[role="dialog"].max-w-lg, [role="dialog"][class*="max-w"]').count().catch(() => 0)
  await page.screenshot({ path: resolve(outDir, '3-submission-modal.png') })
  out.push({ check: 'submission-modal', subDialogVisible, subDialogCentered, rawKeys: await rawKeys(page) })
} catch (e) {
  out.push({ fatal: String(e) })
} finally {
  console.log(JSON.stringify({ results: out, pageErrors: errs, failedApi }, null, 2))
  await browser.close()
}
