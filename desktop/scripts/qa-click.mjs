// Generischer QA-Klick-Test: route|selector|name Tripel (per ';' getrennt).
// Öffnet route, klickt selector, screenshottet als <name>__full.png.
// Nutzung: node scripts/qa-click.mjs "kontakte/firmen|input[type=checkbox]|firmen-bulk"
import { chromium } from 'playwright'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const specs = (process.argv[2] || '').split(';').map((s) => {
  const [route, selector, name] = s.split('|')
  return { route, selector, name }
})

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try { const KEY='cosmi-ui'; const raw=localStorage.getItem(KEY); const p=raw?JSON.parse(raw):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(KEY,JSON.stringify(p)) } catch(e){}
`

const browser = await chromium.launch()
const results = []
for (const spec of specs) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    await page.goto(`${BASE}/#/${spec.route}`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(2500)
    await page.locator(spec.selector).first().click({ timeout: 5000 })
    await page.waitForTimeout(1200)
    await page.screenshot({ path: resolve(outDir, `${spec.name}__full.png`) })
    results.push({ name: spec.name, errors: errors.length, errorMsgs: errors.slice(0, 2) })
  } catch (err) {
    results.push({ name: spec.name, error: String(err).split('\n')[0], errors: errors.length })
  } finally {
    await ctx.close()
  }
}
await browser.close()
console.log(JSON.stringify(results, null, 2))
