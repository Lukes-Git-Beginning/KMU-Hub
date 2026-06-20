import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte report-builder E-5 — style options (palette, top-N, labels) + dashboard pin.
// Run: node scripts/qa-berichte-builder-style.mjs   (dev server on :5173)
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
const barFill = (page) => page.locator('.recharts-bar-rectangle path').first().getAttribute('fill').catch(() => null)
const barCount = (page) => page.locator('.recharts-bar-rectangle path').count()

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

  // Build: Finanzen → Status + Brutto-Betrag (bar)
  await page.getByText('Rechnungen', { exact: true }).first().click()
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: /^Status$/ }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('button', { name: /Brutto-Betrag/ }).first().click()
  await page.waitForTimeout(1200)
  out.fillDefault = await barFill(page)
  await page.screenshot({ path: resolve(outDir, 'st1-default.png'), fullPage: true })

  // Palette → Ozean (2nd select: agg=0, palette=1)
  const selects = page.locator('select')
  await selects.nth(1).selectOption({ label: 'Ozean' }).catch(() => {})
  await page.waitForTimeout(900)
  out.fillOcean = await barFill(page)
  await page.screenshot({ path: resolve(outDir, 'st2-ocean.png'), fullPage: true })

  // Datenbeschriftung an
  await page.locator('label:has-text("Datenbeschriftung") button[role="switch"]').first().click().catch(() => {})
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, 'st3-labels.png'), fullPage: true })

  // Top-N = 3
  out.barsBefore = await barCount(page)
  await page.getByPlaceholder('Alle').fill('3').catch(() => {})
  await page.waitForTimeout(900)
  out.barsAfterTopN = await barCount(page)
  await page.screenshot({ path: resolve(outDir, 'st4-topn.png'), fullPage: true })

  // Save + pin to dashboard
  await page.getByPlaceholder('Berichtsname…').fill('Top-3 Status')
  await page.getByRole('button', { name: /^Speichern$/ }).first().click()
  await page.waitForTimeout(1200)
  await page.getByRole('button', { name: /Meine Berichte/ }).first().click()
  await page.waitForTimeout(900)
  // open the 3-dot menu on the card (last trigger = library card, not the topbar) and pin
  await page.locator('button[aria-haspopup="menu"]').last().click().catch((e) => (out.menuTriggerErr = String(e)))
  await page.waitForTimeout(600)
  out.menuItems = await page.locator('[role="menuitem"]').allInnerTexts().catch(() => [])
  await page.getByRole('menuitem', { name: /Dashboard/ }).click().catch((e) => (out.pinClickErr = String(e)))
  await page.waitForTimeout(900)
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(300)

  // back to dashboard tab
  await page.getByRole('button', { name: 'Dashboard' }).first().click().catch(() => {})
  await page.waitForTimeout(1800)
  out.dashboardHasPinned = (await page.evaluate(() => document.body.innerText)).includes('Top-3 Status')
  await page.screenshot({ path: resolve(outDir, 'st5-dashboard-pinned.png'), fullPage: true })

  out.rawKeys = await scanRawKeys(page)
} catch (e) {
  out.error = String(e)
  await page.screenshot({ path: resolve(outDir, 'st-ERROR.png'), fullPage: true }).catch(() => {})
}

out.pageErrors = errors
out.failedApiRequests = failed
console.log(JSON.stringify(out, null, 2))
await browser.close()
