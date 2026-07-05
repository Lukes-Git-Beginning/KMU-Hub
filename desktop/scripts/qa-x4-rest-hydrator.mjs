// QA X-4 rest hydrator — verifies useHydrateModuleSettings hydrates the 6 new
// stores (workPrefs, vertraegePrefs [user] + financeTenant, wikiSettings,
// dashboardSettings, zeiterfassungSettings [tenant]) from the backend after login.
// Sets NON-DEFAULT server values via API, seeds localStorage with DEFAULTS, logs
// in (DeskEnvironment mounts → hydrator fires), then checks each store carries the
// SERVER value.
import { chromium } from 'playwright'

const API = 'http://localhost:8080'
const FE = 'http://localhost:5173'

const idem = () => `qa-${Date.now()}-${Math.random().toString(36).slice(2)}`

// 1) login via API
const loginResp = await fetch(`${API}/api/v1/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idem() },
  body: JSON.stringify({ email: 'demo@local.test', password: 'Demo1234!' }),
})
const { access_token: token } = await loginResp.json()
if (!token) { console.error('login failed'); process.exit(1) }

// 2) set NON-DEFAULT server values for all 6 stores
const put = (mod, scope, settings) =>
  fetch(`${API}/api/v1/settings/${mod}/${scope}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, 'Idempotency-Key': idem() },
    body: JSON.stringify({ settings }),
  }).then((r) => r.status)

const puts = {
  'work/user': await put('work', 'user', { defaultView: 'list', myTasksGroupBy: 'priority', density: 'compact' }),
  'vertraege/user': await put('vertraege', 'user', { defaultTab: 'archiv', density: 'compact' }),
  'finance/tenant': await put('finance', 'tenant', { chartFramework: 'SKR04' }),
  'wiki/tenant': await put('wiki', 'tenant', { shareDefault: 'public', publicModeEnabled: true }),
  'dashboard/tenant': await put('dashboard', 'tenant', { defaultWidgets: ['kpi-revenue', 'kpi-tasks'], allowedWidgets: ['kpi-revenue', 'kpi-tasks', 'recent-contacts'] }),
  'zeiterfassung/tenant': await put('zeiterfassung', 'tenant', { rounding: '15', holidayRegion: 'AT', autoBreakAfterHours: 8, autoBreakMinutes: 45 }),
}
console.log('PUT status:', puts)

// 3) browser: stub electron, seed DEFAULTS, login, wait for hydration
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const SUPPRESS = `try { const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p)) } catch(e){}`
const SEED = `try {
  localStorage.setItem('cosmi-work-prefs', JSON.stringify({ state: { defaultView:'kanban', myTasksGroupBy:'project', density:'comfortable', defaultProjectId:null, serverInitialized:false }, version: 0 }))
  localStorage.setItem('cosmi-vertraege-prefs', JSON.stringify({ state: { defaultTab:'aktiv', density:'comfortable', defaultReminderDays:null, serverInitialized:false }, version: 0 }))
  localStorage.setItem('cosmi-finance-tenant', JSON.stringify({ state: { chartFramework:'SKR03', serverInitialized:false }, version: 0 }))
  localStorage.setItem('cosmi-wiki-settings', JSON.stringify({ state: { shareDefault:'internal', publicModeEnabled:false, serverInitialized:false }, version: 0 }))
  localStorage.setItem('cosmi-dashboard-settings', JSON.stringify({ state: { defaultWidgets:['kpi-revenue','kpi-tasks','recent-contacts','deal-pipeline'], allowedWidgets:['kpi-revenue','kpi-tasks','recent-contacts','deal-pipeline'], serverInitialized:false }, version: 0 }))
  localStorage.setItem('cosmi-zeiterfassung-settings', JSON.stringify({ state: { autoBreakAfterHours:6, autoBreakMinutes:30, rounding:'none', holidayRegion:'DE', serverInitialized:false }, version: 0 }))
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
await page.waitForTimeout(6000) // login → DeskEnvironment mounts → hydrator fires → persist writes back

const read = async (k) => page.evaluate((key) => { try { return JSON.parse(localStorage.getItem(key)).state } catch { return {} } }, k)
const work = await read('cosmi-work-prefs')
const vert = await read('cosmi-vertraege-prefs')
const fin = await read('cosmi-finance-tenant')
const wiki = await read('cosmi-wiki-settings')
const dash = await read('cosmi-dashboard-settings')
const zeit = await read('cosmi-zeiterfassung-settings')
await browser.close()

const ok = (label, cond, got) => console.log(`  ${cond ? 'OK ✓' : 'FEHLT ✗'}  ${label}  →`, JSON.stringify(got))
console.log('=== X-4 rest hydrator (6 new stores) ===')
ok('work/user     defaultView=list density=compact', work.defaultView === 'list' && work.density === 'compact', { v: work.defaultView, g: work.myTasksGroupBy, d: work.density, init: work.serverInitialized })
ok('vertraege/user defaultTab=archiv', vert.defaultTab === 'archiv' && vert.density === 'compact', { t: vert.defaultTab, d: vert.density, init: vert.serverInitialized })
ok('finance/tenant chartFramework=SKR04', fin.chartFramework === 'SKR04', { cf: fin.chartFramework, init: fin.serverInitialized })
ok('wiki/tenant   shareDefault=public public=true', wiki.shareDefault === 'public' && wiki.publicModeEnabled === true, { s: wiki.shareDefault, p: wiki.publicModeEnabled, init: wiki.serverInitialized })
ok('dashboard/tenant defaultWidgets=[kpi-revenue,kpi-tasks]', Array.isArray(dash.defaultWidgets) && dash.defaultWidgets.length === 2 && dash.defaultWidgets.includes('kpi-revenue'), { dw: dash.defaultWidgets, aw: dash.allowedWidgets, init: dash.serverInitialized })
ok('zeiterfassung/tenant rounding=15 region=AT', zeit.rounding === '15' && zeit.holidayRegion === 'AT' && zeit.autoBreakAfterHours === 8, { r: zeit.rounding, reg: zeit.holidayRegion, br: zeit.autoBreakAfterHours, init: zeit.serverInitialized })

const all = work.defaultView === 'list' && vert.defaultTab === 'archiv' && fin.chartFramework === 'SKR04' && wiki.shareDefault === 'public' && Array.isArray(dash.defaultWidgets) && dash.defaultWidgets.length === 2 && zeit.rounding === '15'
console.log('\n  → Alle 6 zentral hydratisiert:', all ? 'JA ✓' : 'NEIN ✗')
process.exit(all ? 0 : 1)
