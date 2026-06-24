// QA Dialer Supervisor — Echt-Schaltung (localbackend, kein Mock für /dialer).
// Verifiziert gegen das echte Backend (:8080):
//   A) Seed-Daten → Supervisor voll (KPIs, Agents calls_today, Recent-Calls aus SQL-Fix)
//   B) /dialer/supervisor → {} abgefangen → Leer-Zustand rendert (active_agents=0 statt
//      undefined, recent_calls=[] statt .length-Crash) = Beweis für den Normalizer.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots', 'dialer-supervisor')

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
// Lärm reduzieren (NICHT /dialer mocken — das ist der Prüfgegenstand)
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))

const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errors.push('console: ' + m.text()) })

const login = async () => {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const email = page.locator('input[type=email]')
  await email.waitFor({ state: 'visible', timeout: 20000 })
  await email.fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3500)
}

// ---- Pass A: echte Seed-Daten ----
await login()
await page.goto(`${FE}/#/dialer/supervisor`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForFunction(() => /Nachfass|Demo Local|Demo Review|Termin/i.test(document.body.innerText), { timeout: 25000 }).catch(() => {})
await page.waitForTimeout(1500)
await page.screenshot({ path: resolve(outDir, 'A-supervisor-1440.png'), fullPage: false })
const bodyA = await page.evaluate(() => document.body.innerText)
await ctx.setViewportSize ? null : null
await page.setViewportSize({ width: 1024, height: 900 })
await page.waitForTimeout(600)
await page.screenshot({ path: resolve(outDir, 'A-supervisor-1024.png'), fullPage: false })
await page.setViewportSize({ width: 1440, height: 950 })

// ---- Pass B: leerer Response abgefangen (Normalizer-Crash-Test) ----
await ctx.route('**/api/v1/dialer/supervisor', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: '{}' }))
await page.goto(`${FE}/#/dialer/agent`, { waitUntil: 'domcontentloaded', timeout: 20000 }).catch(() => {})
await page.waitForTimeout(800)
await page.goto(`${FE}/#/dialer/supervisor`, { waitUntil: 'domcontentloaded', timeout: 20000 })
await page.waitForTimeout(2000)
await page.screenshot({ path: resolve(outDir, 'B-supervisor-empty-1440.png'), fullPage: false })
const bodyB = await page.evaluate(() => document.body.innerText)

await browser.close()

// ---- Auswertung ----
const has = (s, re) => re.test(s)
console.log('=== PASS A (Seed-Daten) ===')
console.log('  Kampagne sichtbar:', has(bodyA, /Nachfass/i))
console.log('  Agent sichtbar:   ', has(bodyA, /Demo Local|Demo Review/i))
console.log('  Recent-Call (SQL-Fix):', has(bodyA, /Termin vereinbart|Interesse|Kein Interesse/i))
console.log('  undefined/NaN im Text:', has(bodyA, /undefined|NaN/))
console.log('=== PASS B (leerer Response) ===')
console.log('  Seite gerendert (kein Crash):', bodyB.length > 50)
console.log('  undefined/NaN im Text:', has(bodyB, /undefined|NaN/))
console.log('=== Fehler (pageerror/console) ===')
console.log(errors.length ? errors.slice(0, 8).join('\n') : '  KEINE')
