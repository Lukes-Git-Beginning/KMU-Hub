/**
 * V-5 QA — Nummernkreis, echter Audit-User, Demo-Tiefe
 *
 * (a) „Vertrag anlegen" → Vertragsnummer ist automatisch vorbefüllt (V-{JAHR}-001).
 * (b) Anlegen → erneut öffnen → nächste Nummer (V-{JAHR}-002) = Counter erhöht.
 * (c) Audit-Log des neuen Vertrags zeigt echten Auth-User „Markus Weber"
 *     (nicht „Aktueller Benutzer").
 * (d) Vorlagen-Tab → „Vertrag aus Vorlage" legt wirklich einen Vertrag an
 *     (vorher toter Button: isEdit-Bug bei id='').
 *
 * Sub-Terminal: :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/vertraege-demo-depth')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const YEAR = new Date().getFullYear()

function clickButtonByText(page, text) {
  return page.evaluate((t) => {
    const btn = Array.from(document.querySelectorAll('button')).find(
      (b) => (b.textContent || '').trim() === t || (b.textContent || '').trim().includes(t),
    )
    if (btn) { btn.click(); return true }
    return false
  }, text)
}

async function fillNewContractForm(page, title, partner) {
  await page.locator('input[placeholder*="Mietvertrag"]').first().fill(title)
  await page.locator('input[placeholder*="Swisscom"]').first().fill(partner)
  await page.locator('input[type="date"]').first().fill(`${YEAR}-06-17`)
}

const browser = await chromium.launch()
const out = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })

try {
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2000)

  // ── (a) Auto-Nummer beim ersten Anlegen ──────────────────────────
  await clickButtonByText(page, 'Vertrag anlegen')
  await page.locator('input[placeholder*="SV-2026-001"]').first().waitFor({ state: 'visible', timeout: 6000 })
  await page.waitForTimeout(400)
  const num1 = await page.locator('input[placeholder*="SV-2026-001"]').first().inputValue()
  await page.screenshot({ path: resolve(outDir, 'a-new-dialog-autonumber.png'), fullPage: false })
  out.push({
    step: 'a-autonumber',
    number: num1,
    pass: num1 === `V-${YEAR}-001`,
  })

  // Vertrag anlegen
  await fillNewContractForm(page, 'QA Auto-Nummer Vertrag', 'QA Partner AG')
  await clickButtonByText(page, 'Anlegen')
  await page.waitForTimeout(1200)

  // ── (b) Counter erhöht → nächste Nummer ──────────────────────────
  await clickButtonByText(page, 'Vertrag anlegen')
  await page.locator('input[placeholder*="SV-2026-001"]').first().waitFor({ state: 'visible', timeout: 6000 })
  await page.waitForTimeout(400)
  const num2 = await page.locator('input[placeholder*="SV-2026-001"]').first().inputValue()
  out.push({
    step: 'b-counter-bumped',
    number: num2,
    pass: num2 === `V-${YEAR}-002`,
  })
  // Dialog schließen
  await clickButtonByText(page, 'Abbrechen')
  await page.waitForTimeout(600)

  // ── (c) Audit-Log zeigt echten User ──────────────────────────────
  const row = page.locator('table tbody tr').filter({ hasText: 'QA Auto-Nummer Vertrag' }).first()
  await row.waitFor({ state: 'visible', timeout: 6000 })
  await row.click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)
  await page.evaluate(() => {
    const dialog = document.querySelector('[role="dialog"]')
    const cands = Array.from(dialog?.querySelectorAll('*') || []).filter(
      (e) => e.scrollHeight > e.clientHeight + 20 && e.clientHeight > 150,
    )
    cands.sort((a, b) => (b.scrollHeight - b.clientHeight) - (a.scrollHeight - a.clientHeight))
    cands[0]?.scrollTo({ top: cands[0].scrollHeight, behavior: 'instant' })
  })
  await page.waitForTimeout(500)
  const auditText = await page.evaluate(() => document.querySelector('[role="dialog"]')?.textContent || '')
  await page.screenshot({ path: resolve(outDir, 'c-audit-real-user.png'), fullPage: false })
  out.push({
    step: 'c-audit-real-user',
    hasRealUser: /Markus Weber/.test(auditText),
    hasPlaceholderUser: /Aktueller Benutzer/.test(auditText),
    pass: /Markus Weber/.test(auditText) && !/Aktueller Benutzer/.test(auditText),
  })
  // Detail-Modal schließen (Radix → ESC)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(600)

  // ── (d) Vorlage → Vertrag aus Vorlage legt wirklich an ───────────
  await page.evaluate(() => {
    const tab = Array.from(document.querySelectorAll('button')).find((b) => /^Vorlagen/.test((b.textContent || '').trim()))
    tab?.click()
  })
  await page.waitForTimeout(800)
  const tplClicked = await clickButtonByText(page, 'Vertrag aus Vorlage')
  await page.locator('input[placeholder*="SV-2026-001"]').first().waitFor({ state: 'visible', timeout: 6000 })
  await page.waitForTimeout(500)
  // Template-Dialog: Titel „Neuen Vertrag anlegen" (create-Pfad, nicht edit)
  const dialogTitle = await page.evaluate(() => {
    const h2 = Array.from(document.querySelectorAll('h2')).map((e) => e.textContent || '').join(' ')
    return h2
  })
  const isCreatePath = /Neuen Vertrag anlegen/.test(dialogTitle)
  await fillNewContractForm(page, 'QA Vorlagen Vertrag', 'QA Vorlage Partner')
  await clickButtonByText(page, 'Anlegen')
  await page.waitForTimeout(1200)
  // In Aktiv-Tab prüfen, ob der Vertrag existiert
  await page.evaluate(() => {
    const tab = Array.from(document.querySelectorAll('button')).find((b) => /^Aktiv/.test((b.textContent || '').trim()))
    tab?.click()
  })
  await page.waitForTimeout(800)
  const tplCreated = await page.evaluate(() =>
    Array.from(document.querySelectorAll('table tbody tr')).some((r) => /QA Vorlagen Vertrag/.test(r.textContent || '')),
  )
  await page.screenshot({ path: resolve(outDir, 'd-template-created.png'), fullPage: false })
  out.push({
    step: 'd-template-create',
    tplClicked,
    isCreatePath,
    tplCreated,
    pass: tplClicked && isCreatePath && tplCreated,
  })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'console-errors', errors: errs.slice(0, 6), pass: errs.length === 0 })

await ctx.close()
await browser.close()

const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== V-5 QA RESULT: ${allPass ? 'ALL PASS' : 'FAIL'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
process.exit(allPass ? 0 : 1)
