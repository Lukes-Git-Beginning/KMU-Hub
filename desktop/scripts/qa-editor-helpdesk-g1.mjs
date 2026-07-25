/**
 * QA — Helpdesk Editor G1: Werteliste-Option löschen + Reassignment-Migration.
 *   S1 Editor + Wertelisten offen; Tabelle hat „In Bearbeitung"-Status-Chips.
 *   S2 Trash auf „In Bearbeitung" (Basis-Option) → Reassignment-Karte erscheint.
 *   S3 Ziel „Offen" bestätigen → „In Bearbeitung"-Chips verschwinden, Tickets zeigen live „Offen".
 *   S4 „+ Neue Option" hinzufügen → direkt löschbar (kein Dialog, weil neu/unbenutzt).
 *   S5 Keine Page-Errors.
 * Screenshots → .qa-screenshots/editor-helpdesk-g1/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-g1')
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
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})
const tableChip = (label) => page.locator('td span', { hasText: new RegExp(`^${label}$`) })

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2500)
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2500)
  await page.locator('button, [role="button"]', { hasText: /^Wertelisten$/ }).first().click()
  await wait(1200)
  const inProgBefore = await tableChip('In Bearbeitung').count()
  await shot('01-before.png')
  out.push({ step: 'S1 „In Bearbeitung"-Chips in Tabelle', inProgBefore, pass: inProgBefore > 0 })

  // S2 — trash on "In Bearbeitung" → reassignment card
  const trash = page.locator('button[aria-label*="In Bearbeitung"]').first()
  const trashFound = (await trash.count()) > 0
  if (trashFound) { await trash.click(); await wait(600) }
  await shot('02-reassign-card.png')
  const confirmBtn = page.locator('button', { hasText: /^Entfernen$/ })
  out.push({ step: 'S2 Reassignment-Karte erscheint', trashFound, confirmVisible: (await confirmBtn.count()) > 0, pass: trashFound && (await confirmBtn.count()) > 0 })

  // S3 — confirm reassign to "Offen" (default first other option)
  if ((await confirmBtn.count()) > 0) { await confirmBtn.first().click(); await wait(1000) }
  await shot('03-migrated.png')
  {
    const inProgAfter = await tableChip('In Bearbeitung').count()
    out.push({ step: 'S3 „In Bearbeitung" migriert → Chips weg', inProgAfter, pass: inProgAfter === 0 })
  }

  // S4 — add a fresh option then delete it directly (no reassignment dialog)
  await page.locator('button', { hasText: /Neue Option/ }).first().click()
  await wait(700)
  const addedTrash = page.locator('button[aria-label*="Neue Option"]').first()
  const addedFound = (await addedTrash.count()) > 0
  if (addedFound) { await addedTrash.click(); await wait(600) }
  await shot('04-added-deleted.png')
  {
    // After a direct delete there must be NO reassignment card (no "Entfernen" btn)
    const cardOpen = await page.locator('button', { hasText: /^Entfernen$/ }).count()
    out.push({ step: 'S4 Neue Option direkt löschbar (kein Dialog)', addedFound, cardOpen, pass: addedFound && cardOpen === 0 })
  }

  out.push({ step: 'S5 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk G1 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
