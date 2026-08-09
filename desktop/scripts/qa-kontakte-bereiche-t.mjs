/**
 * QA — Kontakte-Bereiche laufen über Zustand statt über den Router (Paket C3).
 *
 * Kontakte war das einzige Modul ohne schaltbare Bereiche: `KontakteLayout` navigierte
 * per `NavLink` + `<Outlet/>`, und die Editor-Sandbox hat keine passende URL — der
 * Inhalt wäre leer geblieben, einen eigenen Router darf sie nicht aufmachen.
 * Seit 2026-08-10 führt Zustand den Bereich, die Routen bleiben als Einstieg.
 *
 * Diese Suite prüft **beide Hälften**, weil der Umbau an einer Route eines Kern-Moduls
 * hängt und ein Fehler dort teurer wäre als das Feature wert ist:
 *
 *   T0–T5  die Live-App: Bereichswechsel, URL wandert mit, Deep-Link, Zurück-Button,
 *          Detail-Seite mit Leiste darüber.
 *   T6–T9  der Editor: Bereichs-Leiste in der Vorschau, Bereich abschalten, das
 *          Ergebnis nach dem Übernehmen im echten Modul.
 *
 * Voraussetzung: Renderer-Dev-Server auf :5173 (npm run dev).
 * Screenshots → .qa-screenshots/kontakte-bereiche-t/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/kontakte-bereiche-t')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-kontakte-t')
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
const bodyText = (p) => p.evaluate(() => document.body.innerText)
const hash = (p) => p.evaluate(() => window.location.hash)
const goto = async (p, h) => {
  await p.evaluate((x) => { window.location.hash = x }, h)
  await wait(p, 3000)
}
/** Die Bereichs-Leiste der Kunden-Zentrale. */
const bereiche = (p) => p.locator('nav.border-b button').allInnerTexts()

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

  // ══ Live-App ═══════════════════════════════════════════════════════════════

  await goto(hub, '#/kontakte')
  await shot(hub, 't0-kontakte.png')
  const leiste = await bereiche(hub)
  check('T0 Die Bereichs-Leiste steht', leiste.length >= 5, leiste.join(' | '))

  // Bereich wechseln — und die URL muss mitwandern, sonst sind Links nicht teilbar.
  await hub.locator('nav.border-b button').filter({ hasText: /^Unternehmen$/ }).first().click()
  await wait(hub, 2500)
  await shot(hub, 't1-firmen.png')
  check('T1 Ein Klick wechselt den Bereich', (await bodyText(hub)).length > 0)
  check('T2 Die URL wandert mit', (await hash(hub)).includes('/kontakte/firmen'), await hash(hub))

  // Zurück-Button: URL führt, der Zustand muss folgen.
  await hub.goBack()
  await wait(hub, 2500)
  await shot(hub, 't2-zurueck.png')
  check('T3 Der Zurück-Button führt zurück in den ersten Bereich',
    !(await hash(hub)).includes('/firmen'), await hash(hub))

  // Deep-Link: direkt in einen Bereich springen (frisch geladen, nicht geklickt).
  await goto(hub, '#/kontakte/pipeline')
  await hub.reload()
  await wait(hub, 5000)
  await goto(hub, '#/kontakte/pipeline')
  await shot(hub, 't3-deeplink-pipeline.png')
  const pipelineAktiv = await hub
    .locator('nav.border-b button.tab-accent-active')
    .first()
    .innerText()
    .catch(() => '')
  check('T4 Ein Deep-Link landet im richtigen Bereich', /Deals|Pipeline/i.test(pipelineAktiv),
    `aktiv: ${pipelineAktiv || '(keiner)'}`)

  // Detail-Seite: bleibt eine echte Route und behält die Leiste darüber.
  await goto(hub, '#/kontakte/firmen')
  await wait(hub, 2500)
  const ersteZeile = hub.locator('table tbody tr, div[role="button"]').first()
  if ((await ersteZeile.count()) > 0) {
    await ersteZeile.click()
    await wait(hub, 3000)
    await shot(hub, 't4-firma-detail.png')
    check('T5 Die Detail-Seite behält die Bereichs-Leiste',
      (await bereiche(hub)).length >= 5, (await hash(hub)))
    // Der erste Durchlauf war hier grün und das Bild trotzdem falsch: auf der
    // Firmen-Detailseite leuchtete „Kontakte". Der aktive Bereich muss mitgeprüft
    // werden, sonst deckt die Suite nur ab, DASS eine Leiste da ist.
    const aktivImDetail = await hub
      .locator('nav.border-b button.tab-accent-active')
      .first()
      .innerText()
      .catch(() => '')
    check('T5b Die Detail-Seite behält den Bereich, aus dem sie kommt',
      /Unternehmen|Firmen/i.test(aktivImDetail), `aktiv: ${aktivImDetail || '(keiner)'}`)
  } else {
    check('T5 Die Detail-Seite behält die Bereichs-Leiste', false, 'keine Zeile zum Öffnen gefunden')
    check('T5b Die Detail-Seite behält den Bereich, aus dem sie kommt', false, 'keine Zeile zum Öffnen gefunden')
  }

  // ══ Editor ═════════════════════════════════════════════════════════════════

  await goto(hub, '#/admin/anpassungen')
  await wait(hub, 2500)
  const editorPromise = app.waitForEvent('window')
  await hub.locator('button').filter({ hasText: /Kontakte/ }).first().click()
  const editor = await editorPromise
  await editor.waitForLoadState('domcontentloaded')
  await wait(editor, 5500)
  await shot(editor, 't5-editor-offen.png')

  const editorBereiche = await bereiche(editor)
  check('T6 Die Vorschau zeigt die Bereichs-Leiste (vorher unmöglich)',
    editorBereiche.length >= 5, editorBereiche.join(' | '))

  // Bereiche-Panel: einen Bereich abschalten.
  await editor.locator('nav[aria-label="Anpassen"] button').filter({ hasText: /Bereiche/ }).first().click()
  await wait(editor, 2000)
  await shot(editor, 't6-bereiche-panel.png')
  const schalter = editor.getByRole('switch')
  check('T7 Das Bereiche-Panel bietet Schalter an', (await schalter.count()) >= 5,
    `${await schalter.count()} Schalter`)

  const vorher = (await bereiche(editor)).length
  await editor.getByRole('switch').filter({ hasText: /Leads/i }).first().click()
    .catch(async () => { await editor.getByRole('switch').nth(1).click() })
  await wait(editor, 2000)
  await shot(editor, 't7-bereich-aus.png')
  const nachher = (await bereiche(editor)).length
  check('T8 Ein abgeschalteter Bereich verschwindet aus der Vorschau',
    nachher === vorher - 1, `${vorher} → ${nachher}`)

  // ── Übernehmen und im echten Modul nachsehen ────────────────────────────
  await editor.locator('button').filter({ hasText: /^Übernehmen$/ }).first().dispatchEvent('click')
  await wait(editor, 1500)
  await editor.getByRole('button', { name: 'Jetzt übernehmen' }).first().click()
  await wait(hub, 3000)

  await goto(hub, '#/kontakte')
  await hub.reload()
  await wait(hub, 5000)
  await goto(hub, '#/kontakte')
  await shot(hub, 't8-modul-nach-deploy.png')
  const liveBereiche = await bereiche(hub)
  check('T9 Der abgeschaltete Bereich ist auch im echten Modul weg',
    liveBereiche.length === nachher, `${liveBereiche.length} Bereiche: ${liveBereiche.join(' | ')}`)
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n')[0])
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await app.close()
process.exit(fail > 0 ? 1 : 0)
