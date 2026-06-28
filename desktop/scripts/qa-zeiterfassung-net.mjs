// Capture: welche Requests geben 4xx/5xx auf der zeiterfassung-Seite?
import { chromium } from 'playwright'
const FE = 'http://localhost:5173'
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const SUPPRESS = `try { const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p)) } catch(e){}`
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}`)
const page = await ctx.newPage()
const bad = []
page.on('response', (res) => { const s = res.status(); if (s >= 400) bad.push(`${s} ${res.request().method()} ${res.url()}`) })
await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.locator('input[type=email]').waitFor({ state: 'visible', timeout: 20000 })
await page.locator('input[type=email]').fill('demo@local.test')
await page.locator('input[type=password]').fill('Demo1234!')
await page.locator('input[type=password]').press('Enter')
await page.waitForTimeout(3500)
await page.goto(`${FE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(3500)
// auch Tabs durchklicken (Woche/Auswertungen/Team/Korrekturen) für deren Requests
for (const tab of ['Woche', 'Auswertungen', 'Team', 'Korrekturen']) {
  try { await page.getByRole('tab', { name: tab }).click({ timeout: 2000 }); await page.waitForTimeout(1500) } catch (e) {}
}
await browser.close()
console.log('=== 4xx/5xx Requests (ohne Vite-HMR) ===')
const real = bad.filter((b) => b.includes('/api/'))
console.log(real.length ? [...new Set(real)].join('\n') : '  KEINE API-Fehler')
