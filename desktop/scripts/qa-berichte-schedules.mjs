import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte B-3 — schedule create (with alert threshold), toggle, delete,
// next-run column. Run: node scripts/qa-berichte-schedules.mjs  (dev :5173)
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
  await page.waitForTimeout(3000)
  await page.getByRole('button', { name: /^Geplant/ }).first().click()
  await page.waitForTimeout(1200)

  out.nextRunHeader = await page.locator('th:has-text("Nächster Lauf")').count()
  out.rowsBefore = await page.locator('table tbody tr').count()
  out.nextRunCells = await page.locator('table tbody tr td:nth-child(5)').allInnerTexts()

  // --- Create with alert threshold ---
  await page.getByRole('button', { name: /Neuer geplanter Bericht/ }).click()
  await page.waitForTimeout(700)
  const dialog = page.locator('[role="dialog"]')
  const sel = dialog.locator('select')
  const opts = await sel.locator('option').all()
  if (opts.length > 1) {
    const val = await opts[1].getAttribute('value')
    if (val) await sel.selectOption(val)
  }
  await dialog.locator('input[placeholder*="Monatlich"]').fill('QA Alert-Bericht')
  await dialog.locator('input[type="number"]').fill('150000')
  await dialog.locator('input[type="email"]').fill('qa@zentria.tech')
  await dialog.getByRole('button', { name: /^Hinzufügen/ }).click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: resolve(outDir, '3b-create-dialog.png') })
  await dialog.getByRole('button', { name: /^Speichern/ }).click()
  await page.waitForTimeout(1500)

  out.rowsAfterCreate = await page.locator('table tbody tr').count()
  out.bellIcons = await page.locator('svg.lucide-bell').count()
  out.activeInfoAfterCreate = (await page.locator('text=/von .* aktiv/').first().innerText().catch(() => '')) || ''

  // --- Toggle first row off ---
  await page.getByRole('button', { name: /Deaktivieren/ }).first().click().catch(async () => {
    await page.locator('table tbody tr').first().locator('button').nth(0).click()
  })
  await page.waitForTimeout(1200)
  out.activeInfoAfterToggle = (await page.locator('text=/von .* aktiv/').first().innerText().catch(() => '')) || ''
  await page.screenshot({ path: resolve(outDir, '3c-after-create-toggle.png'), fullPage: true })

  // --- Delete last row ---
  const delBtns = page.getByRole('button', { name: 'Löschen', exact: true })
  out.deleteBtns = await delBtns.count()
  await delBtns.last().click()
  await page.waitForTimeout(1500)
  out.rowsAfterDelete = await page.locator('table tbody tr').count()

  const text = await page.evaluate(() => document.body.innerText)
  out.rawKeys = [...new Set([...text.matchAll(/\bberichte\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
