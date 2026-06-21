import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte R6-1 — new report sources hr (Mitarbeiter) + zeiterfassung appear in
// the chart-block SourcePicker (Neue Grafik) and render a live preview.
// Path: Neuer Bericht -> edit -> Block einfügen -> Diagramm -> Grafik konfigurieren
//       -> Neue Grafik -> SourcePicker.
// Run: node scripts/qa-berichte-r6-1.mjs   (dev server on :5173)
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte-r6-1')

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

async function pickFields(page, dim, measure) {
  const d = page.getByRole('button', { name: dim }).first()
  if (await has(d)) await d.click().catch(() => {})
  await page.waitForTimeout(400)
  const m = page.getByRole('button', { name: measure }).first()
  if (await has(m)) await m.click().catch(() => {})
  await page.waitForTimeout(1200)
}

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)

  // Neuer Bericht -> opens ReportDocumentEditor in edit mode
  await page.getByRole('button', { name: /Neuer Bericht/ }).first().click()
  await page.waitForTimeout(2000)
  await shot(page, '1-editor.png')
  step('new report editor opened')

  // Block einfügen -> Diagramm
  const addBlock = page.getByRole('button', { name: /Block einfügen/ }).first()
  if (await has(addBlock)) {
    await addBlock.click()
    await page.waitForTimeout(600)
    await shot(page, '2-block-menu.png')
    const chartOpt = page.getByRole('button', { name: /^Diagramm$/ }).first()
    if (await has(chartOpt)) await chartOpt.click()
    await page.waitForTimeout(800)
    step('chart block inserted')
  } else {
    step('Block einfügen NOT FOUND')
  }
  await shot(page, '3-chart-block.png')

  // Grafik konfigurieren -> picker modal
  const cfg = page.getByText(/Grafik konfigurieren/).first()
  if (await has(cfg)) {
    await cfg.click()
    await page.waitForTimeout(800)
    step('picker opened')
  } else {
    step('Grafik konfigurieren NOT FOUND')
  }
  await shot(page, '4-picker.png')

  // Neue Grafik tab
  const newTab = page.getByRole('button', { name: /Neue Grafik/ }).first()
  if (await has(newTab)) {
    await newTab.click()
    await page.waitForTimeout(600)
    step('Neue Grafik tab')
  }
  out.sourcesVisible = await page.evaluate(() => {
    const t = document.body.innerText
    return { mitarbeiter: t.includes('Mitarbeiter'), zeiterfassung: t.includes('Zeiterfassung') }
  })
  await shot(page, '5-source-picker.png')

  // hr: Mitarbeiter -> Abteilung + Resturlaub
  const hr = page.getByText('Mitarbeiter', { exact: true }).first()
  if (await has(hr)) {
    await hr.click()
    await page.waitForTimeout(800)
    await shot(page, '6-hr-fields.png')
    await pickFields(page, /Abteilung/, /Resturlaub/)
    out.hrCharts = await page.locator('.recharts-surface').count()
    await shot(page, '7-hr-preview.png')
    step(`hr preview charts: ${out.hrCharts}`)
  }

  // zeiterfassung -> Tätigkeit + Erfasste Zeit
  const ze = page.getByText('Zeiterfassung', { exact: true }).first()
  if (await has(ze)) {
    await ze.click()
    await page.waitForTimeout(800)
    await pickFields(page, /Tätigkeit/, /Erfasste Zeit/)
    out.zeiterfassungCharts = await page.locator('.recharts-surface').count()
    await shot(page, '8-zeiterfassung-preview.png')
    step(`zeiterfassung preview charts: ${out.zeiterfassungCharts}`)
  }

  out.rawKeys = await scanRawKeys(page)
  out.doubleBraces = await scanDoubleBraces(page)
} catch (e) {
  out.error = String(e)
  await shot(page, 'ERROR.png').catch(() => {})
}

out.pageErrors = errors
console.log(JSON.stringify(out, null, 2))
await browser.close()
