// QA Mock-Exit (Multi-Modul): echter Login gegen :8080, dann nacheinander die
// per CLI übergebenen Routen gegen das echte Backend. Pro Route: Screenshot +
// pageerror-Crashes + fehlgeschlagene /api/v1-Requests + Invalid-Date/Raw-Key-Check.
//
// Aufruf: node scripts/qa-mock-exit-modules.mjs kalender helpdesk wiki
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const routes = process.argv.slice(2)
if (routes.length === 0) {
  console.error('Usage: node scripts/qa-mock-exit-modules.mjs <route> [route...]')
  process.exit(1)
}
const outDir = resolve('.qa-screenshots', 'modules-mock-exit')

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

const perRoute = {}
let currentRoute = 'login'
const errors = []
const failed = []
page.on('pageerror', (e) => { (perRoute[currentRoute] ??= { errors: [], failed: [] }).errors.push(String(e)); errors.push(`[${currentRoute}] ${e}`) })
page.on('response', (res) => {
  if (res.url().includes('/api/v1/') && res.status() >= 400 && !res.url().includes('/notifications')) {
    const line = `${res.status()} ${res.url().replace(/^https?:\/\/[^/]+/, '')}`
    ;(perRoute[currentRoute] ??= { errors: [], failed: [] }).failed.push(line); failed.push(`[${currentRoute}] ${line}`)
  }
})

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const emailField = page.locator('input[type=email]')
  await emailField.waitFor({ state: 'visible', timeout: 20000 })
  await emailField.fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3500)

  for (const route of routes) {
    currentRoute = route
    await page.goto(`${FE}/#/${route}`, { waitUntil: 'domcontentloaded', timeout: 30000 })
    await page.waitForTimeout(2500)
    await page.screenshot({ path: resolve(outDir, `${route}.png`), fullPage: false })
    const body = await page.evaluate(() => document.body.innerText)
    const r = (perRoute[route] ??= { errors: [], failed: [] })
    r.invalidDate = /Invalid Date|NaN:NaN|seconds.*nanos/i.test(body)
    // Raw-i18n-Key-Heuristik: sichtbare Tokens wie "helpdesk.tickets.title"
    r.rawKeys = (body.match(/\b[a-z]+\.[a-z]+\.[a-zA-Z.]{3,}\b/g) || []).filter((k) => !k.includes('.png') && !k.includes('.com')).slice(0, 5)
  }

  console.log('\n=== ERGEBNIS Multi-Modul Mock-Exit ===')
  for (const route of routes) {
    const r = perRoute[route] || { errors: [], failed: [] }
    console.log(`\n[${route}]`)
    console.log('  pageerror:', r.errors.length ? r.errors.slice(0, 2).join(' | ') : 'keine')
    console.log('  failed /api:', r.failed.length ? r.failed.slice(0, 5).join(' | ') : 'keine')
    console.log('  Invalid-Date:', r.invalidDate ? 'JA ⚠' : 'nein')
    console.log('  mögliche Raw-Keys:', r.rawKeys && r.rawKeys.length ? r.rawKeys.join(', ') : 'keine')
  }
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'error.png'), fullPage: false }).catch(() => {})
} finally {
  await browser.close()
}
