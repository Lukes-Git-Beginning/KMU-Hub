// QA KO-9: channel management — create + edit (rename) persist and show in the list.
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

  // --- Create channel ---
  await page.locator('button:has(svg.lucide-plus)').first().click({ timeout: 5000 }).catch((e) => { out.createBtnErr = String(e).split('\n')[0] })
  await page.waitForTimeout(600)
  out.createDialogOpen = await page.getByText(/Neuen Channel erstellen/).first().isVisible().catch(() => false)
  await page.locator('#channel-name').fill('qa-neu-kanal').catch((e) => { out.nameFillErr = String(e).split('\n')[0] })
  await page.getByRole('button', { name: /^Erstellen$/ }).first().click({ timeout: 3000 }).catch((e) => { out.createSubmitErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)
  out.newChannelInList = await page.getByText('qa-neu-kanal').first().isVisible().catch(() => false)
  // switch away + back to confirm persistence
  await page.getByText('vertrieb', { exact: true }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(500)
  out.newChannelPersists = await page.getByText('qa-neu-kanal').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko9-1-created.png'), fullPage: false })

  // --- Edit channel (allgemein = owner -> can edit) ---
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.getByRole('button', { name: /Kanal-Menü/ }).first().click({ timeout: 4000 }).catch((e) => { out.menuErr = String(e).split('\n')[0] })
  await page.waitForTimeout(400)
  out.editMenuVisible = await page.getByText(/^Kanal bearbeiten$/).first().isVisible().catch(() => false)
  await page.getByText(/^Kanal bearbeiten$/).first().click({ timeout: 3000 }).catch((e) => { out.editOpenErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  out.editDialogOpen = await page.getByText(/Name, Beschreibung und Sichtbarkeit/).first().isVisible().catch(() => false)
  await page.locator('#edit-channel-name').fill('allgemein-umbenannt').catch(() => {})
  await page.getByRole('button', { name: /Speichern/ }).first().click({ timeout: 3000 }).catch((e) => { out.saveErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)
  out.renameApplied = await page.getByText('allgemein-umbenannt').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko9-2-renamed.png'), fullPage: false })

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
