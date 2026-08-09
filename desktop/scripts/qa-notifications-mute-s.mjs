/**
 * QA — Benachrichtigungen: kein Ping mehr, und ein Schalter für Ruhe
 * (Darien 2026-08-09: „bitte standardmäßig die Notifications stummschalten, das
 * nicht immer dieser Ping-Sound kommt … und wir brauchen eine Möglichkeit alle
 * Notifikationen stumm zu schalten").
 *
 * Voraussetzung: Renderer-Dev-Server auf :5173 (npm run dev).
 * Screenshots → .qa-screenshots/notifications-mute-s/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/notifications-mute-s')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-mute-s')
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
const readStore = (p) => p.evaluate(() => {
  const raw = localStorage.getItem('cosmi-notifications')
  return raw ? JSON.parse(raw).state : null
})

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
  await hub.reload()
  await wait(hub, 4500)

  // ── S1 · Frische Installation: Ton aus ──────────────────────────────────
  // Zustand schreibt erst bei der ersten Änderung — „kein Eintrag" heißt also
  // ebenfalls „Vorgabe gilt", und die Vorgabe ist jetzt: Ton aus.
  const fresh = await readStore(hub)
  check('S1 Frisch installiert ist der Ton aus', fresh === null || fresh.soundEnabled === false,
    fresh === null ? 'nichts gespeichert → Vorgabe' : `soundEnabled=${fresh.soundEnabled}`)
  check('S1b „Alles stumm" ist aus (nichts wird verschluckt)',
    fresh === null || fresh.muteAll === false,
    fresh === null ? 'nichts gespeichert → Vorgabe' : `muteAll=${fresh.muteAll}`)

  // ── S2 · Bestehende Installation mit eingeschaltetem Ton wird migriert ──
  await hub.evaluate(() => {
    localStorage.setItem('cosmi-notifications', JSON.stringify({
      state: { notifications: [], isDropdownOpen: false, soundEnabled: true },
      version: 0,
    }))
  })
  await hub.reload()
  await wait(hub, 4500)
  const migrated = await readStore(hub)
  check('S2 Ein gespeichertes „Ton an" wird einmalig abgeschaltet',
    migrated?.soundEnabled === false, `soundEnabled=${migrated?.soundEnabled}`)

  // ── S3 · Der Schalter im Benachrichtigungs-Center ───────────────────────
  await hub.evaluate(() => { window.location.hash = '#/notifications' })
  await wait(hub, 3500)
  await shot(hub, 's1-center.png')
  const muteBtn = hub.getByRole('button', { name: /Alle Benachrichtigungen stummschalten/ }).first()
  check('S3 Das Benachrichtigungs-Center hat einen Stumm-Schalter', (await muteBtn.count()) > 0)
  await muteBtn.click()
  await wait(hub, 1200)
  await shot(hub, 's2-stumm.png')
  const muted = await readStore(hub)
  check('S3b Er schaltet wirklich alles stumm', muted?.muteAll === true, `muteAll=${muted?.muteAll}`)
  check('S3c Und sagt es auch',
    (await hub.evaluate(() => document.body.innerText)).includes('Alles stumm'))

  // ── S4 · Die Einstellungen zeigen beide Schalter ────────────────────────
  // Sie leben im Modul-Einstellungs-Fenster (Sidebar), nicht auf einer Route.
  await hub.locator('button, a, [role="button"]').filter({ hasText: /Modul-Einstellun/ }).first().click()
  await wait(hub, 2500)
  const entry = hub.locator('button, a, [role="button"]').filter({ hasText: /^Benachrichtigungen$/ }).first()
  if ((await entry.count()) > 0) {
    await entry.dispatchEvent('click')
    await wait(hub, 2500)
  }
  await shot(hub, 's3-einstellungen.png')
  const settingsText = await hub.evaluate(() => document.body.innerText)
  check('S4 Einstellungen: Stummschalten ist da', settingsText.includes('Alle Benachrichtigungen stummschalten'))
  check('S4b Einstellungen: der Ton hat weiterhin einen eigenen Schalter',
    settingsText.includes('Ton bei neuen Benachrichtigungen'))
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
