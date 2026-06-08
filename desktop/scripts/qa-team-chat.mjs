// QA Team-Chat scharfschalten: open channel, member panel toggle, inline edit,
// auto-mark-read (unread badge clears), no raw keys / page errors.
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
  await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)

  // Open a channel (first # channel in the list)
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 5000 }).catch((e) => { out.channelClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.messagesVisible = await page.locator('.whitespace-pre-wrap').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'team-chat-channel.png'), fullPage: true })

  // Toggle member panel (Users-icon button in the channel header)
  const membersBtn = page.locator('button:has(svg.lucide-users)').first()
  await membersBtn.click({ timeout: 4000 }).catch((e) => { out.membersBtnErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  out.memberPanelVisible = await page.getByText(/Mitglieder/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'team-chat-members.png'), fullPage: true })
  // close it again
  await membersBtn.click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(500)

  // Send an own message so we have something editable, then inline-edit it.
  const composer = page.locator('textarea').last()
  await composer.fill('QA Test Nachricht').catch((e) => { out.composerErr = String(e).split('\n')[0] })
  await composer.press('Enter').catch(() => {})
  await page.waitForTimeout(1000)
  out.ownMessageSent = await page.getByText('QA Test Nachricht').first().isVisible().catch(() => false)

  // Hover the last message, look for the edit (pencil) button
  const lastMsg = page.locator('.group.relative.flex.items-start').last()
  await lastMsg.hover().catch(() => {})
  await page.waitForTimeout(400)
  const pencil = page.locator('button:has(svg.lucide-pencil)').last()
  out.editButtonPresent = await pencil.isVisible().catch(() => false)
  if (out.editButtonPresent) {
    await pencil.click({ timeout: 3000 }).catch((e) => { out.editClickErr = String(e).split('\n')[0] })
    await page.waitForTimeout(400)
    out.editTextareaVisible = await page.locator('textarea').count() > 1
    await page.screenshot({ path: resolve(outDir, 'team-chat-inline-edit.png'), fullPage: true })
    await page.keyboard.press('Escape')
  }

  // Full-text search panel
  const searchBtn = page.locator('button:has(svg.lucide-search)').last()
  await searchBtn.click({ timeout: 4000 }).catch((e) => { out.searchBtnErr = String(e).split('\n')[0] })
  await page.waitForTimeout(600)
  out.searchPanelVisible = await page.getByText(/Nachrichten durchsuchen/).first().isVisible().catch(() => false)
  // widen to all channels, then search a word that exists in mock messages
  await page.getByText(/In allen Channels suchen/).first().click({ timeout: 3000 }).catch(() => {})
  const searchInput = page.getByPlaceholder(/Suchbegriff/).first()
  await searchInput.fill('Meeting').catch((e) => { out.searchFillErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.searchHasResults = await page.locator('mark').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'team-chat-search.png'), fullPage: true })

  // --- Channel leave (member channel: entwicklung → leave enabled) ---
  await page.getByText('entwicklung', { exact: false }).first().click({ timeout: 5000 }).catch((e) => { out.entwicklungClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)
  const menuBtn = page.getByRole('button', { name: /Kanal-Menü/ }).first()
  out.menuVisible = await menuBtn.isVisible().catch(() => false)
  await menuBtn.click({ timeout: 4000 }).catch((e) => { out.menuClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(400)
  out.leaveItemVisible = await page.getByText(/^Kanal verlassen$/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'team-chat-leave-menu.png'), fullPage: false })
  await page.getByText(/^Kanal verlassen$/).first().click({ timeout: 4000 }).catch((e) => { out.leaveClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(400)
  out.leaveDialogVisible = await page.getByText(/Neue Nachrichten siehst du/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'team-chat-leave-dialog.png'), fullPage: false })
  await page.getByRole('button', { name: /Abbrechen/ }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(400)

  // --- Owner channel (allgemein → leave blocked) ---
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.getByRole('button', { name: /Kanal-Menü/ }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(400)
  out.ownerBlockedVisible = await page.getByText(/Eigentümer kann nicht verlassen/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'team-chat-owner-blocked.png'), fullPage: false })
  await page.keyboard.press('Escape').catch(() => {})

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
