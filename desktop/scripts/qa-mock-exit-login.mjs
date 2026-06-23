// QA Mock-Exit (Login-Flow): verifiziert, dass mit RENDERER_VITE_DEV_BYPASS_AUTH=false
// der echte Login-Screen erscheint, der Login gegen das echte Backend läuft und
// danach kontakte echte Daten zeigt. Kein Token-Inject — voller Flow wie ein User.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const API = 'http://localhost:8080'
const outDir = resolve('.qa-screenshots')

// Electron-Stub: getStoredTokens -> null, damit der echte Login-Screen kommt.
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
await ctx.route('**/api/v1/hr/**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: '{}' }))
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  // Login-Screen abwarten
  const emailField = page.locator('input[type=email]')
  await emailField.waitFor({ state: 'visible', timeout: 20000 })
  await page.screenshot({ path: resolve(outDir, 'mock-exit-login-1-screen.png'), fullPage: false })
  const sawLogin = await emailField.isVisible()

  // Echter Login
  await emailField.fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')

  // Auf eingeloggten Zustand warten + zu kontakte
  await page.waitForTimeout(3500)
  await page.goto(`${FE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForFunction(() => /mueller|huber|vogel|brandner/i.test(document.body.innerText), { timeout: 25000 }).catch(() => {})
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'mock-exit-login-2-kontakte.png'), fullPage: false })

  const bodyText = await page.evaluate(() => document.body.innerText)
  const sawSeed = /mueller|huber|vogel|brandner/i.test(bodyText)
  console.log('\n=== ERGEBNIS ===')
  console.log('Login-Screen erschienen:', sawLogin)
  console.log('Kontakte-Seed-Daten sichtbar:', sawSeed)
  console.log('Page errors:', errors.length ? errors.slice(0, 3).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'mock-exit-login-error.png'), fullPage: false }).catch(() => {})
} finally {
  await browser.close()
}
