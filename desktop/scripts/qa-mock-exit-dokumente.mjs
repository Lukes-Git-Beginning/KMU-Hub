// QA Mock-Exit (dokumente): echter Login gegen :8080, dann /dokumente gegen das
// echte document-Backend. Prüft, dass die Ordner (Meine Dateien/Bilder/Dokumente,
// via initialize-user geseedet) rendern, keine pageerror-Crashes auftreten und
// keine rohen {seconds,nanos}-Timestamps / "Invalid Date" sichtbar sind.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots', 'dokumente-mock-exit')

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
const failedReqs = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('response', (res) => {
  const u = res.url()
  if (u.includes('/api/v1/documents/') && res.status() >= 400) failedReqs.push(`${res.status()} ${u.replace(/^https?:\/\/[^/]+/, '')}`)
})

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const emailField = page.locator('input[type=email]')
  await emailField.waitFor({ state: 'visible', timeout: 20000 })
  await emailField.fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3500)

  await page.goto(`${FE}/#/dokumente`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForFunction(() => /Meine Dateien|Bilder|Dokumente|Ordner|Datei/i.test(document.body.innerText), { timeout: 25000 }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, '1-dokumente-list.png'), fullPage: false })

  const bodyText = await page.evaluate(() => document.body.innerText)
  const sawFolders = /Meine Dateien|Bilder|Dokumente/i.test(bodyText)
  const sawInvalidDate = /Invalid Date|NaN|seconds.*nanos/i.test(bodyText)

  console.log('\n=== ERGEBNIS dokumente Mock-Exit ===')
  console.log('Ordner sichtbar (Meine Dateien/Bilder/Dokumente):', sawFolders)
  console.log('Invalid Date / NaN / rohe Timestamps:', sawInvalidDate)
  console.log('Fehlgeschlagene /documents-Requests:', failedReqs.length ? failedReqs.slice(0, 6).join(' | ') : 'keine')
  console.log('Page errors:', errors.length ? errors.slice(0, 3).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'error.png'), fullPage: false }).catch(() => {})
} finally {
  await browser.close()
}
