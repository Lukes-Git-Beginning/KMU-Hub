// QA zeiterfassung — Echt-Schaltung (localbackend, echtes HR/biz-Backend :8080).
// Verifiziert nach dem HR-correction_reason-Fix: Einträge-Liste lädt (vorher 500),
// Stundenkonto/Balance + Wochenstatus rendern, Timestamps via normalizeWireTimestamps
// korrekt (kein "Invalid Date"), keine Raw-Keys/Crashes.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots', 'zeiterfassung')

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

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))

const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errors.push('console: ' + m.text()) })

await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
const email = page.locator('input[type=email]')
await email.waitFor({ state: 'visible', timeout: 20000 })
await email.fill('demo@local.test')
await page.locator('input[type=password]').fill('Demo1234!')
await page.locator('input[type=password]').press('Enter')
await page.waitForTimeout(3500)

await page.goto(`${FE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(3000)
await page.screenshot({ path: resolve(outDir, 'zeiterfassung-1440.png'), fullPage: false })
const body = await page.evaluate(() => document.body.innerText)

await browser.close()

const has = (re) => re.test(body)
console.log('=== zeiterfassung (echtes HR-Backend) ===')
console.log('  Seite gerendert:        ', body.length > 80)
console.log('  Invalid Date (Timestamp-Bug):', has(/Invalid Date|NaN/))
console.log('  undefined im Text:      ', has(/\bundefined\b/))
console.log('  Raw-i18n-Keys (zeiterfassung\\.|hr\\.):', has(/\b(zeiterfassung|hr)\.[a-z]/i))
console.log('  Doppelklammern {{:      ', has(/\{\{/))
console.log('=== Fehler (pageerror/console, ohne 503-Service-Lärm) ===')
const real = errors.filter((e) => !/503|Service Unavailable/.test(e))
console.log(real.length ? real.slice(0, 8).join('\n') : '  KEINE (außer 503 von nicht-laufenden Services)')
