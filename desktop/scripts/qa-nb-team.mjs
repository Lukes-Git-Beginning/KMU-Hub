/**
 * QA — Nachbesserungs-Batch (team SortMenu + trainings render, admin integrations EmptyState).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/nb-team')
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
const rawKeys = (txt) => (txt.match(/\b(team|admin|shared)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '1-team-members.png') })

  // 1) SortMenu present in members toolbar
  const sortBtn = page.getByRole('button', { name: /Sortieren|Name/ })
  const hasSort = (await sortBtn.count()) > 0
  out.push({ step: 'team members SortMenu present', hasSort, pass: hasSort })
  if (hasSort) {
    await sortBtn.first().click()
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, '2-team-sortmenu.png') })
    const menuTxt = await page.evaluate(() => document.querySelector('[role="menu"],[role="listbox"]')?.textContent || document.body.innerText)
    out.push({ step: 'sort options', pass: /Abteilung/.test(menuTxt) && /Status/.test(menuTxt) })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  }

  // 2) Schulungen tab renders (EmptyState wrapper must not break the seeded table)
  const trainingsTab = page.getByRole('button', { name: /Schulungen/ })
  if ((await trainingsTab.count()) > 0) {
    await trainingsTab.first().click()
    await page.waitForTimeout(700)
    await page.screenshot({ path: resolve(outDir, '3-team-schulungen.png') })
    const txt = await page.evaluate(() => document.body.innerText)
    out.push({ step: 'schulungen tab renders', pass: /Schulungskatalog|Katalog|Teilnahme/.test(txt) && errs.length === 0 })
  }

  const leaks = rawKeys(await page.evaluate(() => document.body.innerText))
  out.push({ step: 'no raw i18n keys', leaks, pass: leaks.length === 0 })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 8), pass: errs.length === 0 })
await ctx.close(); await b.close()
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== NB-TEAM QA: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
