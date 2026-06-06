// QA-Phase-3: testet die Block-A-Nachbesserung an Kontakte —
// (1) Raster-Ansicht @ 3 Größen, (2) Detail-Modal @ 3 Größen,
// (3) Consent-Sektion im Modal (Overflow-Check nach ConsentPanel-Fix).
// Nutzung: node scripts/qa-phase3.mjs
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/phase3-qa')

const SIZES = [
  { name: 'full', width: 1440, height: 900 },
  { name: 'half', width: 720, height: 900 },
  { name: 'small', width: 500, height: 800 },
]

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

async function newPage(browser, size) {
  const ctx = await browser.newContext({ viewport: { width: size.width, height: size.height } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(1800)
  return { ctx, page, errors }
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

// ---- (1) GRID VIEW @ 3 Größen ----
for (const size of SIZES) {
  const { ctx, page, errors } = await newPage(browser, size)
  try {
    await page.getByRole('button', { name: 'Raster' }).click({ timeout: 5000 })
    await page.waitForTimeout(1000)
    const file = resolve(outDir, `grid__${size.name}.png`)
    await page.screenshot({ path: file })
    results.push({ flow: 'grid', size: size.name, errors: errors.length, file })
  } catch (err) {
    results.push({ flow: 'grid', size: size.name, error: String(err).split('\n')[0], errors: errors.length })
  } finally {
    await ctx.close()
  }
}

// ---- (2) DETAIL MODAL @ 3 Größen ----
for (const size of SIZES) {
  const { ctx, page, errors } = await newPage(browser, size)
  try {
    // erster Kontakt-Button in der Liste (Name Karl Bauer)
    await page.getByText('Karl Bauer', { exact: false }).first().click({ timeout: 5000 })
    await page.waitForTimeout(1200)
    const dialogOpen = await page.locator('[role="dialog"]').count()
    const file = resolve(outDir, `modal__${size.name}.png`)
    await page.screenshot({ path: file })
    results.push({ flow: 'modal', size: size.name, dialogOpen, errors: errors.length, file })
  } catch (err) {
    results.push({ flow: 'modal', size: size.name, error: String(err).split('\n')[0], errors: errors.length })
  } finally {
    await ctx.close()
  }
}

// ---- (3) CONSENT-Sektion im Modal (Overflow-Check) — schmal + voll ----
for (const size of [SIZES[0], SIZES[2]]) {
  const { ctx, page, errors } = await newPage(browser, size)
  try {
    await page.getByText('Karl Bauer', { exact: false }).first().click({ timeout: 5000 })
    await page.waitForTimeout(1000)
    // im Dialog zur Consent-Sektion scrollen
    const consent = page.getByText(/Einwilligung|Consent|DSGVO/i).first()
    if (await consent.count()) {
      await consent.scrollIntoViewIfNeeded({ timeout: 3000 }).catch(() => {})
      await page.waitForTimeout(600)
    }
    const file = resolve(outDir, `consent__${size.name}.png`)
    await page.screenshot({ path: file })
    results.push({ flow: 'consent', size: size.name, consentFound: await consent.count(), errors: errors.length, file })
  } catch (err) {
    results.push({ flow: 'consent', size: size.name, error: String(err).split('\n')[0], errors: errors.length })
  } finally {
    await ctx.close()
  }
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
