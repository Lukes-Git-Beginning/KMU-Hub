/**
 * QA — Helpdesk Statistik-Dimension (Editor-Feedback Q2/B/Q3, 2026-07-26).
 *
 *   S1 Live-Modul Statistik-Tab: CSAT-Kachel + CSAT-Chart AUSGEBLENDET (kein
 *      echtes Feature), andere Widgets sichtbar (Q3 + Konsum).
 *   S2 Editor: neue „Statistik"-Sektion, Widget-Toggles + CSAT gesperrt (Q2).
 *   S3 Editor: ein Widget („Nach Priorität") ausschalten wirkt (Toggle-Zustand).
 *   S4 Sandbox: nach dem Ausschalten ist „Nach Priorität" im Modul weg (Live).
 *   S5 Keine Page-Errors.
 * Screenshots → .qa-screenshots/editor-helpdesk-statistik/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-statistik')
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
  // ── LIVE MODUL: CSAT ausgeblendet, andere Widgets da (Q3 + Konsum) ──────────
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await wait(2800)
  await waitForText('Zutrittskarte', 15000)
  await page.locator('button', { hasText: 'Statistik' }).first().click()
  await wait(1200)
  await shot('01-live-stats.png')
  {
    const txt = await bodyText()
    const csatGone = !txt.includes('Kundenzufriedenheit')
    const hasOpen = txt.includes('Offene Tickets')
    const hasPerDay = txt.includes('Tickets pro Tag')
    const hasByStatus = txt.includes('Nach Status')
    out.push({
      step: 'S1 Live: CSAT weg, andere Widgets da',
      csatGone, hasOpen, hasPerDay, hasByStatus,
      pass: csatGone && hasOpen && hasPerDay && hasByStatus,
    })
  }

  // ── EDITOR ──────────────────────────────────────────────────────────────
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2800)

  // Sandbox → Statistik-Tab. Before the nav badge appears, both "Statistik"
  // buttons are exact-text: [0] = trio-nav section, [1] = module tab.
  const statBtns = page.getByRole('button', { name: 'Statistik', exact: true })
  const statBtnCount = await statBtns.count()
  if (statBtnCount > 1) await statBtns.nth(1).click()
  await wait(900)
  await shot('02-sandbox-stats.png')
  {
    const txt = await bodyText()
    const csatGone = !txt.includes('Kundenzufriedenheit')
    out.push({ step: 'S2 Sandbox Statistik-Tab: CSAT ausgeblendet', statBtnCount, csatGone, pass: statBtnCount > 1 && csatGone })
  }

  // Open the Statistik editor section (trio-nav [0]).
  await statBtns.nth(0).click()
  await wait(900)
  await shot('03-editor-statistik-panel.png')
  {
    const txt = await bodyText()
    const csatLocked = txt.includes('Braucht das Kundenzufriedenheits-Feature')
    const hasToggles = txt.includes('Offene Tickets') && txt.includes('Nach Priorität')
    out.push({ step: 'S3 Editor: Statistik-Sektion, Toggles + CSAT gesperrt', csatLocked, hasToggles, pass: csatLocked && hasToggles })
  }

  // Toggle "Nach Priorität" off → its chart disappears from the sandbox module.
  // "Nach Priorität" appears twice now (sandbox chart title + panel toggle label);
  // after hiding, the chart is gone → only the panel label remains.
  const prioBefore = await page.getByText('Nach Priorität', { exact: true }).count()
  await page.locator('button[aria-label="Nach Priorität ausblenden"]').first().click()
  await wait(900)
  await shot('04-priority-hidden-live.png')
  const prioAfter = await page.getByText('Nach Priorität', { exact: true }).count()
  out.push({
    step: 'S4 Toggle blendet „Nach Priorität" live im Modul aus',
    prioBefore, prioAfter,
    pass: prioBefore === 2 && prioAfter === 1,
  })

  out.push({ step: 'S5 Keine Page-Errors', errCount: errs.length, pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk Statistik-Dimension — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
