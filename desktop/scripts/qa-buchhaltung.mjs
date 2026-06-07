// QA Buchhaltung: (1) Modul-Einstellungen → Buchhaltung (Persönlich/Für-alle),
// (2) Seite ohne Admin-Tabs, Ausgaben + Transaktionen rendern (TanStack).
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
  return [...new Set([...text.matchAll(/\b(finanzen|buchhaltung|moduleSettings|settings|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)

  // Tab bar of the page — admin tabs should be gone
  out.pageTabs = await page.evaluate(() => {
    const bar = document.querySelectorAll('button')
    return [...bar].map((b) => b.textContent?.trim()).filter((t) => t && t.length < 24)
      .filter((t) => /Dashboard|Rechnungen|Angebote|Gutschriften|Ausgaben|Transaktionen|Mahnwesen|Belegkette|Banking|Export|Stammdaten|Integrationen|Einstellungen/.test(t))
  })

  // Ausgaben tab
  await page.getByRole('button', { name: /^Ausgaben/ }).first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  await page.screenshot({ path: resolve(outDir, 'bh-expenses.png') })
  out.expensesRows = await page.evaluate(() => document.querySelectorAll('.grid.grid-cols-\\[1fr_120px_100px_140px_100px_40px\\] > div, [class*="grid-cols"]').length)
  out.expensesRawKeys = await scanRawKeys(page)

  // Transaktionen tab
  await page.getByRole('button', { name: /^Transaktionen/ }).first().click({ timeout: 5000 })
  await page.waitForTimeout(800)
  await page.screenshot({ path: resolve(outDir, 'bh-transactions.png') })
  out.transactionsRawKeys = await scanRawKeys(page)

  // Modul-Einstellungen overlay → Buchhaltung (preselected on /finanzen)
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 5000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'bh-settings.png'), fullPage: true })
  out.settingsText = await page.evaluate(() => document.body.innerText.replace(/\n{2,}/g, '\n').slice(0, 600))
  out.settingsRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 4)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
