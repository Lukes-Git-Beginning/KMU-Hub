/**
 * QA — Dariens Editor-Feedback vom 2026-08-04, Runde 1.
 *
 *   S1 Der Modul-Editor wird beim Formular-Bearbeiten nicht verlassen (das
 *      eingebettete Verhalten selbst prüft qa-editor-feedback-g.mjs, G1).
 *   S3 Wertelisten: JEDES Set (auch die vordefinierten) hat ein Namensfeld.
 *   S4 Trio-Nav fokussiert die Vorschau: "Statistik" → Statistik-Tab,
 *      "Zusatzfelder" → Ticket-Detail offen, Leiste bleibt bedienbar.
 *   S5 Neue Werteliste an ein Auswahl-Zusatzfeld binden → erscheint im
 *      Ticket-Detail als Dropdown UND in der Statistik als Aufschlüsselung.
 * Screenshots → .qa-screenshots/editor-feedback-0804/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-feedback-0804')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const SET_NAME = 'Eskalationsstufe'
const FIELD_NAME = 'Eskalation'

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
const check = (name, ok, extra = '') => { out.push(`${ok ? 'PASS' : 'FAIL'}  ${name}${extra ? ' · ' + extra : ''}`); ok ? pass++ : fail++ }
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const text = () => page.evaluate(() => document.body.innerText)
const rail = (name) => page.getByRole('button', { name }).first()

try {
  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(3500)

  // ── S1 · Editor wird nicht verlassen ───────────────────────────────────────
  await rail(/Kanäle/i).click()
  await wait(1200)
  await shot('s1a-kanaele-panel.png')
  await page.getByRole('button', { name: /Formular bearbeiten/i }).first().click({ force: true })
  await wait(2000)
  check('S1 Modul-Editor wird nicht verlassen (Draft überlebt)', page.url().includes('editor-window'))
  await shot('s1b-editor-bleibt-offen.png')
  const back = page.getByRole('button', { name: /Zurück zum Modul/i }).first()
  if (await back.count()) { await back.click(); await wait(1500) }

  // ── S3 · Namensfeld für ALLE Wertelisten ───────────────────────────────────
  await rail(/Wertelisten/i).click()
  await wait(1400)
  // Helpdesk bringt zwei vordefinierte Listen mit (ticket_priority, ticket_status).
  const nameInputs = await page.locator('input[aria-label*="Name"]').count()
  check('S3 jedes Werteliste-Set hat ein Namensfeld', nameInputs >= 2, `${nameInputs} Felder`)
  check('S3b irreführender Hinweis entfernt', !(await text()).includes('Im Modul per Klick umbenennen'))
  await shot('s3-wertelisten-namensfelder.png')

  // ── S4 · Nav fokussiert die Vorschau ───────────────────────────────────────
  await rail(/^Statistik/i).click()
  await wait(1800)
  await shot('s4a-nav-statistik.png')
  const statText = await text()
  check('S4 "Statistik" bringt die Vorschau auf den Statistik-Tab', statText.includes('Tickets pro Tag') || statText.includes('Nach Status'))

  await rail(/Zusatzfelder/i).click()
  await wait(1800)
  await shot('s4b-nav-zusatzfelder.png')
  const detail = await text()
  check('S4b "Zusatzfelder" öffnet ein Ticket-Detail in der Vorschau',
    detail.includes('Zusatzfelder') && detail.includes('SLA-Stufe') && detail.includes('Beschreibung'))
  // Das Detail ist Vorschau, kein echter Dialog: die Editor-Leiste muss bedienbar bleiben.
  check('S4c Editor-Leiste bleibt neben dem offenen Detail bedienbar', await rail(/Wertelisten/i).isEnabled())

  // ── S5 · Werteliste anlegen → an Feld binden → wirkt ──────────────────────
  await rail(/Wertelisten/i).click()
  await wait(1200)
  await page.getByRole('button', { name: /Neue Werteliste|Werteliste hinzufügen/i }).first().click()
  await wait(1200)
  await page.locator('input[aria-label*="Name"]').last().fill(SET_NAME)
  await wait(1000)
  await shot('s5a-neue-werteliste.png')

  await rail(/Zusatzfelder/i).click()
  await wait(1400)
  await page.getByRole('button', { name: /Feld anlegen/i }).first().click()
  await wait(1200)
  // Das Namensfeld IM Dialog treffen (nicht die Ticket-Suche in der Sandbox dahinter).
  await page.locator('input[placeholder*="Kundennummer"]').first().fill(FIELD_NAME)
  await page.getByRole('button', { name: /^Auswahl$/i }).first().click()
  await wait(900)
  await shot('s5b-feld-dialog-typ-auswahl.png')

  const fromList = page.getByRole('radio', { name: /Aus Werteliste/i }).first()
  const hasSourceToggle = (await fromList.count()) > 0
  check('S5a Feld-Dialog bietet "Aus Werteliste"', hasSourceToggle)
  if (hasSourceToggle) {
    await fromList.click()
    await wait(700)
    await page.locator('select[aria-label*="Werteliste"]').first().selectOption({ label: SET_NAME })
    await wait(900)
    await shot('s5c-werteliste-gebunden.png')
    check('S5b Dialog zeigt die Optionen der Liste als Vorschau', (await text()).includes('Neue Option'))
  }
  await page.getByRole('button', { name: /Anlegen|Speichern|Erstellen/i }).last().click()
  await wait(1600)
  await shot('s5d-feld-angelegt.png')

  await rail(/Zusatzfelder/i).click()
  await wait(1800)
  check('S5c gebundenes Feld erscheint im Ticket-Detail', (await text()).includes(FIELD_NAME))
  await shot('s5e-ticket-detail-mit-feld.png')

  await rail(/^Statistik/i).click()
  await wait(2000)
  check(`S5d Aufschlüsselung "Nach ${FIELD_NAME}" in der Statistik`, (await text()).includes(`Nach ${FIELD_NAME}`))
  await shot('s5f-statistik-mit-aufschluesselung.png')

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
