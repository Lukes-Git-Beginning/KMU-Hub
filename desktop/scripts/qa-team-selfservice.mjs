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

  out.selfTabClicked = await clickByText(page, 'Self-Service')
  await page.waitForTimeout(1500)

  // Profile tab — identity must be real now (Stefan Vogel), not the fake "Jonas Diaz".
  out.hasStefan = (await page.getByText('Stefan Vogel').count()) > 0
  out.hasJonasDiaz = (await page.getByText('Jonas Diaz').count()) > 0
  await page.screenshot({ path: resolve(outDir, 'team-selfservice-profile.png'), fullPage: true })

  // Requests tab — Stefan's own requests
  await clickByText(page, 'Meine Anträge')
  out.hasSommerurlaub = (await page.getByText(/Sommerurlaub/i).count()) > 0
  await page.screenshot({ path: resolve(outDir, 'team-selfservice-requests.png'), fullPage: true })

  // Open the request dialog
  const newBtn = page.getByText('Antrag stellen').first()
  out.dialogTitleFound = await newBtn.count()
  await page.screenshot({ path: resolve(outDir, 'team-selfservice-requests2.png'), fullPage: true })

  // Salary tab
  await clickByText(page, 'Gehaltsabrechnungen')
  await page.screenshot({ path: resolve(outDir, 'team-selfservice-salary.png'), fullPage: true })

  // ── Create flow: open dialog, fill, submit, expect a new request to appear ──
  await clickByText(page, 'Meine Anträge')
  await page.waitForTimeout(600)
  await clickByText(page, 'Neuer Antrag')
  await page.waitForTimeout(800)
  const trigger = page.locator('[role="combobox"]').first()
  if (await trigger.count()) {
    await trigger.click()
    await page.waitForTimeout(400)
    const opt = page.locator('[role="option"]').first()
    if (await opt.count()) { await opt.click(); await page.waitForTimeout(300) }
  }
  const dates = page.locator('input[type="date"]')
  if ((await dates.count()) >= 2) {
    await dates.nth(0).fill('2026-09-01')
    await dates.nth(1).fill('2026-09-05')
  }
  await clickByText(page, 'Antrag einreichen')
  await page.waitForTimeout(1600)
  out.hasNewSeptember = (await page.getByText(/01\.09\.2026/).count()) > 0
  await page.screenshot({ path: resolve(outDir, 'team-selfservice-aftercreate.png'), fullPage: true })

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
