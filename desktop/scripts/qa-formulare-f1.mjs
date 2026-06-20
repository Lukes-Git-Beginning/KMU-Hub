/**
 * QA — formulare F-1: MSW handler + demo alive.
 * Verifies: forms grid populated, Eingänge groups expand with real submissions,
 * Vorlagen tab populated, 0 pageErrors, 0 console errors, 0 failed /api/v1/formulare
 * requests, no raw i18n keys / {{ }} doubles.
 * Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-f1')
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
page.on('requestfailed', (r) => { if (r.url().includes('/api/v1/formulare')) failedApi.push(`FAILED ${r.method()} ${r.url()} — ${r.failure()?.errorText}`) })
page.on('response', (r) => { if (r.url().includes('/api/v1/formulare') && r.status() >= 400) failedApi.push(`HTTP ${r.status()} ${r.request().method()} ${r.url()}`) })

const out = []
try {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // Forms grid cards
  const formCards = await page.locator('.grid > div').count()
  await page.screenshot({ path: resolve(outDir, '0-forms.png'), fullPage: true })
  out.push({ check: 'forms-tab', formCards, rawKeys: await rawKeys(page) })

  // Stats numbers
  const stats = await page.locator('p.text-2xl').allTextContents().catch(() => [])
  out.push({ check: 'stats', stats })

  // Eingänge tab — expand first two groups
  await page.locator('button:has-text("Eingänge")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1200)
  const groups = await page.locator('button:has(svg) >> text=/Eingänge/').count().catch(() => 0)
  // expand groups
  const groupHeaders = page.locator('.space-y-4 > .rounded-lg > button')
  const gc = await groupHeaders.count()
  for (let i = 0; i < Math.min(gc, 3); i++) {
    await groupHeaders.nth(i).click({ timeout: 3000 }).catch(() => {})
    await page.waitForTimeout(700)
  }
  await page.waitForTimeout(800)
  const submissionRows = await page.locator('table tbody tr').count()
  await page.screenshot({ path: resolve(outDir, '1-eingaenge.png'), fullPage: true })
  out.push({ check: 'eingaenge-tab', groupCount: gc, submissionRows, rawKeys: await rawKeys(page) })

  // Open a submission detail (modal)
  await page.locator('table tbody tr').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const dialogVisible = await page.locator('[role="dialog"]').isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '2-submission-detail.png') })
  out.push({ check: 'submission-detail', dialogVisible, rawKeys: await rawKeys(page) })
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(500)

  // Vorlagen tab
  await page.locator('button:has-text("Vorlagen")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const templateCards = await page.locator('.grid > div').count()
  await page.screenshot({ path: resolve(outDir, '3-vorlagen.png'), fullPage: true })
  out.push({ check: 'vorlagen-tab', templateCards, rawKeys: await rawKeys(page) })
} catch (e) {
  out.push({ fatal: String(e) })
} finally {
  console.log(JSON.stringify({ results: out, pageErrors: errs, failedApi }, null, 2))
  await browser.close()
}
