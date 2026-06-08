// QA Posteingang (Phase 4): snooze/claim/assign/status/tags/forward header,
// threading (multi-message), bulk toolbar, status filter, team-inbox + routing
// in module settings. Screenshots @ 3 widths + raw-key scan + pageErrors.
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
  return [...new Set([...text.matchAll(/\b(kommunikation|chat|moduleSettings|settings|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

async function shot(page, name) {
  await page.screenshot({ path: resolve(outDir, name), fullPage: true })
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const errors = []

const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/kommunikation?bereich=posteingang`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)

  // 1. Select an UNASSIGNED conversation (Michael Brunner = msg-003) → thread
  await page.getByText('Michael Brunner').first().click({ timeout: 6000 }).catch((e) => { out.selectErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.threadBubbles = await page.locator('.whitespace-pre-line').count().catch(() => -1)
  out.assignBtnVisible = await page.getByRole('button', { name: 'Zuweisen' }).first().isVisible().catch(() => false)
  out.claimBtnVisible = await page.getByRole('button', { name: 'Übernehmen' }).first().isVisible().catch(() => false)
  await shot(page, 'inbox-thread-full.png')
  out.threadRawKeys = await scanRawKeys(page)

  // 2. Assign popover (unassigned → button reads "Zuweisen")
  await page.getByRole('button', { name: 'Zuweisen' }).first().click({ timeout: 4000 }).catch((e) => { out.assignErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  out.assignListVisible = await page.getByText('Zuweisen an').first().isVisible().catch(() => false)
  await shot(page, 'inbox-assign-popover.png')
  await page.waitForTimeout(300)

  // 3. Tag add popover
  await page.getByText('Tag hinzufügen').first().click({ timeout: 4000 }).catch((e) => { out.tagErr = String(e).split('\n')[0] })
  await page.waitForTimeout(400)
  out.tagPlaceholderVisible = await page.getByPlaceholder('Neues Tag…').first().isVisible().catch(() => false)
  await shot(page, 'inbox-tag-popover.png')
  await page.waitForTimeout(300)

  // 4. Snooze popover
  await page.getByTitle('Später erinnern').first().click({ timeout: 4000 }).catch((e) => { out.snoozeErr = String(e).split('\n')[0] })
  await page.waitForTimeout(400)
  out.snoozePresetVisible = await page.getByText(/Nächste Woche|Morgen früh|1 Stunde/).first().isVisible().catch(() => false)
  await shot(page, 'inbox-snooze-popover.png')
  await page.waitForTimeout(300)

  // 5. Status selector popover (popover items are portalled → use .last())
  await page.getByTitle('Status ändern').first().click({ timeout: 4000 }).catch((e) => { out.statusErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  await shot(page, 'inbox-status-popover.png')
  await page.getByRole('button', { name: 'Gelöst' }).last().click({ timeout: 3000 }).catch((e) => { out.statusPickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)

  // 6. Bulk selection mode (clean state, no active filter)
  await page.getByTitle('Mehrere auswählen').first().click({ timeout: 3000 }).catch((e) => { out.bulkErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  await page.getByText('Werner Koch').first().click({ timeout: 3000 }).catch(() => {})
  await page.getByText('Maria Huber').first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(400)
  out.bulkToolbarVisible = await page.getByText(/ausgewählt/).first().isVisible().catch(() => false)
  await shot(page, 'inbox-bulk-toolbar.png')
  await page.getByTitle('Mehrere auswählen').first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(400)

  // 7. Status filter chip in list (filter chips live in the list, not portalled)
  await page.getByRole('button', { name: 'Gelöst' }).first().click({ timeout: 3000 }).catch((e) => { out.filterErr = String(e).split('\n')[0] })
  await page.waitForTimeout(600)
  out.filterResolvedCount = await page.locator('button:has-text("Michael Brunner")').count().catch(() => -1)
  await shot(page, 'inbox-status-filter.png')
  await page.getByRole('button', { name: /^Alle$/ }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(400)

  // 8. Narrow widths
  await page.setViewportSize({ width: 720, height: 900 })
  await page.waitForTimeout(600)
  await shot(page, 'inbox-half.png')
  await page.setViewportSize({ width: 500, height: 800 })
  await page.waitForTimeout(600)
  await shot(page, 'inbox-small.png')
  await page.setViewportSize({ width: 1440, height: 950 })
  await page.waitForTimeout(400)

  // 9. Module settings → Kommunikation → Team-Postfächer + Routing
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 5000 }).catch((e) => { out.settingsOpenErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)
  await page.getByText(/^Kommunikation$/).first().click({ timeout: 4000 }).catch((e) => { out.settingsEntryErr = String(e).split('\n')[0] })
  await page.waitForTimeout(900)
  out.settingsTeamInboxes = await page.getByText('Team-Postfächer').first().isVisible().catch(() => false)
  out.settingsRouting = await page.getByText('Routing-Regeln').first().isVisible().catch(() => false)
  await shot(page, 'inbox-settings.png')
  out.settingsRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
