// A-1 QA — admin/Benutzer tab (Demo-Mode, Auth-Bypass=admin) on :5174.
// Captures list / invite dialog / detail modal in DE and EN.
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
const prep = (locale) => `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
  try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}
  try { localStorage.setItem('cosmi-locale', JSON.stringify({ state: { locale: '${locale}' }, version: 0 })) } catch (e) {}
`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()

async function jsClick(page, selector) {
  return page.evaluate((sel) => {
    const el = document.querySelector(sel)
    if (el) { el.click(); return true }
    return false
  }, selector)
}

for (const locale of ['de', 'en']) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(prep(locale))
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  const shot = async (name) => { await page.waitForTimeout(500); await page.screenshot({ path: resolve(outDir, `${locale}-${name}.png`) }) }

  // List
  await page.goto(`${FE}/#/admin/users`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(2600)
  await shot('01-list')

  // Invite dialog
  const inviteBtn = page.getByRole('button', { name: /einladen|invite/i }).first()
  try { await inviteBtn.click({ timeout: 3000 }) } catch (e) { await jsClick(page, 'button') }
  await page.waitForTimeout(700)
  await shot('02-invite')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // Detail modal — first row
  try { await page.locator('ul li button').first().click({ timeout: 3000 }) } catch (e) {}
  await page.waitForTimeout(700)
  await shot('03-detail')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  console.log(`[${locale}] pageerrors: ${errors.length}`, errors.slice(0, 3))
  await ctx.close()
}

await browser.close()
console.log('done')
