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
  // ── TEAM > Abwesenheiten ────────────────────────────────────────────────
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3000)

  const tab = page.getByText(/^Abwesenheit/i).first()
  out.teamTabFound = await tab.count()
  if (out.teamTabFound) {
    await tab.evaluate((el) => el.click())
    await page.waitForTimeout(1800)
  }
  out.teamHasLena = (await page.getByText('Lena Braun').count()) > 0
  out.teamHasMarkus = (await page.getByText('Markus Weber').count()) > 0
  out.teamHasFelix = (await page.getByText('Felix Krause').count()) > 0
  await page.screenshot({ path: resolve(outDir, 'team-absences.png'), fullPage: true })

  // ── DASHBOARD > Abwesenheiten-Widget ───────────────────────────────────
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3000)
  out.dashAbsentMentions = await page.getByText(/abwesend/i).count()
  out.dashHasMarkus = (await page.getByText('Markus Weber').count()) > 0
  await page.screenshot({ path: resolve(outDir, 'dashboard-absences.png'), fullPage: true })

  // ── i18n raw-key scan ──────────────────────────────────────────────────
  const text = await page.evaluate(() => document.body.innerText)
  out.rawKeys = [...new Set([...text.matchAll(/\b(team|dashboard|hr|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
