// QA-Test: öffnet Kontakt-Detail, klickt "+ Tag" und screenshottet den Popover.
import { chromium } from 'playwright'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try { const KEY='cosmi-ui'; const raw=localStorage.getItem(KEY); const p=raw?JSON.parse(raw):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(KEY,JSON.stringify(p)) } catch(e){}
`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const result = {}
try {
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.getByText('Karl Bauer', { exact: false }).first().click({ timeout: 5000 })
  await page.waitForTimeout(1500)
  await page.getByRole('button', { name: /\+ ?tag/i }).first().click({ timeout: 5000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'kontakt-tagpopover__full.png') })
  result.popoverOpened = true
  result.errors = errors.length
  result.errorMsgs = errors.slice(0, 3)
} catch (err) {
  result.error = String(err).split('\n')[0]
  result.errors = errors.length
  result.errorMsgs = errors.slice(0, 3)
}
await browser.close()
console.log(JSON.stringify(result, null, 2))
