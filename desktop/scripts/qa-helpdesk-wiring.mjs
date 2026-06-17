/**
 * QA Helpdesk Wiring
 *
 * Checks:
 *  1. Ticket-Liste lädt (kein MOCK_-Text, keine Crash-Meldung)
 *  2. Status-Filter-Buttons vorhanden
 *  3. Canned-Responses-Panel öffnet
 *  4. Error-State bei simulierter API-Abwesenheit (fetch-Intercept)
 *
 * Screenshots -> scripts/screenshots/helpdesk/
 * Report (JSON) -> stdout
 */
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots/helpdesk')

// Electron API stub (no real Electron in Chromium QA)
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`

// Skip onboarding overlay
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`

// Helper: scan for raw i18n keys in visible text
async function scanRawKeys(page, namespace) {
  const text = await page.evaluate(() => document.body.innerText)
  const pattern = new RegExp(`\\b${namespace}\\.[a-zA-Z][a-zA-Z0-9.]+\\b`, 'g')
  return [...new Set([...text.matchAll(pattern)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })

const browser = await chromium.launch()
const checks = []
const errors = []

// ─────────────────────────────────────────────────────────
// Check 1: Ticket-Liste lädt (Normalfall)
// ─────────────────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errors.push(String(e)))

  const shot = 'helpdesk-ticket-list.png'
  let pass = false
  let notes = ''

  try {
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded', timeout: 25000 })
    await page.waitForTimeout(3000)

    // Verify page rendered (heading or list area)
    const heading = await page.locator('h1, h2, [data-testid="helpdesk-page"]').first().count()
    const rawKeys = await scanRawKeys(page, 'helpdesk')
    const hasMockText = (await page.evaluate(() => document.body.innerText)).includes('MOCK_')
    const hasCrash = await page.locator('[data-testid="error-boundary"], .error-boundary').count()

    notes = `heading=${heading} rawKeys=${rawKeys.length} MOCK_=${hasMockText} crash=${hasCrash}`
    pass = heading > 0 && !hasMockText && hasCrash === 0

    await page.screenshot({ path: resolve(outDir, shot) })
  } catch (e) {
    notes = String(e).split('\n')[0]
  }

  checks.push({ name: 'ticket-list-loads', pass, screenshot: `screenshots/helpdesk/${shot}`, notes })
  await ctx.close()
}

// ─────────────────────────────────────────────────────────
// Check 2: Status-Filter-Buttons vorhanden
// ─────────────────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()

  const shot = 'helpdesk-status-buttons.png'
  let pass = false
  let notes = ''

  try {
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded', timeout: 25000 })
    await page.waitForTimeout(3000)

    // Status filter buttons: "Offen", "In Bearbeitung", "Gelöst", "Geschlossen"
    // or english fallback "open", "resolved" etc.
    const btnTexts = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button'))
      return btns.map((b) => b.innerText.trim().toLowerCase()).filter(Boolean)
    })

    const hasOpen = btnTexts.some((t) => /offen|open/.test(t))
    const hasResolved = btnTexts.some((t) => /gel.st|resolved/.test(t))

    notes = `btns found: ${btnTexts.slice(0, 10).join(', ')}`
    pass = hasOpen || hasResolved

    await page.screenshot({ path: resolve(outDir, shot) })
  } catch (e) {
    notes = String(e).split('\n')[0]
  }

  checks.push({ name: 'status-filter-buttons', pass, screenshot: `screenshots/helpdesk/${shot}`, notes })
  await ctx.close()
}

// ─────────────────────────────────────────────────────────
// Check 3: Canned-Responses-Panel öffnet
// ─────────────────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()

  const shot = 'helpdesk-canned-panel.png'
  let pass = false
  let notes = ''

  try {
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded', timeout: 25000 })
    await page.waitForTimeout(3000)

    // Look for the Canned Responses trigger button (Zap icon or label)
    const zapBtn = page.locator('button:has([data-lucide="zap"]), button:has(.lucide-zap)').first()
    const zapCount = await zapBtn.count()

    if (zapCount > 0) {
      await zapBtn.click({ timeout: 5000 })
      await page.waitForTimeout(1200)

      // Panel should be visible — slide-over or dialog
      const panelVisible = await page
        .locator('[role="dialog"], [data-testid="canned-responses-panel"], .canned-panel')
        .first()
        .count()
      notes = `zapBtn=${zapCount} panelVisible=${panelVisible}`
      pass = panelVisible > 0
    } else {
      // Try text-based fallback
      const fallbackBtn = page
        .getByRole('button', { name: /canned|textbausteine|vorgefertigte/i })
        .first()
      const fbCount = await fallbackBtn.count()
      if (fbCount > 0) {
        await fallbackBtn.click({ timeout: 5000 })
        await page.waitForTimeout(1200)
      }
      notes = `zapBtn=0 fallback=${fbCount}`
      pass = false // Cannot confirm panel without finding button
    }

    await page.screenshot({ path: resolve(outDir, shot) })
  } catch (e) {
    notes = String(e).split('\n')[0]
  }

  checks.push({ name: 'canned-responses-panel', pass, screenshot: `screenshots/helpdesk/${shot}`, notes })
  await ctx.close()
}

// ─────────────────────────────────────────────────────────
// Check 4: Error-State / Loading-Skeleton bei API offline
// ─────────────────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()

  const shot = 'helpdesk-api-offline.png'
  let pass = false
  let notes = ''

  try {
    // Intercept all helpdesk API calls and return 503
    await page.route('**/api/v1/helpdesk/**', (route) => {
      route.fulfill({ status: 503, body: '{"error":"service unavailable"}' })
    })

    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded', timeout: 25000 })
    await page.waitForTimeout(3500)

    const bodyText = await page.evaluate(() => document.body.innerText)

    // Expect either: loading skeleton, error banner, empty-state, or toast notification
    const hasSkeleton = await page.locator('[class*="skeleton"], [class*="animate-pulse"]').count()
    const hasErrorBanner = bodyText.match(/fehler|error|nicht verf.gbar|service.*unavailable/i)
    const hasEmptyState = bodyText.match(/keine tickets|no tickets|kein ticket/i)
    const hasCrash = await page.locator('[data-testid="error-boundary"]').count()

    notes = `skeleton=${hasSkeleton} errorBanner=${!!hasErrorBanner} emptyState=${!!hasEmptyState} crash=${hasCrash}`
    // pass if no unhandled crash and at least one graceful fallback
    pass = hasCrash === 0 && (hasSkeleton > 0 || !!hasErrorBanner || !!hasEmptyState)

    await page.screenshot({ path: resolve(outDir, shot) })
  } catch (e) {
    notes = String(e).split('\n')[0]
  }

  checks.push({ name: 'api-offline-graceful', pass, screenshot: `screenshots/helpdesk/${shot}`, notes })
  await ctx.close()
}

await browser.close()

const allPass = checks.every((c) => c.pass)
const report = { pass: allPass, checks, pageErrors: errors.slice(0, 6) }

console.log(JSON.stringify(report, null, 2))
if (!allPass) {
  process.exit(1)
}