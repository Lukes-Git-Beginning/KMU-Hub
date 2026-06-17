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
    await page.waitForTimeout(1000)
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
  // ── Schulungen tab (Zustand store — verify it works) ──────────────────
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3000)
  out.trainingsTabClicked = await clickByText(page, 'Schulungen')
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'team-trainings.png'), fullPage: true })

  // ── Deactivate: open first member's action menu and deactivate ────────
  await clickByText(page, 'Mitglieder')
  await page.waitForTimeout(1200)
  // count rows before
  out.membersBefore = await page.locator('text=/Inaktiv/').count()
  await page.screenshot({ path: resolve(outDir, 'team-members-before.png'), fullPage: true })

  // open the first action menu (3-dots, Radix DropdownMenu — needs a real click)
  const menuBtn = page.locator('button:has(svg.lucide-ellipsis-vertical)').first()
  out.menuBtnFound = await page.locator('button:has(svg.lucide-ellipsis-vertical)').count()
  if (await menuBtn.count()) {
    await menuBtn.scrollIntoViewIfNeeded()
    await menuBtn.click({ force: true })
    await page.waitForTimeout(800)
  }
  out.menuItems = await page.getByRole('menuitem').allInnerTexts().catch(() => [])
  // click "Deaktivieren" menu item
  const deact = page.getByRole('menuitem', { name: /Deaktivieren|Deactivate/i }).first()
  out.deactivateFound = await deact.count()
  if (await deact.count()) {
    await deact.click()
    await page.waitForTimeout(700)
    // confirm dialog
    const confirm = page.getByRole('button', { name: /Deaktivieren|Bestätigen|Confirm/i }).last()
    if (await confirm.count()) { await confirm.click(); await page.waitForTimeout(1500) }
  }
  out.membersAfter = await page.locator('text=/Inaktiv/').count()
  await page.screenshot({ path: resolve(outDir, 'team-members-after.png'), fullPage: true })

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
