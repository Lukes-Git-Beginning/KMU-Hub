/**
 * QA script — Dashboard D-4: Team-Dashboard
 *  1. TeamWorktime now reads real per-member data via MSW (no client seeding):
 *     distinct names + distinct hour values, all consistent per employee.
 *  2. TeamStatus presence indicators render.
 *  3. ScopeToggle Personal → Team switches to team widgets.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dashboard-d4')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const seed = (ids) => `try{localStorage.setItem('cosmi-dashboard',JSON.stringify({state:{scope:'personal',personalActiveWidgets:${JSON.stringify(ids)},personalLayouts:${JSON.stringify(ids.map((id,i)=>({i:id,x:(i%2)*6,y:Math.floor(i/2)*4,w:6,h:4,minW:3,minH:3})))},teamActiveWidgets:[],teamLayouts:[]},version:2}))}catch(e){}`
const RAW_RE = /(dashboard\.[a-z]|widgets\.[a-z])/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim()).slice(0, 10)
  }, RAW_RE.source)
}

const browser = await chromium.launch()
const out = []

// Scenario 1: TeamWorktime real per-member data + TeamStatus presence
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(seed(['team-worktime', 'team-status']))
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2800)
    await page.evaluate(() => { const el = document.querySelector('.layout'); if (el) el.scrollIntoView() })
    await page.waitForTimeout(700)
    await page.screenshot({ path: resolve(outDir, '1-team-widgets.png') })
    const names = await page.$$eval('[data-testid="team-worktime-employee-name"]', (els) => els.map((e) => e.textContent.trim()))
    const hours = await page.evaluate(() => {
      const txt = document.querySelector('.layout')?.textContent || ''
      return [...txt.matchAll(/(\d+h(?:\s\d+m)?)/g)].map((m) => m[1])
    })
    const distinctHours = [...new Set(hours)]
    out.push({
      scenario: 'team-worktime',
      memberNames: names,
      distinctHourValues: distinctHours.length,
      sampleHours: distinctHours.slice(0, 6),
      rawKeys: await rawKeys(page),
      pageErrors: errs.length,
    })
  } catch (e) { out.push({ scenario: 'team-worktime', error: String(e).split('\n')[0], pageErrors: errs.length }) }
  finally { await ctx.close() }
}

// Scenario 2: ScopeToggle Personal → Team
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2500)
    const teamToggle = page.locator('[data-testid="scope-toggle-team"]').first()
    const hasToggle = await teamToggle.isVisible().catch(() => false)
    let teamHeadline = false
    if (hasToggle) {
      await teamToggle.click({ timeout: 5000 })
      await page.waitForTimeout(1200)
      teamHeadline = await page.evaluate(() => /Team/.test(document.querySelector('h1')?.textContent || ''))
      await page.evaluate(() => { const el = document.querySelector('.layout'); if (el) el.scrollIntoView() })
      await page.waitForTimeout(600)
      await page.screenshot({ path: resolve(outDir, '2-team-scope.png') })
    }
    out.push({ scenario: 'scope-toggle', hasToggle, teamHeadline, rawKeys: await rawKeys(page), pageErrors: errs.length })
  } catch (e) { out.push({ scenario: 'scope-toggle', error: String(e).split('\n')[0], pageErrors: errs.length }) }
  finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
