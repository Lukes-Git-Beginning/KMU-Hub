// QA P1 zeiterfassung — module shell + Stundenkonto badge + unified header widget.
// Usage: node scripts/qa-zeiterfassung-p1.mjs
import { chromium } from 'playwright'
import { mkdir, readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/ze-p1')

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

async function dismissOnboarding(page) {
  for (const name of [/überspringen/i, /skip/i]) {
    const btn = page.getByRole('button', { name }).first()
    try { if (await btn.isVisible({ timeout: 500 })) { await btn.click(); await page.waitForTimeout(300); return } } catch (e) {}
  }
}

// Raw-key heuristic: any visible text containing these literal prefixes = untranslated.
const RAW_PATTERNS = ['zeiterfassung.', 'header.timeTracker.', 'profil.zeiterfassung.']

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

// ── 1. Module shell @ 3 sizes ──
for (const size of [
  { name: 'full', width: 1440, height: 900 },
  { name: 'half', width: 720, height: 900 },
  { name: 'small', width: 500, height: 800 },
]) {
  const ctx = await browser.newContext({ viewport: { width: size.width, height: size.height } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    await page.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(1500)
    await dismissOnboarding(page)
    await page.waitForTimeout(1000)
    const body = (await page.locator('body').innerText()).toLowerCase()
    const raw = RAW_PATTERNS.filter((p) => body.includes(p.toLowerCase()))
    await page.screenshot({ path: resolve(outDir, `shell__${size.name}.png`), fullPage: false })
    results.push({ shot: `shell ${size.name}`, errors: errors.length, rawKeys: raw })
  } catch (err) {
    results.push({ shot: `shell ${size.name}`, error: String(err) })
  } finally { await ctx.close() }
}

// ── 2. Header work-clock widget dropdown @ 1440 ──
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    await page.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(1500)
    await dismissOnboarding(page)
    await page.waitForTimeout(1500)
    const trigger = page.locator('[data-tour="time-tracker"] button').first()
    await trigger.waitFor({ state: 'visible', timeout: 8000 })
    await trigger.click()
    await page.waitForTimeout(900)
    const body = (await page.locator('body').innerText()).toLowerCase()
    const raw = RAW_PATTERNS.filter((p) => body.includes(p.toLowerCase()))
    await page.screenshot({ path: resolve(outDir, `widget-open__full.png`), fullPage: false })
    results.push({ shot: 'widget dropdown', errors: errors.length, rawKeys: raw })
  } catch (err) {
    results.push({ shot: 'widget dropdown', error: String(err) })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
