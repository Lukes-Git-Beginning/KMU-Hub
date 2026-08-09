/**
 * QA — Wertelisten, Teil 3: eine selbst angelegte Liste muss auch nach dem
 * Übernehmen noch bearbeitbar sein (W1 aus der Prüfliste).
 *
 * Vorher gehörte eine im Editor angelegte Liste zu keinem Modul: das Panel kannte
 * nur die fest deklarierten Listen plus den aktuellen Entwurf. Sobald der Entwurf
 * live ging, war die eigene Liste aus dem Panel verschwunden — im Modul benutzt,
 * aber nur noch über das Feld erreichbar, das sie zufällig gebunden hat.
 *
 * Voraussetzung: Renderer-Dev-Server auf :5173 (npm run dev).
 * Screenshots → .qa-screenshots/editor-wertelisten-p/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/editor-wertelisten-p')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-wertelisten-p')
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
const clickRail = async (p, name) =>
  p.locator('nav[aria-label="Anpassen"] button').filter({ hasText: name }).first().click()
const listNames = async (p) => p.getByLabel('Name der Liste').evaluateAll((els) => els.map((e) => e.value))

const NAME = 'Eskalationsstufe'

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

  // ── Liste anlegen und übernehmen ────────────────────────────────────────
  const editorPromise = app.waitForEvent('window')
  await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
  const editor = await editorPromise
  await editor.waitForLoadState('domcontentloaded')
  await wait(editor, 5000)
  await clickRail(editor, 'Wertelisten')
  await wait(editor, 1800)

  await editor.getByRole('button', { name: 'Neue Werteliste' }).first().click()
  await wait(editor, 1200)
  await editor.getByLabel('Name der Liste').last().fill(NAME)
  await wait(editor, 1200)
  await shot(editor, 'p1-liste-angelegt.png')
  check('P0 Die neue Liste steht im Panel', (await listNames(editor)).includes(NAME))

  // Zusätzlich eine benutzte Option entfernen — nach dem Deploy muss der Umzug im
  // Panel weiterhin sichtbar und rücknehmbar sein.
  await editor.getByRole('button', { name: /.Mittel. löschen/ }).first().click()
  await wait(editor, 1000)
  await editor.locator('select').last().selectOption({ label: 'Hoch' })
  await editor.getByRole('button', { name: 'Entfernen' }).first().click()
  await wait(editor, 1500)

  await editor.locator('button').filter({ hasText: /^Übernehmen$/ }).first().dispatchEvent('click')
  await wait(editor, 1500)
  await editor.getByRole('button', { name: 'Jetzt übernehmen' }).first().click()
  await wait(hub, 3500)

  // ── Editor neu öffnen: ist die Liste noch da? ────────────────────────────
  // Erst sicherstellen, dass kein Editor-Fenster mehr offen ist — es gibt nur
  // eines pro Modul, ein zweiter Klick würde sonst kein neues Fenster öffnen.
  for (const w of app.windows()) {
    if (w === hub) continue
    await w.close().catch(() => {})
  }
  await wait(hub, 1500)
  await hub.reload()
  await wait(hub, 4500)
  await hub.evaluate(() => { window.location.hash = '#/admin/anpassungen' })
  await wait(hub, 3500)
  const editor2Promise = app.waitForEvent('window')
  await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
  const editor2 = await editor2Promise
  await editor2.waitForLoadState('domcontentloaded')
  await wait(editor2, 5500)
  await clickRail(editor2, 'Wertelisten')
  await wait(editor2, 2000)
  await shot(editor2, 'p2-nach-uebernehmen.png')
  const namesAfter = await listNames(editor2)
  check('P1 Die eigene Liste ist nach dem Übernehmen weiter im Panel bearbeitbar',
    namesAfter.includes(NAME), namesAfter.join(' · '))

  // Und sie ist keine Dublette: genau einmal.
  check('P2 Sie steht genau einmal in der Liste',
    namesAfter.filter((n) => n === NAME).length === 1,
    `${namesAfter.filter((n) => n === NAME).length}×`)

  // ── P3 · Ein bereits ausgerollter Umzug bleibt sichtbar und rücknehmbar ──
  const panelText = await editor2.evaluate(() => document.body.innerText)
  check('P3 Der ausgerollte Umzug steht weiter als „Entfernt" mit Ziel im Panel',
    /ENTFERNT/i.test(panelText) && panelText.includes('→'))
  const restore = editor2.getByRole('button', { name: /Wiederherstellen/ }).first()
  check('P3b Es gibt einen Weg zurück', (await restore.count()) > 0)
  if ((await restore.count()) > 0) {
    await restore.click()
    await wait(editor2, 1500)
    await shot(editor2, 'p3-wiederhergestellt.png')
    const opts = await editor2.getByLabel(/^Option .* umbenennen$/).evaluateAll((els) => els.map((e) => e.value))
    check('P3c Die Option ist zurück in der Liste', opts.includes('Mittel'), opts.slice(0, 5).join(' · '))
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
