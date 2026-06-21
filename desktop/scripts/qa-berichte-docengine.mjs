import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte on the shared document engine (Phase A). Verifies the report
// reader (A4/print) and block editor render identically after the refactor.
// Run: node scripts/qa-berichte-docengine.mjs   (dev server on :5173)
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte-docengine')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, p) => (p === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
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
async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(berichte|common|document)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function scanDoubleBraces(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\{\{[^}]+\}\}/g)].map((m) => m[0]))]
}
const shot = (page, n) => page.screenshot({ path: resolve(outDir, n), fullPage: true })
const has = async (loc) => (await loc.count().catch(() => 0)) > 0

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = { steps: [] }
const step = (s) => out.steps.push(s)

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)

  // Tab "Berichte"
  await page.getByRole('button', { name: 'Berichte', exact: true }).first().click()
  await page.waitForTimeout(1200)
  await shot(page, '1-library.png')

  // Open a report with content (prefer "Monatsbericht", else the first card).
  const monat = page.getByText(/Monatsbericht/).first()
  if (await has(monat)) {
    await monat.click()
    step('opened Monatsbericht')
  } else {
    await page.locator('h4').first().click()
    step('opened first report')
  }
  await page.waitForTimeout(2000)
  await shot(page, '2-reader.png')

  // Switch to edit mode (block editor)
  const editBtn = page.getByRole('button', { name: /Bearbeiten/ }).first()
  if (await has(editBtn)) {
    await editBtn.click()
    await page.waitForTimeout(1500)
    step('edit mode')
  }
  await shot(page, '3-editor.png')

  // Open insert menu
  const addBlock = page.getByRole('button', { name: /Block einfügen/ }).first()
  if (await has(addBlock)) {
    await addBlock.click()
    await page.waitForTimeout(700)
    step('insert menu open')
  }
  await shot(page, '4-insert-menu.png')

  out.rawKeys = await scanRawKeys(page)
  out.doubleBraces = await scanDoubleBraces(page)
} catch (e) {
  out.error = String(e)
  await shot(page, 'ERROR.png').catch(() => {})
}

out.pageErrors = errors
console.log(JSON.stringify(out, null, 2))
await browser.close()
