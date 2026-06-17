/**
 * QA — helpdesk H-7 SortMenu + real-clock SLA against :5174.
 *  - SLA badges show a believable mix relative to today (overdue / soon / comfy)
 *    and NO frozen "122d" artifact.
 *  - Sorting by SLA asc vs desc reorders the list (most-overdue first vs last).
 * Screenshots: .qa-screenshots/helpdesk-sort/
 *
 * Usage: node scripts/qa-helpdesk-sort.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots', 'helpdesk-sort')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const WIPE = `try{localStorage.removeItem('cosmi-helpdesk')}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(WIPE)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

const firstNr = () => page.locator('tbody tr').first().locator('td').first().textContent()
const slaTexts = () => page.evaluate(() =>
  Array.from(document.querySelectorAll('tbody tr')).map((tr) => {
    const cells = tr.querySelectorAll('td')
    return (cells[6]?.textContent || '').trim() // SLA is the 7th column
  }))
async function sortBy(fieldLabel, dirLabel) {
  await page.getByRole('button', { name: 'Sortieren nach' }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: fieldLabel }).click()
  await page.waitForTimeout(300)
  await page.getByRole('button', { name: 'Sortieren nach' }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: dirLabel }).click()
  await page.waitForTimeout(400)
}

const out = []
try {
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  const sla = await slaTexts()
  await page.screenshot({ path: resolve(outDir, '0-sla-mix.png'), fullPage: true })
  out.push({
    step: 'sla-clock',
    overdue: sla.filter((s) => /überfällig/.test(s)).length,
    remaining: sla.filter((s) => /übrig/.test(s)).length,
    frozenArtifact: sla.filter((s) => /1\d\dd|[0-9]{3,}/.test(s)).length, // e.g. "122d"
    sample: sla.slice(0, 6),
  })

  // sort by SLA ascending → most overdue (earliest due) first
  await sortBy('SLA-Restzeit', 'Aufsteigend')
  const slaAsc = await slaTexts()
  const firstAsc = (await firstNr())?.trim()
  await page.screenshot({ path: resolve(outDir, '1-sla-asc.png'), fullPage: true })

  // sort by SLA descending
  await sortBy('SLA-Restzeit', 'Absteigend')
  const firstDesc = (await firstNr())?.trim()
  await page.screenshot({ path: resolve(outDir, '2-sla-desc.png'), fullPage: true })

  out.push({
    step: 'sort-sla',
    firstAscOverdue: /überfällig/.test(slaAsc[0] || ''),
    firstAsc, firstDesc,
    reordered: firstAsc !== firstDesc,
  })

  // sort by priority ascending
  await sortBy('Priorität', 'Absteigend')
  const prioFirst = await page.locator('tbody tr').first().textContent()
  out.push({ step: 'sort-priority', firstRowCritical: /Kritisch/.test(prioFirst || '') })

  out.push({ pageErrors: errs.slice(0, 6) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 6) })
} finally {
  await ctx.close(); await browser.close()
}

console.log(JSON.stringify(out, null, 2))
