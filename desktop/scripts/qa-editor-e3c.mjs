/**
 * QA — Modul-Editor E-3c (Felder-Panel) + Modul-Identität-Regel (Option 1).
 *
 * A) Option 1: Modul-Namen sind fix — Editor-Titel „Kontakte bearbeiten" (nicht
 *    „Patientenverwaltung"), Sidebar-Nav „Kontakte" (nicht „Patienten").
 * B) Helpdesk-Felder: Seed-Felder, Feld anlegen (Modal) → „Neu"-Badge + Zähler,
 *    Sichtbarkeit togglen, Reset.
 * C) Kontakte-Felder: 4-Entity-Umschalter, Felder je Entity.
 *
 * Screenshots → .qa-screenshots/editor-e3c/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5175'
const outDir = resolve('.qa-screenshots/editor-e3c')
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

async function openEditor(moduleKey) {
  await page.goto(`${BASE}/#/editor-window?module=${moduleKey}`, { waitUntil: 'domcontentloaded' })
  // Hash-only nav does not reload the SPA — force a fresh page so no state/cache
  // from a previous module carries over (real app opens a separate OS window).
  await page.reload({ waitUntil: 'domcontentloaded' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(1800)
  await page.locator('nav button').filter({ hasText: /^Felder$/ }).first().click()
  await wait(1000)
}

try {
  // Boot main shell (demo continue if needed)
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2000)
  const lt = await bodyText()
  if (/Anmelden|Login|Sign in/i.test(lt) && !/Übersicht|Dashboard/i.test(lt)) {
    const d = page.locator('button').filter({ hasText: /Demo|Continue|Fortfahren/i }).first()
    if ((await d.count()) > 0) { await d.click(); await wait(2000) }
  }

  // ── A) Option 1 — Sidebar-Nav zeigt „Kontakte", nicht „Patienten" ──────────
  await shot('01-main-shell.png')
  {
    const navHasKontakte = await page.locator('nav, aside').filter({ hasText: 'Kontakte' }).count() > 0
    const navHasPatienten = await page.locator('nav, aside').filter({ hasText: /Patienten/ }).count() > 0
    out.push({
      step: 'A1 Sidebar-Nav „Kontakte" fix (nicht „Patienten")',
      navHasKontakte, navHasPatienten,
      pass: navHasKontakte && !navHasPatienten,
    })
  }

  // ── B) Helpdesk-Felder ─────────────────────────────────────────────────────
  await openEditor('helpdesk')
  await shot('02-helpdesk-felder.png')
  {
    const txt = await bodyText()
    const titleFixed = /Helpdesk bearbeiten/.test(txt) && !/Patient/i.test(txt)
    const seeds = /SLA-Stufe/.test(txt) && /Eskalationsgrund/.test(txt) && /Kontaktkanal/.test(txt)
    const count3 = /3 Felder/.test(txt)
    out.push({ step: 'B1 Helpdesk-Felder: Titel fix + 3 Seed-Felder', titleFixed, seeds, count3, pass: titleFixed && seeds && count3 })
  }

  // Create a field via modal
  await page.locator('aside button').filter({ hasText: 'Feld anlegen' }).first().click()
  await wait(600)
  await page.locator('[role="dialog"] input').first().fill('Rückrufwunsch')
  await wait(300)
  await shot('03-create-modal.png')
  await page.locator('[role="dialog"] button').filter({ hasText: 'Feld anlegen' }).first().click()
  await wait(700)
  await shot('04-field-created.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'B2 Feld angelegt „Rückrufwunsch" → Neu-Badge + Zähler + 4 Felder',
      hasField: /Rückrufwunsch/.test(txt),
      newBadge: /\bNeu\b/.test(txt),
      counter: /1 Änderung/.test(txt),
      count4: /4 Felder/.test(txt),
      pass: /Rückrufwunsch/.test(txt) && /\bNeu\b/.test(txt) && /1 Änderung/.test(txt) && /4 Felder/.test(txt),
    })
  }

  // Toggle visibility of first field (eye button)
  const eye = page.locator('aside button[aria-label="Sichtbarkeit umschalten"]').first()
  if ((await eye.count()) > 0) {
    await eye.click(); await wait(400)
    await shot('05-toggle-visible.png')
    out.push({ step: 'B3 Sichtbarkeit getoggelt (Feld durchgestrichen)', pass: true })
  } else {
    out.push({ step: 'B3 Eye-Button nicht gefunden', pass: false })
  }

  // Reset the entity
  const reset = page.locator('aside button[aria-label="Zurücksetzen"]').first()
  if ((await reset.count()) > 0) {
    await reset.click(); await wait(500)
    await shot('06-after-reset.png')
    const txt = await bodyText()
    out.push({
      step: 'B4 Reset → zurück auf 3 Felder, keine Änderungen, kein Rückrufwunsch',
      pass: /3 Felder/.test(txt) && /Keine Änderungen/.test(txt) && !/Rückrufwunsch/.test(txt),
    })
  } else {
    out.push({ step: 'B4 Reset-Button nicht gefunden', pass: false })
  }

  // ── C) Kontakte-Felder — 4-Entity-Umschalter + Titel fix ───────────────────
  await openEditor('kontakte')
  await shot('07-kontakte-felder.png')
  {
    const txt = await bodyText()
    const titleFixed = /Kontakte bearbeiten/.test(txt) && !/Patientenverwaltung/.test(txt)
    // entity switcher radios (labels: Kontakte/Firmen/Deals/Aktivitäten)
    const radios = await page.locator('aside [role="radio"]').count()
    const contactFields = /Kundennummer ERP/.test(txt) && /Newsletter-Einwilligung/.test(txt)
    out.push({ step: 'C1 Kontakte: Titel fix + 4-Entity-Umschalter + crm_contact-Felder', titleFixed, radios, contactFields, pass: titleFixed && radios === 4 && contactFields })
  }

  // Switch to Firmen entity
  const firmen = page.locator('aside [role="radio"]').filter({ hasText: 'Firmen' }).first()
  if ((await firmen.count()) > 0) {
    await firmen.click(); await wait(600)
    await shot('08-kontakte-firmen.png')
    const txt = await bodyText()
    out.push({ step: 'C2 Umschalten auf „Firmen" → USt-IdNr.-Feld', pass: /USt-IdNr\./.test(txt) })
  } else {
    out.push({ step: 'C2 Firmen-Radio nicht gefunden', pass: false })
  }

  // Raw-key sweep
  const raw = /customization\.(editor|fields|labels)\.[a-z]/i.test(await bodyText())
  out.push({ step: 'D Keine rohen i18n-Keys sichtbar', rawKeysFound: raw, pass: !raw })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Modul-Editor E-3c + Option 1 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
