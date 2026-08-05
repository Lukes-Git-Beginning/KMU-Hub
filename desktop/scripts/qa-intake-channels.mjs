/**
 * QA — Ticket-Intake per-channel forms (Block A: A1 store, A3 panel, A4 deep-link,
 * A5 new template, A7 self-service binding).
 *
 *   S1 Editor Kanäle panel: each enabled channel (agent + self-service) has its
 *      OWN form picker; agent bound to „Agent-Ticket (intern)", self-service to
 *      „Support-Ticket (Selfservice)". „Neue Ticket-Vorlage" button present.
 *   S2 Deep-link A4: /formulare?edit=form-ticket-agent opens the concrete form in
 *      the editor — shows the agent-specific fields „Gerät / Anlage" + „Fehlercode".
 *   S3 Self-service (A7): Settings → Über Cosmi → IT-Support renders the bound
 *      self-service form (Betreff/Beschreibung/Priorität), requester fields hidden.
 *   S4 No page errors.
 * Screenshots → .qa-screenshots/intake-channels/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/intake-channels')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

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
  // ── S0 Agent new-ticket dialog: bound template fields + internal tools ───────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Ticket', 15000)
  await page.getByRole('button', { name: /Neues Ticket/i }).first().click().catch(() => {})
  await wait(1200)
  await shot('00-agent-new-ticket-dialog.png')
  {
    const txt = await bodyText()
    // Agent form drives subject/description/category + extras; pro tools separate.
    const hasTemplateFields = txt.includes('Gerät / Anlage') && txt.includes('Fehlercode')
    const hasInternal = txt.toLowerCase().includes('interne zuordnung')
    out.push({ step: 'S0 Agent-Dialog: Vorlage-Felder (Gerät/Fehlercode) + interne Zuordnung', hasTemplateFields, hasInternal, pass: hasTemplateFields && hasInternal })
  }
  await page.keyboard.press('Escape').catch(() => {})
  await wait(500)

  // ── S1 Editor Kanäle-Panel: per-channel form pickers ────────────────────────
  let s1 = false
  try {
    await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
    await wait(2800)
    await page.getByRole('button', { name: /^Kanäle/ }).first().click({ timeout: 5000 })
    await wait(900)
    const txt = await bodyText()
    // Both bound forms visible as select options + the new-template button.
    const hasAgentForm = txt.includes('Agent-Ticket')
    const hasSelfForm = txt.includes('Support-Ticket (Selfservice)')
    const hasNewBtn = txt.includes('Neue Ticket-Vorlage')
    s1 = hasAgentForm && hasSelfForm && hasNewBtn
    await shot('01-editor-kanaele-per-channel.png')
    out.push({ step: 'S1 Kanäle-Panel: pro-Kanal Formular-Wähler + neue Vorlage', hasAgentForm, hasSelfForm, hasNewBtn, pass: s1 })
  } catch (e) {
    out.push({ step: 'S1 Kanäle-Panel', error: String(e).split('\n')[0], pass: false })
  }

  // ── S2 Deep-link A4: open the concrete agent form in the editor ─────────────
  await page.goto(`${BASE}/#/formulare?edit=form-ticket-agent`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Gerät', 10000)
  await shot('02-deeplink-agent-form-editor.png')
  {
    const txt = await bodyText()
    const opened = txt.includes('Gerät / Anlage') && txt.includes('Fehlercode')
    out.push({ step: 'S2 Deep-link öffnet konkretes Agent-Formular (Gerät/Anlage + Fehlercode)', opened, pass: opened })
  }

  // ── S3 Self-service form binding (A7) ───────────────────────────────────────
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await wait(2200)
  await page.getByRole('button', { name: /Über Cosmi/i }).first().click().catch(() => {})
  await wait(900)
  await page.getByRole('button', { name: /Support-Ticket erstellen/i }).first().click().catch(() => {})
  await wait(1200)
  await shot('03-selfservice-bound-form.png')
  {
    const txt = await bodyText()
    // Self-service form fields; requester name/email hidden (auto from profile).
    const hasFields = txt.includes('Betreff') && txt.includes('Beschreibung')
    const requesterHidden = !txt.includes('Ihr Name')
    out.push({ step: 'S3 Selfservice rendert gebundenes Formular, Requester versteckt', hasFields, requesterHidden, pass: hasFields && requesterHidden })
  }

  out.push({ step: 'S4 Keine Page-Errors', errCount: errs.length, errs: errs.slice(0, 5), pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e), pass: false })
}

console.log(JSON.stringify(out, null, 2))
const passed = out.filter((o) => o.pass).length
console.log(`\n${passed}/${out.length} passed`)
await b.close()
process.exit(passed === out.length ? 0 : 1)
