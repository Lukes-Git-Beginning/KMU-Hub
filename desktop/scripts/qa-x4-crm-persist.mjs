// QA X-4 settings rollout (crmPrefs) — verifies backend hydration overrides the
// local default. Seeds localStorage cosmi-crm-prefs with DEFAULTS (list/comfortable/
// true), logs in, mounts Kontakte; initFromServer must hydrate the SERVER values
// (grid/compact/false, set via API before this run). Proves load + write-through wiring.
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
// Seed crm prefs with the LOCAL DEFAULTS so a successful hydration is observable.
const SEED_DEFAULT_PREFS = `try { localStorage.setItem('cosmi-crm-prefs', JSON.stringify({ state: { defaultContactView: 'list', density: 'comfortable', showAvatars: true, serverInitialized: false }, version: 0 })) } catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
await ctx.addInitScript(SEED_DEFAULT_PREFS)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}`)
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))

const page = await ctx.newPage()
await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.locator('input[type=email]').waitFor({ state: 'visible', timeout: 20000 })
await page.locator('input[type=email]').fill('demo@local.test')
await page.locator('input[type=password]').fill('Demo1234!')
await page.locator('input[type=password]').press('Enter')
await page.waitForTimeout(3500)

// Read the seeded value BEFORE navigating to Kontakte
const before = await page.evaluate(() => localStorage.getItem('cosmi-crm-prefs'))

await page.goto(`${FE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(3000) // let initFromServer hydrate + persist write back

const after = await page.evaluate(() => localStorage.getItem('cosmi-crm-prefs'))
await browser.close()

const parse = (s) => { try { return JSON.parse(s).state } catch { return {} } }
const b = parse(before), a = parse(after)
console.log('=== X-4 crmPrefs backend hydration ===')
console.log('  vor  Kontakte:', JSON.stringify({ view: b.defaultContactView, density: b.density, avatars: b.showAvatars }))
console.log('  nach Kontakte:', JSON.stringify({ view: a.defaultContactView, density: a.density, avatars: a.showAvatars, serverInit: a.serverInitialized }))
const hydrated = a.defaultContactView === 'grid' && a.density === 'compact' && a.showAvatars === false
console.log('  → Server-Werte hydratisiert (grid/compact/false):', hydrated ? 'JA ✓' : 'NEIN ✗')
