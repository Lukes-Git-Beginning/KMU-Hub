// QA helpdesk — Echt-Schaltung (localbackend, echtes helpdesk-Backend via Gateway :8080).
// Verifiziert nach Lukes tenant-Fix + helpdesk-demo-Seed: Ticket-Liste lädt (6 Tickets,
// vorher 400), Status-/Prioritäts-Badges, SLA/Überfällig, Detail-Modal mit Messages,
// Timestamps via Adapter (kein Invalid Date), keine Raw-Keys/Crashes.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots', 'helpdesk-localbackend')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const SUPPRESS = `try { const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p)) } catch(e){}`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}`)
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))

const page = await ctx.newPage()
const errors = []
const badApi = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errors.push('console: ' + m.text()) })
page.on('response', (res) => { if (res.status() >= 400 && res.url().includes('/api/') && !res.url().includes('/notifications')) badApi.push(`${res.status()} ${res.url()}`) })

await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.locator('input[type=email]').waitFor({ state: 'visible', timeout: 20000 })
await page.locator('input[type=email]').fill('demo@local.test')
await page.locator('input[type=password]').fill('Demo1234!')
await page.locator('input[type=password]').press('Enter')
await page.waitForTimeout(3500)

await page.goto(`${FE}/#/helpdesk`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(3500)
await page.screenshot({ path: resolve(outDir, 'helpdesk-list-1440.png'), fullPage: false })
const listBody = await page.evaluate(() => document.body.innerText)

// Detail öffnen: erste Ticket-Zeile klicken
let detailOpened = false
try {
  await page.getByText('Drucker im 2. OG', { exact: false }).first().click({ timeout: 4000 })
  await page.waitForTimeout(2000)
  await page.screenshot({ path: resolve(outDir, 'helpdesk-detail-1440.png'), fullPage: false })
  detailOpened = true
} catch (e) { /* Detail-Selektor evtl. anders */ }
const detailBody = detailOpened ? await page.evaluate(() => document.body.innerText) : ''

await browser.close()

const has = (s, re) => re.test(s)
console.log('=== helpdesk LISTE (echtes Backend) ===')
console.log('  Seite gerendert:          ', listBody.length > 80)
console.log('  Demo-Tickets sichtbar:    ', /Drucker|VPN-Zugang|Monitor flackert/.test(listBody))
console.log('  Status-Begriffe:          ', /(Offen|Open|Wartend|Pending|Gelöst|Geschlossen)/i.test(listBody))
console.log('  Invalid Date/NaN:         ', has(listBody, /Invalid Date|NaN/))
console.log('  undefined im Text:        ', has(listBody, /\bundefined\b/))
console.log('  Raw-i18n-Keys:            ', has(listBody, /\bhelpdesk\.[a-z]/i))
console.log('  Doppelklammern {{:        ', has(listBody, /\{\{/))
console.log('=== helpdesk DETAIL ===')
console.log('  Detail geöffnet:          ', detailOpened)
console.log('  Message-Text sichtbar:    ', /Toner|Papier|Tonereinheit/.test(detailBody))
console.log('=== API-Fehler (ohne notifications) ===')
console.log(badApi.length ? [...new Set(badApi)].slice(0, 8).join('\n') : '  KEINE')
console.log('=== JS-Fehler (ohne 503) ===')
const real = errors.filter((e) => !/503|Service Unavailable|notifications/.test(e))
console.log(real.length ? real.slice(0, 6).join('\n') : '  KEINE')
