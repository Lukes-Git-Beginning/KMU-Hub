// QA X-4 central hydrator — verifies useHydrateModuleSettings (DeskEnvironment)
// hydrates MULTIPLE prefs stores from the backend after login. Seeds localStorage
// for crm/finance/wiki with DEFAULTS, logs in (DeskEnvironment mounts → hydrator
// fires), then checks all three carry the SERVER values (set via API beforehand).
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
// Seed all three with LOCAL DEFAULTS so a successful hydration is observable.
const SEED = `try {
  localStorage.setItem('cosmi-crm-prefs', JSON.stringify({ state: { defaultContactView: 'list', density: 'comfortable', showAvatars: true, serverInitialized: false }, version: 0 }))
  localStorage.setItem('cosmi-finance-prefs', JSON.stringify({ state: { startTab: 'last', serverInitialized: false }, version: 0 }))
  localStorage.setItem('cosmi-wiki-prefs', JSON.stringify({ state: { defaultView: 'tree', readingWidth: 'normal', sidebarDefaultOpen: true, serverInitialized: false }, version: 0 }))
} catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
await ctx.addInitScript(SEED)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}`)
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))

const page = await ctx.newPage()
await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.locator('input[type=email]').waitFor({ state: 'visible', timeout: 20000 })
await page.locator('input[type=email]').fill('demo@local.test')
await page.locator('input[type=password]').fill('Demo1234!')
await page.locator('input[type=password]').press('Enter')
await page.waitForTimeout(5000) // login → DeskEnvironment mounts → hydrator fires → persist writes back

const read = async (k) => page.evaluate((key) => { try { return JSON.parse(localStorage.getItem(key)).state } catch { return {} } }, k)
const crm = await read('cosmi-crm-prefs')
const fin = await read('cosmi-finance-prefs')
const wiki = await read('cosmi-wiki-prefs')
await browser.close()

console.log('=== X-4 central hydrator (DeskEnvironment) ===')
console.log('  crm    :', JSON.stringify({ view: crm.defaultContactView, density: crm.density, avatars: crm.showAvatars }), '→', crm.defaultContactView === 'grid' && crm.density === 'compact' && crm.showAvatars === false ? 'OK ✓' : 'FEHLT ✗')
console.log('  finance:', JSON.stringify({ startTab: fin.startTab }), '→', fin.startTab === 'rechnungen' ? 'OK ✓' : 'FEHLT ✗')
console.log('  wiki   :', JSON.stringify({ view: wiki.defaultView, width: wiki.readingWidth, sidebar: wiki.sidebarDefaultOpen }), '→', wiki.defaultView === 'flat' && wiki.readingWidth === 'wide' && wiki.sidebarDefaultOpen === false ? 'OK ✓' : 'FEHLT ✗')
const all = crm.defaultContactView === 'grid' && fin.startTab === 'rechnungen' && wiki.defaultView === 'flat'
console.log('  → Alle drei zentral hydratisiert:', all ? 'JA ✓' : 'NEIN ✗')
