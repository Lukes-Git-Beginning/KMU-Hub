import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

// QA: berichte R6-5 final check — all 11 report sources appear in the picker, their
// sampleRows feed a rendered live preview. One (dimension, measure) pair per source.
// Run: node scripts/qa-berichte-r6-5.mjs   (dev server on :5173)
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte-r6-5')

const SOURCES = [
  ['Rechnungen', /Belegart/, /Brutto-Betrag/],
  ['Kontakte & Deals', /Pipeline-Stufe/, /Deal-Wert/],
  ['Projekte & Aufgaben', /^Status$/, /Geschätzte Stunden/],
  ['Tickets', /Priorität/, /Lösungszeit/],
  ['Posteingang', /Kanal/, /Antwortzeit/],
  ['Mitarbeiter', /Abteilung/, /Resturlaub/],
  ['Zeiterfassung', /Tätigkeit/, /Erfasste Zeit/],
  ['Verträge', /Vertragsart/, /Vertragswert/],
  ['Bestellungen', /Kategorie/, /Bestellwert/],
  ['Fuhrpark', /Antrieb/, /Kraftstoffkosten/],
  ['Arbeitsrapporte', /Leistungsart/, /Arbeitsstunden/],
]

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
const has = async (loc) => (await loc.count().catch(() => 0)) > 0

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = { perSource: [], rawKeys: [], doubleBraces: [] }

try {
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3500)
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

  out.sourcesInPicker = await page.evaluate((labels) => {
    const t = document.body.innerText
    return labels.filter((l) => t.includes(l)).length
  }, SOURCES.map((s) => s[0]))

  let i = 0
  for (const [label, dim, measure] of SOURCES) {
    const s = page.getByText(label, { exact: true }).first()
    if (!(await has(s))) { out.perSource.push({ label, ok: false, reason: 'not in picker' }); continue }
    await s.click()
    await page.waitForTimeout(1300)
    const d = page.getByRole('button', { name: dim }).first()
    const dimOk = await has(d)
    if (dimOk) await d.click()
    await page.waitForTimeout(350)
    const m = page.getByRole('button', { name: measure }).first()
    const mOk = await has(m)
    if (mOk) await m.click()
    // Wait actively for the live preview to render rather than a fixed timeout.
    await page.locator('.recharts-surface').first().waitFor({ state: 'visible', timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(1100) // let the bar/line animation settle before the screenshot
    const charts = await page.locator('.recharts-surface').count()
    i++
    await page.screenshot({ path: resolve(outDir, `${String(i).padStart(2, '0')}-${label.replace(/[^a-zA-Z]/g, '')}.png`) })
    out.perSource.push({ label, dimOk, measureOk: mOk, charts, ok: dimOk && mOk && charts > 0 })
    out.rawKeys.push(...(await scanRawKeys(page)))
    out.doubleBraces.push(...(await scanDoubleBraces(page)))
  }
  out.rawKeys = [...new Set(out.rawKeys)]
  out.doubleBraces = [...new Set(out.doubleBraces)]
} catch (e) {
  out.error = String(e)
  await page.screenshot({ path: resolve(outDir, 'ERROR.png') }).catch(() => {})
}

out.pageErrors = errors
out.allOk = out.perSource.length === SOURCES.length && out.perSource.every((s) => s.ok)
console.log(JSON.stringify(out, null, 2))
await browser.close()
