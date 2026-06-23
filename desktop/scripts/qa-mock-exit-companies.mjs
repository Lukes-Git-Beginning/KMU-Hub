// QA Mock-Exit (companies): Firmen-Liste + Detail echt gegen das Backend.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

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
const shot = (n) => page.screenshot({ path: resolve(outDir, `companies-${n}.png`), fullPage: false }).catch(() => {})
const result = { login: false, listSeed: false, detail: false }

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const email = page.locator('input[type=email]')
  await email.waitFor({ state: 'visible', timeout: 20000 })
  await email.fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3500)
  result.login = true

  await page.goto(`${FE}/#/kontakte/firmen`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForFunction(() => /müller|mueller|huber|brandner|söhne|soehne|gmbh|kg/i.test(document.body.innerText), { timeout: 25000 }).catch(() => {})
  await page.waitForTimeout(1000)
  result.listSeed = /müller|mueller|huber|brandner|söhne|gmbh|kg/i.test(await page.evaluate(() => document.body.innerText))
  await shot('1-liste')

  // Erste Firmenzeile öffnen (Tabellenzeile mit GmbH/KG)
  const row = page.locator('tr', { hasText: /gmbh|kg|huber|müller|brandner/i }).first()
  if (await row.count()) {
    await row.click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(1500)
    result.detail = /branche|website|kontakte|umsatz|adresse|domain|\.de|\.at/i.test(await page.evaluate(() => document.body.innerText))
    await shot('2-detail')
  }

  console.log('\n=== COMPANIES-QA ===')
  console.log('Login:           ', result.login)
  console.log('Liste Seed:      ', result.listSeed)
  console.log('Detail geöffnet: ', result.detail)
  console.log('Page errors:     ', errors.length ? errors.slice(0, 4).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await shot('error')
} finally {
  await browser.close()
}
