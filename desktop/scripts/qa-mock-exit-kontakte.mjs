// QA Mock-Exit: verifiziert das echt-geschaltete kontakte-Modul gegen das
// LOKALE Backend (Gateway :8080), nicht MSW. Holt ein echtes JWT vom auth-
// Service, injiziert es in den Electron-Auth-Stub (getStoredTokens) und macht
// einen Screenshot der kontakte-Liste. Voraussetzung: Dev-Server (Mode
// localbackend) auf :5173 + Docker-Stack (auth/crm/gateway) up + Seed eingespielt.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const API = 'http://localhost:8080'
const outDir = resolve('.qa-screenshots')

// 1) Echtes JWT vom Backend holen
const loginRes = await fetch(`${API}/api/v1/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: 'demo@local.test', password: 'Demo1234!' }),
})
const login = await loginRes.json()
const accessToken = login.access_token
const refreshToken = login.refresh_token ?? 'dummy-refresh'
if (!accessToken) {
  console.error('LOGIN FAILED:', JSON.stringify(login).slice(0, 300))
  process.exit(1)
}
console.log('login ok, token len', accessToken.length)

// 2) Electron-Stub mit echtem Auth-Token
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = {
    getStoredTokens: async () => ({ accessToken: ${JSON.stringify(accessToken)}, refreshToken: ${JSON.stringify(refreshToken)} }),
    storeTokens: async () => {},
    clearTokens: async () => {},
  }
  window.electronAPI = new Proxy({}, {
    get: (_t, prop) => {
      if (prop === 'auth') return authApi
      if (prop === 'then') return undefined
      return fallback[prop]
    },
  })
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
// CosmiLaunch-Splash überspringen (sonst hängt die kontakte-View dahinter):
// prefers-reduced-motion (Context) ODER das gespielt-Flag verhindern Fall A.
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
// Noise-Endpoints abfangen: notification/hr-Services sind im Minimal-Stack
// nicht gestartet -> ihre 503/403-Retry-Flut blockiert sonst die Connection-
// Slots und hungert /contacts aus. contacts/companies/auth bleiben echt.
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))
await ctx.route('**/api/v1/hr/**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: '{}' }))
const page = await ctx.newPage()
const errors = []
const apiCalls = []
const contactsBodies = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error' || m.type() === 'warning') errors.push(`[console.${m.type()}] ${m.text().slice(0, 160)}`) })
page.on('response', async (r) => {
  if (r.url().includes('/api/v1/')) apiCalls.push(`${r.status()} ${r.url().replace(API, '')}`)
  if (r.url().includes('/api/v1/contacts')) {
    try { contactsBodies.push(`${r.status()} -> ${(await r.text()).slice(0, 200)}`) } catch (e) { contactsBodies.push(`${r.status()} (body unreadable)`) }
  }
})

try {
  await page.goto(`${FE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  // aktiv warten bis die echten Kontakte gerendert sind (Backend-Call kommt spät,
  // weil notification/hr-Retries die Connection-Slots fluten)
  await page.waitForFunction(() => /Müller|Huber|Vogel|Brandner/.test(document.body.innerText), { timeout: 25000 }).catch(() => {})
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'mock-exit-kontakte.png'), fullPage: false })

  const bodyText = await page.evaluate(() => document.body.innerText)
  const sawSeedData = /Müller|Huber|Vogel|Brandner/.test(bodyText)
  const rawKeys = [...new Set([...bodyText.matchAll(/\b(kontakte|common|contacts)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]

  console.log('\n=== ERGEBNIS ===')
  console.log('Seed-Daten sichtbar (Müller/Huber/Vogel/Brandner):', sawSeedData)
  console.log('Raw i18n keys:', rawKeys.length ? rawKeys.join(', ') : 'keine')
  console.log('Page errors:', errors.length ? errors.slice(0, 3).join(' | ') : 'keine')
  console.log('--- /contacts Calls + Body ---')
  console.log('  ' + (contactsBodies.length ? contactsBodies.join('\n  ') : 'KEIN /contacts-Call!'))
  console.log('--- alle API-Calls an :8080 ---')
  console.log('  ' + apiCalls.join('\n  '))
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'mock-exit-kontakte-error.png'), fullPage: false }).catch(() => {})
} finally {
  await browser.close()
}
