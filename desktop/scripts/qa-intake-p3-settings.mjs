/**
 * QA — Ticket-Intake P3 Settings: CSAT-Einstellung (an/aus + Verzögerung).
 *
 *   S1 Gate: mit tenant-Setting csatEnabled=false (localStorage) ist die
 *      CSAT-Kachel im Statistik-Tab AUSGEBLENDET → die Einstellung steuert die
 *      Sichtbarkeit end-to-end.
 *   S2 Settings-UI: Modul-Einstellungen öffnen → CSAT-Sektion mit an/aus-Schalter
 *      + Verzögerungs-Auswahl sichtbar.
 *   S3 Keine Page-Errors.
 * Screenshots → .qa-screenshots/intake-p3-settings/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/intake-p3-settings')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`
const CSAT_OFF = `try{localStorage.setItem('cosmi-helpdesk', JSON.stringify({state:{csatEnabled:false},version:0}))}catch(e){}`

const b = await chromium.launch({ headless: true })
const out = []
const errs = []
const shot = (page, n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (page, ms) => page.waitForTimeout(ms)

try {
  // ── S1 Gate: csatEnabled=false → CSAT weg im Statistik-Tab ──────────────────
  {
    const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
    await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH); await ctx.addInitScript(CSAT_OFF)
    const page = await ctx.newPage()
    page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
    await wait(page, 2800)
    await page.locator('button', { hasText: 'Statistik' }).first().click()
    await wait(page, 1200)
    await shot(page, '01-gate-off.png')
    const txt = await page.evaluate(() => document.body.innerText)
    const csatGone = !txt.includes('Kundenzufriedenheit')
    const othersThere = txt.includes('Offene Tickets')
    out.push({ step: 'S1 Gate off: CSAT ausgeblendet, andere Widgets da', csatGone, othersThere, pass: csatGone && othersThere })
    await ctx.close()
  }

  // ── S2 Settings-UI: CSAT-Sektion mit Schalter + Verzögerung ─────────────────
  {
    const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
    await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
    const page = await ctx.newPage()
    page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
    await wait(page, 2800)
    await page.getByText('Modul-Einstellungen', { exact: false }).first().click()
    await wait(page, 1600)
    // Zur CSAT-Sektion scrollen (falls unterhalb der Faltung)
    const csatHeading = page.getByText('Kundenzufriedenheit (CSAT)', { exact: false }).first()
    await csatHeading.scrollIntoViewIfNeeded().catch(() => {})
    await wait(page, 500)
    await shot(page, '02-settings-csat.png')
    const txt = await page.evaluate(() => document.body.innerText)
    const hasSection = txt.includes('Kundenzufriedenheit (CSAT)') || txt.includes('Umfrage nach Ticket-Schluss senden')
    const hasDelay = txt.includes('Verzögerung')
    const hasSwitch = (await page.locator('[role="switch"]').count()) > 0
    out.push({ step: 'S2 Settings: CSAT-Sektion + Schalter + Verzögerung', hasSection, hasDelay, hasSwitch, pass: hasSection && hasDelay && hasSwitch })
    await ctx.close()
  }

  out.push({ step: 'S3 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Ticket-Intake P3 Settings — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
