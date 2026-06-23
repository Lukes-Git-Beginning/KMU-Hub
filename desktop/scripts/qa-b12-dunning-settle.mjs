// QA B-12 follow-up: Mahnungen-Tab settled state (längerer Wait + Response-Capture).
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b12-finanzen')
await mkdir(outDir, { recursive: true })
const STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, p) => (p === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))

const page = await ctx.newPage()
page.setDefaultTimeout(12000)
const out = { dunningResponses: [] }
page.on('response', (r) => {
  if (r.url().includes('/finance/dunning')) out.dunningResponses.push(`${r.status()} ${r.url().split('/api/v1')[1]}`)
})

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.locator('input[type=email]').waitFor({ state: 'visible', timeout: 20000 })
  await page.locator('input[type=email]').fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3800)
  await page.goto(`${FE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(2500)

  const tab = page.getByRole('button', { name: /Mahnungen/ }).first()
  await tab.click()
  await page.waitForTimeout(7000) // großzügig: Query muss settlen
  await page.screenshot({ path: resolve(outDir, '4b-mahnungen-settled.png'), fullPage: false }).catch(() => {})
  const txt = await page.evaluate(() => document.body.innerText)
  out.stillLoading = /werden geladen|wird geladen|loading/i.test(txt)
  out.emptyStateShown = /Keine Mahnungen|keine.*Mahnung|nicht überfällig|überfällige/i.test(txt)
  out.hasError = /Fehler|error/i.test(txt)
} catch (e) {
  out.scriptError = String(e).slice(0, 200)
} finally {
  await browser.close()
}
console.log(JSON.stringify(out, null, 2))
