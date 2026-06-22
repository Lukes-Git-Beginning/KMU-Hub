// QA KO-7: unified unread inbox lists channels/DMs with unread; mark-all clears.
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

  // Open the unified unread inbox immediately (before reading any channel)
  await page.getByRole('button', { name: /Alle ungelesenen/ }).first().click({ timeout: 5000 }).catch((e) => { out.openErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  out.panelOpen = await page.getByText(/ungelesene Nachricht/).first().isVisible().catch(() => false)
  // entries for allgemein + entwicklung should be listed in the panel
  out.hasAllgemein = await page.locator('.border-l').getByText('allgemein').first().isVisible().catch(() => false)
  out.hasEntwicklung = await page.locator('.border-l').getByText('entwicklung').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko7-1-inbox.png'), fullPage: false })

  // Click an entry -> opens that channel
  await page.locator('.border-l button').filter({ hasText: 'allgemein' }).first().click({ timeout: 4000 }).catch((e) => { out.entryClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(900)
  out.channelOpened = await page.locator('textarea').last().isVisible().catch(() => false)

  // Reopen inbox and mark all read
  await page.getByRole('button', { name: /Alle ungelesenen/ }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: /Alle gelesen/ }).first().click({ timeout: 4000 }).catch((e) => { out.markAllErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  out.emptyAfterMarkAll = await page.getByText(/Alles gelesen/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko7-2-allread.png'), fullPage: false })

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
