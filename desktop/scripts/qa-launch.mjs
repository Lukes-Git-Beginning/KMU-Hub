// QA: Cosmi Launch-/Login-Screen — Intro-Phasen, Settled (Logo rechts / Login
// links), Reduced-Motion (sofort settled) und Schmalfenster (gestapelt).
// Route /#/launch-preview (dev-only, ohne GuestRoute) gegen den vite.qa-Server :5174.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5174'
const ROUTE = `${BASE}/#/launch-preview`
const outDir = resolve('.qa-screenshots')

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

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(auth|common|security)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = { pageErrors: [] }

async function shot(page, name) {
  await page.screenshot({ path: resolve(outDir, `launch-${name}.png`), fullPage: false })
}

try {
  // ── 1) Desktop: Intro-Phasen + Settled ────────────────────────────────
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(SUPPRESS_ONBOARDING)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => out.pageErrors.push(String(e).split('\n')[0]))

  await page.goto(ROUTE, { waitUntil: 'domcontentloaded', timeout: 25000 })
  // Intro-Frames (Animationszeitachse: C ~ bis 3.7s, OSMI ~4.7–6.6s, settle ~7–8s)
  await page.waitForTimeout(1700);  await shot(page, '1-gather')
  await page.waitForTimeout(2100);  await shot(page, '2-cform')      // ~3.8s
  await page.waitForTimeout(1700);  await shot(page, '3-osmi')       // ~5.5s
  await page.waitForTimeout(1500);  await shot(page, '4-wordmark')   // ~7.0s (settle startet)
  await page.waitForTimeout(2000);  await shot(page, '5-settled')    // ~9.0s (Logo rechts / Login links)

  // Settled-Checks
  out.emailVisible = await page.getByLabel(/e-?mail/i).first().isVisible().catch(() => false)
  out.passwortVisible = await page.locator('input[type="password"]').first().isVisible().catch(() => false)
  out.anmeldenVisible = await page.getByRole('button', { name: /anmelden|login/i }).first().isVisible().catch(() => false)
  out.rawKeys = await scanRawKeys(page)
  await ctx.close()

  // ── 2) Reduced-Motion: sofort settled, kein Intro ─────────────────────
  const ctxR = await browser.newContext({ viewport: { width: 1440, height: 900 }, reducedMotion: 'reduce' })
  await ctxR.addInitScript(ELECTRON_STUB)
  await ctxR.addInitScript(SUPPRESS_ONBOARDING)
  const pageR = await ctxR.newPage()
  pageR.on('pageerror', (e) => out.pageErrors.push('reduced:' + String(e).split('\n')[0]))
  await pageR.goto(ROUTE, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await pageR.waitForTimeout(1500)
  await shot(pageR, '6-reduced-motion')
  out.reducedSettledImmediately = await pageR.getByRole('button', { name: /anmelden|login/i }).first().isVisible().catch(() => false)
  await ctxR.close()

  // ── 3) Schmalfenster: gestapelt ───────────────────────────────────────
  const ctxN = await browser.newContext({ viewport: { width: 820, height: 900 } })
  await ctxN.addInitScript(ELECTRON_STUB)
  await ctxN.addInitScript(SUPPRESS_ONBOARDING)
  const pageN = await ctxN.newPage()
  pageN.on('pageerror', (e) => out.pageErrors.push('narrow:' + String(e).split('\n')[0]))
  await pageN.goto(ROUTE, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await pageN.waitForTimeout(9000)
  await shot(pageN, '7-narrow-settled')
  await ctxN.close()

  // ── 4) Fall A — Startup-Splash auf #/ (Logo-Intro → Fly-in → Dashboard) ──
  // DEV_BYPASS_AUTH (vite dev) gilt als „mit Token": initStartupLaunch() legt
  // das Overlay vor dem ersten Frame über das ladende Dashboard.
  const ctxA = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctxA.addInitScript(ELECTRON_STUB)
  await ctxA.addInitScript(SUPPRESS_ONBOARDING)
  const pageA = await ctxA.newPage()
  pageA.on('pageerror', (e) => out.pageErrors.push('startup:' + String(e).split('\n')[0]))
  await pageA.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  out.startupOverlayAtBoot = await pageA.locator('.cosmi-startup').first().isVisible().catch(() => false)
  await pageA.waitForTimeout(1700);  await shot(pageA, 'A1-intro-gather')
  await pageA.waitForTimeout(2100);  await shot(pageA, 'A2-intro-cform')     // ~3.8s
  await pageA.waitForTimeout(2600);  await shot(pageA, 'A3-intro-wordmark')  // ~6.4s
  await pageA.waitForTimeout(1000);  await shot(pageA, 'A4-flyin')           // ~7.4s (Zoom)
  await pageA.waitForTimeout(1600);  await shot(pageA, 'A5-revealed')        // Overlay weg
  out.startupOverlayGone = !(await pageA.locator('.cosmi-startup').first().isVisible().catch(() => true))
  await ctxA.close()

  // ── 5) Fall B — geteilte Fly-in-Animation isoliert (#/flyin-preview) ─────
  const ctxB = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctxB.addInitScript(ELECTRON_STUB)
  await ctxB.addInitScript(SUPPRESS_ONBOARDING)
  const pageB = await ctxB.newPage()
  pageB.on('pageerror', (e) => out.pageErrors.push('flyin:' + String(e).split('\n')[0]))
  await pageB.goto(`${BASE}/#/flyin-preview`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await pageB.waitForTimeout(160);  await shot(pageB, 'B1-flyin-start')
  await pageB.waitForTimeout(560);  await shot(pageB, 'B2-flyin-mid')
  await pageB.waitForTimeout(1300); await shot(pageB, 'B3-revealed')
  out.flyinOverlayGone = !(await pageB.locator('.cosmi-startup').first().isVisible().catch(() => true))
  await ctxB.close()
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.pageErrors = out.pageErrors.slice(0, 8)
await browser.close()
console.log(JSON.stringify(out, null, 2))
