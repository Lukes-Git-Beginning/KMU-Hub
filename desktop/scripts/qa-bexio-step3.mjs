/**
 * QA — Bexio wizard step 3 (field mapping) layout fix.
 * Bug: grid-cols used commas (invalid CSS → columns collapsed → dropdowns
 * stacked) + dialog had no max-height (overflowed the screen). Verifies the
 * 5-column grid renders side-by-side and the dialog stays within the viewport
 * with the nav buttons visible.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/bexio-review')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`
const NOPOPUP = `window.open=()=>null`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 960 } })
for (const s of [STUB, ONB, NOLAUNCH, NOPOPUP]) await ctx.addInitScript(s)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

try {
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await page.getByText(/Modul-Einstellungen/).first().click({ timeout: 6000 })
  await page.waitForTimeout(1200)
  const card = page.getByRole('button').filter({ hasText: /^Bexio/ }).first()
  await card.scrollIntoViewIfNeeded().catch(() => {})
  await card.click({ timeout: 6000 })
  await page.waitForSelector('[role="dialog"]', { timeout: 8000 })
  await page.waitForTimeout(700)
  // connect
  const connect = page.getByRole('button', { name: /Mit Bexio verbinden|Verbinden/ }).first()
  if (await connect.count()) { await connect.click(); await page.waitForTimeout(2800) }
  // advance to step 3 (2× Weiter)
  for (let i = 0; i < 2; i++) {
    const next = page.getByRole('button', { name: /^Weiter/ }).first()
    if ((await next.count()) && (await next.isEnabled().catch(() => false))) { await next.click(); await page.waitForTimeout(800) }
  }
  await page.screenshot({ path: resolve(outDir, 'FIX-step3-mapping.png') })

  // Measure: dialog within viewport + grid rendered side-by-side (row wider than a single select)
  const m = await page.evaluate(() => {
    const dlg = document.querySelector('[role="dialog"]')
    if (!dlg) return { ok: false, reason: 'no dialog' }
    const r = dlg.getBoundingClientRect()
    const withinViewport = r.top >= -2 && r.bottom <= window.innerHeight + 2
    // find a mapping row: a grid element with >=3 selects
    const rows = [...dlg.querySelectorAll('div')].filter((d) => d.querySelectorAll(':scope > select').length >= 3)
    const firstRow = rows[0]
    let sideBySide = false
    if (firstRow) {
      const selects = [...firstRow.querySelectorAll(':scope > select')]
      // side-by-side ⇒ the 2nd/3rd selects sit to the RIGHT of the first (different x), not stacked below
      const tops = selects.map((s) => Math.round(s.getBoundingClientRect().top))
      sideBySide = new Set(tops).size === 1 // all selects on the same row (same top)
    }
    // nav buttons visible within viewport
    const btns = [...dlg.querySelectorAll('button')].filter((x) => /Weiter|Zurück/.test(x.textContent || ''))
    const navVisible = btns.some((x) => { const b = x.getBoundingClientRect(); return b.bottom <= window.innerHeight + 2 && b.top >= 0 })
    return { ok: true, withinViewport, dialogH: Math.round(r.height), viewportH: window.innerHeight, rowsFound: rows.length, sideBySide, navVisible }
  })
  out.push({ step: 'step3 layout', ...m, pass: m.ok && m.withinViewport && m.sideBySide && m.navVisible })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 8), pass: errs.length === 0 })
await ctx.close(); await b.close()
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== BEXIO STEP-3 FIX QA: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
