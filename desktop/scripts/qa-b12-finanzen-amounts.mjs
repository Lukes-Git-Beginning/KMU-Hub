// QA B-12: finanzen Betrag-Spalte gegen echtes Backend (localbackend mode, :5173).
// Verifiziert, dass Rechnungen/Angebote echte gross_total-Beträge zeigen (nicht 0,00 €)
// und Gutschriften/Mahnungen ohne Fehler rendern. tax_breakdown-Fallback (biz) im Test.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b12-finanzen')
await mkdir(outDir, { recursive: true })

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
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

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))

const page = await ctx.newPage()
page.setDefaultTimeout(12000)
const errors = []
page.on('pageerror', (e) => errors.push('PAGEERR: ' + String(e).slice(0, 140)))
page.on('console', (m) => { if (m.type() === 'error') errors.push('CONSOLE: ' + m.text().slice(0, 140)) })
const shot = (n) => page.screenshot({ path: resolve(outDir, `${n}.png`), fullPage: false }).catch(() => {})
const body = () => page.evaluate(() => document.body.innerText)
const out = {}

async function openTab(re) {
  const tab = page.getByRole('button', { name: re }).first()
  if (await tab.count()) { await tab.click(); await page.waitForTimeout(2000); return true }
  const alt = page.getByText(re).first()
  if (await alt.count()) { await alt.click(); await page.waitForTimeout(2000); return true }
  return false
}

try {
  // --- Login (echtes Backend) ---
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const email = page.locator('input[type=email]')
  await email.waitFor({ state: 'visible', timeout: 20000 })
  await email.fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3800)

  // --- Finanzen ---
  await page.goto(`${FE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(3000)

  // Rechnungen
  out.invoicesTabOpened = await openTab(/Rechnungen/)
  await shot('1-rechnungen')
  const invTxt = await body()
  out.invoiceAmounts = {
    'RE-001 (1.785,00)': invTxt.includes('1.785,00'),
    'RE-002 (1.035,30)': invTxt.includes('1.035,30'),
    'RE-003 (3.451,00)': invTxt.includes('3.451,00'),
  }
  out.invoicesShowZero = /Müller|Bäcker|Elektro/i.test(invTxt) && invTxt.includes('0,00 €') &&
    !invTxt.includes('1.785,00')

  // Angebote
  out.quotesTabOpened = await openTab(/Angebote/)
  await shot('2-angebote')
  const qTxt = await body()
  out.quoteAmounts = {
    'AN-001 (26.180,00)': qTxt.includes('26.180,00'),
    'AN-002 (11.662,00)': qTxt.includes('11.662,00'),
  }

  // Gutschriften (leer erwartet, darf nicht crashen)
  out.creditNotesTabOpened = await openTab(/Gutschriften/)
  await shot('3-gutschriften')
  const cnTxt = await body()
  out.creditNotesRendered = /Gutschrift|keine|Keine/i.test(cnTxt) && !/Fehler|error/i.test(cnTxt)

  // Mahnungen (leer erwartet, darf nicht crashen)
  out.dunningTabOpened = await openTab(/Mahnungen/)
  await shot('4-mahnungen')
  const dTxt = await body()
  out.dunningRendered = /Mahnung|keine|Keine/i.test(dTxt) && !/Fehler|error/i.test(dTxt)

  out.noNaNorRaw = !/NaN|undefined|\{\{/.test(invTxt + qTxt + cnTxt + dTxt)
  out.errors = errors.length ? [...new Set(errors)].slice(0, 6) : 'keine'
} catch (e) {
  out.scriptError = String(e).slice(0, 200)
  await shot('error')
} finally {
  await browser.close()
}
console.log(JSON.stringify(out, null, 2))
