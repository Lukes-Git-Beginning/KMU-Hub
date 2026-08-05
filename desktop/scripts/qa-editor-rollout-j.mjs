/**
 * QA — Dariens Review-Feedback Runde 5 (2026-08-05).
 *
 *   J1 Der Verstellpunkt der Spaltenbreite ist SICHTBAR — aber nur im Editor und
 *      nur solange der Spalten-Bereich offen ist.
 *   J2 Die Breitenangabe im Panel ist Anzeige, das Zurücksetzen ist beschriftet.
 *   J3 „Als Entwurf speichern" merkt sich die Änderungen: der Entwurf lässt sich
 *      fortsetzen und bringt seinen Stand mit (vorher: leerer Editor).
 *   J4 Ein Rollout/Entwurf öffnet auf Klick ein Detail-Fenster mit Terminplanung
 *      und Nachricht; die Nachricht erreicht die Nutzer im Modul.
 *
 * NB: Hash-Navigation statt page.goto — ein echter Reload setzt den In-Memory-
 * Draft-Store zurück und würde genau das wegwerfen, was hier geprüft wird.
 * Screenshots → .qa-screenshots/editor-rollout-j/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-rollout-j')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const RENAMED = 'Anliegen'
const MESSAGE = 'Die Ticket-Liste hat jetzt andere Spalten.'

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
const hash = async (h) => { await page.evaluate((x) => { window.location.hash = x }, h); await wait(2600) }
/** Sichtbarkeit des Zieh-Griffs: opacity-0 heißt „da, aber unsichtbar". */
const gripVisible = () =>
  page.locator('[data-column-resize]').first().evaluate((el) => getComputedStyle(el).opacity !== '0')

