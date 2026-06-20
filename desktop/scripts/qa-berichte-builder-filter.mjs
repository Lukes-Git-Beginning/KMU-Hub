import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte report-builder E-2 — date-range quick picks + filter builder.
// Run: node scripts/qa-berichte-builder-filter.mjs   (dev server on :5173)
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte-builder')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, p) => (p === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`
const scanRawKeys = (page) =>
  page.evaluate(() => [
    ...new Set(
      [...document.body.innerText.matchAll(/\b(berichte|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]),
    ),
  ])

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
const failed = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('requestfailed', (r) => {
  if (r.url().includes('/api/')) failed.push(`${r.method()} ${r.url()}`)
})
const out = {}

async function barTotal(page) {
  // Sum of bar heights as a proxy for "preview changed".
  return page.locator('.recharts-bar-rectangle path').count()
}

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)
  await page.getByRole('button', { name: /Erstellen/ }).first().click()
  await page.waitForTimeout(1000)

  // Source + Status dimension + Brutto measure → bar by status
  await page.getByText('Rechnungen', { exact: true }).first().click()
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: /^Status$/ }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('button', { name: /Brutto-Betrag/ }).first().click()
  await page.waitForTimeout(1000)
  out.barsBaseline = await barTotal(page)
  await page.screenshot({ path: resolve(outDir, 'f1-baseline.png'), fullPage: true })

  // Date range quick pick
  await page.getByRole('button', { name: /Dieses Jahr/ }).first().click()
  await page.waitForTimeout(1000)
  out.barsAfterDateRange = await barTotal(page)
  await page.screenshot({ path: resolve(outDir, 'f2-daterange.png'), fullPage: true })

  // Add a filter (defaults to first filterable field = Status)
  await page.getByRole('button', { name: /Hinzufügen/ }).first().click()
  await page.waitForTimeout(500)
  out.filterRowShown = await page.locator('select').count()
  // Set the enum value select (3rd select: field, operator, value)
  const selects = page.locator('select')
  if ((await selects.count()) >= 3) {
    await selects.nth(2).selectOption({ label: 'Bezahlt' }).catch(() => {})
    await page.waitForTimeout(1000)
  }
  out.barsAfterFilter = await barTotal(page)
  await page.screenshot({ path: resolve(outDir, 'f3-filtered.png'), fullPage: true })

  out.rawKeys = await scanRawKeys(page)
} catch (e) {
  out.error = String(e)
  await page.screenshot({ path: resolve(outDir, 'f-ERROR.png'), fullPage: true }).catch(() => {})
}

out.pageErrors = errors
out.failedApiRequests = failed
console.log(JSON.stringify(out, null, 2))
await browser.close()
