// QA script for profil/zeiterfassung sub-views (Welle 4 reconcile).
// Navigiert zum Profil-Tab (/profil) und wechselt via Hash auf den Zeiterfassung-Tab.
// Usage: node scripts/qa-zeiterfassung.mjs
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/ze-reconcile')

// Electron-API-Stub (noop-Proxy für window.electronAPI)
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`

// Onboarding unterdrücken (setzt cosmi-ui.state.onboardingCompleted = true)
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`

// Roh-Key-Heuristik: sichtbarer Text mit diesen Präfixen = nicht übersetzt
const RAW_PATTERNS = [
  'profil.zeiterfassung.',
  'zeiterfassung.',
  'common.error',
  'api.hr.',
]

async function dismissOnboarding(page) {
  for (const name of [/überspringen/i, /skip/i, /weiter/i]) {
    const btn = page.getByRole('button', { name }).first()
    try {
      if (await btn.isVisible({ timeout: 400 })) {
        await btn.click()
        await page.waitForTimeout(300)
        return
      }
    } catch (_e) {}
  }
}

// Hilfsfunktion: Screenshot + Raw-Key-Scan
async function shot(page, name, errors) {
  const body = (await page.locator('body').innerText()).toLowerCase()
  const rawKeys = RAW_PATTERNS.filter((p) => body.includes(p.toLowerCase()))
  const emojis = /[\u{1F300}-\u{1FAFF}]/u.test(body)
  const path = resolve(outDir, `${name}.png`)
  await page.screenshot({ path, fullPage: false })
  return { shot: name, errors: errors.length, rawKeys, emojis }
}

// Sub-Views die wir testen
const VIEWS = [
  { id: 'overview', label: 'Übersicht' },
  { id: 'today', label: 'Heute' },
  { id: 'week', label: 'Woche' },
  { id: 'month', label: 'Monat' },
  { id: 'reports', label: 'Auswertung' },
  { id: 'team', label: 'Team' },
  { id: 'categories', label: 'Kategorien' },
]

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

// ── 1. Haupt-Zeiterfassung-Tab im Profil aufrufen ──────────────────────────
// Route: /#/profil → dann Zeiterfassung-Tab klicken
for (const view of VIEWS) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    // Navigiere zu Profil
    await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(1200)
    await dismissOnboarding(page)
    await page.waitForTimeout(800)

    // Klicke Zeiterfassung-Tab im Profil (Text-Matching)
    const zeitTab = page.getByRole('tab', { name: /zeiterfassung/i }).first()
    try {
      await zeitTab.waitFor({ state: 'visible', timeout: 5000 })
      await zeitTab.click()
      await page.waitForTimeout(800)
    } catch (_e) {
      // Fallback: direkter Hash-Route-Versuch
      await page.goto(`${BASE}/#/profil?tab=zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 10000 })
      await page.waitForTimeout(800)
    }

    // Sub-View navigieren (via Button/Tab in der Zeiterfassung-Sidebar/Navigation)
    if (view.id !== 'overview') {
      const navBtn = page.getByRole('button', { name: new RegExp(view.label, 'i') }).first()
      try {
        await navBtn.waitFor({ state: 'visible', timeout: 3000 })
        await navBtn.click()
        await page.waitForTimeout(600)
      } catch (_e) {
        // Tab-based navigation fallback
        const navTab = page.getByRole('tab', { name: new RegExp(view.label, 'i') }).first()
        try {
          await navTab.waitFor({ state: 'visible', timeout: 2000 })
          await navTab.click()
          await page.waitForTimeout(600)
        } catch (_e2) {
          errors.push(`nav-miss: konnte View "${view.label}" nicht klicken`)
        }
      }
    }

    results.push(await shot(page, `view__${view.id}`, errors))
  } catch (err) {
    results.push({ shot: `view__${view.id}`, error: String(err) })
  } finally {
    await ctx.close()
  }
}

// ── 2. ManualEntryForm Dialog ─────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(1200)
    await dismissOnboarding(page)
    await page.waitForTimeout(800)

    // Zeiterfassung-Tab
    const zeitTab = page.getByRole('tab', { name: /zeiterfassung/i }).first()
    try {
      await zeitTab.waitFor({ state: 'visible', timeout: 5000 })
      await zeitTab.click()
      await page.waitForTimeout(800)
    } catch (_e) {}

    // Heute-View → Manuell-eintragen Button
    const manualBtn = page.getByRole('button', { name: /manuell|manual/i }).first()
    await manualBtn.waitFor({ state: 'visible', timeout: 5000 })
    await manualBtn.click()
    await page.waitForTimeout(700)
    results.push(await shot(page, 'dialog__manual-entry', errors))
  } catch (err) {
    results.push({ shot: 'dialog__manual-entry', error: String(err) })
  } finally {
    await ctx.close()
  }
}

// ── 3. ExportDialog ────────────────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  try {
    await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(1200)
    await dismissOnboarding(page)
    await page.waitForTimeout(800)

    // Zeiterfassung-Tab
    const zeitTab = page.getByRole('tab', { name: /zeiterfassung/i }).first()
    try {
      await zeitTab.waitFor({ state: 'visible', timeout: 5000 })
      await zeitTab.click()
      await page.waitForTimeout(800)
    } catch (_e) {}

    // Auswertungen-View → Export-Button
    const auswBtn = page.getByRole('button', { name: /auswertung|reports/i }).first()
    try {
      await auswBtn.waitFor({ state: 'visible', timeout: 3000 })
      await auswBtn.click()
      await page.waitForTimeout(600)
    } catch (_e) {}

    const exportBtn = page.getByRole('button', { name: /export/i }).first()
    await exportBtn.waitFor({ state: 'visible', timeout: 5000 })
    await exportBtn.click()
    await page.waitForTimeout(700)
    results.push(await shot(page, 'dialog__export', errors))
  } catch (err) {
    results.push({ shot: 'dialog__export', error: String(err) })
  } finally {
    await ctx.close()
  }
}

await browser.close()

// Ausgabe
console.log('\n=== QA Results: zeiterfassung reconcile ===')
console.log(JSON.stringify(results, null, 2))

const issues = results.filter((r) => r.error || r.errors > 0 || (r.rawKeys && r.rawKeys.length > 0) || r.emojis)
if (issues.length > 0) {
  console.log('\n=== Issues found ===')
  console.log(JSON.stringify(issues, null, 2))
  process.exit(1)
}
console.log('\nAll checks passed.')
