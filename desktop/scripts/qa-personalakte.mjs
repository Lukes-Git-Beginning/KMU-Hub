import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
  try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}
`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(team|hr|notifications|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(3500)

  const tab = page.getByRole('tab', { name: /Personalakte/i }).first()
  const tabVisible = await tab.isVisible().catch(() => false)
  out.tabVisible = tabVisible
  if (tabVisible) {
    await tab.click()
    await page.waitForTimeout(2500)
  } else {
    // Fallback: the tab strip may render as buttons rather than ARIA tabs.
    const btn = page.getByText(/Personalakte/i).first()
    if (await btn.isVisible().catch(() => false)) {
      await btn.click()
      await page.waitForTimeout(2500)
      out.tabVisible = true
    }
  }

  await page.screenshot({ path: resolve(outDir, 'personalakte-empty.png'), fullPage: true })

  // The list only renders once an Akte is picked — that is where the wire
  // adapter (snake_case -> camelCase) and the derived expiry status show up.
  const select = page.locator('select').first()
  if (await select.isVisible().catch(() => false)) {
    const values = await select.locator('option').evaluateAll((opts) =>
      opts.map((o) => o.value).filter((v) => v !== ''),
    )
    out.employeeOptions = values.length
    if (values.length > 0) {
      await select.selectOption(values[0])
      await page.waitForTimeout(2000)
    }
  }

  await page.screenshot({ path: resolve(outDir, 'personalakte.png'), fullPage: true })
  out.rawKeys = await scanRawKeys(page)
  out.bodyExcerpt = (await page.evaluate(() => document.body.innerText)).slice(-900)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
