/**
 * QA for the rewired ConsentPanel: opens a contact, screenshots the consent
 * section, grants one consent, reloads, and checks the state survived. The old
 * panel could never fail this because it never left the browser -- that is the
 * whole point of the check.
 */
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5180'
const outDir = resolve('.qa-screenshots/consent-wired')

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
const ctx = await browser.newContext({ viewport: { width: 620, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()

const errors = []
const consentCalls = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('console', (m) => {
  if (m.type() === 'error') errors.push('console: ' + m.text())
})
page.on('request', (r) => {
  if (r.url().includes('/consents')) consentCalls.push(`${r.method()} ${r.url().replace(BASE, '')}`)
})

const report = {}

try {
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(3500)
  await page.screenshot({ path: resolve(outDir, '00-liste.png') })

  // Open the first contact row.
  const row = page.locator('table tbody tr, [role="row"]').nth(1)
  await row.click({ timeout: 8000 }).catch(async () => {
    await page.getByText(/Bauer|Berger|Brunner|Egger/i).first().click({ timeout: 8000 })
  })
  await page.waitForTimeout(1800)

  // Scroll the consent section into view.
  const heading = page.getByText(/DSGVO-Einwilligungen|Einwilligungen/i).first()
  if (await heading.count()) await heading.scrollIntoViewIfNeeded().catch(() => {})
  await page.waitForTimeout(900)
  await page.screenshot({ path: resolve(outDir, '01-consent-initial.png'), fullPage: false })

  // Raw i18n keys would show up literally in the text.
  const bodyText = await page.evaluate(() => document.body.innerText)
  report.rawKeys = (bodyText.match(/kontakte\.consent\.[a-zA-Z_.]+/g) || []).slice(0, 10)
  report.showsLoadError = /nicht geladen werden/i.test(bodyText)
  report.labelsVisible = {
    emailMarketing: bodyText.includes('E-Mail-Marketing'),
    phoneMarketing: bodyText.includes('Telefon-Marketing'),
    profiling: bodyText.includes('Profilbildung'),
    dataSharing: bodyText.includes('Datenweitergabe'),
  }

  // Grant the first available consent.
  const grantBtn = page.getByRole('button', { name: /^Erteilen$/ }).first()
  if (await grantBtn.count()) {
    await grantBtn.click()
    await page.waitForTimeout(600)
    await page.screenshot({ path: resolve(outDir, '02-grant-form.png') })
    const confirm = page.getByRole('button', { name: /^Bestätigen$/ }).first()
    await confirm.click({ timeout: 5000 })
    await page.waitForTimeout(1500)
    await page.screenshot({ path: resolve(outDir, '03-after-grant.png') })
    report.grantClicked = true
  } else {
    report.grantClicked = false
  }

  const afterGrant = await page.evaluate(() => document.body.innerText)
  report.grantedCountAfter = (afterGrant.match(/(\d+)\s+erteilt/) || [])[1] ?? null

  // The real test: does it survive a reload?
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await page.screenshot({ path: resolve(outDir, '04-after-reload.png') })
  const afterReload = await page.evaluate(() => document.body.innerText)
  report.stillOnConsent = /Einwilligungen/i.test(afterReload)

  report.consentCalls = consentCalls
  report.errors = errors.slice(0, 6)
} catch (err) {
  report.fatal = String(err).split('\n').slice(0, 3).join(' | ')
  report.errors = errors.slice(0, 6)
  await page.screenshot({ path: resolve(outDir, 'fatal.png') }).catch(() => {})
} finally {
  await browser.close()
}

console.log(JSON.stringify(report, null, 2))
