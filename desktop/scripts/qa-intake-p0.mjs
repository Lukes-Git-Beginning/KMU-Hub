/**
 * QA — Ticket-Intake P0 (Daten-Fundament): CSAT auf dem aktiven Wire-Pfad.
 *
 *   S1 Helpdesk-Liste lädt, solved/closed-Tickets sichtbar (Default-Filter 'all').
 *   S2 Resolved-Ticket MIT Seed-Rating (tk-006) → CSAT-Widget zeigt 5/5 + Kommentar
 *      (Beweis: Seed-Ratings jetzt im aktiven Wire-Pfad, nicht im Legacy-Store).
 *   S3 Resolved-Ticket OHNE Rating (tk-009) → CSAT-Formular sichtbar.
 *   S4 Bewertung abgeben (4 Sterne) → wird auf das Wire-Ticket geschrieben,
 *      Widget wechselt in den abgegebenen Zustand (4/5).
 *   S5 Keine Page-Errors.
 * Screenshots → .qa-screenshots/intake-p0/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/intake-p0')
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

async function openTicketBySubject(fragment) {
  const el = page.getByText(fragment, { exact: false }).first()
  await el.scrollIntoViewIfNeeded().catch(() => {})
  await el.click()
  await wait(1000)
}

try {
  // ── S1 Liste lädt, solved sichtbar ─────────────────────────────────────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Ticket', 15000)
  await shot('01-list.png')
  {
    const txt = await bodyText()
    const hasSolved = txt.includes('ERP-System') || txt.includes('Sharepoint')
    out.push({ step: 'S1 Liste lädt, solved-Tickets sichtbar', hasSolved, pass: hasSolved })
  }

  // ── S2 Resolved MIT Seed-Rating (tk-006, ERP-System, Rating 5) ──────────────
  await openTicketBySubject('ERP-System')
  await shot('02-tk006-detail-csat5.png')
  {
    const txt = await bodyText()
    const hasWidget = txt.includes('Kundenfeedback')
    const shows5 = txt.includes('5/5')
    const hasComment = txt.includes('Rechnungslauf lief nach kurzer Zeit')
    out.push({
      step: 'S2 tk-006: Seed-Rating 5/5 + Kommentar im aktiven Pfad',
      hasWidget, shows5, hasComment,
      pass: hasWidget && shows5 && hasComment,
    })
  }

  // ── S3 Resolved OHNE Rating (tk-009, Backup-Fehler) → Formular ──────────────
  // zurück zur Liste
  await page.goBack().catch(() => {})
  await wait(800)
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2200)
  await openTicketBySubject('Backup-Fehler')
  await shot('03-tk009-detail-form.png')
  {
    const txt = await bodyText()
    const hasWidget = txt.includes('Kundenfeedback')
    const hasSubmitBtn = txt.includes('Bewertung abgeben')
    out.push({
      step: 'S3 tk-009: CSAT-Formular sichtbar (noch keine Bewertung)',
      hasWidget, hasSubmitBtn,
      pass: hasWidget && hasSubmitBtn,
    })
  }

  // ── S4 Bewertung abgeben (4 Sterne) → Wire-Write, submitted state ───────────
  const starBtns = page.locator('button:has(.lucide-star)')
  const starCount = await starBtns.count()
  if (starCount >= 4) await starBtns.nth(3).click()
  await wait(400)
  await page.getByText('Bewertung abgeben', { exact: false }).first().click()
  await wait(1500)
  await shot('04-tk009-submitted.png')
  {
    const txt = await bodyText()
    const submitted = txt.includes('4/5')
    out.push({
      step: 'S4 tk-009: 4 Sterne abgegeben → auf Wire geschrieben (4/5)',
      starCount, submitted,
      pass: starCount >= 5 && submitted,
    })
  }

  out.push({ step: 'S5 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Ticket-Intake P0 (CSAT Wire-Pfad) — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
