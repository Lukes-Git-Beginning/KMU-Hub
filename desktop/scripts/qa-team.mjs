// QA team: (1) Lohnvorbereitung-Tab (Lohnlauf: review→freigeben→export),
// (2) Modul-Einstellungen → Team (Persönlich + HR + DATEV-Lohn), keine Raw-Keys.
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
  out.tabs = await page.evaluate(() =>
    [...document.querySelectorAll('button')].map((b) => b.textContent?.trim())
      .filter((t) => t && /Mitarbeiter|Anträge|Abwesenheiten|Korrekturen|Personalakte|Onboarding|Organigramm|Lohnvorbereitung|Schulungen|Self|Modul|Einstellungen|Integrationen/.test(t) && t.length < 24))

  // Lohnvorbereitung tab
  await page.getByRole('button', { name: /Lohnvorbereitung/ }).first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  await page.screenshot({ path: resolve(outDir, 'team-payroll.png'), fullPage: true })
  out.payrollRawKeys = await scanRawKeys(page)
  // approve + export flow
  await page.getByRole('button', { name: /Prüfen & freigeben/ }).first().click({ timeout: 4000 }).catch((e) => { out.approveErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  await page.getByRole('button', { name: /^Export/ }).first().click({ timeout: 4000 }).catch((e) => { out.exportErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1600)
  out.afterExportRawKeys = await scanRawKeys(page)

  // Modul-Einstellungen → Team
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 5000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'team-settings.png'), fullPage: true })
  out.settingsRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 4)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
