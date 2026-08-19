/**
 * QA for the GDPR anonymization detour on the contact delete flow:
 * - deleting a contact with call/advisory history (Hans Müller, ct-001 in the
 *   demo mock) is refused with 409 and offers anonymization instead of a
 *   dead end
 * - the anonymization step gets its own explicit, irreversible-labelled
 *   confirmation and reports a "pending" request, not "anonymized"
 * - the plain delete path for a contact WITHOUT history still works
 */
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5180'
const outDir = resolve('.qa-screenshots/erasure-flow')

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

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 640, height: 960 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()

const errors = []
const gdprCalls = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('console', (m) => {
  if (m.type() === 'error') errors.push('console: ' + m.text())
})
page.on('request', (r) => {
  if (r.url().includes('/gdpr/') || (r.method() === 'DELETE' && r.url().includes('/contacts/'))) {
    gdprCalls.push(`${r.method()} ${r.url().replace(BASE, '')}`)
  }
})
page.on('response', (r) => {
  if (r.url().includes('/gdpr/') || (r.request().method() === 'DELETE' && r.url().includes('/contacts/'))) {
    gdprCalls.push(`  -> ${r.status()} ${r.url().replace(BASE, '')}`)
  }
})

const report = {}
const rawKeyPattern = /kontakte\.(confirm|erasure|toast)\.[a-zA-Z]+/g

async function search(term) {
  const box = page.getByPlaceholder('Kontakt suchen...')
  await box.fill('')
  await box.fill(term)
  await page.waitForTimeout(500)
}

try {
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  // LaunchOverlay (modules/auth/LaunchOverlay.tsx) covers the app with a
  // boot/fly-in animation on every fresh load; it unmounts (.cosmi-startup
  // removed) once done. Wait for that instead of a guessed timeout.
  await page.locator('.cosmi-startup').waitFor({ state: 'detached', timeout: 20000 }).catch(() => {})
  await page.getByPlaceholder('Kontakt suchen...').waitFor({ state: 'visible', timeout: 25000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, '00-liste.png') })

  // ---- Blocked contact: Hans Müller (ct-001) always 409s on hard delete ----
  await search('Müller')
  await page.screenshot({ path: resolve(outDir, '01-gefiltert-mueller.png') })

  const row = page.getByText('Hans Müller', { exact: false }).first()
  await row.click({ timeout: 8000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, '02-detailpanel.png') })

  const deleteBtn = page.getByTitle('Löschen').first()
  await deleteBtn.click({ timeout: 8000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, '03-delete-confirm.png') })

  const alertDialog = page.getByRole('alertdialog')
  await alertDialog.getByRole('button', { name: 'Löschen' }).click({ timeout: 8000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, '04-conflict-dialog.png') })

  const conflictText = await page.evaluate(() => document.body.innerText)
  report.conflictDialogShown = /Kontakt kann nicht gelöscht werden/.test(conflictText)
  report.conflictShowsServerReason = /call campaign history/.test(conflictText)
  report.conflictRawKeys = conflictText.match(rawKeyPattern) ?? []

  await page.getByRole('alertdialog').getByRole('button', { name: 'Zur Anonymisierung' }).click({ timeout: 8000 })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '05-erasure-confirm.png') })

  const erasureText = await page.evaluate(() => document.body.innerText)
  report.erasureDialogShown = /Kontakt anonymisieren/.test(erasureText)
  report.erasureMentionsIrreversible = /nicht rückgängig/.test(erasureText)
  report.erasureRawKeys = erasureText.match(rawKeyPattern) ?? []

  await page.getByRole('alertdialog').getByRole('button', { name: 'Anonymisierung beantragen' }).click({ timeout: 8000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, '06-nach-antrag.png') })

  const afterRequestText = await page.evaluate(() => document.body.innerText)
  report.pendingToastShown = /Antrag auf Anonymisierung erstellt/.test(afterRequestText)
  report.wronglyClaimsAnonymized = /Kontakt anonymisiert(?!.{0,5}beantrag)/.test(afterRequestText)
  report.afterRequestRawKeys = afterRequestText.match(rawKeyPattern) ?? []

  // ---- Unblocked contact: plain delete still works ----
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  await search('Weber')
  await page.waitForTimeout(500)
  const weberRow = page.getByText('Claudia Weber', { exact: false }).first()
  await weberRow.click({ timeout: 8000 })
  await page.waitForTimeout(1000)
  await page.getByTitle('Löschen').first().click({ timeout: 8000 })
  await page.waitForTimeout(500)
  await page.getByRole('alertdialog').getByRole('button', { name: 'Löschen' }).click({ timeout: 8000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, '07-normaler-delete.png') })
  const afterPlainDelete = await page.evaluate(() => document.body.innerText)
  report.plainDeleteSucceeded = /Kontakt gelöscht/.test(afterPlainDelete)
  report.plainDeleteShowedConflict = /Kontakt kann nicht gelöscht werden/.test(afterPlainDelete)

  report.gdprCalls = gdprCalls
  report.errors = errors.slice(0, 10)
} catch (err) {
  report.fatal = String(err).split('\n').slice(0, 5).join(' | ')
  report.errors = errors.slice(0, 10)
  await page.screenshot({ path: resolve(outDir, 'fatal.png') }).catch(() => {})
} finally {
  await browser.close()
}

console.log(JSON.stringify(report, null, 2))
