// A-3 QA — admin/Lizenz (module activation) on :5174, DE + EN.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5174'
const outDir = resolve('.qa-admin-a3')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const prep = (locale) => `
  try { const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p)); } catch(e){}
  try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}
  try { localStorage.setItem('cosmi-locale', JSON.stringify({state:{locale:'${locale}'},version:0})) } catch(e){}
`
await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()

for (const locale of ['de', 'en']) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 }, reducedMotion: 'reduce' })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(prep(locale))
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  const shot = async (n) => { await page.waitForTimeout(500); await page.screenshot({ path: resolve(outDir, `${locale}-${n}.png`) }) }

  await page.goto(`${FE}/#/admin/license`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(2800)
  await shot('01-overview')

  // Activate the first inactive module switch.
  const toggled = await page.evaluate(() => {
    const sw = [...document.querySelectorAll('[role="switch"]')].find((s) => s.getAttribute('aria-checked') === 'false')
    if (sw) { sw.click(); return sw.getAttribute('aria-label') }
    return null
  })
  await page.waitForTimeout(700)
  await shot('02-after-activate')

  // Survives navigation?
  await page.goto(`${FE}/#/admin/roles`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(700)
  await page.goto(`${FE}/#/admin/license`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const stillOn = await page.evaluate((label) => {
    if (!label) return 'n/a'
    const s = [...document.querySelectorAll('[role="switch"]')].find((x) => x.getAttribute('aria-label') === label)
    return s ? s.getAttribute('aria-checked') : 'missing'
  }, toggled)

  console.log(`[${locale}] activated=${toggled} survivesNav=${stillOn} pageerrors=${errors.length}`, errors.slice(0, 2))
  await ctx.close()
}
await browser.close()
console.log('done')
