// QA finanzen P2.5e: Banking (Filter + Match-Persistenz), Belegkette (Liste +
// Empty), Recurring-Detail (erzeugte Rechnungen befüllt), Stunden→Rechnung
// (select → preview → echter Create). Screenshots + Raw-Key-Scan + pageErrors.
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
  return [...new Set([...text.matchAll(/\b(finanzen|buchhaltung|moduleSettings|settings|common|shared)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function dialogText(page) {
  return page.evaluate(() => {
    const d = document.querySelector('[role="dialog"]')
    return d ? d.innerText.replace(/\n{2,}/g, '\n').slice(0, 1600) : null
  })
}
async function openTab(page, name) {
  await page.getByRole('button', { name }).first().click({ timeout: 6000 })
  await page.waitForTimeout(900)
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 960 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3000)

  // ---- Banking ----
  const bank = {}
  try {
    await openTab(page, /^Banking/)
    await page.waitForTimeout(600)
    await page.screenshot({ path: resolve(outDir, 'p25e-banking-all.png') })
    bank.rawKeys = await scanRawKeys(page)
    // counts before
    bank.filtersBefore = await page.evaluate(() =>
      [...document.querySelectorAll('button')].map((b) => b.textContent.trim()).filter((t) => /^(Alle|Zugeordnet|Vorgeschlagen|Offen) \(/.test(t)),
    )
    // accept first suggested match
    const accept = page.locator('button[title="Zuordnung bestätigen"]').first()
    bank.acceptBtnCount = await accept.count()
    if (bank.acceptBtnCount) {
      await accept.click({ timeout: 4000 }).catch((e) => { bank.acceptErr = String(e).split('\n')[0] })
      await page.waitForTimeout(1200)
      bank.filtersAfter = await page.evaluate(() =>
        [...document.querySelectorAll('button')].map((b) => b.textContent.trim()).filter((t) => /^(Alle|Zugeordnet|Vorgeschlagen|Offen) \(/.test(t)),
      )
      await page.screenshot({ path: resolve(outDir, 'p25e-banking-after-match.png') })
    }
  } catch (e) { bank.error = String(e).split('\n')[0] }
  out.banking = bank

  // ---- Belegkette ----
  const beleg = {}
  try {
    await openTab(page, /^Belegkette/)
    await page.waitForTimeout(600)
    beleg.chainCount = await page.locator('button:has-text("CHF"), button:has-text("EUR")').count()
    await page.screenshot({ path: resolve(outDir, 'p25e-belegkette.png') })
    beleg.rawKeys = await scanRawKeys(page)
    // empty state via search
    const search = page.getByPlaceholder('Kunde oder Belegnummer suchen…')
    if (await search.count()) {
      await search.fill('zzzznomatch')
      await page.waitForTimeout(500)
      await page.screenshot({ path: resolve(outDir, 'p25e-belegkette-empty.png') })
      beleg.emptyText = await page.evaluate(() => document.body.innerText.includes('Keine') )
      await search.fill('')
    }
  } catch (e) { beleg.error = String(e).split('\n')[0] }
  out.belegkette = beleg

  // ---- Recurring detail (generated invoices populated) ----
  const rec = {}
  try {
    await openTab(page, /^Wiederkehrend/)
    await page.waitForTimeout(600)
    const row = page.locator('div[role="button"].cursor-pointer, tbody tr, div[role="button"]').first()
    await row.click({ timeout: 4000 }).catch((e) => { rec.rowErr = String(e).split('\n')[0] })
    await page.waitForTimeout(900)
    rec.dialogOpen = !!(await page.locator('[role="dialog"]').count())
    const dt = await dialogText(page)
    rec.hasGeneratedSection = dt ? dt.includes('Erzeugte Rechnungen') : false
    // count clickable generated-invoice rows (font-mono RE-numbers inside dialog buttons)
    rec.generatedRows = await page.locator('[role="dialog"] button:has(.font-mono)').count()
    rec.text = dt
    await page.screenshot({ path: resolve(outDir, 'p25e-recurring-detail.png') })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(400)
  } catch (e) { rec.error = String(e).split('\n')[0] }
  out.recurring = rec

  // ---- Stunden → Rechnung (real create) ----
  const hours = {}
  try {
    await page.getByRole('button', { name: /Stunden abrechnen/ }).first().click({ timeout: 6000 })
    await page.waitForTimeout(900)
    hours.dialogOpen = !!(await page.locator('[role="dialog"]').count())
    await page.screenshot({ path: resolve(outDir, 'p25e-hours-select.png') })
    hours.selectRawKeys = await scanRawKeys(page)
    // select all → preview
    await page.getByRole('button', { name: /Alle auswählen/ }).first().click({ timeout: 4000 }).catch((e) => { hours.selAllErr = String(e).split('\n')[0] })
    await page.waitForTimeout(400)
    await page.getByRole('button', { name: /^Vorschau/ }).first().click({ timeout: 4000 }).catch((e) => { hours.previewErr = String(e).split('\n')[0] })
    await page.waitForTimeout(700)
    await page.screenshot({ path: resolve(outDir, 'p25e-hours-preview.png') })
    hours.previewText = await dialogText(page)
    // create button should be disabled until customer name entered
    const createBtn = page.getByRole('button', { name: /Rechnung erstellen|Erstellen/ }).last()
    hours.createDisabledNoCustomer = await createBtn.isDisabled().catch(() => null)
    const custInput = page.locator('[role="dialog"] input[type="text"]').first()
    if (await custInput.count()) {
      await custInput.fill('Muster Kunde GmbH')
      await page.waitForTimeout(300)
    }
    hours.createEnabledWithCustomer = !(await createBtn.isDisabled().catch(() => true))
    await createBtn.click({ timeout: 4000 }).catch((e) => { hours.createErr = String(e).split('\n')[0] })
    await page.waitForTimeout(1200)
    hours.dialogClosedAfterCreate = !(await page.locator('[role="dialog"]').count())
    await page.screenshot({ path: resolve(outDir, 'p25e-hours-after-create.png') })
  } catch (e) { hours.error = String(e).split('\n')[0] }
  out.hours = hours
} catch (err) {
  out.fatal = String(err).split('\n')[0]
}

out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
