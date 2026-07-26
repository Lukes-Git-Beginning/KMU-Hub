/**
 * QA — Helpdesk Editor-Feedback Runde 2 (Darien lokal-Review, 2026-07-26).
 *
 *   S1 Ticket OHNE Beschreibung (hd-tk-003 „Neuer Mitarbeiter") → KEINE
 *      „Beschreibung"-Überschrift mehr (Punkt A).
 *   S2 Ticket MIT Beschreibung (Zutrittskarte) → Überschrift vorhanden.
 *   S3 Undo/Redo: Toolbar-Pfeile funktionieren — Änderung anlegen → rückgängig →
 *      wiederherstellen; Button-Zustände stimmen (Punkt C).
 *   S4 Keine Page-Errors.
 * Screenshots → .qa-screenshots/editor-helpdesk-feedback2/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-feedback2')
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
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})
const descHeadings = () => page.locator('h4', { hasText: 'Beschreibung' }).count()
const nameInputs = () => page.locator('input[aria-label="Name der Liste"]').count()

try {
  // ── LIVE APP: Beschreibung-Sektion (Punkt A) ───────────────────────────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Neuer Mitarbeiter', 15000)

  // S1 — ticket without a description → heading gone.
  await page.locator('tr[role="button"]', { hasText: 'Neuer Mitarbeiter' }).first().click()
  await wait(1000)
  await shot('01-no-description.png')
  {
    const count = await descHeadings()
    out.push({ step: 'S1 Kein „Beschreibung" bei leerem Feld', descHeadings: count, pass: count === 0 })
  }
  await page.keyboard.press('Escape').catch(() => {})
  await wait(400)

  // S2 — ticket WITH a description (Zutrittskarte now seeded) → heading present.
  await page.locator('tr[role="button"]', { hasText: 'Zutrittskarte' }).first().click()
  await wait(1000)
  await shot('02-with-description.png')
  {
    const count = await descHeadings()
    out.push({ step: 'S2 „Beschreibung" da, wenn Feld gefüllt', descHeadings: count, pass: count > 0 })
  }
  await page.keyboard.press('Escape').catch(() => {})
  await wait(400)

  // ── EDITOR: Undo/Redo (Punkt C) ────────────────────────────────────────────
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2800)
  await page.locator('nav[aria-label="Anpassen"] button', { hasText: 'Wertelisten' }).first().click()
  await wait(800)

  const undoBtn = page.locator('button[aria-label="Rückgängig"]').first()
  const redoBtn = page.locator('button[aria-label="Wiederholen"]').first()

  const undoDisabled0 = await undoBtn.isDisabled()
  // make a change: create a new value list (one atomic undo step)
  await page.locator('button', { hasText: 'Neue Werteliste' }).first().click()
  await wait(700)
  const afterCreate = await nameInputs()
  const undoEnabled = !(await undoBtn.isDisabled())
  await shot('03-after-create.png')

  // undo → the new list is gone again
  await undoBtn.click()
  await wait(700)
  const afterUndo = await nameInputs()
  const redoEnabled = !(await redoBtn.isDisabled())
  await shot('04-after-undo.png')

  // redo → the new list returns
  await redoBtn.click()
  await wait(700)
  const afterRedo = await nameInputs()
  await shot('05-after-redo.png')

  out.push({
    step: 'S3 Undo/Redo funktioniert + Button-Zustände',
    undoDisabled0, afterCreate, undoEnabled, afterUndo, redoEnabled, afterRedo,
    pass: undoDisabled0 && afterCreate === 1 && undoEnabled && afterUndo === 0 && redoEnabled && afterRedo === 1,
  })

  out.push({ step: 'S4 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk Editor-Feedback Runde 2 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
