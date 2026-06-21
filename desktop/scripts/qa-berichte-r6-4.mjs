import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte R6-4 — the 5 existing sources deepened to >=12 fields. Each source is
// exercised with a NEWLY ADDED dimension + measure to confirm they render.
// Run: node scripts/qa-berichte-r6-4.mjs   (dev server on :5173)
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte-r6-4')

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
const out = { steps: [], rawKeys: [], doubleBraces: [] }
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
  if (!(await has(s))) { step(`${source} NOT in picker`); return }
  await s.click()
  await page.waitForTimeout(800)
  // count fields available (dimension + measure rows are buttons with checkboxes)
  const d = page.getByRole('button', { name: dim }).first()
  if (await has(d)) await d.click(); else step(`${source}: dim ${dim} MISSING`)
  await page.waitForTimeout(400)
  const m = page.getByRole('button', { name: measure }).first()
  if (await has(m)) await m.click(); else step(`${source}: measure ${measure} MISSING`)
  await page.waitForTimeout(1200)
  const charts = await page.locator('.recharts-surface').count()
  await shot(page, file)
  const rk = await scanRawKeys(page)
  const db = await scanDoubleBraces(page)
  out.rawKeys.push(...rk)
  out.doubleBraces.push(...db)
  step(`${source}: ${charts} chart(s), newDim=${dim.source}, newMeasure=${measure.source}`)
}

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)
  await openPicker(page)

  await pickSource(page, 'Rechnungen', /Belegart/, /Steuerbetrag/, '1-finanzen.png')
  await pickSource(page, 'Tickets', /SLA-Status/, /Zufriedenheit/, '2-helpdesk.png')
  await pickSource(page, 'Kontakte & Deals', /Lead-Quelle/, /Gewichteter Wert/, '3-kontakte.png')
  await pickSource(page, 'Projekte & Aufgaben', /Aufgabentyp/, /Erfasste Stunden/, '4-work.png')
  await pickSource(page, 'Posteingang', /Kategorie/, /Bearbeitungszeit/, '5-kommunikation.png')

  out.rawKeys = [...new Set(out.rawKeys)]
  out.doubleBraces = [...new Set(out.doubleBraces)]
} catch (e) {
  out.error = String(e)
  await shot(page, 'ERROR.png').catch(() => {})
}

out.pageErrors = errors
console.log(JSON.stringify(out, null, 2))
await browser.close()
