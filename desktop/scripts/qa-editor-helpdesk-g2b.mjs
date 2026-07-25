/**
 * QA — Helpdesk G2b: Zusatzfelder sind IM TICKET editierbar (Darien-Feedback).
 *
 *   S1 Ticket „Zutrittskarte…" (hd-tk-010) öffnen → Zusatzfelder mit Werten +
 *      SLA-Stufe ist ein Dropdown (Wert „Priorität"), Eskalationsgrund ein Text.
 *   S2 SLA-Stufe auf „Kritisch" umstellen → Wert übernommen (editierbar).
 *   S3 Eskalationsgrund neu eintippen → Wert übernommen.
 *   S4 Editor-Panel „Zusatzfelder": Typ+Optionen sichtbar („Auswahl · Standard, …").
 *   S5 Keine Page-Errors.
 * Screenshots → .qa-screenshots/editor-helpdesk-g2b/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-g2b')
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

// The SLA-Stufe custom-field select is the one carrying option value "Priorität".
const slaSelect = () => page.locator('select').filter({ has: page.locator('option[value="Priorität"]') }).first()

try {
  // ── LIVE APP ───────────────────────────────────────────────────────────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Zutrittskarte', 15000)

  // S1 — open the ticket he opened (hd-tk-010) → editable Zusatzfelder pre-filled.
  await page.locator('tr[role="button"]', { hasText: 'Zutrittskarte' }).first().click()
  await wait(1200)
  await shot('01-detail-editable.png')
  {
    const selCount = await slaSelect().count()
    const slaVal = selCount ? await slaSelect().inputValue() : ''
    const escInput = page.locator('input[value="Mitarbeiter kommt nicht ins Büro"]')
    const escFilled = (await escInput.count()) > 0
    out.push({ step: 'S1 Zusatzfelder editierbar + vorbefüllt', selCount, slaVal, escFilled, pass: selCount > 0 && slaVal === 'Priorität' && escFilled })
  }

  // S2 — change SLA-Stufe dropdown to "Kritisch".
  await slaSelect().selectOption('Kritisch')
  await wait(500)
  await shot('02-sla-changed.png')
  {
    const slaVal = await slaSelect().inputValue()
    out.push({ step: 'S2 SLA-Stufe auf „Kritisch" umstellbar', slaVal, pass: slaVal === 'Kritisch' })
  }

  // S3 — type a new Eskalationsgrund.
  const escInput = page.locator('input[value="Mitarbeiter kommt nicht ins Büro"]').first()
  let escOk = false
  if ((await escInput.count()) > 0) {
    await escInput.fill('Türsteuerung komplett ausgefallen')
    await wait(400)
    escOk = (await page.locator('input[value="Türsteuerung komplett ausgefallen"]').count()) > 0
  }
  await shot('03-esc-typed.png')
  out.push({ step: 'S3 Eskalationsgrund eintippbar', escOk, pass: escOk })
  await page.keyboard.press('Escape').catch(() => {})
  await wait(500)

  // ── EDITOR ───────────────────────────────────────────────────────────────
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2800)
  await page.locator('nav[aria-label="Anpassen"] button', { hasText: 'Zusatzfelder' }).first().click()
  await wait(1000)
  await shot('04-panel-types.png')
  {
    const txt = await bodyText()
    const hasTypeLine = txt.includes('Auswahl · Standard, Priorität, Kritisch')
    out.push({ step: 'S4 Panel zeigt Typ + Optionen', hasTypeLine, pass: hasTypeLine })
  }

  out.push({ step: 'S5 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk G2b — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
