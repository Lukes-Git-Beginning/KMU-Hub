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
const out = {}

try {
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3000)
  out.titleText = await page.locator('h1').first().innerText().catch(() => '')

  // Organigramm tab → select a node to reveal the email/call buttons
  out.orgTabClicked = await clickByText(page, 'Organigramm')
  await page.waitForTimeout(1200)
  // click first org node card (a name) to open the detail panel
  const node = page.getByText(/Stefan Vogel|Vogel/).first()
  if (await node.count()) { await node.evaluate((el) => el.click()); await page.waitForTimeout(1000) }
  out.hasEmailBtn = (await page.getByText('E-Mail', { exact: true }).count()) > 0
  await page.screenshot({ path: resolve(outDir, 'team-orgchart.png'), fullPage: true })

  const text = await page.evaluate(() => document.body.innerText)
  out.rawKeys = [...new Set([...text.matchAll(/\b(team|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
  out.doubleBraces = (text.match(/\{\{[a-zA-Z]/g) || []).length
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