try {
  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(3800)

  // ── J1 · Verstellpunkt nur im Spalten-Bereich sichtbar ────────────────────
  await shot('j1a-editor-ohne-spaltenbereich.png')
  check('J1a Ohne Spalten-Bereich bleibt der Verstellpunkt unsichtbar', (await gripVisible()) === false)

  await rail(/^Spalten/i).click()
  await wait(1600)
  await shot('j1b-editor-spaltenbereich.png')
  check('J1b Im Spalten-Bereich ist der Verstellpunkt sichtbar', (await gripVisible()) === true)

  // ── J2 · Breite ziehen → Anzeige + beschriftetes Zurücksetzen ─────────────
  const handle = page.locator('[data-column-resize]').nth(1)
  const box = await handle.boundingBox()
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.mouse.down()
  await page.mouse.move(box.x + 90, box.y + box.height / 2, { steps: 8 })
  await page.mouse.up()
  await wait(1500)
  await shot('j2-breite-mit-zuruecksetzen.png')
  const panelText = await text()
  check('J2a Die Breite steht als Angabe im Panel', /Breite\s+\d+\s*%/.test(panelText))
  check('J2b Das Zurücksetzen ist beschriftet, nicht nur eine Zahl', panelText.includes('Zurücksetzen'))

  // ── J3 · Umbenennen, als Entwurf speichern ───────────────────────────────
  await page.getByRole('button', { name: /Spalte Betreff umbenennen/i }).first().click()
  await wait(600)
  const input = page.locator('input[aria-label*="umbenennen"]').first()
  await input.fill(RENAMED)
  await input.press('Enter')
  await wait(1500)
  check('J3a Umbenennen wirkt in der Vorschau', (await headers()).some((h) => h.includes(RENAMED)))
  await page.locator('button').filter({ hasText: /^Als Entwurf speichern$/ }).first().dispatchEvent('click')
  await wait(1600)
  await shot('j3b-entwurf-gespeichert.png')

  // ── Hub: Entwurf steht in der Liste ──────────────────────────────────────
  // Auf die Zeile warten statt auf eine feste Wartezeit: die Liste kommt über
  // einen Refetch herein, ein innerText-Schnappschuss ist sonst ein Wettlauf.
  await hash('#/admin/anpassungen')
  const row = page.locator('div[role="button"]').filter({ hasText: /Helpdesk — Anpassung/ }).last()
  await row.waitFor({ state: 'visible', timeout: 15000 })
  await shot('j3c-hub-mit-entwurf.png')
  check('J3b Der Entwurf steht in „Rollouts & Entwürfe"', (await row.innerText()).includes('Entwurf'))

  // ── J4 · Zeile klickbar → Detail-Fenster ─────────────────────────────────
  await row.click()
  await wait(1400)
  await shot('j4a-rollout-detail.png')
  const modalText = await text()
  // Die Abschnitts-Überschriften sind per CSS uppercase — innerText liefert sie
  // großgeschrieben, also case-insensitiv prüfen (Lehre aus Runde 3).
  const modalLower = modalText.toLowerCase()
  check('J4a Klick auf die Zeile öffnet das Detail-Fenster',
    modalLower.includes('was sich ändert') && modalLower.includes('nachricht an die nutzer'))
  check('J4b Das Fenster sagt, was der Rollout ändert', /\d+\s+Spalten?/.test(modalText),
    (modalText.match(/\d+ (Spalten?|Begriffe?|Wertelisten?)/g) ?? []).join(' | '))

  // Nachricht hinterlegen (blur speichert)
  const note = page.locator('textarea').first()
  await note.fill(MESSAGE)
  await note.blur()
  await wait(1200)
  check('J4c Nachricht wird am Rollout gespeichert', (await text()).includes('Nachricht gespeichert'))

  // Terminieren + Termin wieder entfernen
  await page.getByRole('button', { name: /^Terminieren$/ }).first().click()
  await wait(1400)
  await shot('j4b-terminiert.png')
  check('J4d Terminieren setzt den Rollout auf „Geplant"', (await text()).includes('Geplant'))
  await page.getByRole('button', { name: /Termin entfernen/i }).first().click()
  await wait(1400)
  check('J4e Termin entfernen lässt ihn als Entwurf liegen', (await text()).includes('Termin entfernt'))

  // ── J3 (Kern) · „Weiter bearbeiten" bringt den Stand mit ─────────────────
  // Zwei Hälften, weil der Editor in einem eigenen OS-Fenster aufgeht: der Hub
  // legt den Entwurf zur Übergabe ab, das Editor-Fenster nimmt ihn auf. Der
  // electronAPI-Stub führt openWindow(...) nicht aus, darum wird der zweite Teil
  // hier per Navigation nachgestellt — der Übergabeweg ist derselbe.
  await page.getByRole('button', { name: /Weiter bearbeiten/i }).first().click()
  await wait(1200)
  const stashed = await page.evaluate(() => {
    const raw = localStorage.getItem('cosmi:customization:resume-draft')
    if (!raw) return null
    const d = JSON.parse(raw)
    return { module: d.moduleKey, labels: Object.keys(d.payload?.labels?.de ?? {}).length }
  })
  check('J3c Fortsetzen übergibt den Entwurf samt Änderungen an den Editor',
    stashed?.module === 'helpdesk' && stashed.labels > 0, JSON.stringify(stashed))

  await hash('#/editor-window?module=helpdesk')
  await wait(3000)
  await shot('j3d-entwurf-fortgesetzt.png')
  const resumedCols = await headers()
  check('J3d Der Editor öffnet den Entwurf MIT seinen Änderungen',
    resumedCols.some((h) => h.includes(RENAMED)), resumedCols.join(' | '))

  // ── J4 · Jetzt übernehmen aus dem Detail heraus ──────────────────────────
  await hash('#/admin/anpassungen')
  await page.locator('div[role="button"]').filter({ hasText: /Helpdesk/ }).last().click()
  await wait(1400)
  await page.getByRole('button', { name: /Jetzt übernehmen/i }).first().click()
  await wait(2000)
  await shot('j4c-uebernommen.png')
  check('J4f „Jetzt übernehmen" schaltet den Rollout live', (await text()).includes('Live'))

  // ── Die Nachricht erreicht die Nutzer im Modul ───────────────────────────
  await hash('#/helpdesk')
  await wait(1500)
  await shot('j4d-nachricht-im-modul.png')
  const moduleText = await text()
  check('J4g Die Nachricht steht im Modul', moduleText.includes(MESSAGE))
  check('J4h Sie ist als Änderungs-Hinweis erkennbar', moduleText.includes('Änderung an diesem Modul'))
  check('J4i Die umbenannte Spalte ist live', (await headers()).some((h) => h.includes(RENAMED)))

  // Und im Editor darf der Hinweis NICHT stören
  await hash('#/editor-window?module=helpdesk')
  await wait(2500)
  check('J4j Im Editor erscheint der Hinweis nicht', !(await text()).includes(MESSAGE))

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
