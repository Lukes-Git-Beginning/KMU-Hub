/**
 * QA — Zwei-Fenster-Fall (Darien 2026-08-05: „wenn ich als Entwurf speichern
 * drücke, dann speichert er keinen Entwurf, auch beim direkten Deployen").
 *
 * In Electron läuft der Editor in einem EIGENEN Fenster mit eigenem JS-Heap; die
 * bisherige QA lief in einer einzigen Page und hat das nie abgebildet. Hier sind
 * es zwei Pages im selben Browser-Context — gleicher Origin, getrennte Heaps,
 * geteilter localStorage: genau die Electron-Konstellation.
 *
 *   K1 Ein im Editor-Fenster gespeicherter Entwurf steht im Hub-Fenster.
 *   K2 Ein im Editor-Fenster übernommener Rollout steht dort als „Live".
 *   K3 Der Entwurf lässt sich vom Hub aus mit seinem Stand fortsetzen.
 * Screenshots → .qa-screenshots/editor-twowindow-k/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-twowindow-k')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const RENAMED = 'Anliegen'

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1500, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)

const out = []
let pass = 0, fail = 0
const check = (n, ok, extra = '') => { out.push(`${ok ? 'PASS' : 'FAIL'}  ${n}${extra ? ' · ' + extra : ''}`); ok ? pass++ : fail++ }

// Fenster 1 = Cosmi-Hauptfenster, Fenster 2 = Editor. Eigene Heaps, ein Origin.
const hub = await ctx.newPage()
const editor = await ctx.newPage()
const errs = []
for (const [name, p] of [['hub', hub], ['editor', editor]]) {
  p.on('pageerror', (e) => errs.push(`${name}: ${String(e).split('\n')[0]}`))
  p.on('console', (m) => {
    if (m.type() === 'error' && !/ERR_CONNECTION_REFUSED|Failed to load resource/.test(m.text())) {
      errs.push(`${name} console: ${m.text()}`)
    }
  })
}
const wait = (p, ms) => p.waitForTimeout(ms)
const shot = (p, n) => p.screenshot({ path: resolve(outDir, n), fullPage: false })
const textOf = (p) => p.evaluate(() => document.body.innerText)

try {
  await hub.goto(`${BASE}/#/admin/anpassungen`, { waitUntil: 'domcontentloaded' })
  await wait(hub, 3500)
  await editor.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(editor, 3800)

  // ── Im Editor-Fenster: Spalte umbenennen, als Entwurf speichern ───────────
  await editor.getByRole('button', { name: /^Spalten/i }).first().click()
  await wait(editor, 1500)
  await editor.getByRole('button', { name: /Spalte Betreff umbenennen/i }).first().click()
  await wait(editor, 600)
  const input = editor.locator('input[aria-label*="umbenennen"]').first()
  await input.fill(RENAMED)
  await input.press('Enter')
  await wait(editor, 1500)
  await editor.locator('button').filter({ hasText: /^Als Entwurf speichern$/ }).first().dispatchEvent('click')
  await wait(editor, 1800)
  await shot(editor, 'k1a-editor-gespeichert.png')

  // ── Im Hub-Fenster: steht er in der Liste? ────────────────────────────────
  // MIT Reload — das ist der Fall, an dem es hing: „gespeichert" muss auch einen
  // Neustart überstehen, nicht nur einen Fensterwechsel.
  await hub.reload({ waitUntil: 'domcontentloaded' })
  await wait(hub, 3500)
  await shot(hub, 'k1b-hub-liste.png')
  const hubRow = hub.locator('div[role="button"]').filter({ hasText: /Helpdesk — Anpassung/ }).last()
  const hasRow = (await hubRow.count()) > 0
  check('K1 Der im Editor-Fenster gespeicherte Entwurf steht im Hub', hasRow,
    hasRow ? (await hubRow.innerText()).replace(/\n/g, ' · ') : (await textOf(hub)).slice(-120))

  // ── Fortsetzen: Stand muss mitkommen ──────────────────────────────────────
  if (hasRow) {
    await hubRow.click()
    await wait(hub, 1400)
    await shot(hub, 'k3a-detail.png')
    const detailText = (await textOf(hub)).toLowerCase()
    check('K2 Das Detail-Fenster kennt die Änderungen des Entwurfs',
      /\d+\s+spalten?/.test(detailText) || /\d+\s+begriff/.test(detailText))
    await hub.getByRole('button', { name: /Weiter bearbeiten/i }).first().click()
    await wait(hub, 1200)
    // Übergabe liegt im geteilten Speicher — das Editor-Fenster liest sie beim Start.
    const stash = await hub.evaluate(() => localStorage.getItem('cosmi:customization:resume-draft'))
    check('K3 Der Entwurf wird zur Übergabe an das Editor-Fenster abgelegt', Boolean(stash))
    const editor2 = await ctx.newPage()
    await editor2.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
    await wait(editor2, 4000)
    await shot(editor2, 'k3b-fortgesetzt.png')
    const cols = await editor2.locator('table thead th').allInnerTexts()
    check('K4 Das neue Editor-Fenster öffnet den Entwurf mit seinem Stand',
      cols.some((h) => h.includes(RENAMED)), cols.join(' | '))
    await editor2.close()
  }

  // ── Deploy aus dem Editor-Fenster → Hub sieht „Live" ──────────────────────
  await editor.locator('button').filter({ hasText: /^Übernehmen$/ }).first().dispatchEvent('click')
  await wait(editor, 900)
  await editor.locator('button').filter({ hasText: 'Jetzt übernehmen' }).first().click({ force: true })
  await wait(editor, 2200)
  await hub.reload({ waitUntil: 'domcontentloaded' })
  await wait(hub, 3500)
  await shot(hub, 'k2-hub-nach-deploy.png')
  const liveRow = hub.locator('div[role="button"]').filter({ hasText: /Helpdesk — Anpassung/ }).last()
  const liveOk = (await liveRow.count()) > 0 && (await liveRow.innerText()).includes('Live')
  check('K5 Der im Editor-Fenster übernommene Rollout steht im Hub als „Live"', liveOk,
    (await liveRow.count()) > 0 ? (await liveRow.innerText()).replace(/\n/g, ' · ') : 'keine Zeile')

  check('Keine Seitenfehler', errs.length === 0, errs.slice(0, 3).join(' | '))
} catch (e) {
  out.push('ABBRUCH: ' + String(e).split('\n')[0])
  await shot(hub, 'zz-abbruch-hub.png')
  fail++
}

console.log(out.join('\n'))
console.log(`\n${pass}/${pass + fail} grün`)
await b.close()
process.exit(fail > 0 ? 1 : 0)
