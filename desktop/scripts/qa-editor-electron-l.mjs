/**
 * QA — der echte Electron-Fall (Darien 2026-08-05: „als Entwurf speichern geht,
 * nur speichert er den Entwurf nicht — wenn man den Entwurf öffnet, ist es
 * wieder das aktuelle Layout").
 *
 * Die bisherigen Suiten fahren einen Browser. Der Editor läuft in der App aber
 * als eigenes BrowserWindow, geöffnet über IPC — ein anderer Renderer-Prozess,
 * eigener Heap, echte Fenster-Lebensdauer. Genau darin steckte der Fehler, und
 * genau das bildet nur ein echter Electron-Start ab.
 *
 * Voraussetzung: der Renderer-Dev-Server läuft auf :5173 (npm run dev).
 * Eigenes --user-data-dir, damit eine parallel laufende App nicht kollidiert.
 * Screenshots → .qa-screenshots/editor-electron-l/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/editor-electron-l')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-electron')
await rm(userDataDir, { recursive: true, force: true })

const RENAMED = 'Anliegen'

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
    } catch { /* egal */ }
  })
  // Erst nach einem Neuladen greifen Onboarding-Flag und Launch-Skip — sonst
  // läuft die Startanimation noch, wenn der Test schon klickt.
  await hub.reload()
  await wait(hub, 4000)
  await hub.evaluate(() => { window.location.hash = '#/admin/anpassungen' })
  await wait(hub, 4000)
  await shot(hub, 'l0-hub.png')

  // ── Editor über den echten IPC-Weg öffnen ────────────────────────────────
  const editorPromise = app.waitForEvent('window')
  await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
  const editor = await editorPromise
  await editor.waitForLoadState('domcontentloaded')
  await wait(editor, 4500)
  await shot(editor, 'l1-editor.png')
  check('L0 Der Editor öffnet als eigenes Fenster', (await editor.title()).length >= 0)

  // Ein zweiter Klick darf KEIN weiteres Fenster aufmachen.
  const before = app.windows().length
  await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
  await wait(hub, 1500)
  check('L1 Ein zweiter Klick öffnet kein zweites Editor-Fenster',
    app.windows().length === before, `${before} → ${app.windows().length}`)

  // ── Im Editor: Spalte umbenennen + als Entwurf speichern ─────────────────
  await editor.getByRole('button', { name: /^Spalten/i }).first().click()
  await wait(editor, 1800)
  await editor.getByRole('button', { name: /Spalte Betreff umbenennen/i }).first().click()
  await wait(editor, 700)
  const input = editor.locator('input[aria-label*="umbenennen"]').first()
  await input.fill(RENAMED)
  await input.press('Enter')
  await wait(editor, 1800)
  check('L2 Umbenennen wirkt in der Editor-Vorschau', (await headers(editor)).some((h) => h.includes(RENAMED)))

  // Layout mit dazu — genau das, was Darien vermisst hat: Breite ziehen,
  // Reihenfolge tauschen, eine Spalte ausblenden.
  const grip = editor.locator('[data-column-resize]').nth(1)
  const box = await grip.boundingBox()
  await editor.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await editor.mouse.down()
  await editor.mouse.move(box.x + 120, box.y + box.height / 2, { steps: 8 })
  await editor.mouse.up()
  await wait(editor, 1200)
  const movehandle = editor.getByRole('button', { name: /Spalte Ticket-Nr\.? verschieben/i }).first()
  await movehandle.focus()
  await editor.keyboard.press('Space')
  await wait(editor, 400)
  await editor.keyboard.press('ArrowDown')
  await wait(editor, 400)
  await editor.keyboard.press('Space')
  await wait(editor, 1200)
  await editor.getByRole('switch', { name: /Spalte Kategorie ausblenden/i }).first().click()
  await wait(editor, 1200)
  const draftCols = await headers(editor)
  await shot(editor, 'l2a-layout-im-entwurf.png')

  await editor.locator('button').filter({ hasText: /^Als Entwurf speichern$/ }).first().dispatchEvent('click')
  await wait(editor, 2000)
  await shot(editor, 'l2-gespeichert.png')

  // Was steht wirklich im Entwurf? (Der Kern von Dariens Beobachtung.)
  const stored = await editor.evaluate(() => {
    const raw = localStorage.getItem('cosmi:customization:drafts')
    if (!raw) return { ok: false, reason: 'kein Eintrag' }
    const parsed = JSON.parse(raw)
    const d = parsed.drafts?.[0]
    if (!d) return { ok: false, reason: 'kein Entwurf' }
    return {
      ok: true,
      labels: Object.keys(d.payload?.labels?.de ?? {}),
      areas: Object.keys(d.payload?.moduleAreas?.helpdesk ?? {}).length,
    }
  })
  check('L3 Der gespeicherte Entwurf trägt den Namen', stored.ok && stored.labels?.length > 0,
    JSON.stringify(stored.labels))
  check('L3b Der gespeicherte Entwurf trägt AUCH das Layout (Breite/Reihenfolge/Sichtbarkeit)',
    (stored.areas ?? 0) > 0, `moduleAreas-Einträge: ${stored.areas}`)

  // ── Editor schließen, Entwurf vom Hub aus fortsetzen ─────────────────────
  await editor.close()
  await wait(hub, 1500)
  await hub.reload()
  await wait(hub, 4000)
  const row = hub.locator('div[role="button"]').filter({ hasText: /Helpdesk — Anpassung/ }).last()
  const hasRow = (await row.count()) > 0
  check('L4 Der Entwurf steht im Hub (auch nach Neuladen)', hasRow)
  await shot(hub, 'l3-hub-mit-entwurf.png')

  {
    const viaCardPromise = app.waitForEvent('window')
    await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
    const viaCard = await viaCardPromise
    await viaCard.waitForLoadState('domcontentloaded')
    await wait(viaCard, 5000)
    await shot(viaCard, 'l5-ueber-kachel.png')
    const cardCols = await headers(viaCard)
    check('L8 Editor über die Modul-Kachel setzt den offenen Entwurf fort',
      cardCols.join('|') === draftCols.join('|'), cardCols.join(' | '))
    check('L9 Der Editor sagt, welchen Entwurf er geladen hat',
      (await viaCard.evaluate(() => document.body.innerText)).includes('Entwurf:'))
    await viaCard.close()
    await wait(hub, 1000)
  }

  if (hasRow) {
    await row.click()
    await wait(hub, 1500)
    const editor2Promise = app.waitForEvent('window')
    await hub.getByRole('button', { name: /Weiter bearbeiten/i }).first().click()
    const editor2 = await editor2Promise
    await editor2.waitForLoadState('domcontentloaded')
    await wait(editor2, 5000)
    await shot(editor2, 'l4-fortgesetzt.png')
    const cols = await headers(editor2)
    check('L5 Der fortgesetzte Entwurf bringt den Namen mit', cols.some((h) => h.includes(RENAMED)),
      cols.join(' | '))
    check('L6 Der fortgesetzte Entwurf bringt das Layout mit',
      cols.join('|') === draftCols.join('|'), `Entwurf: ${draftCols.join(' | ')}  ·  geöffnet: ${cols.join(' | ')}`)
    const widthBack = await editor2.locator('table thead th').first().evaluate((el) => el.style.width)
    check('L7 Die gezogene Breite kommt mit', /%$/.test(widthBack || ''), widthBack || '(leer)')
  }
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n')[0])
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await app.close()
process.exit(fail > 0 ? 1 : 0)
