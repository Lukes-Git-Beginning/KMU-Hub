/**
 * QA — Ticket-Intake P1a: Agent-Neu-Dialog reicht ALLE Felder durch.
 *
 *   S1 Neu-Dialog öffnen, Betreff + Beschreibung + Kontakt + SLA-Stufe füllen.
 *   S2 Ticket erstellen → erscheint in der Liste.
 *   S3 Ticket öffnen → Beschreibung, Kontaktperson und SLA-Stufe sind DA
 *      (früher gingen Beschreibung/Kontakt/Felder beim Anlegen verloren).
 *   S4 Keine Page-Errors.
 * Screenshots → .qa-screenshots/intake-p1a/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/intake-p1a')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const SUBJECT = 'QA Intake P1a Testticket'
const DESC = 'P1a-Testbeschreibung die beim Anlegen persistieren muss.'
const CONTACT = 'Testkontakt Meier'

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
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Ticket', 15000)

  // ── S1 Dialog öffnen + füllen ───────────────────────────────────────────────
  await page.getByRole('button', { name: /Neues Ticket/i }).first().click()
  await waitForText('Neues Ticket erstellen', 8000)
  await wait(600)
  await page.getByPlaceholder('Kurze Beschreibung des Problems...').fill(SUBJECT)
  await page.getByPlaceholder('Detaillierte Problembeschreibung...').fill(DESC)
  await page.getByPlaceholder('Name der Kontaktperson...').fill(CONTACT)
  // SLA-Stufe: the only select carrying the option "Kritisch".
  const slaSelect = page.locator('select').filter({ hasText: 'Kritisch' }).first()
  const slaFound = await slaSelect.count()
  if (slaFound) await slaSelect.selectOption({ label: 'Priorität' })
  await shot('01-dialog-filled.png')
  out.push({ step: 'S1 Neu-Dialog gefüllt (Betreff/Beschreibung/Kontakt/SLA)', slaFound, pass: slaFound > 0 })

  // ── S2 Erstellen ────────────────────────────────────────────────────────────
  await page.getByRole('button', { name: /erstellen/i }).last().click()
  await wait(1600)
  await shot('02-after-create.png')
  {
    const txt = await bodyText()
    out.push({ step: 'S2 Ticket erstellt, erscheint in der Liste', inList: txt.includes(SUBJECT), pass: txt.includes(SUBJECT) })
  }

  // ── S3 Öffnen → Felder persistiert ──────────────────────────────────────────
  await page.getByText(SUBJECT, { exact: false }).first().click()
  await wait(1200)
  await shot('03-new-ticket-detail.png')
  {
    const txt = await bodyText()
    const hasDesc = txt.includes('P1a-Testbeschreibung')
    const hasContact = txt.includes(CONTACT)
    const hasSla = txt.includes('Priorität')
    out.push({
      step: 'S3 Detail: Beschreibung + Kontakt + SLA-Stufe persistiert',
      hasDesc, hasContact, hasSla,
      pass: hasDesc && hasContact && hasSla,
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
console.log(`\n=== QA Ticket-Intake P1a (Agent-Durchreichung) — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
