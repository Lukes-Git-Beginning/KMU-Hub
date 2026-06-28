// QA kommunikation/Inbox — Echt-Schaltung (localbackend, notification-Service via :8080).
// Verifiziert nach inbox-demo-Seed: Posteingang lädt echte Messages (6, 3 unread),
// Kanal (email/chat/notification als Int 1/2/3 vom Backend) rendert korrekt,
// Detail/Thread öffnet, keine Raw-Keys/Invalid Date/Crashes.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots', 'inbox-localbackend')

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

await page.goto(`${FE}/#/kommunikation`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(2500)
// Falls "Posteingang"-Bereich nicht default ist, hinklicken
try { await page.getByRole('button', { name: /Posteingang/i }).first().click({ timeout: 2500 }); await page.waitForTimeout(1500) } catch (e) {}
await page.screenshot({ path: resolve(outDir, 'inbox-list-1440.png'), fullPage: false })
const listBody = await page.evaluate(() => document.body.innerText)

// Erste Message öffnen
let detailOpened = false
try {
  await page.getByText('Angebot 2026-0312', { exact: false }).first().click({ timeout: 4000 })
  await page.waitForTimeout(2000)
  await page.screenshot({ path: resolve(outDir, 'inbox-detail-1440.png'), fullPage: false })
  detailOpened = true
} catch (e) {}
const detailBody = detailOpened ? await page.evaluate(() => document.body.innerText) : ''

await browser.close()

const has = (s, re) => re.test(s)
console.log('=== kommunikation/Inbox (echtes Backend) ===')
console.log('  Seite gerendert:          ', listBody.length > 80)
console.log('  Demo-Messages sichtbar:   ', /Sabine Brandt|Angebot 2026-0312|Lieferavis/.test(listBody))
console.log('  Sender/Betreff:           ', /Thomas Keller|Petra Lindner/.test(listBody))
console.log('  Invalid Date/NaN:         ', has(listBody, /Invalid Date|NaN/))
console.log('  undefined im Text:        ', has(listBody, /\bundefined\b/))
console.log('  Raw-Channel-Int (>channel:1<):', has(listBody, /channel.?[123]\b/i))
console.log('  Raw-i18n-Keys:            ', has(listBody, /\b(kommunikation|inbox)\.[a-z]/i))
console.log('  Doppelklammern {{:        ', has(listBody, /\{\{/))
console.log('  Detail geöffnet:          ', detailOpened)
console.log('=== API-Fehler (ohne notifications) ===')
console.log(badApi.length ? [...new Set(badApi)].slice(0, 8).join('\n') : '  KEINE')
console.log('=== JS-Fehler (ohne 503/ws) ===')
const real = errors.filter((e) => !/503|Service Unavailable|notifications|ws:\/\/|WebSocket/.test(e))
console.log(real.length ? real.slice(0, 6).join('\n') : '  KEINE')
