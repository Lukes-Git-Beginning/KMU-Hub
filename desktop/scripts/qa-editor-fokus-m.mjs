/**
 * QA — Fokus-Kopplung in BEIDE Richtungen (Darien 2026-08-06: „wenn ich im
 * Modul-Editor im Modul auf Statistik klicke, wechselt das Menü links und rechts
 * nicht").
 *
 * Bis hierher zog nur die Leiste die Vorschau. Jetzt meldet das Modul zurück, wo
 * der Nutzer steht — und die Leiste + das rechte Panel folgen. Die Suite prüft
 * beide Richtungen, vor allem die Stellen wo sie sich gegenseitig schieben
 * könnten (Leisten-Klick → Vorschau → Meldung → Leiste → …).
 *
 * Echter Electron-Start, weil der Editor ein eigenes BrowserWindow ist (die
 * Browser-Suiten sehen das zweite Fenster nicht — Lehre aus Session #35).
 * Voraussetzung: Renderer-Dev-Server auf :5173 (npm run dev).
 * Screenshots → .qa-screenshots/editor-fokus-m/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/editor-fokus-m')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-fokus-m')
await rm(userDataDir, { recursive: true, force: true })

const out = []
let pass = 0, fail = 0
const check = (n, ok, extra = '') => { out.push(`${ok ? 'PASS' : 'FAIL'}  ${n}${extra ? ' · ' + extra : ''}`); ok ? pass++ : fail++ }

const app = await electron.launch({
  args: [resolve('out/main/index.js'), `--user-data-dir=${userDataDir}`],
  env: { ...process.env, ELECTRON_RENDERER_URL: 'http://localhost:5173', NODE_ENV: 'development' },
})

const wait = (p, ms) => p.waitForTimeout(ms)
const shot = (p, n) => p.screenshot({ path: resolve(outDir, n) })

/** Welcher Eintrag in der linken Leiste ist hervorgehoben? '' = keiner. */
const railActive = async (p) => {
  const active = p.locator('nav[aria-label="Anpassen"] button[aria-current="true"]')
  return (await active.count()) === 0 ? '' : (await active.first().innerText()).trim()
}
const bodyText = async (p) => p.evaluate(() => document.body.innerText)
/** Einen Eintrag der linken Leiste klicken — gescopt, weil „Statistik" auch im Modul steht. */
const clickRail = async (p, name) => {
  await p.locator('nav[aria-label="Anpassen"] button').filter({ hasText: name }).first().click()
}

