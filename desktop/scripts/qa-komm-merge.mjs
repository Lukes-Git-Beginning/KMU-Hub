// QA Kommunikation merge: area switcher (Team|Posteingang), /chat redirect,
// single nav entry, module-settings entry, no raw keys.
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
  return [...new Set([...text.matchAll(/\b(kommunikation|chat|moduleSettings|settings|layout|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
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
  // 1. Open module
  await page.goto(`${BASE}/#/kommunikation`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  out.switcherTeam = await page.getByRole('button', { name: /^Team$/ }).first().isVisible().catch(() => false)
  out.switcherInbox = await page.getByRole('button', { name: /Posteingang/ }).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'komm-merge-default.png'), fullPage: true })

  // 2. Switch to Team area → chat layout (empty state title)
  await page.getByRole('button', { name: /^Team$/ }).first().click({ timeout: 4000 }).catch((e) => { out.teamClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  out.urlAfterTeam = await page.evaluate(() => location.hash)
  await page.screenshot({ path: resolve(outDir, 'komm-merge-team.png'), fullPage: true })
  out.teamRawKeys = await scanRawKeys(page)

  // 3. Switch back to Posteingang
  await page.getByRole('button', { name: /Posteingang/ }).first().click({ timeout: 4000 }).catch((e) => { out.inboxClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  out.urlAfterInbox = await page.evaluate(() => location.hash)

  // 4. /chat redirect
  await page.goto(`${BASE}/#/chat`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(1500)
  out.chatRedirectHash = await page.evaluate(() => location.hash)

  // 5. Single nav entry: no standalone "Chat", but "Kommunikation"
  const navTexts = await page.evaluate(() =>
    [...document.querySelectorAll('nav a, aside a, a')].map((a) => a.textContent?.trim()).filter(Boolean),
  )
  out.navHasKommunikation = navTexts.some((t) => /Kommunikation/.test(t))
  out.navHasStandaloneChat = navTexts.some((t) => /^Chat$/.test(t))

  // 6. Module settings entry
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 5000 }).catch((e) => { out.settingsOpenErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)
  await page.getByText(/^Kommunikation$/).first().click({ timeout: 4000 }).catch((e) => { out.settingsEntryErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  out.settingsPanelHasDisplay = await page.getByText(/Startbereich/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'komm-merge-settings.png'), fullPage: true })
  out.settingsRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
