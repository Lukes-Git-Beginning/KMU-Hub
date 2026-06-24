// A-0 Ist-Research-Screenshots fuer den Admin-Hub (Demo-Mode, Auth-Bypass=admin).
// Setzt cosmi:launch-played, um die Launch-Animation zu ueberspringen.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5173'
const outDir = resolve('.qa-admin-a0')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const PREP = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
  try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}
`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(PREP)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

const tabs = ['it', 'security', 'billing', 'integrations']
const results = []
for (const tab of tabs) {
  try {
    await page.goto(`${FE}/#/admin/${tab}`, { waitUntil: 'domcontentloaded', timeout: 30000 })
    await page.waitForTimeout(2600)
    const file = resolve(outDir, `tab-${tab}.png`)
    await page.screenshot({ path: file, fullPage: false })
    results.push({ tab, file, errors: errors.length })
  } catch (err) {
    results.push({ tab, error: String(err) })
  }
}
await browser.close()
console.log(JSON.stringify(results, null, 2))
