// G0-10: der Buchungslink in der Seiten-Vorschau muss auf die Astro-Seite
// (zentria.tech/book/…) zeigen, nicht auf die nicht existente Subdomain
// booking.zentria.tech.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5173'
const outDir = resolve('.qa-etappe1')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const PREP = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
  try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}
`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(PREP)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

await page.goto(`${FE}/#/kalender`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(3000)

const out = { steps: [] }
const tab = page.getByRole('tab', { name: /terminbuchung/i }).or(page.getByText(/terminbuchung/i)).first()
await tab.click()
await page.waitForTimeout(2200)
await page.screenshot({ path: resolve(outDir, 'booking-tab.png'), fullPage: false })

// "Vorschau" renders BookingPagePreview — that is where the shareable link sits.
const previewBtn = page.getByRole('button', { name: /vorschau/i }).first()
if (await previewBtn.count()) {
  await previewBtn.click()
  await page.waitForTimeout(2200)
  await page.screenshot({ path: resolve(outDir, 'booking-preview.png'), fullPage: false })
  out.steps.push('opened preview')
} else {
  out.steps.push('preview button not found')
}

// The link is carried in href/title attributes, not in visible text — the
// button only shows the slug.
const html = await page.content()
out.mentionsOldDomain = html.includes('booking.zentria.tech')
out.mentionsNewDomain = /zentria\.tech\/book/.test(html)
out.hrefs = await page.locator('a[href*="zentria"]').evaluateAll((els) =>
  els.map((e) => e.getAttribute('href')),
)
out.urlSnippets = [...new Set(html.match(/https?:\/\/[^\s"'<>]*zentria[^\s"'<>]*/g) ?? [])].slice(0, 8)
out.pageErrors = errors.slice(0, 5)

await browser.close()
console.log(JSON.stringify(out, null, 2))
