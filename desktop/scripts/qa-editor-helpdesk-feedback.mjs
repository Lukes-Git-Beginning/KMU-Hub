/**
 * QA — Helpdesk Editor-Feedback (Darien lokal-Review, 2026-07-26).
 *
 *   S1 Ticket-Detail „Zusatzfelder"-Sektion aufgeräumt: schlichte Überschrift,
 *      KEIN Einstellungs-Icon (Punkt Z).
 *   S2 Editor: Vorschau-Toggle entfernt — kein Button „Vorschau", Trio-Nav +
 *      Properties-Panel dauerhaft sichtbar (Punkt 1).
 *   S3 Wertelisten: die 2 vordefinierten Sets zeigen statischen Titel + Hinweis
 *      „Im Modul per Klick umbenennen", KEIN Namens-Input (Punkt 7b).
 *   S4 „Neue Werteliste" anlegen → neue Karte mit editierbarem Namen erscheint
 *      (Punkt 7a).
 *   S5 Keine Page-Errors.
 * Screenshots → .qa-screenshots/editor-helpdesk-feedback/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-feedback')
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
  // ── LIVE APP: Ticket-Detail Zusatzfelder aufgeräumt (Punkt Z) ───────────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Zutrittskarte', 15000)
  await page.locator('tr[role="button"]', { hasText: 'Zutrittskarte' }).first().click()
  await wait(1200)
  await shot('01-detail-zusatzfelder.png')
  {
    const heading = page.locator('h4', { hasText: 'Zusatzfelder' }).first()
    const headingCount = await heading.count()
    const svgInHeading = headingCount ? await heading.locator('svg').count() : -1
    out.push({
      step: 'S1 Zusatzfelder-Überschrift schlicht, kein Icon',
      headingCount, svgInHeading,
      pass: headingCount > 0 && svgInHeading === 0,
    })
  }
  await page.keyboard.press('Escape').catch(() => {})
  await wait(400)

  // ── EDITOR: Vorschau-Toggle entfernt (Punkt 1) ─────────────────────────────
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2800)
  await shot('02-editor-toolbar.png')
  {
    const previewBtn = await page.locator('button', { hasText: 'Vorschau' }).count()
    const previewExact = await page.getByText('Vorschau', { exact: true }).count()
    const navVisible = await page.locator('nav[aria-label="Anpassen"]').isVisible().catch(() => false)
    const inspectorVisible = (await bodyText()).includes('So passt du an')
    out.push({
      step: 'S2 Vorschau-Toggle + Sandbox-Label weg, Panels dauerhaft sichtbar',
      previewBtn, previewExact, navVisible, inspectorVisible,
      pass: previewBtn === 0 && previewExact === 0 && navVisible && inspectorVisible,
    })
  }

  // ── EDITOR: Wertelisten — statischer Set-Titel (Punkt 7b) ───────────────────
  await page.locator('nav[aria-label="Anpassen"] button', { hasText: 'Wertelisten' }).first().click()
  await wait(1000)
  await shot('03-wertelisten-static-name.png')
  {
    const hintCount = await page.locator('text=Im Modul per Klick umbenennen').count()
    const nameInputCount = await page.locator('input[aria-label="Name der Liste"]').count()
    out.push({
      step: 'S3 Vordefinierte Sets: statischer Titel + Hinweis, kein Namens-Input',
      hintCount, nameInputCount,
      pass: hintCount >= 2 && nameInputCount === 0,
    })
  }

  // ── EDITOR: Neue Werteliste anlegen (Punkt 7a) ─────────────────────────────
  {
    const addBtn = page.locator('button', { hasText: 'Neue Werteliste' }).first()
    const addBtnCount = await addBtn.count()
    if (addBtnCount) await addBtn.click()
    await wait(800)
    await shot('04-neue-werteliste.png')
    const nameInput = page.locator('input[aria-label="Name der Liste"]').first()
    const newInputCount = await page.locator('input[aria-label="Name der Liste"]').count()
    const newName = newInputCount ? await nameInput.inputValue() : ''
    out.push({
      step: 'S4 „Neue Werteliste" anlegbar, editierbarer Name',
      addBtnCount, newInputCount, newName,
      pass: addBtnCount > 0 && newInputCount === 1 && newName === 'Neue Werteliste',
    })
  }

  out.push({ step: 'S5 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk Editor-Feedback — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
