import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte R6-3 — fuhrpark (Fuhrpark) + rapporte (Arbeitsrapporte) sources in the
// chart-block SourcePicker render a live preview.
// Run: node scripts/qa-berichte-r6-3.mjs   (dev server on :5173)
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte-r6-3')

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
  return [...new Set([...text.matchAll(/\b(berichte|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
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

async function openPicker(page) {
  await page.getByRole('button', { name: /Neuer Bericht/ }).first().click()
  await page.waitForTimeout(2000)
  await page.getByRole('button', { name: /Block einfügen/ }).first().click()
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: /^Diagramm$/ }).first().click()
  await page.waitForTimeout(800)
  await page.getByText(/Grafik konfigurieren/).first().click()
  await page.waitForTimeout(800)
  await page.getByRole('button', { name: /Neue Grafik/ }).first().click()
  await page.waitForTimeout(600)
}
async function pickSource(page, source, dim, measure, file) {
  const s = page.getByText(source, { exact: true }).first()
  if (!(await has(s))) { step(`${source} NOT in picker`); return 0 }
  await s.click()
  await page.waitForTimeout(800)
  const d = page.getByRole('button', { name: dim }).first()
  if (await has(d)) await d.click()
  await page.waitForTimeout(400)
  const m = page.getByRole('button', { name: measure }).first()
  if (await has(m)) await m.click()
  await page.waitForTimeout(1200)
  const charts = await page.locator('.recharts-surface').count()
  await shot(page, file)
  step(`${source}: ${charts} chart(s)`)
  return charts
}

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)
  await openPicker(page)

  out.sourcesVisible = await page.evaluate(() => {
    const t = document.body.innerText
    return { fuhrpark: t.includes('Fuhrpark'), rapporte: t.includes('Arbeitsrapporte') }
  })
  await shot(page, '1-source-picker.png')

  out.fuhrparkCharts = await pickSource(page, 'Fuhrpark', /Antrieb/, /Kraftstoffkosten/, '2-fuhrpark-preview.png')
  out.rapporteCharts = await pickSource(page, 'Arbeitsrapporte', /Leistungsart/, /Arbeitsstunden/, '3-rapporte-preview.png')

  out.rawKeys = await scanRawKeys(page)
  out.doubleBraces = await scanDoubleBraces(page)
} catch (e) {
  out.error = String(e)
  await shot(page, 'ERROR.png').catch(() => {})
}

out.pageErrors = errors
console.log(JSON.stringify(out, null, 2))
await browser.close()
