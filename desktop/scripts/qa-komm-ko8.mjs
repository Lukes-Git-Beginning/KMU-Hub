// QA KO-8: inbox slash commands — /umfrage (poll) + /erinnerung (reminder) are
// real and appear in the thread; voting updates; /giphy is a labelled stub.
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
  await page.goto(`${BASE}/#/kommunikation?bereich=posteingang`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2800)

  // Select the first conversation (list items are buttons with a bg-primary-light avatar)
  const firstConv = page.locator('button').filter({ has: page.locator('.bg-primary-light') }).first()
  out.convCount = await page.locator('button').filter({ has: page.locator('.bg-primary-light') }).count()
  await firstConv.click({ timeout: 5000 }).catch((e) => { out.convClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(900)
  out.composerVisible = await page.locator('textarea').last().isVisible().catch(() => false)

  // Open slash palette and pick "Umfrage"
  const composer = page.locator('textarea').last()
  await composer.fill('/').catch((e) => { out.composerErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  out.slashPaletteVisible = await page.getByText(/Befehle/).first().isVisible().catch(() => false)
  await page.getByRole('button').filter({ hasText: /\/umfrage/ }).first().click({ timeout: 3000 }).catch((e) => { out.pollOpenErr = String(e).split('\n')[0] })
  await page.waitForTimeout(600)
  out.pollDialogOpen = await page.getByText(/Umfrage erstellen/).first().isVisible().catch(() => false)

  // Fill poll
  await page.getByPlaceholder(/Worüber soll abgestimmt/).fill('Mittagessen?').catch(() => {})
  const optionInputs = page.getByPlaceholder(/Option \d/)
  await optionInputs.nth(0).fill('Pizza').catch(() => {})
  await optionInputs.nth(1).fill('Sushi').catch(() => {})
  await page.getByRole('button', { name: /Umfrage senden/ }).first().click({ timeout: 3000 }).catch((e) => { out.pollSendErr = String(e).split('\n')[0] })
  await page.waitForTimeout(900)
  out.pollInThread = await page.getByText('Mittagessen?').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko8-1-poll.png'), fullPage: false })

  // Vote on the poll
  await page.getByRole('button').filter({ hasText: /Pizza/ }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(600)
  out.voteRegistered = await page.getByText(/1 Stimme/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko8-2-vote.png'), fullPage: false })

  // Reminder via slash
  await composer.fill('/').catch(() => {})
  await page.waitForTimeout(400)
  await page.getByRole('button').filter({ hasText: /\/erinnerung/ }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(500)
  out.reminderDialogOpen = await page.getByText(/Erinnerung erstellen/).first().isVisible().catch(() => false)
  await page.getByPlaceholder(/Woran möchtest du erinnert/).fill('Angebot nachfassen').catch(() => {})
  await page.getByRole('button', { name: /Erinnerung setzen/ }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(800)
  out.reminderInThread = await page.getByText('Angebot nachfassen').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko8-3-reminder.png'), fullPage: false })

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
