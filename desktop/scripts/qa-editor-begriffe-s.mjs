/**
 * QA — überleben umbenannte Beschriftungen das „Übernehmen"? (Paket C2)
 *
 * Die Falle: `applyDraftToTenant` schreibt nur Labels, deren Key in
 * `LABEL_WHITELIST` steht. Der Editor ließ aber deutlich mehr umbenennen, als dort
 * gelistet war — Tabs, Statistik-Kacheln, die Überschriften im Ticket-Fenster. Man
 * benannte um, die Vorschau zeigte den neuen Namen, und beim Übernehmen war er
 * still weg. Kein Fehler, keine Meldung.
 *
 * Der Wächter `mocks/data/__tests__/label-whitelist.test.ts` hält die Liste ab
 * jetzt vollständig; diese Suite prüft die andere Hälfte — dass der eingetragene
 * Key den ganzen Weg bis ins echte Modul geht. Genommen werden bewusst zwei
 * Beschriftungen, die vor dem Fix verworfen wurden, und beide Bedienarten:
 * Tab = Doppelklick (`interactive`), Statistik-Kachel = einfacher Klick.
 *
 * Nebenbei: der Chip-Baustein ist nach `components/shared/VsChip.tsx` gezogen
 * (C1) — die Suite sieht nach, dass die Prioritäts-Chips danach noch stehen.
 *
 * Gegenprobe gefahren (2026-08-10): nimmt man die beiden Keys wieder aus der
 * Whitelist, bleiben S1/S2 **grün** und S3–S6 fallen um — die Vorschau zeigt den
 * neuen Namen, das Modul den alten. Genau diese Kombination macht den Fehler so
 * teuer: im Editor sieht alles richtig aus. Die Suite fängt ihn also wirklich.
 *
 * Voraussetzung: Renderer-Dev-Server auf :5173 (npm run dev).
 * Screenshots → .qa-screenshots/editor-begriffe-s/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/editor-begriffe-s')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-begriffe-s')
await rm(userDataDir, { recursive: true, force: true })

/** Zwei Beschriftungen, die vor dem Fix beim Übernehmen verloren gingen. */
const TAB_ALT = 'Wissensdatenbank'
const TAB_NEU = 'Hilfe-Artikel'
const KACHEL_ALT = 'Offene Tickets'
const KACHEL_NEU = 'Unerledigt'

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
const goto = async (p, hash) => {
  await p.evaluate((h) => { window.location.hash = h }, hash)
  await wait(p, 3500)
}
/** Beschriftung im Eingabefeld ersetzen — `innerText` sieht Eingabefelder nicht. */
const tippen = async (p, text) => {
  const feld = p.locator('input:focus').first()
  await feld.fill(text)
  await feld.press('Enter')
  await wait(p, 1200)
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
    } catch { /* egal */ }
  })
  await hub.reload()
  await wait(hub, 4000)

  // ── Ausgangslage im echten Modul ─────────────────────────────────────────
  await goto(hub, '#/helpdesk')
  await shot(hub, 's0-modul-vorher.png')
  const vorher = await bodyText(hub)
  check('S0 Ausgangslage: der Tab heißt noch wie geliefert', vorher.includes(TAB_ALT))
  // Der Chip-Baustein liegt seit C1 in shared/ — rendert er nach dem Umzug noch?
  const chips = await hub.locator('table tbody tr td:nth-child(4)').allInnerTexts()
  check('S0b Die Prioritäts-Chips stehen (VsChip aus shared/)',
    chips.length > 0 && chips.some((c) => c.trim().length > 0), chips.slice(0, 4).join(' | '))

  // ── Im Editor umbenennen ────────────────────────────────────────────────
  await goto(hub, '#/admin/anpassungen')
  await wait(hub, 2500)
  const editorPromise = app.waitForEvent('window')
  await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
  const editor = await editorPromise
  await editor.waitForLoadState('domcontentloaded')
  await wait(editor, 5000)
  await shot(editor, 's1-editor-offen.png')

  // Tab: steckt in einem Schalter → Doppelklick (interactive).
  await editor.locator('button').filter({ hasText: new RegExp(`^${TAB_ALT}`) }).first().dblclick()
  await wait(editor, 800)
  await tippen(editor, TAB_NEU)
  await shot(editor, 's2-tab-umbenannt.png')
  check('S1 Der neue Tab-Name steht in der Vorschau',
    (await bodyText(editor)).includes(TAB_NEU))

  // Statistik-Kachel: freistehend → einfacher Klick. Erst in den Statistik-Tab.
  await editor.locator('button').filter({ hasText: /^Statistik/ }).first().click()
  await wait(editor, 2000)
  await editor.getByTitle('Klicken zum Umbenennen').filter({ hasText: KACHEL_ALT }).first().click()
  await wait(editor, 800)
  await tippen(editor, KACHEL_NEU)
  await shot(editor, 's3-kachel-umbenannt.png')
  check('S2 Der neue Kachel-Name steht in der Vorschau',
    (await bodyText(editor)).includes(KACHEL_NEU))

  // ── Übernehmen ──────────────────────────────────────────────────────────
  await editor.locator('button').filter({ hasText: /^Übernehmen$/ }).first().dispatchEvent('click')
  await wait(editor, 1500)
  await shot(editor, 's4-deploy-dialog.png')
  await editor.getByRole('button', { name: 'Jetzt übernehmen' }).first().click()
  await wait(hub, 3000)

  // ── Im ECHTEN Modul nachsehen — hier ging es vorher verloren ────────────
  await goto(hub, '#/helpdesk')
  await hub.reload()
  await wait(hub, 5000)
  await goto(hub, '#/helpdesk')
  await shot(hub, 's5-modul-nach-deploy.png')
  const nachher = await bodyText(hub)

  check('S3 Der umbenannte Tab überlebt das Übernehmen',
    nachher.includes(TAB_NEU), nachher.includes(TAB_ALT) ? `heißt noch „${TAB_ALT}"` : '')
  check('S4 Der alte Tab-Name ist im Modul verschwunden', !nachher.includes(TAB_ALT))

  await hub.locator('button').filter({ hasText: /^Statistik/ }).first().click()
  await wait(hub, 2500)
  await shot(hub, 's6-modul-statistik.png')
  const statistik = await bodyText(hub)
  check('S5 Die umbenannte Statistik-Kachel überlebt das Übernehmen',
    statistik.includes(KACHEL_NEU), statistik.includes(KACHEL_ALT) ? `heißt noch „${KACHEL_ALT}"` : '')
  check('S6 Der alte Kachel-Name ist im Modul verschwunden', !statistik.includes(KACHEL_ALT))
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n')[0])
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await app.close()
process.exit(fail > 0 ? 1 : 0)
