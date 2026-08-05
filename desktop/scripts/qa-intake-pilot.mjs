/**
 * QA — Ticket-Intake pilot completion (P4 self-service, Herkunfts-Reiter, P6 editor).
 *
 *   S1 Helpdesk: origin sub-tabs (Alle/Agent/Selfservice) + „Zusammenführen"-Toggle
 *      show because agent + self-service channels are on by default.
 *   S2 Settings → „Über Cosmi": IT-Support section + „Support-Ticket erstellen".
 *   S3 Dialog: fill Betreff + Beschreibung + consent → send → „Vielen Dank" + ref.
 *   S4 Helpdesk → Selfservice tab: the new ticket is there (channel = selfservice).
 *   S5 Editor: #/editor-window?module=helpdesk → „Kanäle" section + 3 toggles.
 *   S6 No page errors.
 * Screenshots → .qa-screenshots/intake-pilot/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/intake-pilot')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const SUBJECT = 'QA Pilot Selfservice-Ticket'
const DESC = 'P4-Selfservice: intern gemeldet, requester aus dem Profil.'

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
page.on('console', (m) => { if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED|Failed to load resource/.test(m.text())) errs.push('console: ' + m.text()) })
const out = []
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const bodyText = () => page.evaluate(() => document.body.innerText)
const waitForText = (x, timeout = 15000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

try {
  // ── S1 Herkunfts-Reiter ─────────────────────────────────────────────────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Ticket', 15000)
  await shot('01-helpdesk-source-tabs.png')
  {
    const txt = await bodyText()
    const hasTabs = txt.includes('Selfservice') && txt.includes('Zusammenführen')
    out.push({ step: 'S1 Herkunfts-Reiter (Alle/Agent/Selfservice) + Zusammenführen', hasTabs, pass: hasTabs })
  }

  // ── S2 Settings → Über → IT-Support ─────────────────────────────────────────
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await wait(2200)
  await page.getByRole('button', { name: /Über Cosmi/i }).first().click().catch(() => {})
  await wait(900)
  await waitForText('Support-Ticket erstellen', 8000)
  await shot('02-settings-it-support.png')
  {
    const txt = await bodyText()
    const hasEntry = txt.includes('Support-Ticket erstellen')
    out.push({ step: 'S2 Settings IT-Support-Einstieg', hasEntry, pass: hasEntry })
  }

  // ── S3 Selfservice dialog ───────────────────────────────────────────────────
  await page.getByRole('button', { name: /Support-Ticket erstellen/i }).first().click()
  await wait(900)
  const dlg = page.locator('[role="dialog"]').last()
  await dlg.locator('input[type="text"]').first().fill(SUBJECT)
  await dlg.locator('textarea').first().fill(DESC)
  await dlg.locator('input[type="checkbox"]').first().check().catch(() => {})
  const reportsAs = (await dlg.textContent())?.includes('Gemeldet als') ?? false
  await shot('03-selfservice-filled.png')
  await dlg.getByRole('button', { name: /Ticket senden|Wird gesendet/i }).click()
  await wait(1600)
  await shot('04-selfservice-submitted.png')
  {
    const txt = await bodyText()
    const created = txt.includes('Vielen Dank') && txt.includes('Dein Ticket')
    out.push({ step: 'S3 Selfservice-Formular erstellt ein Ticket', reportsAs, created, pass: created })
  }

  // ── S4 Helpdesk Selfservice-Tab zeigt das Ticket ────────────────────────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2600)
  await waitForText('Ticket', 12000)
  await page.getByRole('button', { name: /^Selfservice/ }).first().click().catch(() => {})
  await wait(900)
  await waitForText(SUBJECT, 6000).catch(() => {})
  await shot('05-helpdesk-selfservice-tab.png')
  {
    const txt = await bodyText()
    out.push({ step: 'S4 Selfservice-Ticket erscheint im Selfservice-Reiter', inList: txt.includes(SUBJECT), pass: txt.includes(SUBJECT) })
  }

  // ── S5 Editor Kanäle-Panel ──────────────────────────────────────────────────
  let editorOk = false
  try {
    await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
    await wait(2600)
    // Open the Kanäle section in the left rail.
    await page.getByRole('button', { name: /^Kanäle/ }).first().click({ timeout: 4000 })
    await wait(700)
    const txt = await bodyText()
    editorOk = txt.includes('Agent (im Modul)') && txt.includes('Selfservice') && txt.includes('Extern')
    await shot('06-editor-kanaele-panel.png')
  } catch { /* editor window may not fully render headless */ }
  out.push({ step: 'S5 Editor Kanäle-Panel (3 Toggles)', editorOk, pass: editorOk })

  out.push({ step: 'S6 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 400), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Ticket-Intake Pilot (P4 + Herkunft + P6 Editor) — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
