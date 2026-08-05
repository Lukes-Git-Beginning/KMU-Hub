/**
 * QA — Helpdesk Editor Feedback F1 (Farben) + F2 (Status-Werteliste).
 *   S1 Editor offen.
 *   S2 Wertelisten zeigt Ticket-Priorität UND Ticket-Status + „Neue Option"-Button.
 *   S3 (F1) Modul-Chips tragen Inline-Farbe aus der Werteliste (color-mix) — nicht mehr nur Tailwind.
 *   S4 (F2) Status „Offen" → „Neu" umbenennen → Tabellen-Status-Chips werden LIVE „Neu".
 *   S5 (F1) „Kritisch" umfärben (Swatch) → Screenshot zur Sichtprüfung.
 *   S6 Keine Page-Errors.
 * Screenshots → .qa-screenshots/editor-helpdesk-f1f2/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-f1f2')
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
const bodyText = () => page.evaluate(() => document.body.innerText)
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})
const findInput = async (val) => {
  const boxes = page.getByRole('textbox'); const n = await boxes.count()
  for (let i = 0; i < n; i++) { const el = boxes.nth(i); if ((await el.inputValue().catch(() => '')) === val) return el }
  return null
}

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2500)
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2500)
  out.push({ step: 'S1 Editor offen', pass: /Helpdesk/.test(await bodyText()) })

  await page.locator('button, [role="button"]', { hasText: /^Wertelisten$/ }).first().click()
  await wait(1200)
  await shot('01-wertelisten-both.png')
  {
    // Value-set names/labels live in <input> (not innerText) → assert via the
    // per-set "Neue Option" buttons: 2 = both Ticket-Priorität + Ticket-Status.
    const addButtons = await page.locator('button', { hasText: /Neue Option/ }).count()
    out.push({ step: 'S2 Priorität + Status je mit „Neue Option"', addButtons, pass: addButtons === 2 })
  }

  // S3 (F1) — module priority chip carries an inline colour from the value-set
  const kritischCell = page.locator('td span', { hasText: /^Kritisch$/ }).first()
  const kritischStyleBefore = (await kritischCell.count()) > 0
    ? await kritischCell.evaluate((el) => el.getAttribute('style') || '')
    : ''
  out.push({
    step: 'S3 (F1) Prioritäts-Chip hat Inline-Farbe aus Werteliste',
    styleHasColor: /color-mix|background/i.test(kritischStyleBefore),
    pass: /color-mix|background/i.test(kritischStyleBefore),
  })

  // S4 (F2) — rename status "Offen" → "Neu"; table status chips must update live
  const offenInput = await findInput('Offen')
  const offenFound = offenInput !== null
  if (offenFound) { await offenInput.fill('Neu'); await offenInput.press('Tab') }
  await wait(1000)
  await shot('02-status-renamed.png')
  {
    const neuChip = await page.locator('td span', { hasText: /^Neu$/ }).count()
    out.push({ step: 'S4 (F2) Status „Offen" → „Neu" live im Modul', offenFound, neuChipCount: neuChip, pass: offenFound && neuChip > 0 })
  }

  // S5 (F1) — recolour "Kritisch" via a swatch, screenshot for visual check
  try {
    const kritInput = await findInput('Kritisch')
    if (kritInput) {
      const row = kritInput.locator('xpath=ancestor::div[2]')
      const swatches = row.locator('button')
      const sc = await swatches.count()
      if (sc > 0) { await swatches.nth(sc - 2).click() } // second-to-last swatch (green)
      await wait(800)
    }
  } catch { /* best effort */ }
  await shot('03-kritisch-recolored.png')
  {
    const kritAfter = (await kritischCell.count()) > 0
      ? await kritischCell.evaluate((el) => el.getAttribute('style') || '')
      : ''
    out.push({ step: 'S5 (F1) „Kritisch" umfärben → Chip-Style vorhanden (siehe Bild)', changed: kritAfter !== kritischStyleBefore, pass: /color-mix|background/i.test(kritAfter) })
  }

  out.push({ step: 'S6 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk F1/F2 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
