/**
 * QA — Helpdesk-Pilot P2 (Editor-Pivot): Wertelisten-Konsum.
 *
 * Beweist: das Modul liest jetzt die Werteliste `ticket_priority` (statt fester
 * i18n-Enums) — Bearbeiten im Wertelisten-Panel wirkt LIVE in der Modul-Vorschau.
 *   S1 Helpdesk-Editor öffnet.
 *   S2 Wertelisten-Panel (links „Wertelisten") zeigt ticket_priority; low = „Rückfrage" (Tenant-Layer).
 *   S3 Option „Mittel" → „Standard" umbenennen → Tabellen-Chips im Modul werden LIVE „Standard".
 *   S4 Keine Page-Errors / rohen Keys.
 *
 * Screenshots → .qa-screenshots/editor-helpdesk-p2/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-p2')
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
  out.push({ step: 'S1 Helpdesk-Editor offen', pass: /Helpdesk/.test(await bodyText()) })

  // S2 — open the Wertelisten panel (left trio nav)
  const wlNav = page.locator('button, [role="button"]', { hasText: /^Wertelisten$/ }).first()
  await wlNav.click()
  await wait(1200)
  await shot('02-wertelisten-panel.png')
  {
    const txt = await bodyText()
    // Tenant layer renamed low → "Rückfrage"; the value-set editor must show it.
    out.push({ step: 'S2 Wertelisten-Panel zeigt ticket_priority (Tenant „Rückfrage")', hasRueckfrage: /Rückfrage/.test(txt), pass: /Rückfrage/.test(txt) })
  }

  // S3 — rename option "Mittel" → "Standard" and expect the module table chips to update live
  const boxes = page.getByRole('textbox')
  const n = await boxes.count()
  let medInput = null
  for (let i = 0; i < n; i++) {
    const el = boxes.nth(i)
    if ((await el.inputValue().catch(() => '')) === 'Mittel') { medInput = el; break }
  }
  const found = medInput !== null
  if (found) {
    await medInput.fill('Standard')
    await medInput.press('Tab')
  }
  await wait(1200)
  await shot('03-renamed-live.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'S3 Option „Mittel" → „Standard" wirkt LIVE im Modul (Chips + Filter)',
      inputFound: found,
      hasStandard: /Standard/.test(txt),
      dirty: /Änderung/.test(txt),
      pass: found && /Standard/.test(txt),
    })
  }

  const raw = /helpdesk\.(tabs|table|stats|ticket|priority)\.|customization\.editor\.[a-z]+\.[a-z]/i.test(await bodyText())
  out.push({ step: 'S4 Keine rohen i18n-Keys', rawKeysFound: raw, pass: !raw })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk-Pilot P2 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
