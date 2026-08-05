/**
 * QA — Ticket-Intake P3 (Kern): CSAT nutzbar im Statistik-Tab.
 *
 *   S1 Statistik-Tab: CSAT-Kachel „Kundenzufriedenheit" ist SICHTBAR (früher
 *      ausgeblendet) und zeigt den echten Schnitt der 3 Seed-Ratings (5,4,3 → 4.0/5).
 *   S2 CSAT-Verteilungs-Chart (CSATAggregate) sichtbar mit „3 Bewertungen".
 *   S3 Keine Page-Errors.
 * Screenshots → .qa-screenshots/intake-p3/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/intake-p3')
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
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Ticket', 15000)
  await page.locator('button', { hasText: 'Statistik' }).first().click()
  await wait(1300)
  await shot('01-statistik-csat.png')

  const txt = await bodyText()
  const hasCsatLabel = txt.includes('Kundenzufriedenheit')
  const hasAvg = txt.includes('4.0')
  const hasCount = /3\s*Bewertungen/.test(txt)
  out.push({ step: 'S1 CSAT-Kachel sichtbar + echter Schnitt 4.0/5', hasCsatLabel, hasAvg, pass: hasCsatLabel && hasAvg })
  out.push({ step: 'S2 CSAT-Verteilungs-Chart mit „3 Bewertungen"', hasCount, pass: hasCount })
  out.push({ step: 'S3 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Ticket-Intake P3 (CSAT nutzbar) — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
