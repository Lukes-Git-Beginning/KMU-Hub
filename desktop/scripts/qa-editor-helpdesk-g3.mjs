/**
 * QA — Helpdesk Editor G3: Wissensdatenbank auf der Block-Dokument-Engine.
 *
 *   S1 KB-Tab: „Neuer Artikel"-Button + Artikel-Karten (Preview = Text, kein JSON).
 *   S2 Seed-Artikel öffnen → DocumentReader rendert den Inhalt (kein rohes JSON).
 *   S3 „Bearbeiten" → Block-Editor + editierbarer Titel.
 *   S4 „Neuer Artikel" → öffnet direkt im Block-Editor (Titel „Neuer Artikel"),
 *      Titel ändern → Speichern → Karte erscheint in der Liste.
 *   S5 Keine Page-Errors.
 * Screenshots → .qa-screenshots/editor-helpdesk-g3/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-g3')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const bodyText = () => page.evaluate(() => document.body.innerText)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

try {
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Helpdesk', 15000)

  // Switch to the Wissensdatenbank tab.
  await page.locator('button, [role="button"]', { hasText: /^Wissensdatenbank$/ }).first().click()
  await wait(1200)
  await shot('01-kb-list.png')
  {
    const txt = await bodyText()
    const hasNew = txt.includes('Neuer Artikel')
    const hasCard = txt.includes('VPN-Einrichtung')
    const noRawJson = !/\[\{"id":"row/.test(txt)
    out.push({ step: 'S1 KB-Liste: Neuer-Artikel-Button + Karten (kein JSON)', hasNew, hasCard, noRawJson, pass: hasNew && hasCard && noRawJson })
  }

  // S2 — open a seed article → DocumentReader renders it (no raw JSON).
  await page.locator('button', { hasText: 'VPN-Einrichtung' }).first().click()
  await wait(1000)
  await shot('02-reader.png')
  {
    const txt = await bodyText()
    const rendered = txt.includes('Anleitung zur Einrichtung des VPN-Zugangs')
    const noRawJson = !txt.includes('[{"id"') && !txt.includes('"type":"text"')
    out.push({ step: 'S2 Seed-Artikel via DocumentReader (kein JSON)', rendered, noRawJson, pass: rendered && noRawJson })
  }

  // S3 — edit → block editor + editable title.
  await page.locator('button', { hasText: /^Bearbeiten$/ }).first().click()
  await wait(1000)
  await shot('03-editor.png')
  {
    const titleInput = page.locator('input[value="VPN-Einrichtung für Mitarbeiter"]')
    const hasTitleInput = (await titleInput.count()) > 0
    // The block editor mounts an editable text surface (contenteditable via TipTap).
    const editable = await page.locator('[contenteditable="true"]').count()
    out.push({ step: 'S3 Block-Editor + editierbarer Titel', hasTitleInput, editableCount: editable, pass: hasTitleInput && editable > 0 })
  }
  // Cancel back to view.
  await page.locator('button', { hasText: /^Abbrechen$/ }).first().click().catch(() => {})
  await wait(600)
  // Back to list.
  await page.locator('button', { hasText: 'Zurück zur Übersicht' }).first().click().catch(() => {})
  await wait(800)

  // S4 — create a new article → opens in edit mode → rename → save → card appears.
  await page.locator('button', { hasText: /^Neuer Artikel$/ }).first().click()
  await wait(1200)
  await shot('04a-new-editing.png')
  let newTitleOk = false
  {
    const titleInput = page.locator('input[value="Neuer Artikel"]')
    newTitleOk = (await titleInput.count()) > 0
    if (newTitleOk) {
      await titleInput.first().fill('QA Testartikel')
      await wait(300)
    }
  }
  // Save.
  await page.locator('button', { hasText: /^Speichern$/ }).first().click().catch(() => {})
  await wait(1200)
  // Back to list to confirm the card.
  await page.locator('button', { hasText: 'Zurück zur Übersicht' }).first().click().catch(() => {})
  await wait(1000)
  await shot('04b-new-card.png')
  {
    const txt = await bodyText()
    const cardThere = txt.includes('QA Testartikel')
    out.push({ step: 'S4 Neuer Artikel im Block-Editor → Titel → Karte in Liste', newTitleOk, cardThere, pass: newTitleOk && cardThere })
  }

  out.push({ step: 'S5 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk G3 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
