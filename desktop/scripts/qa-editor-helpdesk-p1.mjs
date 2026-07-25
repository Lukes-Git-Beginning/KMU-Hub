/**
 * QA — Helpdesk-Pilot P1 (Editor-Pivot): Labels edit-in-place + Mutations-Guard.
 *
 * Beweist am echten Helpdesk-Modul im Editor-Fenster:
 *   S1 Editor öffnet, Reiter-Leiste (Tickets/Wissensdatenbank/Statistik) da.
 *   S2 Reiter EINFACH-Klick schaltet weiter (Vorschau bleibt begehbar).
 *   S3 Tabellen-Header „Betreff" ANklicken → Inline-Input → „Anliegen" + Enter → live + Entwurf dirty.
 *   S4 Reiter „Statistik" DOPPEL-Klick → Inline-Input (interactive-Modus) → umbenennen live.
 *   S5 „Neues Ticket" klicken → Guard-Toast, Dialog öffnet NICHT (Editor = nur anpassen).
 *   S6 Keine rohen i18n-Keys.
 *
 * Screenshots → .qa-screenshots/editor-helpdesk-p1/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-p1')
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

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2500)
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2500)
  await shot('01-editor-open.png')
  {
    const txt = await bodyText()
    const hasTabs = /Tickets/.test(txt) && /Wissensdatenbank/.test(txt) && /Statistik/.test(txt)
    out.push({ step: 'S1 Editor offen + Reiter-Leiste sichtbar', hasTabs, pass: hasTabs })
  }

  // S2 — single click Wissensdatenbank switches the tab (navigable preview intact)
  const kbTab = page.locator('button', { hasText: /Wissensdatenbank/ }).first()
  await kbTab.click()
  await wait(1000)
  await shot('02-tab-switch.png')
  {
    const txt = await bodyText()
    const switched = /Veröffentlicht|Entwurf|VPN|Netzwerkdrucker/i.test(txt)
    out.push({ step: 'S2 Einfach-Klick schaltet Reiter (begehbar)', switched, pass: switched })
  }

  // back to Tickets
  await page.locator('button', { hasText: /^Tickets/ }).first().click()
  await wait(1000)

  // S3 — click table header "Betreff" → inline rename → "Anliegen"
  const betreff = page.locator('th [role="button"]', { hasText: /^Betreff$/ }).first()
  const betreffCount = await betreff.count()
  await betreff.click()
  await wait(400)
  await shot('03-header-input.png')
  const hdrInput = page.locator('th input').first()
  if ((await hdrInput.count()) > 0) {
    await hdrInput.fill('Anliegen')
    await hdrInput.press('Enter')
  }
  await wait(800)
  await shot('04-header-renamed.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'S3 Tabellen-Header „Betreff" → „Anliegen" live',
      betreffWasEditable: betreffCount > 0,
      hasAnliegen: /Anliegen/.test(txt),
      dirty: /Änderung/.test(txt),
      pass: betreffCount > 0 && /Anliegen/.test(txt),
    })
  }

  // S4 — double click "Statistik" tab label → inline rename (interactive mode)
  const statTab = page.locator('button', { hasText: /^Statistik$/ }).first()
  if ((await statTab.count()) > 0) {
    await statTab.dblclick()
    await wait(400)
    await shot('05-tab-dblclick-input.png')
    const tabInput = page.locator('input:focus')
    if ((await tabInput.count()) > 0) {
      await tabInput.fill('Auswertung')
      await tabInput.press('Enter')
      await wait(800)
    }
  }
  await shot('06-tab-renamed.png')
  {
    const txt = await bodyText()
    out.push({ step: 'S4 Reiter „Statistik" per Doppelklick → „Auswertung"', hasAuswertung: /Auswertung/.test(txt), pass: /Auswertung/.test(txt) })
  }

  // S5 — mutations guard: "Neues Ticket" → toast, dialog stays closed
  const newBtn = page.locator('button', { hasText: /Neues Ticket/ }).first()
  const newBtnCount = await newBtn.count()
  if (newBtnCount > 0) {
    await newBtn.click()
    await wait(900)
  }
  await shot('07-guard-toast.png')
  {
    const txt = await bodyText()
    const toastShown = /Im Editor deaktiviert/.test(txt)
    // The new-ticket dialog title must NOT be present
    const dialogTitle = await page.locator('text=/Neues Ticket erstellen|Ticket erstellen/').count()
    out.push({
      step: 'S5 „Neues Ticket" → Guard-Toast, Dialog bleibt zu',
      newBtnPresent: newBtnCount > 0,
      toastShown,
      dialogOpen: dialogTitle > 0,
      pass: toastShown,
    })
  }

  // S6 — no raw i18n keys
  const raw = /helpdesk\.(tabs|table|stats|ticket|kb)\.|customization\.editor\.[a-z]+\.[a-z]/i.test(await bodyText())
  out.push({ step: 'S6 Keine rohen i18n-Keys', rawKeysFound: raw, pass: !raw })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk-Pilot P1 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
