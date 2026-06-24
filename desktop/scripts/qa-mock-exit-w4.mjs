// QA Welle 4: wiki + helpdesk against the REAL backend (localbackend mode, :5173).
// Verifies the mock-exit renders: wiki article content decodes from base64 to a
// readable TipTap doc, timestamps show as dates (not "Invalid Date"/{seconds}),
// helpdesk KB/stats render, no JS errors / raw i18n keys.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const GW = 'http://localhost:8080'
const outDir = resolve('.qa-screenshots/mock-exit-w4')
await mkdir(outDir, { recursive: true })

// Authenticate against the real backend and inject the tokens via the
// electronAPI.auth.getStoredTokens stub so the renderer's auth store boots
// authenticated (no login form is shown in this dev shell).
const loginRes = await fetch(`${GW}/api/v1/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: 'demo@local.test', password: 'DemoPass123!' }),
})
const loginJson = await loginRes.json()
const ACCESS = loginJson.access_token
const REFRESH = loginJson.refresh_token
console.log('[auth] login status', loginRes.status, 'token?', !!ACCESS)

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const TOKENS = { accessToken: ${JSON.stringify(ACCESS)}, refreshToken: ${JSON.stringify(REFRESH)} }
  const authApi = { getStoredTokens: async () => TOKENS, storeTokens: async () => {}, clearTokens: async () => {} }
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

const page = await ctx.newPage()
page.setDefaultTimeout(15000)
const errors = []
page.on('pageerror', (e) => errors.push('PAGEERR: ' + String(e).slice(0, 160)))
page.on('console', (m) => { if (m.type() === 'error') errors.push('CONSOLE: ' + m.text().slice(0, 160)) })
const shot = (n) => page.screenshot({ path: resolve(outDir, `${n}.png`), fullPage: false }).catch(() => {})
const scan = async (label) => {
  const txt = await page.evaluate(() => document.body.innerText)
  const rawKeys = (txt.match(/\b(wiki|helpdesk|common)\.[a-z][a-zA-Z.]+/g) || []).slice(0, 8)
  const invalidDate = /Invalid Date|NaN|\{"seconds"|seconds":\s*\d/.test(txt)
  const base64Leak = /eyJ0eXBl|eyJ0/.test(txt) // base64 of {"type... leaking into UI
  console.log(`[${label}] rawKeys=${rawKeys.length ? JSON.stringify(rawKeys) : 'none'} invalidDate=${invalidDate} base64Leak=${base64Leak}`)
}

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const email = page.locator('input[type=email]')
  if (await email.count().catch(() => 0)) {
    try {
      await email.waitFor({ state: 'visible', timeout: 5000 })
      await email.fill('demo@local.test')
      await page.locator('input[type=password]').fill('DemoPass123!')
      await page.locator('input[type=password]').press('Enter')
      await page.waitForTimeout(3500)
      console.log('[login] submitted via form')
    } catch { console.log('[login] form present but not submittable, proceeding') }
  } else {
    console.log('[login] no form — already authenticated or dev bypass')
  }
  await shot('00-after-login')

  // --- Wiki ---
  await page.goto(`${FE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await shot('10-wiki-list')
  await scan('wiki-list')
  // open the umlaut article (has real paragraph content) to verify the content
  // base64->object adapter renders a readable TipTap doc in the editor
  const art = page.getByText(/^Umlaut /).first()
  if (await art.count()) {
    await art.click(); await page.waitForTimeout(3000); await shot('11-wiki-article'); await scan('wiki-article')
    const hasUmlaut = await page.evaluate(() => document.body.innerText.includes('Übergänge') || document.body.innerText.includes('Straßen'))
    console.log('[wiki-article] umlaut content visible in DOM:', hasUmlaut)
  } else console.log('[wiki] no umlaut article entry found to open')

  // --- Helpdesk ---
  await page.goto(`${FE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await shot('20-helpdesk')
  await scan('helpdesk')
  // try a KB tab if present
  const kb = page.getByRole('button', { name: /Wissens|Knowledge|KB|Artikel/i }).first()
  if (await kb.count()) { await kb.click(); await page.waitForTimeout(2000); await shot('21-helpdesk-kb'); await scan('helpdesk-kb') }

  console.log('\nJS errors:', errors.length ? errors : 'none')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e).slice(0, 300))
  await shot('99-error')
} finally {
  await browser.close()
}
console.log('Screenshots:', outDir)
