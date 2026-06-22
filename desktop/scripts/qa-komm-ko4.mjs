// QA KO-4: bookmark a message, see it in the bookmarks panel, persist across switch.
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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2800)
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)

  // Hover a real message bubble and click its bookmark button
  const msg = page.locator('.group.relative.flex.items-start').nth(1)
  await msg.scrollIntoViewIfNeeded().catch(() => {})
  await msg.hover().catch(() => {})
  await page.waitForTimeout(500)
  const bookmarkBtn = msg.locator('button:has(svg.lucide-bookmark)').first()
  out.bookmarkBtnVisible = await bookmarkBtn.isVisible().catch(() => false)
  await bookmarkBtn.click({ timeout: 3000 }).catch((e) => { out.bmClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)

  // Open bookmarks panel via sidebar "Lesezeichen" entry
  await page.getByRole('button', { name: /Lesezeichen/ }).first().click({ timeout: 4000 }).catch((e) => { out.panelOpenErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  out.panelOpen = await page.getByText(/gespeichert/).first().isVisible().catch(() => false)
  out.panelHasItem = await page.locator('.border-l').getByText(/#allgemein/).first().isVisible().catch(() => false)
  out.emptyShown = await page.getByText(/Noch keine gespeicherten/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko4-1-bookmarks-panel.png'), fullPage: false })

  // Switch channel and back — bookmark must persist
  await page.getByText('entwicklung', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: /Lesezeichen/ }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(600)
  out.persistsAfterSwitch = await page.locator('.border-l').getByText(/#allgemein/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko4-2-persist.png'), fullPage: false })

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
