// QA-Flow Beratungsprotokoll (P8): Kontakt öffnen → Tab "Beratungsprotokolle" →
// "Neues Protokoll" → Vollbild-Editor. Screenshots @ Breiten + Raw-i18n-Key-Scan
// + pageerror-Erfassung. Nutzung: node scripts/qa-advisory.mjs ["Kontaktname"]
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

// Detect raw i18n keys leaking into the DOM (e.g. "advisory.field.date").
const RAW_KEY_RE = /\b(advisory|kontakte|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  const hits = [...text.matchAll(/\b(advisory|kontakte|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0])
  return [...new Set(hits)]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

const SIZES = [
  { name: 'full', width: 1440, height: 900 },
  { name: 'half', width: 800, height: 900 },
]

for (const size of SIZES) {
  const ctx = await browser.newContext({ viewport: { width: size.width, height: size.height } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  const step = { size: size.name }
  try {
    await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(2500)
    await page.getByText(CONTACT, { exact: false }).first().click({ timeout: 5000 })
    await page.waitForTimeout(1500)

    // Tab to Beratungsprotokolle
    await page.getByRole('tab', { name: /Beratungsprotokolle/i }).click({ timeout: 5000 })
    await page.waitForTimeout(800)
    await page.screenshot({ path: resolve(outDir, `advisory-tab__${size.name}.png`) })
    step.tabRawKeys = await scanRawKeys(page)

    // New protocol → editor
    await page.getByRole('button', { name: /Neues Protokoll/i }).click({ timeout: 5000 })
    await page.waitForTimeout(1500)
    await page.screenshot({ path: resolve(outDir, `advisory-editor__${size.name}.png`), fullPage: true })
    step.editorRawKeys = await scanRawKeys(page)
    step.editorUrl = page.url()
  } catch (err) {
    step.error = String(err).split('\n')[0]
  }
  step.pageErrors = errors.slice(0, 3)
  results.push(step)
  await ctx.close()
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
