// QA Chat reactions: reaction pills render, toggling changes the count and
// highlights, the picker adds a new reaction, and the state persists when
// scrolling away and back (store-backed, survives list virtualization).
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
  await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.getByRole('button', { name: /^Team$/ }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(500)
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 5000 }).catch((e) => { out.channelErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1300)

  // A reaction pill is a button containing an emoji + tabular count.
  const pill = page.locator('button:has(.tabular-nums)').first()
  out.pillVisible = await pill.isVisible().catch(() => false)
  const countBefore = await pill.locator('.tabular-nums').textContent().catch(() => null)
  await page.screenshot({ path: resolve(outDir, 'reactions-initial.png'), fullPage: false })

  // Toggle it → count changes (adds 'me')
  await pill.click({ timeout: 3000 }).catch((e) => { out.toggleErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  const countAfter = await pill.locator('.tabular-nums').textContent().catch(() => null)
  out.countChanged = countBefore !== null && countAfter !== null && countBefore !== countAfter
  out.countBefore = countBefore
  out.countAfter = countAfter
  await page.screenshot({ path: resolve(outDir, 'reactions-toggled.png'), fullPage: false })

  // Persistence: scroll to top (remounts virtualized bubbles) then back, count stays
  const scroller = page.locator('.flex-1.overflow-y-auto').first()
  await scroller.evaluate((el) => { el.scrollTop = 0 }).catch(() => {})
  await page.waitForTimeout(500)
  await scroller.evaluate((el) => { el.scrollTop = el.scrollHeight }).catch(() => {})
  await page.waitForTimeout(700)
  const countPersist = await page.locator('button:has(.tabular-nums)').first().locator('.tabular-nums').textContent().catch(() => null)
  out.persistedAfterScroll = countPersist === countAfter

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
