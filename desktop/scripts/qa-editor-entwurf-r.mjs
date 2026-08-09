/**
 * QA — Entwürfe: der richtige Stand und ein eigener Name (Darien 2026-08-09).
 *
 * Sein Wortlaut: *„manchmal wenn man den gespeicherten Entwurf öffnet, dann öffnet
 * er es so wie es aktuell ist oder beim zweiten Versuch die geänderte Version,
 * aber die man vor 2 mal abspeichern hatte"* — und *„man muss die beim Entwurf
 * abspeichern auch benennen können, sonst heißen drei Entwürfe für Helpdesk alle
 * gleich"*.
 *
 * Die Ursache war ein liegengebliebener Übergabe-Zettel: der Hub legt den Entwurf
 * in geteilten Speicher, damit das startende Editor-Fenster ihn liest. War das
 * Fenster schon offen, wurde es nur fokussiert — der Zettel blieb liegen und wurde
 * vom NÄCHSTEN Start eingelöst, mehrere Speicherungen später. Genau das stellt R3
 * nach.
 *
 * Voraussetzung: Renderer-Dev-Server auf :5173 (npm run dev) UND ein aktueller
 * `out/main` (der Main-Prozess meldet jetzt, ob ein Fenster neu entstand).
 * Screenshots → .qa-screenshots/editor-entwurf-r/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/editor-entwurf-r')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-entwurf-r')
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
const headers = (p) => p.locator('table thead th').allInnerTexts()
const clickRail = async (p, name) =>
  p.locator('nav[aria-label="Anpassen"] button').filter({ hasText: name }).first().click()

/** Die Betreff-Spalte im Editor umbenennen — der Marker, an dem der Stand hängt. */
const renameSubjectColumn = async (editor, from, to) => {
  await editor.getByRole('button', { name: new RegExp(`Spalte ${from} umbenennen`) }).first().click()
  await wait(editor, 600)
  const input = editor.locator('input[aria-label*="umbenennen"]').first()
  await input.fill(to)
  await input.press('Enter')
  await wait(editor, 1200)
}

const saveDraft = async (editor) => {
  await editor.locator('button').filter({ hasText: /^Als Entwurf speichern$/ }).first().dispatchEvent('click')
  await wait(editor, 1200)
}

