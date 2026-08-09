/**
 * QA — Wertelisten, Durchstich bis ins Modul (Darien 2026-08-06: „die gehen noch
 * nicht zu 100 Prozent").
 *
 * Prüft die Prüfliste W1–W7 aus .planning/wertelisten-und-fokus-naechste-runde.md:
 * jede Änderung an einer Liste muss im Modul ankommen — in der Tabelle, in den
 * Filtern, im Detail UND in der Statistik. Deploy/Rollback (W10/W11) macht die
 * zweite Suite.
 *
 * Echter Electron-Start (der Editor ist ein eigenes Fenster).
 * Voraussetzung: Renderer-Dev-Server auf :5173 (npm run dev).
 * Screenshots → .qa-screenshots/editor-wertelisten-n/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/editor-wertelisten-n')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-wertelisten-n')
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
const bodyText = async (p) => p.evaluate(() => document.body.innerText)
const clickRail = async (p, name) =>
  p.locator('nav[aria-label="Anpassen"] button').filter({ hasText: name }).first().click()
const moduleTab = (p, name) => p.locator('.min-h-full button').filter({ hasText: name }).first()
/** Alle Auswahlmöglichkeiten der <select>-Filter im Modul. */
const selectChoices = async (p) =>
  p.locator('.min-h-full select').evaluateAll((els) =>
    els.flatMap((el) => Array.from(el.options).map((o) => o.textContent.trim())),
  )
