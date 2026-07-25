/**
 * QA — Helpdesk Editor G2: Zusatzfelder (custom fields) edit-in-place sichtbar.
 *
 *   LIVE APP (#/helpdesk):
 *     S1 Ticket mit Werten öffnen → „Zusatzfelder"-Sektion zeigt SLA-Stufe/
 *        Eskalationsgrund/Kontaktkanal mit echten Werten.
 *     S2 „Neues Ticket"-Dialog → Custom-Felder erscheinen als Eingaben.
 *   EDITOR SANDBOX (#/editor-window?module=helpdesk):
 *     S3 Ticket im Sandbox öffnen → Zusatzfelder-Sektion rendert (alle def. Felder).
 *     S4 „Zusatzfelder"-Panel (Nav umbenannt) → Intro-Zeile + Feld-Liste.
 *     S5 Feld umbenennen (SLA-Stufe → Service-Level) → Sandbox-Detail zeigt es LIVE.
 *     S6 Keine Page-Errors.
 * Screenshots → .qa-screenshots/editor-helpdesk-g2/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-g2')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const bodyText = () => page.evaluate(() => document.body.innerText)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

try {
  // ── LIVE APP ───────────────────────────────────────────────────────────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('VPN-Verbindung', 15000)

  // S1 — open a ticket that has demo custom values (hd-tk-002 "VPN-Verbindung…")
  await page.locator('tr[role="button"]', { hasText: 'VPN-Verbindung' }).first().click()
  await wait(1200)
  await shot('01-detail-values.png')
  {
    const txt = await bodyText()
    const hasSection = /Zusatzfelder|Zusatz-?felder/i.test(txt) || txt.includes('SLA-Stufe')
    const hasSla = txt.includes('SLA-Stufe')
    const hasEsc = txt.includes('Eskalationsgrund')
    const hasChannel = txt.includes('Kontaktkanal')
    const hasValue = txt.includes('Kritisch') && txt.includes('E-Mail')
    out.push({ step: 'S1 Zusatzfelder-Sektion mit Werten (Live)', hasSection, hasSla, hasEsc, hasChannel, hasValue, pass: hasSla && hasEsc && hasChannel && hasValue })
  }

  // Close the detail modal (DetailModal = Radix Dialog, Escape) before the dialog.
  await page.keyboard.press('Escape').catch(() => {})
  await wait(700)

  // S2 — new-ticket dialog shows custom-field inputs
  await page.locator('button', { hasText: /^Neues Ticket$/ }).first().click().catch(() => {})
  await wait(1000)
  await shot('02-new-ticket-inputs.png')
  {
    const txt = await bodyText()
    const dialogOpen = txt.includes('Neues Ticket erstellen')
    const inDialog = txt.includes('SLA-Stufe') && txt.includes('Eskalationsgrund') && txt.includes('Kontaktkanal')
    // sla_tier is a select → its option "Standard" appears ONLY in the new-ticket
    // form (not in the read-only detail), so it discriminates the two.
    const slaOption = await page.locator('option', { hasText: /^Standard$/ }).count()
    out.push({ step: 'S2 Custom-Felder als Eingaben im Neu-Dialog', dialogOpen, inDialog, slaOption, pass: dialogOpen && inDialog && slaOption > 0 })
  }
  // close dialog
  await page.keyboard.press('Escape').catch(() => {})
  await wait(500)

  // ── EDITOR SANDBOX ───────────────────────────────────────────────────────────
  // NB: the ticket detail is a centered DetailModal (dims the nav/panel) — so we
  // interact with the panel FIRST, then open the modal, then close it before the
  // next panel interaction.
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2800)

  const openNavPanel = () =>
    page.locator('nav[aria-label="Anpassen"] button', { hasText: 'Zusatzfelder' }).first().click()

  // S3 — open the Felder panel (nav renamed to "Zusatzfelder") → intro + field list.
  await openNavPanel()
  await wait(1000)
  await shot('03-felder-panel.png')
  {
    const txt = await bodyText()
    const hasIntro = /erscheinen im Detail|Anlege-Formular/i.test(txt)
    const hasFieldRow = txt.includes('SLA-Stufe')
    out.push({ step: 'S3 „Zusatzfelder"-Panel: Intro + Feld-Liste', hasIntro, hasFieldRow, pass: hasIntro && hasFieldRow })
  }

  // S4 — open a ticket in the sandbox → detail modal shows Zusatzfelder (baseline).
  await page.locator('tr[role="button"]', { hasText: 'VPN-Verbindung' }).first().click().catch(() => {})
  await wait(1200)
  await shot('04-sandbox-detail.png')
  {
    const txt = await bodyText()
    const hasSla = txt.includes('SLA-Stufe')
    const hasEsc = txt.includes('Eskalationsgrund')
    const hasChannel = txt.includes('Kontaktkanal')
    out.push({ step: 'S4 Zusatzfelder rendern im Sandbox-Detail', hasSla, hasEsc, hasChannel, pass: hasSla && hasEsc && hasChannel })
  }
  // Close the modal so the panel is interactable again.
  await page.keyboard.press('Escape').catch(() => {})
  await wait(700)

  // S5 — rename a field (SLA-Stufe → Service-Level) in the panel, then reopen the
  //      ticket → the detail shows the new label live (draft reaches the module).
  let renamed = false
  await openNavPanel().catch(() => {})
  await wait(600)
  const fieldRow = page.locator('[role="button"]', { hasText: /^SLA-Stufe$/ }).first()
  if ((await fieldRow.count()) > 0) {
    await fieldRow.click()
    await wait(900)
    // The FieldEditorModal has a label text input pre-filled with "SLA-Stufe".
    const byValue = page.locator('input')
    const count = await byValue.count()
    for (let i = 0; i < count && !renamed; i++) {
      const el = byValue.nth(i)
      const val = await el.inputValue().catch(() => '')
      if (val === 'SLA-Stufe') { await el.fill('Service-Level'); renamed = true }
    }
    await shot('05a-rename-modal.png')
    await page.locator('button', { hasText: /^Speichern$/ }).first().click().catch(() => {})
    await wait(1000)
  }
  // Reopen the ticket → detail should now show "Service-Level".
  await page.locator('tr[role="button"]', { hasText: 'VPN-Verbindung' }).first().click().catch(() => {})
  await wait(1000)
  await shot('05b-after-rename.png')
  {
    const txt = await bodyText()
    const detailShowsNew = txt.includes('Service-Level')
    out.push({ step: 'S5 Feld umbenannt → Sandbox-Detail zeigt „Service-Level" live', renamed, detailShowsNew, pass: renamed && detailShowsNew })
  }

  out.push({ step: 'S6 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk G2 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
