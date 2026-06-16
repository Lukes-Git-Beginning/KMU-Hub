// QA finanzen Audit-Depth: prüft die neuen GoBD-Audit-Sektionen in
// Quote-, CreditNote-, Recurring-, Expense- und Transaction-Detail-Modals.
// Screenshots + Raw-Key-Scan.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots')

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
    return d ? d.innerText.replace(/\n{2,}/g, '\n').slice(0, 2000) : null
  })
}

async function closeDialog(page) {
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)
}

async function openTab(page, name) {
  await page.getByRole('button', { name }).first().click({ timeout: 6000 })
  await page.waitForTimeout(1000)
}

async function openFirstRow(page) {
  const candidates = ['div[role="button"].cursor-pointer', 'div[role="button"]', 'tbody tr', '[data-row]']
  for (const sel of candidates) {
    const loc = page.locator(sel).first()
    if (await loc.count()) {
      await loc.click({ timeout: 4000 }).catch(() => {})
      await page.waitForTimeout(1000)
      const has = await page.locator('[role="dialog"]').count()
      if (has) return true
    }
  }
  return false
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 960 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3500)

  // ---- Invoice (Referenz) ----
  try {
    await openTab(page, /^Rechnungen/)
    const opened = await openFirstRow(page)
    out.invoice = { opened }
    if (opened) {
      await page.screenshot({ path: resolve(outDir, 'qa-invoice-detail.png') })
      const text = await dialogText(page)
      out.invoice.text = text
      out.invoice.rawKeys = await scanRawKeys(page)
      out.invoice.hasAuditSection = text?.includes('GoBD') ?? false
      // PDF-Download-Button vorhanden?
      const pdfBtn = page.locator('[role="dialog"] button:has-text("PDF")').first()
      out.invoice.hasPdfButton = !!(await pdfBtn.count())
    }
    await closeDialog(page)
  } catch (e) {
    out.invoice = { error: String(e).split('\n')[0] }
  }

  // ---- Quote ----
  try {
    await openTab(page, /^Angebote/)
    const opened = await openFirstRow(page)
    out.quote = { opened }
    if (opened) {
      await page.screenshot({ path: resolve(outDir, 'qa-quote-detail.png') })
      const text = await dialogText(page)
      out.quote.text = text
      out.quote.rawKeys = await scanRawKeys(page)
      out.quote.hasAuditSection = text?.includes('GoBD') ?? false
      out.quote.hasHistoryEntry = text?.includes('Erstellt') || text?.includes('Created') || false
      // PDF-Button
      const pdfBtn = page.locator('[role="dialog"] button:has-text("PDF"), [role="dialog"] button:has-text("Pdf")').first()
      out.quote.hasPdfButton = !!(await pdfBtn.count())
    }
    await closeDialog(page)
  } catch (e) {
    out.quote = { error: String(e).split('\n')[0] }
  }

  // ---- CreditNote ----
  try {
    await openTab(page, /^Gutschriften/)
    const opened = await openFirstRow(page)
    out.creditNote = { opened }
    if (opened) {
      await page.screenshot({ path: resolve(outDir, 'qa-creditnote-detail.png') })
      const text = await dialogText(page)
      out.creditNote.text = text
      out.creditNote.rawKeys = await scanRawKeys(page)
      out.creditNote.hasAuditSection = text?.includes('GoBD') ?? false
      out.creditNote.hasHistoryEntry = text?.includes('Erstellt') || text?.includes('Created') || false
      // PDF-Button
      const pdfBtn = page.locator('[role="dialog"] button:has-text("PDF"), [role="dialog"] button:has-text("Pdf")').first()
      out.creditNote.hasPdfButton = !!(await pdfBtn.count())
    }
    await closeDialog(page)
  } catch (e) {
    out.creditNote = { error: String(e).split('\n')[0] }
  }

  // ---- Wiederkehrend ----
  try {
    await openTab(page, /^Wiederkehrend/)
    const opened = await openFirstRow(page)
    out.recurring = { opened }
    if (opened) {
      await page.screenshot({ path: resolve(outDir, 'recurring-detail.png') })
      const text = await dialogText(page)
      out.recurring.text = text
      out.recurring.rawKeys = await scanRawKeys(page)
      out.recurring.hasHistorySection = /verlauf|history|cronologia|historique/i.test(text ?? '')
      out.recurring.hasEndDate = /enddatum|end date|data di fine|date de fin/i.test(text ?? '')
      out.recurring.hasUpcomingRuns = /nächste läufe|upcoming|prossime|prochaines/i.test(text ?? '')
    }
    await closeDialog(page)
  } catch (e) {
    out.recurring = { error: String(e).split('\n')[0] }
  }

  // ---- Ausgaben ----
  try {
    await openTab(page, /^Ausgaben/)
    const opened = await openFirstRow(page)
    out.expense = { opened }
    if (opened) {
      await page.screenshot({ path: resolve(outDir, 'expense-detail.png') })
      const text = await dialogText(page)
      out.expense.text = text
      out.expense.rawKeys = await scanRawKeys(page)
      out.expense.hasHistorySection = /verlauf|history|cronologia|historique/i.test(text ?? '')
      out.expense.hasAccountSection = /kontierung|account assignment|imputazione|imputation/i.test(text ?? '')
    }
    await closeDialog(page)
  } catch (e) {
    out.expense = { error: String(e).split('\n')[0] }
  }

  // ---- Transaktionen ----
  try {
    await openTab(page, /^Transaktionen/)
    const opened = await openFirstRow(page)
    out.transaction = { opened }
    if (opened) {
      await page.screenshot({ path: resolve(outDir, 'transaction-detail.png') })
      const text = await dialogText(page)
      out.transaction.text = text
      out.transaction.rawKeys = await scanRawKeys(page)
      out.transaction.hasHistorySection = /buchungshistorie|booking history|storico contabile|historique de comptabilisation/i.test(text ?? '')
    }
    await closeDialog(page)
  } catch (e) {
    out.transaction = { error: String(e).split('\n')[0] }
  }

} catch (err) {
  out.fatal = String(err).split('\n')[0]
}

out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
