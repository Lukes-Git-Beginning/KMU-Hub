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
  return [...new Set([...text.matchAll(/\b(kalender|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
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

  // Open a recurring event (Daily Standup = weekly)
  await page.getByText('Daily Standup').first().click({ timeout: 8000 })
  await page.waitForTimeout(1000)
  // Detail panel should show a translated recurrence label, not a raw key
  out.detailHasRecurrence = await page.getByText(/Wöchentlich|Täglich/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'recurring-detail.png'), fullPage: false })
  out.rawKeysDetail = await scanRawKeys(page)

  // Edit -> Save should trigger the scope dialog
  await page.getByRole('button', { name: /^Bearbeiten$/ }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(800)
  await page.getByRole('button', { name: /^Speichern$/ }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(800)

  out.dialogVisible = await page.getByText('Serientermin bearbeiten').first().isVisible().catch(() => false)
  out.scopeThis = await page.getByText('Nur dieser Termin').first().isVisible().catch(() => false)
  out.scopeFuture = await page.getByText('Dieser und alle folgenden').first().isVisible().catch(() => false)
  out.scopeAll = await page.getByText('Alle Termine der Serie').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'recurring-dialog.png'), fullPage: false })
  out.rawKeysDialog = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close(); await browser.close()
console.log(JSON.stringify(out, null, 2))
