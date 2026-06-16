// QA Auth-Reset: forgot-password + reset-password (with/without token).
// Verifies layout, i18n (no raw keys), client→MSW path, missing-token guard.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
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

function scanRawKeys(page) {
  return page.evaluate(() =>
    [...new Set([...document.body.innerText.matchAll(/\b(auth|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))],
  )
}
const bodyText = (page) => page.evaluate(() => document.body.innerText.replace(/\n{2,}/g, '\n').slice(0, 320))

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  // 1. Forgot-password page
  await page.goto(`${BASE}/#/forgot-password`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(1800)
  await page.screenshot({ path: resolve(outDir, 'auth-forgot.png') })
  out.forgotRawKeys = await scanRawKeys(page)
  out.forgotHasEmail = await page.locator('#email').count()
  out.forgotText = await bodyText(page)

  // fill + submit -> enumeration-safe "sent" state
  await page.fill('#email', 'pilot@example.com')
  await page.locator('button[type="submit"]').first().click()
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'auth-forgot-sent.png') })
  out.forgotSentText = await bodyText(page)

  // 2. Reset-password WITH token
  await page.goto(`${BASE}/#/reset-password?token=demo-token-123`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, 'auth-reset.png') })
  out.resetRawKeys = await scanRawKeys(page)
  out.resetHasPassword = await page.locator('#password').count()
  out.resetHasConfirm = await page.locator('#confirmPassword').count()
  out.resetText = await bodyText(page)

  // 3. Reset-password WITHOUT token -> missing-token guard
  await page.goto(`${BASE}/#/reset-password`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, 'auth-reset-notoken.png') })
  out.noTokenText = await bodyText(page)
  out.noTokenRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
