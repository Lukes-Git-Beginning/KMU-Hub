// QA Mock-Exit (work + finanzen): Assessment gegen das echte Backend.
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

const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push('PAGEERR: ' + String(e).slice(0, 120)))
page.on('console', (m) => { if (m.type() === 'error') errors.push('CONSOLE: ' + m.text().slice(0, 120)) })
const shot = (n) => page.screenshot({ path: resolve(outDir, `wf-${n}.png`), fullPage: false }).catch(() => {})

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const email = page.locator('input[type=email]')
  await email.waitFor({ state: 'visible', timeout: 20000 })
  await email.fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3500)

  // work
  await page.goto(`${FE}/#/work`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(3500)
  const workTxt = await page.evaluate(() => document.body.innerText)
  await shot('1-work')

  // finanzen
  await page.goto(`${FE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(3500)
  const finTxt = await page.evaluate(() => document.body.innerText)
  await shot('2-finanzen')

  console.log('\n=== WORK+FINANZEN ASSESSMENT ===')
  console.log('work zeigt Tasks/Projekte:', /landing|launch|aufgabe|projekt|task/i.test(workTxt))
  console.log('work NaN/Invalid Date?:    ', /NaN|Invalid Date/i.test(workTxt))
  console.log('finanzen zeigt RE/Beträge: ', /RE-2026|€|müller|rechnung/i.test(finTxt))
  console.log('finanzen NaN/undefined?:   ', /NaN|undefined|\{\{/i.test(finTxt))
  console.log('errors:', errors.length ? [...new Set(errors)].slice(0, 5).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await shot('error')
} finally {
  await browser.close()
}
