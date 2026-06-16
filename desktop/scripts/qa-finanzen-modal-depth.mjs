// QA finanzen Modal-Tiefe: öffnet je Tab das erste Detail-Modal, prüft die neuen
// Tiefen-Sektionen (Kundenkonto, Mahnverlauf, Verknüpfungen, erzeugte Rechnungen,
// Lieferanten-Rollup, verwandte Buchungen), scannt Raw-i18n-Keys und testet die
// Modal-zu-Modal-Cross-Navigation inkl. Zurück-Button.
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
    return d ? d.innerText.replace(/\n{2,}/g, '\n').slice(0, 1400) : null
  })
}

async function closeDialog(page) {
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(300)
}

async function openTab(page, name) {
  await page.getByRole('button', { name }).first().click({ timeout: 6000 })
  await page.waitForTimeout(900)
}

async function openFirstRow(page) {
  // Listen-Zeilen sind div[role=button].cursor-pointer; Ledger-Tabs ggf. anders.
  const candidates = ['div[role="button"].cursor-pointer', 'div[role="button"]', 'tbody tr', '[data-row]']
  for (const sel of candidates) {
    const loc = page.locator(sel).first()
    if (await loc.count()) {
      await loc.click({ timeout: 4000 }).catch(() => {})
      await page.waitForTimeout(900)
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

const tabs = [
  { key: 'invoices', name: /^Rechnungen/, shot: 'md-invoice.png' },
  { key: 'quotes', name: /^Angebote/, shot: 'md-quote.png' },
  { key: 'recurring', name: /^Wiederkehrend/, shot: 'md-recurring.png' },
  { key: 'creditNotes', name: /^Gutschriften/, shot: 'md-creditnote.png' },
  { key: 'expenses', name: /^Ausgaben/, shot: 'md-expense.png' },
  { key: 'transactions', name: /^Transaktionen/, shot: 'md-transaction.png' },
]

try {
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3000)

  for (const tab of tabs) {
    const r = {}
    try {
      await openTab(page, tab.name)
      const opened = await openFirstRow(page)
      r.opened = opened
      if (opened) {
        await page.screenshot({ path: resolve(outDir, tab.shot) })
        r.text = await dialogText(page)
        r.rawKeys = await scanRawKeys(page)
        r.hasBackBtn = !!(await page.locator('[role="dialog"] button[aria-label="Zurück"]').count())
      }
    } catch (e) {
      r.error = String(e).split('\n')[0]
    }
    out[tab.key] = r
    await closeDialog(page)
  }

  // Cross-Navigation: Rechnung öffnen → ersten Kundenkonto-Beleg klicken → Zurück-Button erwartet
  try {
    await openTab(page, /^Rechnungen/)
    await openFirstRow(page)
    const cross = {}
    // Klick auf einen Kundenkonto-Beleg (font-mono Nummer in einem Button im Dialog)
    const docBtn = page.locator('[role="dialog"] button:has(.font-mono)').first()
    cross.docBtnCount = await docBtn.count()
    if (cross.docBtnCount) {
      await docBtn.click({ timeout: 4000 }).catch((e) => { cross.clickErr = String(e).split('\n')[0] })
      await page.waitForTimeout(900)
      cross.backBtnAfter = !!(await page.locator('[role="dialog"] button[aria-label="Zurück"]').count())
      await page.screenshot({ path: resolve(outDir, 'md-crossnav.png') })
      cross.text = await dialogText(page)
      // Zurück testen
      const back = page.locator('[role="dialog"] button[aria-label="Zurück"]').first()
      if (await back.count()) {
        await back.click().catch(() => {})
        await page.waitForTimeout(700)
        cross.afterBackText = (await dialogText(page))?.slice(0, 120)
      }
    }
    out.crossNav = cross
  } catch (e) {
    out.crossNav = { error: String(e).split('\n')[0] }
  }
} catch (err) {
  out.fatal = String(err).split('\n')[0]
}

out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
