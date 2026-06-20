import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte report-builder (E-1) — source picker, field picker, live preview, viz switcher.
// Run: node scripts/qa-berichte-builder.mjs   (dev server on :5173)
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
async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(berichte|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
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
const failed = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('requestfailed', (r) => {
  const u = r.url()
  if (u.includes('/api/')) failed.push(`${r.method()} ${u} — ${r.failure()?.errorText}`)
})
const out = {}

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)

  // --- Open "Erstellen" tab ---
  await page.getByRole('button', { name: /Erstellen/ }).first().click()
  await page.waitForTimeout(1200)
  out.emptyState = (await page.evaluate(() => document.body.innerText)).includes('Datenquelle')
  await page.screenshot({ path: resolve(outDir, '1-empty.png'), fullPage: true })

  // --- Pick source: Rechnungen (finanzen) ---
  await page.getByText('Rechnungen', { exact: true }).first().click()
  await page.waitForTimeout(800)
  out.fieldPickerShown = (await page.evaluate(() => document.body.innerText)).includes('Gruppieren nach')
  await page.screenshot({ path: resolve(outDir, '2-source-picked.png'), fullPage: true })

  // --- Pick dimension (Rechnungsdatum) + measure (Brutto-Betrag) ---
  await page.getByRole('button', { name: /Rechnungsdatum/ }).first().click()
  await page.waitForTimeout(400)
  await page.getByRole('button', { name: /Brutto-Betrag/ }).first().click()
  await page.waitForTimeout(1200)
  out.rechartsAfterPick = await page.locator('.recharts-surface').count()
  await page.screenshot({ path: resolve(outDir, '3-autoviz-line.png'), fullPage: true })

  // --- Switch through visualizations ---
  const vizShots = [
    ['Balken', '4-bar'],
    ['Donut', '5-donut'],
    ['Tabelle', '6-table'],
    ['Kennzahl', '7-kpi'],
    ['Tacho', '8-gauge'],
  ]
  for (const [name, file] of vizShots) {
    await page.getByRole('button', { name: new RegExp(`^${name}$`) }).first().click()
    await page.waitForTimeout(900)
    await page.screenshot({ path: resolve(outDir, `${file}.png`), fullPage: true })
  }

  // --- Combo needs a 2nd measure (Netto-Betrag) ---
  await page.getByRole('button', { name: /Netto-Betrag/ }).first().click()
  await page.waitForTimeout(500)
  await page.getByRole('button', { name: /^Kombi$/ }).first().click()
  await page.waitForTimeout(900)
  await page.screenshot({ path: resolve(outDir, '9-combo.png'), fullPage: true })

  // --- Switch source to CRM to confirm registry works ---
  await page.getByText('Kontakte & Deals', { exact: true }).first().click()
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: /Pipeline-Stufe/ }).first().click()
  await page.waitForTimeout(400)
  await page.getByRole('button', { name: /Deal-Wert/ }).first().click()
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, '10-crm-bar.png'), fullPage: true })

  out.rawKeys = await scanRawKeys(page)
  out.doubleBraces = await scanDoubleBraces(page)
} catch (e) {
  out.error = String(e)
  await page.screenshot({ path: resolve(outDir, 'ERROR.png'), fullPage: true }).catch(() => {})
}

out.pageErrors = errors
out.failedApiRequests = failed
console.log(JSON.stringify(out, null, 2))
await browser.close()
