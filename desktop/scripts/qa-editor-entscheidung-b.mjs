/**
 * Kein Test — Bildmaterial für zwei Entscheidungen (Darien 2026-08-09).
 *
 * B1: Was passiert, wenn eine Spalte sehr breit gezogen wird? (Untergrenze?)
 * B2: Wie hoch baut die Spalten-Zeile im rechten Panel? (kompakter?)
 *
 * Screenshots → .qa-screenshots/editor-entscheidung-b/
 */
import { _electron as electron } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'

const outDir = resolve('.qa-screenshots/editor-entscheidung-b')
await mkdir(outDir, { recursive: true })
const userDataDir = resolve(tmpdir(), 'cosmi-qa-entscheidung-b')
await rm(userDataDir, { recursive: true, force: true })

const app = await electron.launch({
  args: [resolve('out/main/index.js'), `--user-data-dir=${userDataDir}`],
  env: { ...process.env, ELECTRON_RENDERER_URL: 'http://localhost:5173', NODE_ENV: 'development' },
})
const wait = (p, ms) => p.waitForTimeout(ms)

const hub = await app.firstWindow()
await hub.waitForLoadState('domcontentloaded')
await hub.evaluate(() => {
  const K = 'cosmi-ui'
  const raw = localStorage.getItem(K)
  const p = raw ? JSON.parse(raw) : { state: {}, version: 0 }
  p.state = { ...(p.state || {}), onboardingCompleted: true }
  localStorage.setItem(K, JSON.stringify(p))
  sessionStorage.setItem('cosmi:launch-played', '1')
  localStorage.removeItem('cosmi:customization:drafts')
})
await hub.reload()
await wait(hub, 4000)
await hub.evaluate(() => { window.location.hash = '#/admin/anpassungen' })
await wait(hub, 4000)

const editorPromise = app.waitForEvent('window')
await hub.locator('button').filter({ hasText: /Helpdesk/ }).first().click()
const editor = await editorPromise
await editor.waitForLoadState('domcontentloaded')
await wait(editor, 5500)
await editor.locator('nav[aria-label="Anpassen"] button').filter({ hasText: 'Spalten' }).first().click()
await wait(editor, 2000)

// ── B1 · Ausgangslage, dann eine Spalte extrem breit ziehen ───────────────
await editor.screenshot({ path: resolve(outDir, 'b1-vorher.png') })

const grip = editor.locator('[data-column-resize]').nth(1)
const box = await grip.boundingBox()
if (box) {
  await editor.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await editor.mouse.down()
  // Weit nach rechts: das ist der Fall, um den es bei der Untergrenze geht.
  await editor.mouse.move(box.x + 620, box.y + box.height / 2, { steps: 14 })
  await editor.mouse.up()
  await wait(editor, 1500)
}
await editor.screenshot({ path: resolve(outDir, 'b1-extrem-gezogen.png') })

// ── B2 · Das Panel mit gesetzten Breiten (die dreizeiligen Einträge) ──────
await editor.locator('aside, .w-\\[320px\\]').first().screenshot({ path: resolve(outDir, 'b2-panel.png') }).catch(async () => {
  await editor.screenshot({ path: resolve(outDir, 'b2-panel.png') })
})

console.log('Screenshots liegen in .qa-screenshots/editor-entscheidung-b/')
await app.close()
