import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte (Reports/BI) — all 4 tabs + drilldown modal + DATEV toggle.
// Run: node scripts/qa-berichte.mjs   (dev server on :5173)
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte')

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
async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(berichte|notifications|wiki|formulare|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function scanDoubleBraces(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\{\{[^}]+\}\}/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

const clickTab = async (re) => {
  await page.getByRole('button', { name: re }).first().click()
  await page.waitForTimeout(1500)
}

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(4000) // allow hero auto-load

  // --- Tab 1: Dashboard ---
  out.kpiCards = await page.locator('.text-2xl.font-semibold').count()
  out.rechartsSurfaces = await page.locator('.recharts-surface').count()
  const dashText = await page.evaluate(() => document.body.innerText)
  out.heroLoaded = !dashText.includes('Noch keine Daten geladen')
  await page.screenshot({ path: resolve(outDir, '1-dashboard.png'), fullPage: true })

  // --- Drilldown modal (KPI click) ---
  await page.getByText('Umsatz (MTD)').first().click()
  await page.waitForTimeout(900)
  const dialog = page.locator('[role="dialog"]')
  out.drilldownOpen = await dialog.isVisible().catch(() => false)
  out.drilldownChart = await dialog.locator('.recharts-surface').count().catch(() => 0)
  await page.screenshot({ path: resolve(outDir, '1b-drilldown.png') })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)

  // --- Tab 2: Erstellen ---
  await clickTab(/^Erstellen/)
  const selects = page.locator('select')
  out.erstellenOptions = await selects.first().locator('option').count()
  const opts = await selects.first().locator('option').all()
  if (opts.length > 1) {
    const val = await opts[1].getAttribute('value')
    if (val) await selects.first().selectOption(val)
  }
  await page.waitForTimeout(400)
  out.generateDisabled = await page.getByRole('button', { name: /Bericht generieren/ }).isDisabled().catch(() => 'n/a')
  await page.screenshot({ path: resolve(outDir, '2-erstellen.png'), fullPage: true })

  // --- Tab 3: Geplant ---
  await clickTab(/^Geplant/)
  out.scheduleRows = await page.locator('table tbody tr').count()
  await page.screenshot({ path: resolve(outDir, '3-geplant.png'), fullPage: true })

  // --- Tab 4: DATEV + variant toggle ---
  await clickTab(/^DATEV$/)
  await page.waitForTimeout(2000)
  out.datevRowsBwa = await page.locator('table tbody tr').count()
  out.datevHeaderBwa = await page.locator('th:has-text("Aktuelles Jahr")').count()
  await page.screenshot({ path: resolve(outDir, '4-datev-bwa.png'), fullPage: true })
  // toggle to SuSa
  await page.getByRole('button', { name: /Summen/ }).first().click()
  await page.waitForTimeout(2000)
  out.datevHeaderSusa = await page.locator('th:has-text("Konto"), th:has-text("Saldo")').count()
  out.datevRowsSusa = await page.locator('table tbody tr').count()
  await page.screenshot({ path: resolve(outDir, '4b-datev-susa.png'), fullPage: true })

  out.rawKeys = await scanRawKeys(page)
  out.doubleBraces = await scanDoubleBraces(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
