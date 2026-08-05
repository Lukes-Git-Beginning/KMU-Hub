/**
 * QA — Dariens Feedback Runde 4 (2026-08-05), Spalten-Panel.
 *
 *   I1 Eingebaute Spalten (Betreff, Status, SLA …) lassen sich umbenennen —
 *      im Panel, und der Name landet in derselben Quelle wie der Klick auf die
 *      Überschrift in der Vorschau.
 *   I2 Die Reihenfolge der Spalten lässt sich anordnen; die Tabelle folgt.
 *   I3 Die Breite lässt sich am rechten Rand der Überschrift ziehen; das <th>
 *      trägt sie und das Panel zeigt sie an.
 *   I4 Eingebaute Spalten bleiben unlöschbar — mit sichtbarer Begründung.
 *   I5 Regression: Ein-/Ausblenden wirkt weiterhin.
 * Screenshots → .qa-screenshots/editor-spalten-i/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-spalten-i')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const RENAMED = 'Anliegen'

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1600, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
page.on('console', (m) => {
  if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED|Failed to load resource/.test(m.text())) errs.push('console: ' + m.text())
})

const out = []
let pass = 0, fail = 0
const check = (n, ok, extra = '') => { out.push(`${ok ? 'PASS' : 'FAIL'}  ${n}${extra ? ' · ' + extra : ''}`); ok ? pass++ : fail++ }
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const text = () => page.evaluate(() => document.body.innerText)
const rail = (name) => page.getByRole('button', { name }).first()
const headers = () => page.locator('table thead th').allInnerTexts()

try {
  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(3500)
  await rail(/^Spalten/i).click()
  await wait(1600)
  await shot('i0-panel.png')

  const startCols = await headers()
  check('I0 Panel und Tabelle stehen', startCols.length > 3, startCols.join(' | '))

  // ── I1 · eingebaute Spalte umbenennen ────────────────────────────────────
  await page.getByRole('button', { name: /Spalte Betreff umbenennen/i }).first().click()
  await wait(600)
  const input = page.locator('input[aria-label*="umbenennen"]').first()
  await input.fill(RENAMED)
  await input.press('Enter')
  await wait(1800)
  await shot('i1-eingebaute-spalte-umbenannt.png')
  const renamedCols = await headers()
  check('I1a Eingebaute Spalte lässt sich im Panel umbenennen',
    renamedCols.some((h) => h.includes(RENAMED)), renamedCols.join(' | '))
  check('I1b Das Panel zeigt denselben Namen (eine Quelle)', (await text()).includes(RENAMED))

  // ── I4 · unlöschbar, aber mit Begründung ─────────────────────────────────
  check('I4 Eingebaute Spalten tragen die Begründung, warum sie bleiben',
    (await text()).toLowerCase().includes('nicht löschen'))

  // ── I5 · Regression Sichtbarkeit (vor dem Ziehen, damit kein Drag nachwirkt) ─
  const colsBeforeHide = await headers()
  // Der Sichtbarkeits-Schalter ist role="switch" (aria-checked), NICHT button.
  await page.getByRole('switch', { name: /Spalte Kategorie ausblenden/i }).first().click()
  await wait(1500)
  const colsAfterHide = await headers()
  await shot('i5-spalte-ausgeblendet.png')
  check('I5 Ausblenden wirkt weiterhin',
    colsAfterHide.length === colsBeforeHide.length - 1, `${colsBeforeHide.length} → ${colsAfterHide.length}`)

  // ── I2 · Reihenfolge ─────────────────────────────────────────────────────
  // dnd-kit: Griff fokussieren, Leertaste hebt auf, Pfeil verschiebt, Leertaste legt ab.
  const before = await headers()
  const grip = page.getByRole('button', { name: /Spalte Ticket-Nr\.? verschieben/i }).first()
  await grip.focus()
  await page.keyboard.press('Space')
  await wait(500)
  await page.keyboard.press('ArrowDown')
  await wait(500)
  await page.keyboard.press('Space')
  await wait(1800)
  await shot('i2-reihenfolge-getauscht.png')
  const after = await headers()
  check('I2a Die Reihenfolge im Panel wirkt in der Tabelle',
    before[0] !== after[0] && after.includes(before[0]),
    `vorher: ${before.slice(0, 3).join(' | ')} → nachher: ${after.slice(0, 3).join(' | ')}`)
  check('I2b Es geht keine Spalte verloren', before.length === after.length, `${before.length} → ${after.length}`)

  // ── I3 · Breite ziehen ───────────────────────────────────────────────────
  const handle = page.locator('[data-column-resize]').first()
  const key = await handle.getAttribute('data-column-resize')
  const box = await handle.boundingBox()
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.mouse.down()
  await page.mouse.move(box.x + 80, box.y + box.height / 2, { steps: 8 })
  await page.mouse.move(box.x + 160, box.y + box.height / 2, { steps: 8 })
  await page.mouse.up()
  await wait(1500)
  await shot('i3-breite-gezogen.png')
  const width = await page.locator('table thead th').first().evaluate((el) => el.style.width)
  check('I3a Das <th> trägt die gezogene Breite', /%$/.test(width || ''), `${key} → ${width || '(leer)'}`)
  check('I3b Das Panel zeigt die Breite in Prozent', /Breite\s+\d+\s*%/.test(await text()))
  const fixed = await page.locator('table').first().evaluate((el) => el.style.tableLayout)
  check('I3c Tabelle schaltet auf feste Breiten um', fixed === 'fixed', fixed || '(leer)')
  // Der erste Zug friert die Ist-Breiten ein: die Tabelle darf dabei nicht
  // gleichverteilt zusammenklappen, alle übrigen Spalten behalten eine Breite.
  const allSized = await page.locator('table thead th').evaluateAll(
    (cells) => cells.every((c) => c.style.width !== ''),
  )
  check('I3d Alle übrigen Spalten behalten ihre bisherige Breite', allSized)

  // ── I6 · Übernehmen: Name + Reihenfolge + Breite überleben den Deploy ─────
  // Genau hier hing der Fehler: der Deploy filtert Labels gegen LABEL_WHITELIST,
  // in der die Spaltenüberschriften fehlten — umbenennen sah in der Vorschau gut
  // aus und war nach „Übernehmen" wieder weg.
  const draftCols = await headers()
  const apply = page.locator('button').filter({ hasText: /^Übernehmen$/ }).first()
  await apply.dispatchEvent('click')
  await wait(900)
  await page.locator('button').filter({ hasText: 'Jetzt übernehmen' }).first().click({ force: true })
  await wait(2500)
  await shot('i6a-uebernommen.png')

  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(3000)
  await shot('i6b-live-modul.png')
  const liveCols = await headers()
  check('I6a Umbenannte Spalte ist auch live da', liveCols.some((h) => h.includes(RENAMED)), liveCols.join(' | '))
  check('I6b Reihenfolge ist auch live übernommen',
    liveCols[0] === draftCols[0], `Entwurf: ${draftCols[0]} · live: ${liveCols[0]}`)
  const liveWidth = await page.locator('table thead th').first().evaluate((el) => el.style.width)
  check('I6c Breite ist auch live übernommen', /%$/.test(liveWidth || ''), liveWidth || '(leer)')
  check('I6d Ausgeblendete Spalte bleibt live aus',
    !liveCols.some((h) => h.trim() === 'Kategorie'), liveCols.join(' | '))

  check('Keine Seitenfehler', errs.length === 0, errs.slice(0, 3).join(' | '))
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n')[0])
  await shot('zz-abbruch.png')
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await b.close()
process.exit(fail > 0 ? 1 : 0)
