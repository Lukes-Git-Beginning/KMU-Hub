// QA-Flow-Test Kontakt-Detail: öffnet /kontakte, klickt einen Kontakt und
// screenshottet das sich öffnende Detail-Panel bei 3 Fenstergrößen.
// Nutzung: node scripts/qa-contact.mjs
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const CONTACT = process.argv[2] || 'Karl Bauer'

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

const SIZES = [
  { name: 'full', width: 1440, height: 900 },
  { name: 'half', width: 720, height: 900 },
  { name: 'small', width: 500, height: 800 },
]

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

for (const size of SIZES) {
  const ctx = await browser.newContext({ viewport: { width: size.width, height: size.height } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(2500)
    await page.getByText(CONTACT, { exact: false }).first().click({ timeout: 5000 })
    await page.waitForTimeout(2000)
    const file = resolve(outDir, `kontakt-detail__${size.name}.png`)
    await page.screenshot({ path: file, fullPage: false })
    results.push({ size: size.name, errors: errors.length, errorMsgs: errors.slice(0, 2), file })
  } catch (err) {
    results.push({ size: size.name, error: String(err).split('\n')[0], errors: errors.length })
  } finally {
    await ctx.close()
  }
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
