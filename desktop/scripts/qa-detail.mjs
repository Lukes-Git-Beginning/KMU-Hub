// QA-Flow-Test: öffnet eine Listen-Route, klickt das erste Element und
// screenshottet die sich öffnende Detail-Seite — testet den echten User-Flow
// (Liste → Klick → Detail), nicht nur den Direkt-Aufruf einer URL.
// Nutzung: node scripts/qa-detail.mjs
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`

// route = Listen-Route, clickSelector = was angeklickt wird, name = Output-Datei
const flows = [
  { route: 'kontakte/firmen', clickSelector: 'tbody tr', name: 'firmen-detail' },
  { route: 'kontakte/pipeline', clickSelector: '[role="button"]', name: 'pipeline-detail' },
]

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

for (const flow of flows) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    await page.goto(`${BASE}/#/${flow.route}`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(2500)
    const target = page.locator(flow.clickSelector).first()
    await target.click({ timeout: 5000 })
    await page.waitForTimeout(2000)
    const urlAfter = await page.evaluate(() => location.hash)
    const file = resolve(outDir, `${flow.name}__full.png`)
    await page.screenshot({ path: file, fullPage: false })
    results.push({ flow: flow.name, urlAfter, errors: errors.length, file })
  } catch (err) {
    results.push({ flow: flow.name, error: String(err).split('\n')[0], errors: errors.length })
  } finally {
    await ctx.close()
  }
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
