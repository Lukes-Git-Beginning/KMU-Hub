/**
 * QA — Helpdesk-Pilot P3 (Editor-Pivot): R4 Tab-Sichtbarkeit (moduleAreas).
 *
 * Beweist: Reiter lassen sich im Editor pro Mandant ausblenden — live in der Vorschau.
 *   S1 Helpdesk-Editor offen, alle 3 Reiter da (Tickets/Wissensdatenbank/Statistik).
 *   S2 „Bereiche"-Panel (links) zeigt die 3 Reiter mit „Sichtbar"-Schaltern.
 *   S3 „Statistik" ausblenden → der Statistik-REITER verschwindet LIVE aus der Modul-Leiste.
 *   S4 Keine Page-Errors.
 *
 * Screenshots → .qa-screenshots/editor-helpdesk-p3/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-p3')
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
const bodyText = () => page.evaluate(() => document.body.innerText)
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})
const tabBtn = () => page.locator('button', { hasText: /^Statistik$/ })

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2500)
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2500)
  await shot('01-editor-open.png')
  {
    const statTabBefore = await tabBtn().count()
    out.push({ step: 'S1 Editor offen, Statistik-Reiter vorhanden', statTabBefore, pass: statTabBefore > 0 })
  }

  // S2 — open "Bereiche" section
  await page.locator('button', { hasText: /^Bereiche$/ }).first().click()
  await wait(1000)
  await shot('02-bereiche-panel.png')
  {
    const switches = await page.locator('[role="switch"]').count()
    out.push({ step: 'S2 Bereiche-Panel mit Schaltern', switches, pass: switches >= 3 })
  }

  // S3 — hide "Statistik" and expect the tab to vanish from the module bar
  const statSwitch = page.locator('[role="switch"][aria-label*="Statistik"]').first()
  const switchFound = (await statSwitch.count()) > 0
  if (switchFound) {
    await statSwitch.click()
    await wait(1000)
  }
  await shot('03-statistik-hidden.png')
  {
    const statTabAfter = await tabBtn().count()
    const txt = await bodyText()
    out.push({
      step: 'S3 „Statistik" ausgeblendet → Reiter weg aus der Modul-Leiste',
      switchFound,
      statTabAfter,
      dirty: /Änderung/.test(txt),
      pass: switchFound && statTabAfter === 0,
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
console.log(`\n=== QA Helpdesk-Pilot P3 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
