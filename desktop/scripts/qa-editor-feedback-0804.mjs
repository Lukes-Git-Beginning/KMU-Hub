/**
 * QA — Dariens Editor-Feedback vom 2026-08-04 (4 Punkte).
 *
 *   S1 F1a  Kanäle → "Formular bearbeiten": öffnet in Electron ein EIGENES Fenster
 *           (IPC openFormWindow), damit der Editor-Draft nicht verloren geht.
 *   S2 F1b  Web-Fallback: derselbe Klick routet nach /formulare?edit=<id> — der
 *           Navigations-Blocker lässt die editor-eigene Navigation jetzt durch.
 *   S3 F2b  Wertelisten: JEDES Set (auch die vordefinierten) hat ein Namensfeld.
 *   S4 F3   Trio-Nav fokussiert die Vorschau: "Statistik" → Statistik-Tab,
 *           "Zusatzfelder" → Ticket-Detail offen.
 *   S5 F2a/c Neue Werteliste an ein Auswahl-Zusatzfeld binden → erscheint im
 *           Ticket-Detail als Dropdown UND in der Statistik als Aufschlüsselung.
 * Screenshots → .qa-screenshots/editor-feedback-0804/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-feedback-0804')
await mkdir(outDir, { recursive: true })

/** Generic electronAPI stub, but with a REAL editor.openFormWindow that records calls. */
const STUB_ELECTRON = `
const noop=()=>Promise.resolve();
const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};
window.__openedForms=[];
window.electronAPI=new Proxy(noop,{
  get:(t,p)=> p==='editor'
    ? {openWindow:()=>Promise.resolve(), openFormWindow:(id)=>{window.__openedForms.push(id);return Promise.resolve()}}
    : h.get(t,p),
  apply:()=>Promise.resolve()
});`
/** Same, but WITHOUT the editor bridge → exercises the web fallback (navigate). */
const STUB_WEB = `
const noop=()=>Promise.resolve();
const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};
window.electronAPI=new Proxy(noop,{
  get:(t,p)=> p==='editor' ? {openWindow:()=>Promise.resolve()} : h.get(t,p),
  apply:()=>Promise.resolve()
});`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const SET_NAME = 'Eskalationsstufe'
const FIELD_NAME = 'Eskalation'

const b = await chromium.launch({ headless: true })
const out = []
let pass = 0
let fail = 0
const check = (name, ok, extra = '') => {
  out.push(`${ok ? 'PASS' : 'FAIL'}  ${name}${extra ? ' · ' + extra : ''}`)
  ok ? pass++ : fail++
}

async function newPage(stub) {
  const ctx = await b.newContext({ viewport: { width: 1600, height: 1000 } })
  await ctx.addInitScript(stub)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(NOLAUNCH)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
  page.on('console', (m) => {
    if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED|Failed to load resource/.test(m.text())) errs.push('console: ' + m.text())
  })
  return page
}
const errs = []

