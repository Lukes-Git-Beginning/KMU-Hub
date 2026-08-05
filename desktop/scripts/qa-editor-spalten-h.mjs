/**
 * QA — Dariens Feedback Runde 3 (2026-08-05), Spalten-Panel.
 *
 *   H1 Eine NEU angelegte Werteliste taucht sofort im Spalten-Panel auf
 *      ("Wertelisten ohne Spalte") und wird mit einem Klick zur echten Spalte —
 *      ohne dass man vorher ein Zusatzfeld anlegen muss.
 *   H2 Spalten sind bearbeitbar: neue anlegen, bestehende umbenennen/Werteliste
 *      wechseln, löschen.
 *   H3 Regression: die so erzeugte Spalte steht in der Tabelle und in der Statistik.
 * Screenshots → .qa-screenshots/editor-spalten-h/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-spalten-h')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const SET_NAME = 'Risikostufe'
const NEW_COL = 'Bearbeiter-Notiz'

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

  // ── H1 · Werteliste anlegen → erscheint SOFORT bei den Spalten ─────────────
  await rail(/Wertelisten/i).click()
  await wait(1300)
  await page.getByRole('button', { name: /Neue Werteliste|Werteliste hinzufügen/i }).first().click()
  await wait(1200)
  await page.locator('input[aria-label*="Name"]').last().fill(SET_NAME)
  await wait(1000)

  await rail(/^Spalten/i).click()
  await wait(1600)
  await shot('h1a-spalten-mit-freier-werteliste.png')
  // Die Überschrift ist per CSS uppercase — innerText liefert sie großgeschrieben.
  const panel = (await text()).toLowerCase()
  check('H1a Neue Werteliste steht unter "Wertelisten ohne Spalte"',
    panel.includes('wertelisten ohne spalte') && panel.includes(SET_NAME.toLowerCase()))

  // Ein Klick macht daraus eine Spalte
  await page.getByRole('button', { name: new RegExp(SET_NAME, 'i') }).first().click()
  await wait(1800)
  await shot('h1b-werteliste-als-spalte.png')
  const cols = await headers()
  check('H1b Ein Klick macht sie zur echten Spalte in der Übersicht',
    cols.some((h) => h.includes(SET_NAME)), cols.join(' | '))
  // Sie war die einzige Liste ohne Spalte → der ganze Block muss verschwinden.
  // Priorität/Status zählen NICHT als frei: die haben längst eine eingebaute Spalte.
  const afterAdd = (await text()).toLowerCase()
  check('H1c Der Block "ohne Spalte" ist danach leer/weg', !afterAdd.includes('wertelisten ohne spalte'))

  // ── H3 · und in der Statistik ─────────────────────────────────────────────
  await rail(/^Statistik/i).click()
  await wait(1800)
  check(`H3 Statistik-Katalog kennt "Nach ${SET_NAME}"`, (await text()).includes(`Nach ${SET_NAME}`))
  await shot('h3-statistik-katalog.png')

  // ── H2 · Spalten bearbeiten: neu anlegen ──────────────────────────────────
  await rail(/^Spalten/i).click()
  await wait(1500)
  await page.getByRole('button', { name: /Neue Spalte/i }).first().click()
  await wait(1200)
  await page.locator('input[placeholder*="Kundennummer"]').first().fill(NEW_COL)
  await page.getByRole('button', { name: /Anlegen|Speichern|Erstellen/i }).last().click()
  await wait(1800)
  await shot('h2a-neue-spalte-angelegt.png')
  const cols2 = await headers()
  check('H2a "Neue Spalte" legt eine Spalte an und schaltet sie ein',
    cols2.some((h) => h.includes(NEW_COL)), cols2.join(' | '))

  // ── H2 · bearbeiten (umbenennen) ──────────────────────────────────────────
  await page.getByRole('button', { name: new RegExp(`Spalte ${NEW_COL} bearbeiten`, 'i') }).first().click()
  await wait(1200)
  await shot('h2b-spalte-bearbeiten.png')
  const dialogOpen = (await text()).includes('Feldtyp')
  check('H2b Klick auf die Spalte öffnet den Bearbeiten-Dialog', dialogOpen)
  if (dialogOpen) {
    await page.locator('input[placeholder*="Kundennummer"]').first().fill(`${NEW_COL} v2`)
    await page.getByRole('button', { name: /Speichern|Übernehmen/i }).last().click()
    await wait(1800)
    const cols3 = await headers()
    check('H2c Umbenennen wirkt in der Übersicht', cols3.some((h) => h.includes(`${NEW_COL} v2`)), cols3.join(' | '))
  }

  // ── H2 · löschen ──────────────────────────────────────────────────────────
  await page.getByRole('button', { name: new RegExp(`Spalte ${NEW_COL} v2 bearbeiten`, 'i') }).first().click()
  await wait(1200)
  const delBtn = page.getByRole('button', { name: /Löschen|Feld löschen/i }).first()
  const hasDelete = (await delBtn.count()) > 0
  if (hasDelete) { await delBtn.click(); await wait(1800) }
  await shot('h2d-spalte-geloescht.png')
  const cols4 = await headers()
  check('H2d Spalte lässt sich löschen', hasDelete && !cols4.some((h) => h.includes(NEW_COL)), cols4.join(' | '))

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
