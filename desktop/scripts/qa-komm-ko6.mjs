// QA KO-6: search results jump to the message; unread badge clears and stays cleared.
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

  // --- Unread: #allgemein starts with a badge; open it -> badge clears ---
  const allgemeinRow = page.locator('button', { hasText: /^#?\s*allgemein/ }).first()
  out.unreadBadgeBefore = await page.locator('button:has-text("allgemein") .bg-destructive, button:has-text("allgemein") [class*="destructive"]').first().isVisible().catch(() => false)
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1200)
  // switch away and back
  await page.getByText('vertrieb', { exact: true }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  // After reading, allgemein badge should be gone
  out.allgemeinBadgeGone = !(await page.locator('button:has-text("allgemein") [class*="destructive"]').first().isVisible().catch(() => false))
  await page.screenshot({ path: resolve(outDir, 'ko6-1-unread.png'), fullPage: false })

  // --- Search jump ---
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(800)
  const searchBtn = page.locator('button:has(svg.lucide-search)').last()
  await searchBtn.click({ timeout: 4000 }).catch((e) => { out.searchBtnErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  await page.getByText(/In allen Channels/).first().click({ timeout: 2000 }).catch(() => {})
  const searchInput = page.getByPlaceholder(/Suchbegriff|Nachrichten|durchsuchen/i).first()
  await searchInput.fill('Meeting').catch((e) => { out.searchFillErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.searchHasResults = await page.locator('mark').first().isVisible().catch(() => false)
  // Click the first result button (in the search panel on the right)
  const firstResult = page.locator('.border-l button').filter({ has: page.locator('mark') }).first()
  await firstResult.click({ timeout: 4000 }).catch((e) => { out.resultClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(700)
  // The flashed message wrapper carries bg-warning/15
  out.jumpFlashVisible = await page.locator('[class*="bg-warning"]').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko6-2-jump.png'), fullPage: false })

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
