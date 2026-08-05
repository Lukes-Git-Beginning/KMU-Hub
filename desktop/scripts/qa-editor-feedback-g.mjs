/**
 * QA — Dariens Editor-Feedback Runde 2 (2026-08-04).
 *
 *   G1 Der Ticket-Formular-Editor liegt IM Modul-Editor: "Formular bearbeiten"
 *      tauscht die Editor-Fläche gegen den Builder, ohne den Editor zu verlassen;
 *      "Zurück zum Modul" bringt die Vorschau zurück.
 *   G2 Spalten der Ticket-Übersicht sind konfigurierbar: eingebaute abschaltbar,
 *      Zusatzfeld-Spalten (inkl. gebundener Werteliste) zuschaltbar — als Chip.
 *   G3 Der Statistik-Katalog listet die Aufschlüsselung eines Zusatzfeldes als
 *      Schalter, schon im Entwurf (ohne Übernehmen).
 * Screenshots → .qa-screenshots/editor-feedback-g/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-feedback-g')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const SET_NAME = 'Dringlichkeit'
const FIELD_NAME = 'Dringlichkeit'

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

try {
  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(3500)

  // ══ G1 · Form-Builder auf der Editor-Fläche ══════════════════════════════
  await rail(/Kanäle/i).click()
  await wait(1200)
  await page.getByRole('button', { name: /Formular bearbeiten/i }).first().click({ force: true })
  await wait(2500)
  await shot('g1a-builder-im-editor.png')

  const stayed = page.url().includes('editor-window')
  check('G1a Editor wird NICHT verlassen', stayed, page.url().split('#')[1] ?? '')
  const builderText = await text()
  const builderVisible = builderText.includes('Zurück zum Modul') && builderText.includes('Ticket-Formular:')
  check('G1b Formular-Builder liegt auf der Editor-Fläche', builderVisible)
  // Die Editor-Leiste bleibt daneben stehen
  check('G1c Editor-Leiste bleibt sichtbar', builderText.includes('Wertelisten') && builderText.includes('Statistik'))

  await page.getByRole('button', { name: /Zurück zum Modul/i }).first().click()
  await wait(2000)
  await shot('g1d-zurueck-zum-modul.png')
  // Kein Placeholder-Text prüfen (innerText liest den nicht) — die Filterleiste der Liste.
  const backText = await text()
  check('G1d "Zurück zum Modul" zeigt wieder die Vorschau', backText.includes('Alle Status') && backText.includes('Zusammenführen'))

  // ══ Vorbereitung: Werteliste + gebundenes Feld ═══════════════════════════
  await rail(/Wertelisten/i).click()
  await wait(1200)
  await page.getByRole('button', { name: /Neue Werteliste|Werteliste hinzufügen/i }).first().click()
  await wait(1200)
  await page.locator('input[aria-label*="Name"]').last().fill(SET_NAME)
  await wait(900)

  await rail(/Zusatzfelder/i).click()
  await wait(1400)
  await page.getByRole('button', { name: /Feld anlegen/i }).first().click()
  await wait(1200)
  await page.locator('input[placeholder*="Kundennummer"]').first().fill(FIELD_NAME)
  await page.getByRole('button', { name: /^Auswahl$/i }).first().click()
  await wait(800)
  await page.getByRole('radio', { name: /Aus Werteliste/i }).first().click()
  await wait(600)
  await page.locator('select[aria-label*="Werteliste"]').first().selectOption({ label: SET_NAME })
  await wait(700)
  await page.getByRole('button', { name: /Anlegen|Speichern|Erstellen/i }).last().click()
  await wait(1600)

  // ══ G3 · Statistik-Katalog kennt das neue Feld (im Entwurf) ══════════════
  await rail(/^Statistik/i).click()
  await wait(1800)
  await shot('g3-statistik-katalog.png')
  const statPanel = await text()
  check('G3 Statistik-Katalog listet "Nach ' + FIELD_NAME + '" als Schalter', statPanel.includes(`Nach ${FIELD_NAME}`))

  // Abschalten muss in der Vorschau wirken
  const toggle = page.getByRole('switch', { name: new RegExp(`Nach ${FIELD_NAME} ausblenden`, 'i') }).first()
  const hadToggle = (await toggle.count()) > 0
  if (hadToggle) {
    await toggle.click()
    await wait(1500)
    await shot('g3b-nach-abschalten.png')
  }
  check('G3b Der Schalter ist bedienbar (Entwurf, ohne Übernehmen)', hadToggle)

  // ══ G2 · Spalten der Übersicht ═══════════════════════════════════════════
  await rail(/^Spalten/i).click()
  await wait(1800)
  await shot('g2a-spalten-panel.png')
  const colPanel = await text()
  check('G2a Sektion "Spalten" existiert mit den eingebauten Spalten', colPanel.includes('SLA') && colPanel.includes('Priorität'))
  check('G2b Zusatzfeld wird als zuschaltbare Spalte angeboten', colPanel.includes(FIELD_NAME))

  // Zusatzfeld-Spalte einschalten → erscheint in der Tabelle
  const colSwitch = page.getByRole('switch', { name: new RegExp(`Spalte ${FIELD_NAME} einblenden`, 'i') }).first()
  const hasColSwitch = (await colSwitch.count()) > 0
  if (hasColSwitch) { await colSwitch.click(); await wait(1800) }
  await shot('g2c-spalte-eingeschaltet.png')
  const headers = await page.locator('table thead th').allInnerTexts()
  check('G2c eingeschaltete Spalte steht in der Übersicht', headers.some((h) => h.includes(FIELD_NAME)), headers.join(' | '))

  // Eingebaute Spalte abschalten → verschwindet
  const slaOff = page.getByRole('switch', { name: /Spalte SLA ausblenden/i }).first()
  const hasSlaOff = (await slaOff.count()) > 0
  if (hasSlaOff) { await slaOff.click(); await wait(1800) }
  const headers2 = await page.locator('table thead th').allInnerTexts()
  check('G2d eingebaute Spalte lässt sich entfernen', hasSlaOff && !headers2.some((h) => h.trim() === 'SLA'), headers2.join(' | '))
  await shot('g2d-sla-entfernt.png')

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
