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
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.pageErrors = out.pageErrors.slice(0, 8)
await browser.close()
console.log(JSON.stringify(out, null, 2))