try {
  const hub = await app.firstWindow()
  await hub.waitForLoadState('domcontentloaded')
  await hub.evaluate(() => {
    try {
      const K = 'cosmi-ui'
      const raw = localStorage.getItem(K)
      const p = raw ? JSON.parse(raw) : { state: {}, version: 0 }
      p.state = { ...(p.state || {}), onboardingCompleted: true }
      localStorage.setItem(K, JSON.stringify(p))
      sessionStorage.setItem('cosmi:launch-played', '1')
      // Ohne Alt-Entwürfe starten — sonst setzt der Editor einen fort und die
      // Vorschau steht woanders als diese Suite annimmt.
      localStorage.removeItem('cosmi:customization:drafts')
    } catch { /* egal */ }
  })
  await hub.reload()
  await wait(hub, 4000)
  await hub.evaluate(() => { window.location.hash = '#/admin/anpassungen' })
  await wait(hub, 4000)

  const editorPromise = app.waitForEvent('window')
  await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
  const editor = await editorPromise
  await editor.waitForLoadState('domcontentloaded')
  await wait(editor, 5000)
  await shot(editor, 'm0-editor.png')
  check('M0 Der Editor öffnet als eigenes Fenster', (await editor.locator('nav[aria-label="Anpassen"]').count()) > 0)
  check('M0b Die Leiste startet ohne Auswahl', (await railActive(editor)) === '', await railActive(editor))

  // ── Richtung Modul → Editor: der Statistik-Reiter IM Modul ───────────────
  const moduleTab = (name) =>
    editor.locator('main button, .min-h-full button').filter({ hasText: name }).first()
  await moduleTab(/^Statistik$/).click()
  await wait(editor, 2000)
  await shot(editor, 'm1-modul-statistik.png')
  check('M1 Modul-Reiter „Statistik" hebt die Leiste auf Statistik', (await railActive(editor)) === 'Statistik', await railActive(editor))
  check('M1b Das rechte Panel zeigt den Statistik-Katalog',
    (await bodyText(editor)).includes('Blende Kacheln und Diagramme'))

  // ── Zurück auf die Liste: der verlassene Ort gibt die Leiste wieder frei ─
  // (Der Zähler steht ohne Leerzeichen im DOM: „Tickets(11)".)
  await moduleTab(/^Tickets\s*\(\d+\)$/).click()
  await wait(editor, 1800)
  await shot(editor, 'm2-zurueck-liste.png')
  check('M2 Zurück auf die Liste löst die Statistik-Auswahl wieder', (await railActive(editor)) === '', await railActive(editor))

  // ── Ein Ticket öffnen: Zusatzfelder-Kontext — und es bleibt DAS Ticket ──
  const rows = editor.locator('table tbody tr')
  const rowCount = await rows.count()
  const targetRow = rows.nth(Math.min(2, rowCount - 1))
  // Die Ticket-Nr-Zelle: ein Wert, kein umbenennbares Label — der Klick fällt also
  // auf die Zeile durch und öffnet das Detail.
  const nrCell = targetRow.locator('td').first()
  const ticketNr = (await nrCell.innerText()).trim()
  await nrCell.click()
  await wait(editor, 2000)
  await shot(editor, 'm3-detail-offen.png')
  check('M3 Ein geöffnetes Ticket hebt die Leiste auf Zusatzfelder',
    (await railActive(editor)) === 'Zusatzfelder', await railActive(editor))
  const dialog = editor.locator('[role="dialog"]').first()
  const dialogText = (await dialog.count()) > 0 ? await dialog.innerText() : ''
  // Der eigentliche Stolperstein: die Meldung darf den Fokus-Handler NICHT
  // auslösen, sonst springt das Fenster auf das erste Ticket zurück.
  check('M4 Das geöffnete Ticket bleibt das angeklickte (springt nicht auf das erste)',
    dialogText.includes(ticketNr), `angeklickt: ${ticketNr}`)

  await editor.keyboard.press('Escape')
  await wait(editor, 1500)
  check('M5 Detail schließen gibt die Leiste wieder frei', (await railActive(editor)) === '', await railActive(editor))

  // ── Gegenprobe: die alte Richtung (Leiste → Vorschau) lebt weiter ────────
  await clickRail(editor, 'Statistik')
  await wait(editor, 2000)
  await shot(editor, 'm6-leiste-statistik.png')
  const afterRail = await bodyText(editor)
  check('M6 Leiste „Statistik" schiebt die Vorschau weiterhin auf den Statistik-Reiter',
    afterRail.includes('Blende Kacheln und Diagramme') && (await railActive(editor)) === 'Statistik',
    await railActive(editor))

  await clickRail(editor, 'Spalten')
  await wait(editor, 2000)
  await shot(editor, 'm7-leiste-spalten.png')
  check('M7 Leiste „Spalten" bringt die Liste zurück UND bleibt ausgewählt (kein Zurückschieben)',
    (await railActive(editor)) === 'Spalten' && (await editor.locator('table thead th').count()) > 0,
    `${await railActive(editor)} · ${await editor.locator('table thead th').count()} Spalten`)

  await clickRail(editor, 'Zusatzfelder')
  await wait(editor, 2200)
  await shot(editor, 'm8-leiste-felder.png')
  check('M8 Leiste „Zusatzfelder" öffnet weiterhin ein Ticket und bleibt ausgewählt',
    (await railActive(editor)) === 'Zusatzfelder' && (await editor.locator('[role="dialog"]').count()) > 0,
    await railActive(editor))

  // Und ein zweites Mal derselbe Klick muss immer noch greifen (Nonce-Pfad).
  await editor.keyboard.press('Escape')
  await wait(editor, 1200)
  await clickRail(editor, 'Zusatzfelder')
  await wait(editor, 2000)
  check('M9 Derselbe Leisten-Eintrag zweimal geklickt fokussiert erneut',
    (await editor.locator('[role="dialog"]').count()) > 0)
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n')[0])
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await app.close()
process.exit(fail > 0 ? 1 : 0)
