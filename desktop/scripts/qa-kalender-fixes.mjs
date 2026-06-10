import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

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
  return [...new Set([...text.matchAll(/\b(kalender|common|settings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
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

try {
  await page.goto(`${BASE}/#/kalender`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3000)

  // Default view (week) — work hours 08-17 + now indicator expected
  out.weekHasGrid = await page.getByText('08:00').first().isVisible().catch(() => false)
  out.weekHas17 = await page.getByText('17:00').first().isVisible().catch(() => false)
  out.weekHas07 = await page.getByText('07:00').first().isVisible().catch(() => false) // should be FALSE now
  await page.screenshot({ path: resolve(outDir, 'kalender-week.png'), fullPage: false })
  out.rawKeysWeek = await scanRawKeys(page)

  // Day view
  await page.getByRole('button', { name: /^Tag$/ }).first().click().catch(() => {})
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'kalender-day.png'), fullPage: false })

  // Month view
  await page.getByRole('button', { name: /^Monat$/ }).first().click().catch(() => {})
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'kalender-month.png'), fullPage: false })

  // Navigate one month back (to a month with a fixed holiday for verification)
  await page.locator('button.p-1\\.5').first().click().catch(() => {})
  await page.waitForTimeout(1000)
  out.monthShowsHoliday = await page.getByText(/Tag der Arbeit|Neujahr|Weihnachtsfeiertag|Heilige Drei|Deutschen Einheit/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'kalender-month-prev.png'), fullPage: false })
  out.rawKeysMonth = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close(); await browser.close()
console.log(JSON.stringify(out, null, 2))
