// QA: production web build hides self-registration (link + route).
// Run against `npm run preview:web` (port 5175).
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5175'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, p) => (p === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
// Skip the launch fly-in so the login form is on screen when we look.
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  await page.goto(`${BASE}/#/login`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.locator('input[type="password"]').first().waitFor({ state: 'visible', timeout: 15000 })
  await page.waitForTimeout(800)
  await page.screenshot({ path: resolve(outDir, 'register-closed-login.png'), fullPage: true })
  out.loginFormVisible = true
  out.registerLinkCount = await page.locator('a[href="#/register"]').count()
  out.noAccountTextVisible = await page.getByText(/Noch kein Konto/).count()

  await page.goto(`${BASE}/#/register`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.screenshot({ path: resolve(outDir, 'register-closed-route.png'), fullPage: true })
  out.registerRouteUrl = page.url()
  out.registerFormVisible = await page.locator('input[name="firstName"], input[autocomplete="given-name"]').count()
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
