// QA Mentions inbox: open the mentions panel from the channel sidebar,
// verify mock mentions render with channel name + broadcast badge, click
// a mention to jump to its channel, scan raw keys / page errors.
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
  await page.waitForTimeout(600)

  // Open mentions panel from sidebar entry
  out.entryVisible = await page.getByRole('button', { name: /Erwähnungen/ }).first().isVisible().catch(() => false)
  await page.getByRole('button', { name: /Erwähnungen/ }).first().click({ timeout: 4000 }).catch((e) => { out.openErr = String(e).split('\n')[0] })
  await page.waitForTimeout(900)

  out.panelTitleVisible = await page.getByText(/^Erwähnungen$/).first().isVisible().catch(() => false)
  out.mentionContentVisible = await page.getByText(/Auth-Refactor/).first().isVisible().catch(() => false)
  out.broadcastBadgeVisible = await page.getByText(/^An alle$/).first().isVisible().catch(() => false)
  out.channelTagVisible = await page.getByText(/#entwicklung/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'mentions-panel.png'), fullPage: false })
  out.panelRawKeys = await scanRawKeys(page)

  // Click a mention → jumps to its channel (entwicklung)
  await page.getByText(/Auth-Refactor/).first().click({ timeout: 4000 }).catch((e) => { out.clickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(900)
  // After jump the channel header should show the channel name and panel closes
  out.jumpedToChannel = await page.getByRole('heading', { name: /entwicklung/ }).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'mentions-jump.png'), fullPage: false })
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
