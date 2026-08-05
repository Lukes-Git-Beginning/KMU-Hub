/**
 * QA — Ticket-Intake P1b: Custom-Fields-Edit-Persistenz Overlay→Wire + Seed-Migration.
 *
 *   S1 tk-006-Detail: Zusatzfelder kommen jetzt VOM WIRE (Seed-Migration) —
 *      SLA-Stufe „Kritisch", Eskalationsgrund „…Monatsabschluss gefährdet",
 *      Kontaktkanal „Vor Ort".
 *   S2 SLA-Stufe (Select) → „Standard" ändern (commit-on-change) + Eskalationsgrund
 *      (Text) neu tippen + Blur (commit-on-blur).
 *   S3 Weg-navigieren (#/) + zurück (#/helpdesk) → HelpdeskPage unmountet, der
 *      Session-Buffer ist WEG. tk-006 erneut öffnen: SLA „Standard" + neuer
 *      Eskalationstext sind DA → aus dem Wire persistiert (nicht nur Buffer).
 *   S4 Keine Page-Errors.
 * Screenshots → .qa-screenshots/intake-p1b/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/intake-p1b')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const NEW_ESCALATION = 'P1b Edit persistiert auf dem Wire'

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

async function openTk006() {
  await page.getByText('ERP-System', { exact: false }).first().click()
  await wait(1000)
}

try {
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Ticket', 15000)

  // Input-VALUES stehen NICHT im innerText → per .inputValue() prüfen.
  const slaSelect = () => page.locator('select').filter({ hasText: 'Kritisch' }).first()
  const escInput = () => page.getByPlaceholder('Wert eingeben…').first()

  // ── S1 Seed-Werte vom Wire ──────────────────────────────────────────────────
  await openTk006()
  await shot('01-tk006-wire-seeds.png')
  {
    const slaVal = await slaSelect().inputValue()
    const escVal = await escInput().inputValue()
    const hasChannel = (await bodyText()).includes('Vor Ort')
    out.push({
      step: 'S1 Zusatzfelder vom Wire (Seed-Migration)',
      slaVal, escVal, hasChannel,
      pass: slaVal === 'Kritisch' && escVal.includes('Monatsabschluss gefährdet') && hasChannel,
    })
  }

  // ── S2 Editieren: SLA-Select → Standard + Eskalations-Text + Blur ───────────
  await slaSelect().selectOption({ label: 'Standard' })
  await wait(700)
  await escInput().fill(NEW_ESCALATION)
  await escInput().blur()
  await wait(900)
  await shot('02-tk006-edited.png')
  out.push({ step: 'S2 SLA-Select + Eskalations-Text editiert (commit)', pass: true })

  // ── S3 Weg + zurück (Buffer weg) → aus dem Wire persistiert ─────────────────
  await page.evaluate(() => { window.location.hash = '#/' })
  await wait(1400)
  await page.evaluate(() => { window.location.hash = '#/helpdesk' })
  await wait(2400)
  await waitForText('Ticket', 15000)
  await openTk006()
  await shot('03-tk006-after-remount.png')
  {
    const slaVal = await slaSelect().inputValue()
    const escVal = await escInput().inputValue()
    out.push({
      step: 'S3 Nach Remount: SLA „Standard" + Eskalationstext aus dem Wire',
      slaVal, escVal,
      pass: slaVal === 'Standard' && escVal === NEW_ESCALATION,
    })
  }

  out.push({ step: 'S4 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Ticket-Intake P1b (Edit-Persistenz + Seed-Migration) — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
