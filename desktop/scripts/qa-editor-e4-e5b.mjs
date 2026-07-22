/**
 * QA — Modul-Editor E-4 (Galerie) + E-5b (Rollout-/Entwurfs-Liste + Rollback).
 *
 * A) Anpassungen-Hub: Galerie-Kacheln (Kontakte/Helpdesk) mit Dimensions-Chips +
 *    Status „Standard", Rollout-Liste leer.
 * B) Deploy „Jetzt" aus dem Kontakte-Editor → Hub zeigt Rollout „Live" + Kachel
 *    „Angepasst" + Zurückrollen.
 * C) Rollback → Toast + Status „Ersetzt", Kachel zurück auf „Standard".
 * D) „Als Entwurf speichern" (Helpdesk) → Rollout-Liste „Entwurf" + Öffnen/Löschen.
 *
 * Screenshots → .qa-screenshots/editor-e4-e5b/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5175'
const outDir = resolve('.qa-screenshots/editor-e4-e5b')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1360, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []
const bodyText = () => page.evaluate(() => document.body.innerText)
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

// NB: client-side hash navigation (NOT page.goto, which reloads the document and
// resets the in-memory drafts store — wiping deploys/drafts made in the editor).
// The real app keeps the main window's store alive across editor deploys.
async function gotoHub() {
  await page.evaluate(() => { window.location.hash = '#/admin/anpassungen' })
  await waitForText('Modul-Editor', 15000)
  await wait(1200)
}
async function openEditor(moduleKey) {
  await page.evaluate((k) => { window.location.hash = `#/editor-window?module=${k}` }, moduleKey)
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(1500)
}

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2000)

  // ── A) Hub: gallery + empty rollouts ───────────────────────────────────────
  await gotoHub()
  await shot('01-hub-initial.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'A Galerie (Kontakte+Helpdesk, Chips, Standard) + Rollouts leer',
      cards: /Kontakte/.test(txt) && /Helpdesk/.test(txt),
      chips: /Begriffe/.test(txt) && /Werteliste/.test(txt) && /Feld-Typ/.test(txt),
      standard: /Standard/.test(txt),
      emptyRollouts: /Noch keine Entwürfe oder Rollouts/.test(txt),
      pass: /Kontakte/.test(txt) && /Feld-Typ/.test(txt) && /Standard/.test(txt) && /Noch keine Entwürfe oder Rollouts/.test(txt),
    })
  }

  // ── B) Deploy now from the Kontakte editor ─────────────────────────────────
  await openEditor('kontakte')
  await page.locator('nav button').filter({ hasText: /^Begriffe$/ }).first().click()
  await wait(700)
  // rename first term
  const term = page.locator('aside input').first()
  await term.fill('Klienten')
  await wait(500)
  // open deploy dialog (footer "Übernehmen" may be overlapped → dispatch click)
  const apply = page.locator('button').filter({ hasText: /^Übernehmen$/ }).first()
  await apply.dispatchEvent('click')
  await wait(700)
  await shot('02-deploy-dialog.png')
  await page.locator('button').filter({ hasText: 'Jetzt übernehmen' }).first().click({ force: true })
  await waitForText('Änderungen übernommen', 8000)
  await wait(600)
  await shot('02b-after-deploy-click.png')

  // ── back to hub → rollout Live + card Angepasst + rollback ──────────────────
  await gotoHub()
  await shot('03-hub-after-deploy.png')
  {
    const txt = await bodyText()
    // NB: subtitle contains "Live-System", so test the unambiguous signals only.
    out.push({
      step: 'B Nach Deploy: Kachel „Angepasst" + Rollout-Zeile + Zurückrollen',
      customized: /Angepasst/.test(txt),
      rollbackBtn: /Zurückrollen/.test(txt),
      pass: /Angepasst/.test(txt) && /Zurückrollen/.test(txt),
    })
  }

  // ── C) Rollback ─────────────────────────────────────────────────────────────
  await page.locator('button').filter({ hasText: 'Zurückrollen' }).first().click()
  await wait(900)
  await shot('04-after-rollback.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'C Rollback → „Ersetzt" + Kachel zurück auf „Standard"',
      superseded: /Ersetzt/.test(txt),
      backToStandard: /Standard/.test(txt),
      pass: /Ersetzt/.test(txt) && /Standard/.test(txt),
    })
  }

  // ── D) Save-as-draft from Helpdesk editor ──────────────────────────────────
  await openEditor('helpdesk')
  await page.locator('nav button').filter({ hasText: /^Felder$/ }).first().click()
  await wait(800)
  // toggle first field visibility → makes the draft dirty
  await page.locator('aside button[aria-label="Sichtbarkeit umschalten"]').first().dispatchEvent('click')
  await wait(400)
  // footer "Als Entwurf speichern"
  await page.locator('button').filter({ hasText: 'Als Entwurf speichern' }).first().dispatchEvent('click')
  await wait(700)

  await gotoHub()
  await shot('05-hub-with-draft.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'D Entwurf gespeichert (Helpdesk) → Rollout-Liste „Entwurf"',
      draftRow: /Entwurf/.test(txt),
      pass: /Entwurf/.test(txt),
    })
  }

  const raw = /customization\.(editor|fields|labels)\.[a-z]/i.test(await bodyText())
  out.push({ step: 'E Keine rohen i18n-Keys', rawKeysFound: raw, pass: !raw })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Modul-Editor E-4 + E-5b — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
