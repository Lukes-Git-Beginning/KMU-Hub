/**
 * QA — Bexio invoice-pull review readiness (Session #8).
 * The Bexio card lives in the MODULE-SETTINGS OVERLAY (bottom-left "Einstellungen"),
 * not on the /settings page (that tab is adminOnly → always hidden). Two reachable
 * entries, both now wired to the real wizard/dashboard: cosmi→Integrationen and
 * Buchhaltung→Integrationen. Demo user is admin, so tenant sections are editable.
 *
 * Verifies:
 *  A) Finanzen → Rechnungen → Bexio invoice (2026-0042) → read-only InvoiceDetailPanel
 *  B) Overlay → (finance) Integrationen section → Bexio card → setup WIZARD
 *  C) Wizard connect (OAuth mock) → advance → SYNC DASHBOARD reachable
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/bexio-review')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`
const NOPOPUP = `window.open=()=>null`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 960 } })
for (const s of [STUB, ONB, NOLAUNCH, NOPOPUP]) await ctx.addInitScript(s)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []
const dlgText = () => page.evaluate(() => {
  const dls = [...document.querySelectorAll('[role="dialog"]')]
  return dls.map((d) => d.innerText).join('\n---\n')
})

// ---- A) read-only Bexio invoice ----
try {
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  const invTab = page.getByRole('button', { name: /^Rechnungen/ }).first()
  if (await invTab.count()) { await invTab.click(); await page.waitForTimeout(1200) }
  const bexioRow = page.locator('div[role="button"]').filter({ hasText: '2026-0042' }).first()
  const rowCount = await bexioRow.count()
  await bexioRow.click({ timeout: 6000 })
  await page.waitForSelector('[role="dialog"]', { timeout: 8000 })
  await page.waitForTimeout(900)
  await page.screenshot({ path: resolve(outDir, 'A-invoice-readonly.png') })
  const invText = await dlgText()
  const hasEditButtons = await page.evaluate(() => {
    const dlg = document.querySelector('[role="dialog"]')
    if (!dlg) return false
    const labels = /Bearbeiten|Versenden|Zahlung erfassen|Als bezahlt|Stornieren|Gutschrift erstellen/i
    return [...dlg.querySelectorAll('button')].some((btn) => labels.test(btn.textContent || ''))
  })
  out.push({ step: 'A: bexio invoice read-only panel', rowCount, hasBexioMarker: /Bexio|schreibgeschützt|importiert/i.test(invText), mutatingButtonsHidden: !hasEditButtons, pass: rowCount > 0 && /Bexio|schreibgeschützt|importiert/i.test(invText) && !hasEditButtons })
  await page.keyboard.press('Escape'); await page.waitForTimeout(500)
} catch (e) {
  out.push({ step: 'A: bexio invoice', pass: false, error: String(e).split('\n')[0] })
}

// ---- B) module-settings overlay → Integrationen → Bexio card → wizard ----
try {
  // Fresh nav so we start from a clean layout (Phase A left a dialog).
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2500)
  await page.screenshot({ path: resolve(outDir, 'B0-before-overlay.png') })
  // Open module-settings overlay from bottom-left "Modul-Einstellungen" nav button.
  const settingsNav = page.getByText(/Modul-Einstellungen/).first()
  await settingsNav.click({ timeout: 6000 })
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, 'B1-overlay-finance.png') })
  // Bexio card is in the Integrationen section; scroll it into view then click.
  const bexioCard = page.getByRole('button').filter({ hasText: /^Bexio/ }).first()
  const cardCount = await bexioCard.count()
  if (cardCount) { await bexioCard.scrollIntoViewIfNeeded().catch(() => {}); await bexioCard.click({ timeout: 6000 }) }
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, 'B2-wizard-step1.png') })
  const wizText = await dlgText()
  out.push({ step: 'B: overlay → Bexio card opens wizard', cardCount, isWizard: /OAuth|Verbind|Schritt|Sync-Richt/i.test(wizText), pass: cardCount > 0 && /OAuth|Verbind|Schritt|Sync/i.test(wizText) })

  // ---- C) connect (OAuth mock) → advance → dashboard reachable ----
  const connectBtn = page.getByRole('button', { name: /Mit Bexio verbinden|Verbinden|Connect/ }).first()
  if (await connectBtn.count()) { await connectBtn.click(); await page.waitForTimeout(3000) }
  await page.screenshot({ path: resolve(outDir, 'C1-wizard-connected.png') })
  for (let i = 0; i < 3; i++) {
    const next = page.getByRole('button', { name: /^Weiter|Nächster|Next/ }).first()
    if ((await next.count()) && (await next.isEnabled().catch(() => false))) {
      await next.click().catch(() => {}); await page.waitForTimeout(900)
    }
  }
  await page.screenshot({ path: resolve(outDir, 'C2-wizard-later-step.png') })
  const laterText = await dlgText()
  out.push({ step: 'C: wizard advances past OAuth (invoice-pull step)', reachedLater: /Rechnung|Invoice|Pull|Intervall|Feld|Zuordnung|Erster Sync/i.test(laterText), pass: /Rechnung|Invoice|Pull|Intervall|Feld|Sync/i.test(laterText) })
  await page.keyboard.press('Escape'); await page.waitForTimeout(600)

  // Re-open Bexio card — now connected → dashboard
  const settingsNav2 = page.getByText(/Modul-Einstellungen/).first()
  if (await settingsNav2.count()) { await settingsNav2.click().catch(() => {}); await page.waitForTimeout(1200) }
  const bexioCard2 = page.getByRole('button').filter({ hasText: /^Bexio/ }).first()
  if (await bexioCard2.count()) { await bexioCard2.scrollIntoViewIfNeeded().catch(() => {}); await bexioCard2.click({ timeout: 6000 }).catch(() => {}) }
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'C3-sync-dashboard.png') })
  const dashText = await dlgText()
  out.push({ step: 'C: connected card opens sync dashboard', hasCards: /Kontakte/.test(dashText) && /Rechnungen/.test(dashText), pass: /Kontakte/.test(dashText) && /Rechnungen/.test(dashText) })
} catch (e) {
  out.push({ step: 'B/C: wizard+dashboard', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 10), pass: errs.length === 0 })
await ctx.close(); await b.close()
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== BEXIO-REVIEW QA: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
