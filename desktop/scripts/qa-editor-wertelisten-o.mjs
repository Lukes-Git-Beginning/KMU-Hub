/**
 * QA — Wertelisten, Teil 2: überlebt es das „Übernehmen"? (W10/W11)
 *
 * Der Verdacht aus der Prüfliste: die Vorschau verspricht beim Entfernen einer
 * benutzten Option „Bestehende Einträge werden geändert auf: X" — und nach dem
 * Deploy fiel das Modul auf den entfernten Wert zurück, weil die Umzugs-Tabelle
 * nur im Editor existierte. Diese Suite geht den ganzen Weg: bearbeiten →
 * übernehmen → im ECHTEN Modul nachsehen → zurückrollen → wieder nachsehen.
 *
 * Voraussetzung: Renderer-Dev-Server auf :5173 (npm run dev).
 * Screenshots → .qa-screenshots/editor-wertelisten-o/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/editor-wertelisten-o')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-wertelisten-o')
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
/** Prioritäts-Chips der Ticket-Tabelle (Spalte 4). */
const priorities = async (p) => p.locator('table tbody tr td:nth-child(4)').allInnerTexts()
const goto = async (p, hash) => {
  await p.evaluate((h) => { window.location.hash = h }, hash)
  await wait(p, 3500)
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

  // ── Ausgangslage im echten Modul festhalten ──────────────────────────────
  await goto(hub, '#/helpdesk')
  await shot(hub, 'o0-modul-vorher.png')
  const before = await priorities(hub)
  const mittelBefore = before.filter((c) => c.includes('Mittel')).length
  const hochBefore = before.filter((c) => c.includes('Hoch')).length
  check('O0 Ausgangslage: das Modul zeigt „Mittel"', mittelBefore > 0, `${mittelBefore}× Mittel, ${hochBefore}× Hoch`)

  // ── Im Editor: umbenennen + eine benutzte Option entfernen ───────────────
  await goto(hub, '#/admin/anpassungen')
  await wait(hub, 2500)
  await shot(hub, 'o0b-hub.png')
  const editorPromise = app.waitForEvent('window')
  await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
  const editor = await editorPromise
  await editor.waitForLoadState('domcontentloaded')
  await wait(editor, 5000)
  await clickRail(editor, 'Wertelisten')
  await wait(editor, 1800)

  await editor.getByLabel(/Option .Hoch. umbenennen/).first().fill('Dringend')
  await wait(editor, 1200)
  await editor.getByRole('button', { name: /.Mittel. löschen/ }).first().click()
  await wait(editor, 1200)
  // Das Umzugsziel bewusst wählen — der Auswahlkasten des Dialogs ist der letzte
  // im Fenster (davor stehen die Filter des Moduls). Ohne diese Wahl liefe der
  // Test gegen den Vorgabewert und würde nicht prüfen, was er behauptet.
  await editor.locator('select').last().selectOption({ label: 'Dringend' })
  await editor.getByRole('button', { name: 'Entfernen' }).first().click()
  await wait(editor, 1800)
  await shot(editor, 'o1-editor-vorschau.png')
  const previewPrios = await editor.locator('.min-h-full table tbody tr td:nth-child(4)').allInnerTexts()
  check('O1 Die Vorschau zieht die Einträge auf „Dringend" um',
    previewPrios.filter((c) => c.includes('Mittel')).length === 0 && previewPrios.some((c) => c.includes('Dringend')),
    previewPrios.slice(0, 5).join(' | '))

  // ── Übernehmen ──────────────────────────────────────────────────────────
  await editor.locator('button').filter({ hasText: /^Übernehmen$/ }).first().dispatchEvent('click')
  await wait(editor, 1500)
  await shot(editor, 'o2-deploy-dialog.png')
  await editor.getByRole('button', { name: 'Jetzt übernehmen' }).first().click()
  await wait(hub, 3000)

  // ── Im ECHTEN Modul nachsehen ───────────────────────────────────────────
  await goto(hub, '#/helpdesk')
  await hub.reload()
  await wait(hub, 5000)
  await goto(hub, '#/helpdesk')
  await shot(hub, 'o3-modul-nach-deploy.png')
  const after = await priorities(hub)
  check('O2 Nach dem Übernehmen heißt „Hoch" im Modul „Dringend"',
    after.some((c) => c.includes('Dringend')), after.slice(0, 5).join(' | '))
  check('O3 Nach dem Übernehmen ist kein Eintrag mehr auf der entfernten Option',
    after.filter((c) => c.includes('Mittel')).length === 0,
    `noch ${after.filter((c) => c.includes('Mittel')).length}× Mittel (vorher ${mittelBefore}×)`)
  // Und sie sind dort gelandet, wo der Dialog es versprochen hat: bei „Dringend"
  // stehen jetzt die eigenen plus die umgezogenen Einträge.
  check('O3b Die umgezogenen Einträge tragen das gewählte Ziel',
    after.filter((c) => c.includes('Dringend')).length === mittelBefore + hochBefore,
    `${after.filter((c) => c.includes('Dringend')).length}× Dringend, erwartet ${mittelBefore + hochBefore}`)

  // ── Zurückrollen ────────────────────────────────────────────────────────
  await goto(hub, '#/admin/anpassungen')
  await shot(hub, 'o4-hub-rollout.png')
  const rolloutRow = hub.locator('div[role="button"]').filter({ hasText: /Helpdesk/ }).last()
  if ((await rolloutRow.count()) > 0) {
    await rolloutRow.click()
    await wait(hub, 1500)
    const rollbackBtn = hub.getByRole('button', { name: 'Zurückrollen' }).first()
    const hasRollback = (await rollbackBtn.count()) > 0
    check('O4 Der Rollout lässt sich zurückrollen', hasRollback)
    if (hasRollback) {
      await rollbackBtn.click()
      await wait(hub, 2500)
      await goto(hub, '#/helpdesk')
      await shot(hub, 'o5-modul-nach-rollback.png')
      const back = await priorities(hub)
      check('O5 Zurückrollen bringt die Einträge zurück',
        back.filter((c) => c.includes('Mittel')).length === mittelBefore
        && !back.some((c) => c.includes('Dringend')),
        `${back.filter((c) => c.includes('Mittel')).length}× Mittel · ${back.slice(0, 4).join(' | ')}`)
    }
  } else {
    check('O4 Der Rollout steht im Hub', false, 'keine Zeile gefunden')
  }
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n').slice(0, 4).join(' ⏎ '))
  for (const [i, w] of app.windows().entries()) {
    await w.screenshot({ path: resolve(outDir, `zz-abbruch-fenster-${i}.png`) }).catch(() => {})
  }
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await app.close()
process.exit(fail > 0 ? 1 : 0)
