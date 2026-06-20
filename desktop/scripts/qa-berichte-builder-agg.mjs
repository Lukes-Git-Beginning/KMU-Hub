import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte report-builder E-3 — aggregation switch (sum / avg / count).
// Run: node scripts/qa-berichte-builder-agg.mjs   (dev server on :5173)
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
const yTicks = (page) =>
  page.locator('.recharts-yAxis .recharts-cartesian-axis-tick-value').allInnerTexts()
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
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)
  await page.getByRole('button', { name: /Erstellen/ }).first().click()
  await page.waitForTimeout(1000)
  await page.getByText('Rechnungen', { exact: true }).first().click()
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: /^Status$/ }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('button', { name: /Brutto-Betrag/ }).first().click()
  await page.waitForTimeout(1200)

  out.summarizeShown = (await page.evaluate(() => document.body.innerText)).includes('Zusammenfassen')
  out.ticksSum = await yTicks(page)
  await page.screenshot({ path: resolve(outDir, 'a1-sum.png'), fullPage: true })

  const aggSelect = page.locator('select').first()
  await aggSelect.selectOption({ label: 'Durchschnitt' })
  await page.waitForTimeout(1100)
  out.ticksAvg = await yTicks(page)
  await page.screenshot({ path: resolve(outDir, 'a2-avg.png'), fullPage: true })

  await aggSelect.selectOption({ label: 'Anzahl' })
  await page.waitForTimeout(1100)
  out.ticksCount = await yTicks(page)
  await page.screenshot({ path: resolve(outDir, 'a3-count.png'), fullPage: true })

  out.rawKeys = await scanRawKeys(page)
} catch (e) {
  out.error = String(e)
  await page.screenshot({ path: resolve(outDir, 'a-ERROR.png'), fullPage: true }).catch(() => {})
}

out.pageErrors = errors
console.log(JSON.stringify(out, null, 2))
await browser.close()
