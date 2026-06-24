// A-2 QA — admin/Rollen (RBAC matrix) on :5174, DE + EN. Captures matrix,
// a toggle + role-detail modal, and verifies the toggle survives navigation.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
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

  await page.goto(`${FE}/#/admin/roles`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(2600)
  await shot('01-matrix')

  // Toggle a non-admin checkbox (the first editable, non-checked cell) and re-shot
  const toggled = await page.evaluate(() => {
    const boxes = [...document.querySelectorAll('button[role="checkbox"]')]
    const target = boxes.find((b) => !b.disabled && b.getAttribute('aria-checked') === 'false')
    if (target) { target.click(); return target.getAttribute('aria-label') }
    return null
  })
  await page.waitForTimeout(600)
  await shot('02-after-toggle')

  // Role detail modal (first role card)
  if (locale === 'de') {
    await page.locator('button:has-text("Administrator")').first().click().catch(() => {})
    await page.waitForTimeout(700)
    await shot('03-role-detail')
    await page.keyboard.press('Escape')
  }

  // Toggle persistence across navigation
  await page.goto(`${FE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(800)
  await page.goto(`${FE}/#/admin/roles`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const stillChecked = await page.evaluate((label) => {
    if (!label) return 'n/a'
    const b = [...document.querySelectorAll('button[role="checkbox"]')].find((x) => x.getAttribute('aria-label') === label)
    return b ? b.getAttribute('aria-checked') : 'missing'
  }, toggled)

  console.log(`[${locale}] toggled=${toggled} survivesNav=${stillChecked} pageerrors=${errors.length}`, errors.slice(0, 2))
  await ctx.close()
}

await browser.close()
console.log('done')
