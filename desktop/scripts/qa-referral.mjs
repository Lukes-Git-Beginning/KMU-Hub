// QA P8-2: Empfohlen-von-Feld im Kontakt-Detail + Empfehler-Report in Auswertungen.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const CONTACT = process.argv[2] || 'Karl Bauer'

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
  return [...new Set([...text.matchAll(/\b(advisory|kontakte|common|crm)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
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
  // Detail panel: pick a referrer
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.getByText(CONTACT, { exact: false }).first().click({ timeout: 5000 })
  await page.waitForTimeout(1200)
  await page.getByRole('button', { name: /Empfehler wählen/i }).click({ timeout: 5000 })
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, 'referral-picker.png') })
  // Choose first listed contact
  const opts = page.locator('div[role="presentation"], [data-radix-popper-content-wrapper]').locator('button')
  const count = await opts.count()
  out.pickerOptions = count
  if (count > 0) { await opts.nth(0).click({ timeout: 3000 }).catch((e) => { out.pickErr = String(e).split('\n')[0] }) }
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, 'referral-detail.png') })
  out.detailRawKeys = await scanRawKeys(page)

  // Auswertungen report
  await page.goto(`${BASE}/#/kontakte/auswertungen`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.getByText(/Top-Empfehler/i).first().scrollIntoViewIfNeeded().catch(() => {})
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, 'referral-report.png') })
  out.reportText = await page.evaluate(() => {
    const cards = [...document.querySelectorAll('div')].filter((d) => d.textContent?.includes('Top-Empfehler'))
    return cards.length ? cards[cards.length - 1].innerText.slice(0, 200) : 'card-not-found'
  })
  out.reportRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 3)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
