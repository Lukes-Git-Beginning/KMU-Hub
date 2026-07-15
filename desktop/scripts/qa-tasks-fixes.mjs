/**
 * QA — task UX fixes (Darien review):
 *  A1 bigger complete affordance in My Tasks
 *  A2 back arrow in task detail goes to the project board (not My Tasks)
 *  A3 project board task click opens the full task page (not the mini modal)
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/tasks-fixes')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

try {
  // ── A1 · My Tasks + bigger complete button ─────────────────────────
  await page.goto(`${BASE}/#/work/my-tasks`, { waitUntil: 'domcontentloaded' })
  await page.getByRole('button', { name: /Abschließen/ }).first().waitFor({ state: 'visible', timeout: 15000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, 'a1-my-tasks.png') })
  const box = await page.getByRole('button', { name: /Abschließen/ }).first().boundingBox()
  out.push({ step: 'A1 complete-button-size', height: box?.height, width: box?.width, pass: !!box && box.height >= 28 })

  // Open a task → full detail page
  await page.getByText('Plugin-API v2 Architektur').first().click()
  await page.waitForTimeout(1500)
  const detailUrl = page.url()
  await page.screenshot({ path: resolve(outDir, 'a2-detail.png') })
  out.push({ step: 'open-task→full-detail', url: detailUrl.split('#')[1], pass: /\/tasks\//.test(detailUrl) })

  // ── A2 · Back arrow → project board (not My Tasks) ─────────────────
  await page.getByRole('button', { name: 'Zurück' }).first().click()
  await page.waitForTimeout(1200)
  const backUrl = page.url()
  await page.screenshot({ path: resolve(outDir, 'a3-after-back.png') })
  out.push({
    step: 'A2 back→project-board',
    url: backUrl.split('#')[1],
    pass: /\/work\/projects\//.test(backUrl) && !/\/tasks\//.test(backUrl) && !/my-tasks/.test(backUrl),
  })

  // ── A3 · Board task card → full task page (not mini modal) ─────────
  // We are on the project board now; click a task card by its title.
  await page.waitForTimeout(500)
  const cardClicked = await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('*')).find(
      (n) => /Dashboard Widgets|Plugin-API v2|CI\/CD Pipeline/.test(n.textContent || '') && n.childElementCount <= 3,
    )
    if (el) { (el.closest('[role="button"], button, [data-card]') || el).dispatchEvent(new MouseEvent('mousedown', { bubbles: true })) }
    return !!el
  })
  // Prefer a real click on a card title
  const cardTitle = page.getByText(/Dashboard Widgets implementieren/).first()
  if (await cardTitle.count()) await cardTitle.click({ force: true })
  await page.waitForTimeout(1500)
  const boardClickUrl = page.url()
  await page.screenshot({ path: resolve(outDir, 'a4-board-click.png') })
  out.push({ step: 'A3 board-card→full-page', cardClicked, url: boardClickUrl.split('#')[1], pass: /\/tasks\//.test(boardClickUrl) })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 8), pass: errs.length === 0 })
await ctx.close(); await b.close()
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== TASKS-FIXES QA: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
