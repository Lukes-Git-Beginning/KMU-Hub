// QA KO-3: threads have real seeded replies and new replies persist.
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

  // Click the first thread indicator (MessageSquare button with reply count text)
  const threadIndicator = page.locator('button:has(svg.lucide-message-square)').filter({ hasText: /Antwort/ }).first()
  out.threadIndicatorVisible = await threadIndicator.isVisible().catch(() => false)
  await threadIndicator.click({ timeout: 4000 }).catch((e) => { out.threadClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)

  // Thread panel open with seeded replies (not the empty state)
  out.threadPanelOpen = await page.getByText(/^Thread$|Thread$/).first().isVisible().catch(() => false)
  out.threadHasReplies = await page.locator('text=/Klingt gut|Danke für|Daily besprechen|Sehe ich genauso|Ist erledigt|Guter Punkt|Unterlagen|Top, dann/').first().isVisible().catch(() => false)
  out.threadEmptyShown = await page.getByText(/Noch keine Antworten|No replies/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko3-1-thread-open.png'), fullPage: false })

  // Send a reply in the thread (thread input is the last textarea)
  const replyInput = page.locator('textarea').last()
  await replyInput.fill('THREAD-REPLY-9988').catch((e) => { out.replyFillErr = String(e).split('\n')[0] })
  await replyInput.press('Enter').catch(() => {})
  await page.waitForTimeout(1200)
  out.replyAppears = await page.getByText('THREAD-REPLY-9988').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko3-2-reply-sent.png'), fullPage: false })

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
