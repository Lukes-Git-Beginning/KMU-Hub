// QA KO-2: group DMs. Open the new-message dialog, pick multiple people,
// create a group DM, verify it shows in the sidebar and accepts messages.
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

  // Open the new-message dialog (PenSquare button, aria-label "Neue Nachricht")
  await page.getByRole('button', { name: 'Neue Nachricht' }).first().click({ timeout: 5000 }).catch((e) => { out.openErr = String(e).split('\n')[0] })
  await page.waitForTimeout(700)
  out.dialogVisible = await page.getByText(/Wähle eine Person|Mehrere Personen/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko2-1-dialog.png'), fullPage: false })

  // Pick two people from the directory list
  const peopleButtons = page.locator('[role="dialog"] button').filter({ hasText: /\w+ \w+/ })
  await peopleButtons.nth(0).click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(300)
  await peopleButtons.nth(1).click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(400)
  out.groupHintShown = await page.getByText(/Mehrere Personen ausgewählt/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko2-2-selected.png'), fullPage: false })

  // Submit — button text "Gruppe starten (2)"
  await page.getByRole('button', { name: /Gruppe starten/ }).first().click({ timeout: 4000 }).catch((e) => { out.submitErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  // A group DM should now be open (Users icon in sidebar) and selectable
  out.groupOpened = await page.locator('.lucide-users').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko2-3-group-open.png'), fullPage: false })

  // Send a message into the group
  const composer = page.locator('textarea').last()
  await composer.fill('Gruppen-Hallo zusammen').catch(() => {})
  await composer.press('Enter').catch(() => {})
  await page.waitForTimeout(1000)
  out.groupMessageSent = await page.getByText('Gruppen-Hallo zusammen').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko2-4-group-message.png'), fullPage: false })

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
