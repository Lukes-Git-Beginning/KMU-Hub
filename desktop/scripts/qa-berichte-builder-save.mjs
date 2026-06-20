import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte report-builder E-4 — save, library, open (state restore).
// Run: node scripts/qa-berichte-builder-save.mjs   (dev server on :5173)
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
    ...new Set([...document.body.innerText.matchAll(/\b(berichte|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0])),
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

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)
  await page.getByRole('button', { name: /Erstellen/ }).first().click()
  await page.waitForTimeout(1000)

  // Build a report: CRM → Branche + Deal-Wert
  await page.getByText('Kontakte & Deals', { exact: true }).first().click()
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: /^Branche$/ }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('button', { name: /Deal-Wert/ }).first().click()
  await page.waitForTimeout(1000)

  // Name + save
  await page.getByPlaceholder('Berichtsname…').fill('Umsatz nach Branche')
  await page.waitForTimeout(200)
  await page.getByRole('button', { name: /^Speichern$/ }).first().click()
  await page.waitForTimeout(1200)
  out.afterSaveText = (await page.evaluate(() => document.body.innerText)).includes('Aktualisieren')
  await page.screenshot({ path: resolve(outDir, 's1-saved.png'), fullPage: true })

  // Go to library
  await page.getByRole('button', { name: /Meine Berichte/ }).first().click()
  await page.waitForTimeout(1000)
  out.libraryHasReport = (await page.evaluate(() => document.body.innerText)).includes('Umsatz nach Branche')
  await page.screenshot({ path: resolve(outDir, 's2-library.png'), fullPage: true })

  // Open it back into the builder
  await page.getByText('Umsatz nach Branche', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  const txt = await page.evaluate(() => document.body.innerText)
  out.restoredEditing = txt.includes('Aktualisieren')
  out.restoredName = await page.getByPlaceholder('Berichtsname…').inputValue().catch(() => '')
  out.restoredHasChart = (await page.locator('.recharts-surface').count()) > 0
  await page.screenshot({ path: resolve(outDir, 's3-reopened.png'), fullPage: true })

  out.rawKeys = await scanRawKeys(page)
} catch (e) {
  out.error = String(e)
  await page.screenshot({ path: resolve(outDir, 's-ERROR.png'), fullPage: true }).catch(() => {})
}

out.pageErrors = errors
out.failedApiRequests = failed
console.log(JSON.stringify(out, null, 2))
await browser.close()
