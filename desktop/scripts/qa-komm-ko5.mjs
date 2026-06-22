// QA KO-5: file attachments — image preview + real download.
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

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(chat|common|kommunikation)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, acceptDownloads: true })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2800)
  await page.getByText('design', { exact: true }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1200)

  // Image attachment preview (png placeholder) + document card
  out.imagePreviewVisible = await page.locator('img[alt="dashboard-mockup-v3.png"]').first().isVisible().catch(() => false)
  out.docCardVisible = await page.getByText('design-spezifikation.pdf').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko5-1-attachments.png'), fullPage: false })

  // Download the document — the doc download button is always visible (aria "Herunterladen")
  const dlBtn = page.getByRole('button', { name: 'Herunterladen' }).first()
  out.downloadBtnVisible = await dlBtn.isVisible().catch(() => false)
  const [download] = await Promise.all([
    page.waitForEvent('download', { timeout: 5000 }).catch(() => null),
    dlBtn.click({ timeout: 3000 }).catch(() => {}),
  ])
  out.downloadTriggered = !!download
  out.downloadFilename = download ? download.suggestedFilename() : null

  await page.screenshot({ path: resolve(outDir, 'ko5-2-download.png'), fullPage: false })
  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