try {
  // ══ S1 · F1a — Electron: eigenes Fenster ═════════════════════════════════
  let page = await newPage(STUB_ELECTRON)
  const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
  const wait = (ms) => page.waitForTimeout(ms)
  const text = () => page.evaluate(() => document.body.innerText)

  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(3500)
  await page.getByRole('button', { name: /Kanäle/i }).first().click()
  await wait(1200)
  await shot('s1a-kanaele-panel.png')

  await page.getByRole('button', { name: /Formular bearbeiten/i }).first().click({ force: true })
  await wait(1500)
  const opened = await page.evaluate(() => window.__openedForms ?? [])
  check('S1 F1a Electron öffnet Formular in eigenem Fenster', opened.length === 1 && !!opened[0], `id=${opened[0] ?? '—'}`)
  const stillEditor = page.url().includes('editor-window')
  check('S1b Modul-Editor bleibt stehen (Draft überlebt)', stillEditor)
  await shot('s1b-editor-bleibt-offen.png')

  // ══ S3 · F2b — Namensfeld für ALLE Wertelisten ═══════════════════════════
  await page.getByRole('button', { name: /Wertelisten/i }).first().click()
  await wait(1400)
  // Helpdesk bringt zwei vordefinierte Listen mit (ticket_priority, ticket_status).
  const nameInputs = await page.locator('input[aria-label*="Name"]').count()
  check('S3 F2b jedes Werteliste-Set hat ein Namensfeld', nameInputs >= 2, `${nameInputs} Felder`)
  const noHint = !(await text()).includes('Im Modul per Klick umbenennen')
  check('S3b irreführender Hinweis entfernt', noHint)
  await shot('s3-wertelisten-namensfelder.png')

  // ══ S4 · F3 — Nav fokussiert die Vorschau ════════════════════════════════
  await page.getByRole('button', { name: /^Statistik/i }).first().click()
  await wait(1800)
  await shot('s4a-nav-statistik.png')
  const onStats = (await text()).includes('Tickets pro Tag') || (await text()).includes('Nach Status')
  check('S4 F3 "Statistik" bringt die Vorschau auf den Statistik-Tab', onStats)

  await page.getByRole('button', { name: /Zusatzfelder/i }).first().click()
  await wait(1800)
  await shot('s4b-nav-zusatzfelder.png')
  const detail = await text()
  const onDetail = detail.includes('Zusatzfelder') && detail.includes('SLA-Stufe') && detail.includes('Beschreibung')
  check('S4b "Zusatzfelder" öffnet ein Ticket-Detail in der Vorschau', onDetail)
  // Das Detail ist Vorschau, kein echter Dialog: die Editor-Leiste muss bedienbar bleiben.
  const railLive = await page.getByRole('button', { name: /Wertelisten/i }).first().isEnabled()
  check('S4c Editor-Leiste bleibt neben dem offenen Detail bedienbar', railLive)

  // ══ S5 · F2a/c — Werteliste anlegen → an Feld binden → wirkt ═════════════
  await page.getByRole('button', { name: /Wertelisten/i }).first().click()
  await wait(1200)
  await page.getByRole('button', { name: /Neue Werteliste|Werteliste hinzufügen/i }).first().click()
  await wait(1200)
  const newName = page.locator('input[aria-label*="Name"]').last()
  await newName.fill(SET_NAME)
  await wait(1000)
  await shot('s5a-neue-werteliste.png')

  // Feld anlegen und an die Liste binden
  await page.getByRole('button', { name: /Zusatzfelder/i }).first().click()
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
    const preview = await text()
    check('S5b Dialog zeigt die Optionen der Liste als Vorschau', preview.includes('Neue Option'))
  }
  await page.getByRole('button', { name: /Anlegen|Speichern|Erstellen/i }).last().click()
  await wait(1600)
  await shot('s5d-feld-angelegt.png')

  // Wirkt es im Ticket-Detail?
  await page.getByRole('button', { name: /Zusatzfelder/i }).first().click()
  await wait(1800)
  const detailText = await text()
  check('S5c gebundenes Feld erscheint im Ticket-Detail', detailText.includes(FIELD_NAME))
  await shot('s5e-ticket-detail-mit-feld.png')

  // Und in der Statistik?
  await page.getByRole('button', { name: /^Statistik/i }).first().click()
  await wait(2000)
  const statText = await text()
  check('S5d Aufschlüsselung "Nach ' + FIELD_NAME + '" in der Statistik', statText.includes(`Nach ${FIELD_NAME}`))
  await shot('s5f-statistik-mit-aufschluesselung.png')

  await page.context().close()

  // ══ S2 · F1b — Web-Fallback: Blocker lässt die Navigation durch ══════════
  page = await newPage(STUB_WEB)
  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.getByRole('button', { name: /Kanäle/i }).first().click()
  await page.waitForTimeout(1200)
  await page.getByRole('button', { name: /Formular bearbeiten/i }).first().click({ force: true })
  await page.waitForTimeout(2500)
  const navigated = page.url().includes('/formulare')
  check('S2 F1b Web-Fallback routet ins Formular (Blocker durchlässig)', navigated, page.url().split('#')[1] ?? '')
  await page.screenshot({ path: resolve(outDir, 's2-web-fallback-formular.png') })
  await page.context().close()

  check('Keine Seitenfehler', errs.length === 0, errs.slice(0, 3).join(' | '))
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n')[0])
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await b.close()
process.exit(fail > 0 ? 1 : 0)