const openModuleTile = async (hub) => {
  await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
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
      localStorage.removeItem('cosmi:customization:drafts')
      localStorage.removeItem('cosmi:customization:resume-draft')
    } catch { /* egal */ }
  })
  await hub.reload()
  await wait(hub, 4000)
  await hub.evaluate(() => { window.location.hash = '#/admin/anpassungen' })
  await wait(hub, 4000)

  // ── R1 · Entwurf benennen ────────────────────────────────────────────────
  const editorPromise = app.waitForEvent('window')
  await openModuleTile(hub)
  const editor = await editorPromise
  await editor.waitForLoadState('domcontentloaded')
  await wait(editor, 5000)
  await clickRail(editor, 'Spalten')
  await wait(editor, 1500)
  await renameSubjectColumn(editor, 'Betreff', 'Stand-A')
  await saveDraft(editor)
  await shot(editor, 'r1-namensdialog.png')

  const dialogText = await editor.evaluate(() => document.body.innerText)
  check('R1 Beim Speichern fragt der Editor nach einem Namen', dialogText.includes('Name des Entwurfs'))
  const nameField = editor.getByLabel('Name des Entwurfs')
  const suggested = await nameField.inputValue()
  check('R1b Ein Vorschlag ist vorbelegt (Modul + Zeitpunkt)', suggested.includes('Helpdesk'), suggested)
  await nameField.fill('Nur die Spalten')
  await editor.getByRole('button', { name: /^Speichern$/ }).first().click()
  await wait(editor, 1500)
  await shot(editor, 'r2-benannt.png')
  check('R2 Der Name steht im Editor-Banner', (await editor.evaluate(() => document.body.innerText)).includes('Nur die Spalten'))

  // ── R3 · Der eigentliche Fehler: liegengebliebener Übergabe-Zettel ───────
  // Kachel klicken, während der Editor offen ist → das Fenster wird nur
  // fokussiert. Danach im Editor WEITER arbeiten und speichern. Ein alter Zettel
  // würde beim nächsten Start diesen zweiten Stand überschreiben.
  const windowsBefore = app.windows().length
  await openModuleTile(hub)
  await wait(hub, 2500)
  check('R3 Ein zweiter Klick öffnet kein weiteres Fenster', app.windows().length === windowsBefore)
  const noteAfterFocus = await hub.evaluate(() => localStorage.getItem('cosmi:customization:resume-draft'))
  check('R3b Der Übergabe-Zettel wird wieder eingesammelt', noteAfterFocus === null,
    noteAfterFocus ? 'liegt noch da' : 'weg')
  // Gespeicherte Arbeit darf NICHT nachfragen — „1 Änderung" in der Fußzeile heißt
  // „weicht von Live ab", nicht „ungespeichert".
  check('R3c Nach dem Speichern kommt keine Rückfrage',
    !(await editor.evaluate(() => document.body.innerText)).includes('Entwurf laden?'))

  await renameSubjectColumn(editor, 'Stand-A', 'Stand-B')
  await saveDraft(editor)
  await wait(editor, 1200)
  await shot(editor, 'r3-stand-b.png')
  check('R4 Weiteres Speichern fragt nicht erneut nach dem Namen',
    !(await editor.evaluate(() => document.body.innerText)).includes('Name des Entwurfs'))

  // ── R4b · Wirklich ungespeicherte Arbeit wird geschützt ─────────────────
  await renameSubjectColumn(editor, 'Stand-B', 'Stand-C-ungespeichert')
  await openModuleTile(hub)
  await wait(editor, 2500)
  await shot(editor, 'r3b-rueckfrage.png')
  const askText = await editor.evaluate(() => document.body.innerText)
  check('R4b Ungespeicherte Arbeit löst die Rückfrage aus', askText.includes('Entwurf laden?'))
  await editor.getByRole('button', { name: 'Behalten was ich habe' }).first().click()
  await wait(editor, 1200)
  check('R4c „Behalten" lässt die Arbeit stehen',
    (await headers(editor)).some((h) => h.includes('Stand-C-ungespeichert')),
    (await headers(editor)).join(' | '))
  // Für den nächsten Schritt: den ungespeicherten Stand verwerfen, indem der
  // gespeicherte Entwurf bewusst geladen wird.
  await openModuleTile(hub)
  await wait(editor, 2000)
  await editor.getByRole('button', { name: 'Entwurf laden' }).first().click()
  await wait(editor, 1500)
  check('R4d „Entwurf laden" holt den gespeicherten Stand zurück',
    (await headers(editor)).some((h) => h.includes('Stand-B')),
    (await headers(editor)).join(' | '))

  await editor.close()
  await wait(hub, 1500)

  // ── R5 · Erneut öffnen: der ZULETZT gespeicherte Stand ───────────────────
  const stored = await hub.evaluate(() => {
    const raw = localStorage.getItem('cosmi:customization:drafts')
    if (!raw) return 'kein Eintrag'
    const p = JSON.parse(raw)
    return (p.drafts ?? []).map((d) => `${d.name}[${d.status}]`).join(', ')
  })
  out.push(`INFO  Entwürfe im Speicher vor dem Öffnen: ${stored}`)
  const editor2Promise = app.waitForEvent('window')
  await openModuleTile(hub)
  const editor2 = await editor2Promise
  await editor2.waitForLoadState('domcontentloaded')
  await wait(editor2, 5500)
  await shot(editor2, 'r4-wieder-geoeffnet.png')
  const cols = await headers(editor2)
  check('R5 Der Editor öffnet den ZULETZT gespeicherten Stand',
    cols.some((h) => h.includes('Stand-B')) && !cols.some((h) => h.includes('Stand-A')),
    cols.join(' | '))
  check('R5b Und er sagt, welcher Entwurf geladen ist',
    (await editor2.evaluate(() => document.body.innerText)).includes('Nur die Spalten'))

  // ── R6 · Zwei Entwürfe desselben Moduls sind unterscheidbar ──────────────
  await editor2.close()
  await wait(hub, 1200)
  await hub.reload()
  await wait(hub, 4500)
  await hub.evaluate(() => { window.location.hash = '#/admin/anpassungen' })
  await wait(hub, 3000)
  await shot(hub, 'r5-hub-liste.png')
  const hubText = await hub.evaluate(() => document.body.innerText)
  check('R6 Der benannte Entwurf steht so in der Liste', hubText.includes('Nur die Spalten'))
  check('R6b Und nicht mehr unter dem Einheitsnamen', !hubText.includes('Helpdesk — Anpassung'))

  // ── R7 · Umbenennen im Nachhinein ───────────────────────────────────────
  const row = hub.locator('div[role="button"]').filter({ hasText: /Nur die Spalten/ }).last()
  if ((await row.count()) > 0) {
    await row.click()
    await wait(hub, 1500)
    const renameField = hub.getByLabel('Entwurf umbenennen').first()
    await renameField.fill('Spalten + Prioritäten')
    await renameField.press('Enter')
    await wait(hub, 2000)
    await shot(hub, 'r6-umbenannt.png')
    const after = await hub.evaluate(() => document.body.innerText)
    check('R7 Ein Entwurf lässt sich später umbenennen', after.includes('Spalten + Prioritäten'))
  } else {
    check('R7 Ein Entwurf lässt sich später umbenennen', false, 'Zeile nicht gefunden')
  }
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n').slice(0, 3).join(' ⏎ '))
  for (const [i, w] of app.windows().entries()) {
    await w.screenshot({ path: resolve(outDir, `zz-abbruch-${i}.png`) }).catch(() => {})
  }
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await app.close()
process.exit(fail > 0 ? 1 : 0)
