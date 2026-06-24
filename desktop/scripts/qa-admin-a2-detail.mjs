// Focused capture of the A-2 role-detail modal (assigned users).
import { chromium } from 'playwright'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5174'
const outDir = resolve('.qa-admin-a2')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const PREP = `
  try { const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p)); } catch(e){}
  try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}
  try { localStorage.setItem('cosmi-locale', JSON.stringify({state:{locale:'de'},version:0})) } catch(e){}
`
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(PREP)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
await page.goto(`${FE}/#/admin/roles`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(3000)
// Click the role summary card for "Projektleiterin" (2 users). The role cards
// live in the summary grid and contain the "Nutzer" count.
const clicked = await page.evaluate(() => {
  const cards = [...document.querySelectorAll('button')].filter((b) => /Nutzer/.test(b.textContent || '') && /Projektleiterin/.test(b.textContent || ''))
  if (cards[0]) { cards[0].click(); return cards[0].textContent.replace(/\s+/g, ' ').trim() }
  return null
})
await page.waitForTimeout(900)
await page.screenshot({ path: resolve(outDir, 'de-03-role-detail.png') })
console.log(JSON.stringify({ clicked, pageerrors: errors.length, errors: errors.slice(0, 2) }))
await browser.close()
