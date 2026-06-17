import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

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

async function clickByText(page, text) {
  let loc = page.getByRole('button', { name: text }).first()
  if (!(await loc.count())) loc = page.locator(`button:has-text(${JSON.stringify(text)})`).first()
  if (!(await loc.count())) loc = page.getByText(text, { exact: true }).first()
  if (await loc.count()) {
    await loc.evaluate((el) => el.click())
    await page.waitForTimeout(1200)
    return true
  }
  return false
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const downloads = []
page.on('download', (d) => downloads.push(d.suggestedFilename()))
const out = {}

try {
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3000)

  out.tabClicked = await clickByText(page, 'Personalakte')
  await page.waitForTimeout(1500)

  // Real employee names (Stefan Vogel) instead of fake (Anna Müller / Jonas Diaz)?
  out.hasStefan = (await page.getByText('Stefan Vogel').count()) > 0
  out.hasAnnaMueller = (await page.getByText('Anna Müller').count()) > 0
  out.hasJonasDiaz = (await page.getByText('Jonas Diaz').count()) > 0
  await page.screenshot({ path: resolve(outDir, 'team-pdocs-list.png'), fullPage: true })

  // Preview: click first Eye button
  const eyeBtn = page.locator('button[title]').filter({ has: page.locator('svg') }).first()
  // more robust: click a button with the preview title
  const previewBtn = page.locator('button').filter({ hasText: '' }).nth(0)
  // open preview via the first row's preview button (title attribute)
  const firstPreview = page.locator('button[title="Vorschau"], button[title="Preview"]').first()
  if (await firstPreview.count()) {
    await firstPreview.evaluate((el) => el.click())
    await page.waitForTimeout(900)
  }
  out.previewBadgeVisible = (await page.getByText(/Demo-Vorschau/i).count()) > 0
  await page.screenshot({ path: resolve(outDir, 'team-pdocs-preview.png'), fullPage: true })

  // Download from the preview footer — must produce a real file (no toast-stub).
  const dlPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null)
  await clickByText(page, 'Herunterladen')
  const dl = await dlPromise
  out.downloadFile = dl ? dl.suggestedFilename() : null

  const text = await page.evaluate(() => document.body.innerText)
  out.rawKeys = [...new Set([...text.matchAll(/\b(team|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
  out.doubleBraces = (text.match(/\{\{[a-zA-Z]/g) || []).length
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.downloads = downloads
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
