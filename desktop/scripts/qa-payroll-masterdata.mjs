// QA Lohn-Stammdaten: employee detail → Lohn-Stammdaten section (hr_only),
// completeness badge, edit+save flow, then Lohnvorbereitung incomplete warning.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
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
  return [...new Set([...text.matchAll(/\b(team|moduleSettings|settings|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)

  // Open first employee detail panel.
  const card = page.locator('[class*="cursor-pointer"], button, div').filter({ hasText: /Unbekannt|@/ }).first()
  await card.click({ timeout: 5000 }).catch((e) => { out.cardClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)

  // Lohn-Stammdaten section present? (hr_only)
  const section = page.getByText('Lohn-Stammdaten', { exact: false }).first()
  out.sectionVisible = await section.isVisible().catch(() => false)
  if (out.sectionVisible) {
    await section.click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(500)
    out.incompleteBadge = await page.getByText('Unvollständig', { exact: false }).first().isVisible().catch(() => false)
    await page.screenshot({ path: resolve(outDir, 'payroll-master-collapsed.png'), fullPage: true })

    // Edit flow
    await page.getByRole('button', { name: /Bearbeiten/ }).first().click({ timeout: 4000 }).catch((e) => { out.editErr = String(e).split('\n')[0] })
    await page.waitForTimeout(500)
    out.editModeInputs = await page.locator('input[type="date"], input[type="number"]').count()
    await page.screenshot({ path: resolve(outDir, 'payroll-master-edit.png'), fullPage: true })
    // Fill IBAN-ish text field if present, then save.
    await page.getByRole('button', { name: /^Speichern$/ }).first().click({ timeout: 4000 }).catch((e) => { out.saveErr = String(e).split('\n')[0] })
    await page.waitForTimeout(500)
    out.detailRawKeys = await scanRawKeys(page)
  }

  // Close detail panel before switching tabs.
  await page.keyboard.press('Escape')
  await page.waitForTimeout(600)

  // Lohnvorbereitung tab incomplete warning ("N unvollständig" badge).
  await page.getByRole('button', { name: /Lohnvorbereitung/ }).first().click({ timeout: 5000 }).catch((e) => { out.payrollTabErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.payrollIncompleteWarning = await page.getByText(/\d+ unvollständig/i).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'payroll-prep-warning.png'), fullPage: true })
  out.payrollRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 4)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
