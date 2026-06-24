// A-1 stateful check — invite a user, confirm it appears as a pending row and
// the counts update; then deactivate via the detail modal.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5174'
const outDir = resolve('.qa-admin-a1')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const PREP = `
  try { const KEY='cosmi-ui'; const raw=localStorage.getItem(KEY); const p=raw?JSON.parse(raw):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(KEY,JSON.stringify(p)); } catch(e){}
  try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}
  try { localStorage.setItem('cosmi-locale', JSON.stringify({state:{locale:'de'},version:0})) } catch(e){}
`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(PREP)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

await page.goto(`${FE}/#/admin/users`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(2600)

// Open invite, fill, submit
await page.getByRole('button', { name: /einladen/i }).first().click()
await page.waitForTimeout(600)
await page.locator('#invite-email').fill('max.mustermann@firma.de')
await page.waitForTimeout(200)
await page.getByRole('button', { name: /Einladung senden/i }).click()
await page.waitForTimeout(1200)
await page.screenshot({ path: resolve(outDir, 'de-04-after-invite.png') })

const newUserVisible = await page.evaluate(() =>
  document.body.textContent.includes('max.mustermann@firma.de'))
const invitedCount = await page.evaluate(() => {
  const btns = [...document.querySelectorAll('button')]
  const b = btns.find((x) => /Eingeladen/.test(x.textContent || ''))
  return b ? b.textContent.replace(/\s+/g, ' ').trim() : 'n/a'
})

// Navigate away and back to prove state survives navigation
await page.goto(`${FE}/#/admin/it`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(800)
await page.goto(`${FE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1500)
const survivesNav = await page.evaluate(() =>
  document.body.textContent.includes('max.mustermann@firma.de'))

console.log(JSON.stringify({ newUserVisible, invitedCount, survivesNav, pageerrors: errors.length, errors: errors.slice(0, 3) }, null, 2))
await browser.close()
