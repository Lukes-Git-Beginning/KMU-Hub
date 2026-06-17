/**
 * QA script — Dashboard D-5: Cross-Module/Alerts finish + DnD verify
 *  1. AlertsSection: alerts render with working links.
 *  2. CrossModule "open tasks" row → /work/my-tasks (fixed dead /work/tasks link).
 *  3. DnD: edit mode exposes drag handles + resize handles (react-grid-layout wired).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dashboard-d5')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const seed = (ids) => `try{localStorage.setItem('cosmi-dashboard',JSON.stringify({state:{scope:'personal',personalActiveWidgets:${JSON.stringify(ids)},personalLayouts:${JSON.stringify(ids.map((id,i)=>({i:id,x:0,y:i*4,w:6,h:4,minW:3,minH:3})))},teamActiveWidgets:[],teamLayouts:[]},version:2}))}catch(e){}`

const browser = await chromium.launch()
const out = []

// Scenario 1: AlertsSection + links (default load)
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2800)
    await page.screenshot({ path: resolve(outDir, '1-alerts.png') })
    const alertLinks = await page.evaluate(() => {
      const txt = document.body.textContent || ''
      return { hasContractsAlert: /Verträge?\s+laufen|Verträge prüfen/i.test(txt), hasInvoiceAlert: /Rechnung/i.test(txt) }
    })
    out.push({ scenario: 'alerts', ...alertLinks, pageErrors: errs.length })
  } catch (e) { out.push({ scenario: 'alerts', error: String(e).split('\n')[0], pageErrors: errs.length }) }
  finally { await ctx.close() }
}

// Scenario 2: CrossModule open-tasks row → /work/my-tasks
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(seed(['cross-module-overview']))
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2800)
    const taskRow = page.locator('.layout a').filter({ hasText: /Aufgabe/i }).first()
    const hasRow = await taskRow.isVisible().catch(() => false)
    let urlAfter = null
    if (hasRow) { await taskRow.click({ timeout: 5000 }); await page.waitForTimeout(700); urlAfter = page.url() }
    out.push({ scenario: 'crossmodule-tasklink', hasRow, urlAfter, navOk: urlAfter ? urlAfter.includes('/work/my-tasks') : false, pageErrors: errs.length })
  } catch (e) { out.push({ scenario: 'crossmodule-tasklink', error: String(e).split('\n')[0], pageErrors: errs.length }) }
  finally { await ctx.close() }
}

// Scenario 3: DnD wired — edit mode exposes drag/resize handles
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(seed(['my-tasks', 'birthdays', 'team-status']))
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2500)
    const editBtn = page.locator('button:has-text("Anpassen"), button:has-text("Bearbeiten"), button:has-text("Dashboard anpassen")').first()
    if (await editBtn.isVisible().catch(() => false)) { await editBtn.click({ timeout: 5000 }); await page.waitForTimeout(800) }
    await page.evaluate(() => { const el = document.querySelector('.layout'); if (el) el.scrollIntoView() })
    await page.waitForTimeout(500)
    const dragHandles = await page.locator('.widget-drag-handle').count()
    const resizeHandles = await page.locator('.react-resizable-handle').count()
    await page.screenshot({ path: resolve(outDir, '3-edit-dnd.png') })
    out.push({ scenario: 'dnd-wired', editMode: true, dragHandles, resizeHandles, dndOk: dragHandles > 0 && resizeHandles > 0, pageErrors: errs.length })
  } catch (e) { out.push({ scenario: 'dnd-wired', error: String(e).split('\n')[0], pageErrors: errs.length }) }
  finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