/** Texte der Chips in der Prioritäts-Spalte der Ticket-Tabelle. */
const cellTexts = async (p, columnIndex) =>
  p.locator(`.min-h-full table tbody tr td:nth-child(${columnIndex})`).allInnerTexts()

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

  await clickRail(editor, 'Wertelisten')
  await wait(editor, 1800)
  await shot(editor, 'n0-panel.png')
  const panelText = await bodyText(editor)
  check('W0 Das Panel zeigt die Listen des Moduls', panelText.includes('Priorität') || panelText.includes('Hoch'))

  // Welche Optionen die Prioritätsliste WIRKLICH hat (der Demo-Mandant hat
  // „Niedrig" schon zu „Rückfrage" umbenannt) — nicht raten, auslesen.
  const optionNames = await editor.getByLabel(/^Option .* umbenennen$/).evaluateAll((els) => els.map((e) => e.value))
  out.push(`INFO  Optionen im Panel: ${optionNames.join(' · ')}`)
  const [firstOpt] = optionNames

  // ── W2 · Option umbenennen ───────────────────────────────────────────────
  const hochInput = editor.getByLabel(/Option .Hoch. umbenennen/).first()
  await hochInput.fill('Dringend')
  await wait(editor, 1500)
  await shot(editor, 'n1-umbenannt.png')
  const prioCells = await cellTexts(editor, 4)
  check('W2a Umbenennen wirkt in der Ticket-Liste', prioCells.some((c) => c.includes('Dringend')),
    prioCells.slice(0, 4).join(' | '))
  const choices = await selectChoices(editor)
  check('W2b Umbenennen wirkt im Filter', choices.includes('Dringend'), choices.join(', ').slice(0, 120))

  await moduleTab(editor, /^Statistik$/).click()
  await wait(editor, 1800)
  await shot(editor, 'n2-statistik.png')
  check('W2c Umbenennen wirkt in der Statistik-Aufschlüsselung', (await bodyText(editor)).includes('Dringend'))
  await clickRail(editor, 'Wertelisten')
  await wait(editor, 1500)
  await moduleTab(editor, /^Tickets\s*\(\d+\)$/).click()
  await wait(editor, 1500)

  // ── W3 · Farbe ändern ────────────────────────────────────────────────────
  // Die Farbe der Option ändern, die in der ERSTEN Tabellenzeile steht — sonst
  // prüft der Test einen Chip, den er gar nicht angefasst hat.
  const firstRowPriority = (await cellTexts(editor, 4))[0].trim()
  const chip = () => editor.locator('.min-h-full table tbody tr td:nth-child(4) span').first()
  const beforeColor = await chip().evaluate((el) => getComputedStyle(el).backgroundColor).catch(() => '')
  await editor.getByRole('button', { name: new RegExp(`Farbe Rot für .${firstRowPriority}.`) }).first().click()
  await wait(editor, 1500)
  const afterColor = await chip().evaluate((el) => getComputedStyle(el).backgroundColor).catch(() => '')
  await shot(editor, 'n3-farbe.png')
  check('W3 Farbwechsel erreicht den Chip in der Liste', beforeColor !== afterColor && !!afterColor,
    `${firstRowPriority}: ${beforeColor} → ${afterColor}`)

  // ── W4 · Option hinzufügen ───────────────────────────────────────────────
  await editor.getByRole('button', { name: 'Neue Option' }).first().click()
  await wait(editor, 1200)
  const newOpt = editor.getByLabel(/Option .Neue Option. umbenennen/).first()
  await newOpt.fill('Blocker')
  await wait(editor, 1500)
  await shot(editor, 'n4-neue-option.png')
  const choices2 = await selectChoices(editor)
  check('W4 Eine neue Option ist sofort im Modul wählbar', choices2.includes('Blocker'),
    choices2.join(', ').slice(0, 140))

  // ── W5 · Option deaktivieren ─────────────────────────────────────────────
  // Die erste Option der Liste ausblenden (heißt beim Demo-Mandanten „Rückfrage").
  await editor.getByRole('button', { name: new RegExp(`.${firstOpt}. ein- oder ausblenden`) }).first().click()
  await wait(editor, 1500)
  await shot(editor, 'n5-ausgeblendet.png')
  const choices3 = await selectChoices(editor)
  check('W5a Eine ausgeblendete Option verschwindet aus der Auswahl', !choices3.includes(firstOpt),
    `${firstOpt} weg? · ${choices3.join(', ').slice(0, 120)}`)

  // ── W6 · Option löschen, die in Benutzung ist ────────────────────────────
  const prioBefore = await cellTexts(editor, 4)
  const mittelCount = prioBefore.filter((c) => c.includes('Mittel')).length
  await editor.getByRole('button', { name: /.Mittel. löschen/ }).first().click()
  await wait(editor, 1200)
  await shot(editor, 'n6-umzugsdialog.png')
  const dialogVisible = (await bodyText(editor)).includes('Bestehende Einträge mit diesem Wert')
  check('W6a Löschen einer benutzten Option fragt nach einem Umzugsziel', dialogVisible)
  if (dialogVisible) {
    // Ziel bewusst wählen: „Dringend" (die umbenannte Option). Der Auswahlkasten
    // des Dialogs ist der LETZTE im Fenster — die davor sind die Filter des
    // Moduls, und einen davon zu verstellen würde die Liste filtern statt
    // umzuziehen (dann sagt die Prüfung darunter nichts mehr aus).
    await editor.locator('select').last().selectOption({ label: 'Dringend' })
    await editor.getByRole('button', { name: 'Entfernen' }).first().click()
    await wait(editor, 2000)
    await shot(editor, 'n7-umgezogen.png')
    const prioAfter = await cellTexts(editor, 4)
    check('W6b Die Vorschau zieht die Datensätze mit um',
      prioAfter.filter((c) => c.includes('Mittel')).length === 0,
      `vorher ${mittelCount}× Mittel, nachher ${prioAfter.filter((c) => c.includes('Mittel')).length}×`)
    // Die Überschrift steht per CSS in Großbuchstaben — innerText liefert sie so.
    check('W6c Die entfernte Option steht als „Entfernt" mit Ziel im Panel',
      /ENTFERNT/i.test(await bodyText(editor)) && (await bodyText(editor)).includes('→'))
    const choices4 = await selectChoices(editor)
    check('W6d Die entfernte Option ist nicht mehr wählbar', !choices4.includes('Mittel'),
      choices4.join(', ').slice(0, 140))
  }

  // ── W9 · Reihenfolge der Optionen ────────────────────────────────────────
  // Über die Tastatur, weil der Zieh-Griff das auch können muss (dnd-kit:
  // Leertaste greift, Pfeil bewegt, Leertaste legt ab).
  const orderBefore = await editor.getByLabel(/^Option .* umbenennen$/).evaluateAll((els) => els.map((e) => e.value))
  const rowsBefore = orderBefore.slice(0, 3)
  await editor.getByRole('button', { name: new RegExp(`Option ${rowsBefore[0]} verschieben`) }).first().focus()
  await editor.keyboard.press('Space')
  await wait(editor, 400)
  await editor.keyboard.press('ArrowDown')
  await wait(editor, 400)
  await editor.keyboard.press('Space')
  await wait(editor, 1500)
  await shot(editor, 'n7b-reihenfolge.png')
  const orderAfter = await editor.getByLabel(/^Option .* umbenennen$/).evaluateAll((els) => els.map((e) => e.value))
  check('W9a Optionen lassen sich umsortieren',
    orderAfter[0] === rowsBefore[1] && orderAfter[1] === rowsBefore[0],
    `${rowsBefore.join(' → ')}  wurde  ${orderAfter.slice(0, 3).join(' → ')}`)
  // Die nach oben geschobene Option muss im Prioritäts-Menü ganz vorne stehen —
  // direkt hinter „Alle Prioritäten". (Die vorher ausgeblendete taucht dort
  // zurecht nicht mehr auf, deshalb wird sie nicht mitverglichen.)
  const choicesOrder = await selectChoices(editor)
  const menuStart = choicesOrder.indexOf('Alle Prioritäten')
  const firstInMenu = choicesOrder[menuStart + 1]
  const movedUp = orderAfter.find((label) => choicesOrder.includes(label))
  check('W9b Die neue Reihenfolge wirkt in der Auswahl im Modul', firstInMenu === movedUp,
    `Panel: ${movedUp} · Menü: ${firstInMenu}`)

  // ── W1 · Neue Werteliste anlegen ─────────────────────────────────────────
  await editor.getByRole('button', { name: 'Neue Werteliste' }).first().click()
  await wait(editor, 1500)
  const setNames = editor.getByLabel('Name der Liste')
  await setNames.last().fill('Eskalationsstufe')
  await wait(editor, 1500)
  await shot(editor, 'n8-neue-liste.png')
  // Der Listenname lebt in einem Eingabefeld — der steht nicht im innerText.
  const listNames = await setNames.evaluateAll((els) => els.map((e) => e.value))
  check('W1a Die neue Liste steht im Panel', listNames.includes('Eskalationsstufe'), listNames.join(' · '))

  await clickRail(editor, 'Spalten')
  await wait(editor, 1800)
  await shot(editor, 'n9-spalten-panel.png')
  check('W1b Die neue Liste wird als Spalte angeboten (sonst hat sie kein Zuhause)',
    (await bodyText(editor)).includes('Eskalationsstufe'))

  // Und die Zählung in der Leiste muss die Liste kennen.
  const railBadge = await editor.locator('nav[aria-label="Anpassen"] button').filter({ hasText: 'Wertelisten' })
    .first().innerText()
  check('W1c Die Leiste zählt die Wertelisten-Änderungen', /\d/.test(railBadge), railBadge.replace(/\n/g, ' '))
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n')[0])
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await app.close()
process.exit(fail > 0 ? 1 : 0)
