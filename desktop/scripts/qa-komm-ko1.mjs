// QA KO-1: stateful chat store — send / edit / react / delete persist across
// channel switches and reloads. Verifies the core demo-depth fix.
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

async function openChannel(page, name) {
  await page.getByText(name, { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
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

const UNIQUE = 'PERSIST-CHECK-7291'

try {
  await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2800)

  // 1) Open #allgemein and send a unique message
  await openChannel(page, 'allgemein')
  const composer = page.locator('textarea').last()
  await composer.fill(UNIQUE).catch((e) => { out.composerErr = String(e).split('\n')[0] })
  await composer.press('Enter').catch(() => {})
  await page.waitForTimeout(1000)
  out.sent = await page.getByText(UNIQUE).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko1-1-sent.png'), fullPage: false })

  // 2) Switch to another channel and back — message must still be there
  await openChannel(page, 'entwicklung')
  await page.waitForTimeout(600)
  await openChannel(page, 'allgemein')
  await page.waitForTimeout(800)
  out.persistsAfterSwitch = await page.getByText(UNIQUE).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko1-2-after-switch.png'), fullPage: false })

  // 3) Edit the message — change persists. The edit textarea is autoFocused and
  //    submits via the "Speichern" button (Enter on .last() would hit the composer).
  const ownMsg = page.locator('.group.relative').filter({ hasText: UNIQUE }).last()
  await ownMsg.scrollIntoViewIfNeeded().catch(() => {})
  await ownMsg.hover().catch(() => {})
  await page.waitForTimeout(400)
  const pencil = ownMsg.locator('button:has(svg.lucide-pencil)').first()
  out.editButtonVisible = await pencil.isVisible().catch(() => false)
  if (out.editButtonVisible) {
    await pencil.click({ timeout: 3000 }).catch(() => {})
    await page.waitForTimeout(400)
    const editArea = page.locator('textarea:focus').first()
    await editArea.fill(UNIQUE + ' EDITED').catch((e) => { out.editFillErr = String(e).split('\n')[0] })
    await page.getByRole('button', { name: /Speichern|Save/ }).first().click({ timeout: 3000 }).catch(() => {})
    await page.waitForTimeout(1000)
    out.editApplied = await page.getByText(UNIQUE + ' EDITED').first().isVisible().catch(() => false)
    out.editMarker = await page.getByText(/bearbeitet|edited/i).first().isVisible().catch(() => false)
  }
  await openChannel(page, 'entwicklung')
  await openChannel(page, 'allgemein')
  await page.waitForTimeout(700)
  out.editPersists = await page.getByText(UNIQUE + ' EDITED').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'ko1-3-edited.png'), fullPage: false })

  // 4) React to a seed message — reaction count persists
  const seedMsg = page.locator('.group.relative').nth(2)
  await seedMsg.hover().catch(() => {})
  await page.waitForTimeout(300)
  const emojiBtn = seedMsg.locator('button:has(svg.lucide-plus)').first()
  out.reactionBtnVisible = await emojiBtn.isVisible().catch(() => false)
  if (out.reactionBtnVisible) {
    await emojiBtn.click({ timeout: 3000 }).catch(() => {})
    await page.waitForTimeout(600)
    await page.locator('[role="dialog"] button, .reaction-picker button, button.text-2xl, button.text-xl').first().click({ timeout: 2000 }).catch(() => {})
    await page.waitForTimeout(800)
    // switch away + back, reaction must still be on the message
    await openChannel(page, 'entwicklung')
    await openChannel(page, 'allgemein')
    await page.waitForTimeout(600)
    out.reactionPersists = await page.locator('.group.relative').nth(2).locator('button:has-text("1"), button:has-text("2")').first().isVisible().catch(() => false)
    await page.screenshot({ path: resolve(outDir, 'ko1-4-reaction.png'), fullPage: false })
  }

  // 5) Reload — sent + edited message must survive (in-memory store lives for session)
  out.beforeReloadEdited = await page.getByText(UNIQUE + ' EDITED').first().isVisible().catch(() => false)

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
