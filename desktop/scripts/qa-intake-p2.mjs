/**
 * QA — Ticket-Intake P2: shared intake engine (form → Helpdesk ticket).
 *
 *   S1 Neues Formular aus der Vorlage "Support-Ticket (Helpdesk)" erstellen.
 *   S2 Editor: Intake-Panel zeigt "Helpdesk-Ticket" als Ziel + Feld-Zuordnung
 *      (Rollen mit dem zugeordneten Feld, z.B. "Name des Anfragenden — Ihr Name").
 *   S3 Feld-Config eines Feldes: der Rollen-Dropdown ist da (nur weil gebunden).
 *   S4 Vorschau ausfüllen + absenden → "Helpdesk-Ticket erstellt" + Referenz.
 *   S5 Helpdesk: das neue Ticket erscheint mit gemapptem Betreff/Beschreibung/
 *      Priorität/Kategorie/Kontakt.
 *   S6 Keine Page-Errors.
 * Screenshots → .qa-screenshots/intake-p2/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/intake-p2')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`
const NODRAFT = `try{localStorage.removeItem('cosmi-formulare-draft')}catch(e){}`

const FORM_NAME = 'QA P2 Support-Formular'
const SUBJECT = 'QA P2 Ticket aus Formular'
const DESC = 'P2-Engine: Formular-Einreichung wurde zum Ticket gemappt.'
const REQNAME = 'Formular Tester'
const REQMAIL = 'tester@extern.example'

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
await ctx.addInitScript(NOLAUNCH); await ctx.addInitScript(NODRAFT)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
page.on('console', (m) => { if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED|Failed to load resource/.test(m.text())) errs.push('console: ' + m.text()) })
const out = []
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const bodyText = () => page.evaluate(() => document.body.innerText)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

try {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await wait(3200)
  await waitForText('Formular', 15000)

  // ── S1 Neues Formular aus der Ticket-Vorlage ────────────────────────────────
  await page.getByRole('button', { name: /Neues Formular/i }).first().click()
  await waitForText('von Vorlage', 8000).catch(() => {})
  await wait(500)
  const dlg1 = page.locator('[role="dialog"]').last()
  await dlg1.locator('input[type="text"]').first().fill(FORM_NAME)
  // template <select> — pick by known seed id (value), robust vs label wording.
  await dlg1.locator('select').first().selectOption('tmpl-ticket-intake')
  await shot('01-new-form-dialog.png')
  await page.getByRole('button', { name: /Formular erstellen/i }).click()
  await wait(1600)
  {
    const txt = await bodyText()
    out.push({ step: 'S1 Formular aus Ticket-Vorlage erstellt', inList: txt.includes(FORM_NAME), pass: txt.includes(FORM_NAME) })
  }

  // ── S2 Editor öffnen → Intake-Panel + Feld-Zuordnung ────────────────────────
  await page.locator(`[role="button"]:has-text("${FORM_NAME}")`).first().locator('button').first().click({ timeout: 6000 })
  await wait(400)
  await page.locator('[role="menuitem"]:has-text("Bearbeiten")').first().click({ timeout: 5000 })
  await wait(1200)
  // scroll to the intake panel (below the builder grid)
  await page.getByText('Bei Einreichung', { exact: false }).first().scrollIntoViewIfNeeded().catch(() => {})
  await wait(400)
  await shot('02-editor-intake-panel.png')
  {
    const txt = await bodyText()
    const hasPanel = txt.includes('Bei Einreichung')
    const hasTarget = txt.includes('Helpdesk-Ticket')
    // coverage checklist: role + assigned field
    const mapsRequester = txt.includes('Name des Anfragenden') && txt.includes('Ihr Name')
    const mapsSubject = txt.includes('Betreff')
    out.push({
      step: 'S2 Intake-Panel zeigt Ziel + Feld-Zuordnung',
      hasPanel, hasTarget, mapsRequester, mapsSubject,
      pass: hasPanel && hasTarget && mapsRequester && mapsSubject,
    })
  }

  // ── S3 Feld-Config → Rollen-Dropdown ist da (nur weil gebunden) ─────────────
  let roleDropdownSeen = false
  let roleSelected = ''
  try {
    const row = page.locator('.group:has-text("Betreff")').first()
    await row.scrollIntoViewIfNeeded().catch(() => {})
    await row.hover().catch(() => {})
    await wait(300)
    await row.locator('button[title="Bearbeiten"]').first().click({ timeout: 4000 })
    await wait(700)
    const cfg = page.locator('[role="dialog"]').last()
    roleDropdownSeen = (await cfg.locator('text=Feld-Rolle im Ziel').count()) > 0
    if (roleDropdownSeen) {
      roleSelected = ((await cfg.locator('select').first().locator('option:checked').textContent().catch(() => '')) || '').trim()
      await shot('03-field-role-dropdown.png')
    }
    await page.keyboard.press('Escape').catch(() => {})
    await wait(400)
  } catch { /* keep soft — panel coverage in S2 already proves roles work */ }
  out.push({
    step: 'S3 Rollen-Dropdown im Feld-Config + reflektiert die Rolle',
    roleDropdownSeen, roleSelected,
    pass: roleDropdownSeen && /Betreff/.test(roleSelected),
  })

  // ── S4 Vorschau ausfüllen + absenden → Ticket erstellt ──────────────────────
  await page.locator('button:has-text("Vorschau")').first().click({ timeout: 5000 })
  await wait(1000)
  const pv = page.locator('[role="dialog"]').last()
  // text inputs order in the template: Betreff, Ihr Name, Bestell-/Kundennummer
  const texts = pv.locator('input[type="text"]')
  await texts.nth(0).fill(SUBJECT)
  await pv.locator('textarea').first().fill(DESC)
  const selects = pv.locator('select')
  await selects.nth(0).selectOption({ label: 'Hoch' }).catch(() => {})
  await selects.nth(1).selectOption({ label: 'Technisch' }).catch(() => {})
  await texts.nth(1).fill(REQNAME)
  await pv.locator('input[type="email"]').first().fill(REQMAIL)
  await texts.nth(2).fill('BE-2026-999') // extra → custom_field
  await pv.locator('input[type="checkbox"]').first().check().catch(() => {})
  await shot('04-preview-filled.png')
  await pv.getByRole('button', { name: /Absenden|Wird erstellt/i }).click({ timeout: 5000 })
  await wait(1800)
  await shot('05-preview-submitted.png')
  {
    const txt = await bodyText()
    const created = txt.includes('Helpdesk-Ticket erstellt')
    const hasRef = txt.includes('Referenz:')
    out.push({ step: 'S4 Vorschau-Einreichung erzeugt ein Ticket', created, hasRef, pass: created && hasRef })
  }

  // ── S5 Helpdesk: Ticket erscheint + gemappt ─────────────────────────────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2600)
  await waitForText('Ticket', 15000)
  await waitForText(SUBJECT, 8000).catch(() => {})
  await shot('06-helpdesk-list.png')
  {
    const txt = await bodyText()
    out.push({ step: 'S5a Ticket erscheint im Helpdesk', inList: txt.includes(SUBJECT), pass: txt.includes(SUBJECT) })
  }
  await page.getByText(SUBJECT, { exact: false }).first().click().catch(() => {})
  await wait(1200)
  await shot('07-ticket-detail.png')
  {
    const txt = await bodyText()
    const hasDesc = txt.includes('P2-Engine')
    const hasReq = txt.includes(REQNAME)
    const hasCat = txt.toLowerCase().includes('technisch')
    const hasPrio = /hoch/i.test(txt)
    out.push({
      step: 'S5b Ticket-Detail: Beschreibung + Kontakt + Kategorie + Priorität gemappt',
      hasDesc, hasReq, hasCat, hasPrio,
      pass: hasDesc && hasReq && hasCat && hasPrio,
    })
  }

  out.push({ step: 'S6 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 400), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Ticket-Intake P2 (shared engine: Formular → Ticket) — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
