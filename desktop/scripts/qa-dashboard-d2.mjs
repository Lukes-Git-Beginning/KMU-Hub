/**
 * QA script — Dashboard D-2: Demo-Tiefe / dead buttons
 *  1. MyTasks row → navigates to /work/my-tasks (was: dead role=button)
 *  2. Absences row → navigates to /team (was: dead cursor-pointer)
 *  3. ProfileWidgetSuggestions "+" → adds widget + toast + dismiss (was: no handler)
 *  4. Widget grid visual: Birthdays (MSW), CrossModule unread (real number)
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dashboard-d2')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const PROFILE = `try{localStorage.setItem('cosmi-profiles',JSON.stringify({state:{businessProfileId:'dienstleistung',devShowAllModules:true},version:2}))}catch(e){}`
const seed = (ids) => `try{localStorage.setItem('cosmi-dashboard',JSON.stringify({state:{scope:'personal',personalActiveWidgets:${JSON.stringify(ids)},personalLayouts:${JSON.stringify(ids.map((id,i)=>({i:id,x:0,y:i*4,w:6,h:4,minW:3,minH:3})))},teamActiveWidgets:[],teamLayouts:[]},version:2}))}catch(e){}`

const RAW_RE = /(dashboard\.[a-z]|widgets\.[a-z]|suggestions\.)/
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

async function navTest(label, widgetId, expectPath) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(seed([widgetId]))
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2500)
    await page.evaluate(() => { const el = document.querySelector('.layout'); if (el) el.scrollIntoView() })
    await page.waitForTimeout(500)
    await page.screenshot({ path: resolve(outDir, `${label}-widget.png`) })
    const row = page.locator('.layout [role="button"]').first()
    const hasRow = await row.isVisible().catch(() => false)
    let urlAfter = null
    if (hasRow) {
      await row.click({ timeout: 5000 })
      await page.waitForTimeout(700)
      urlAfter = page.url()
    }
    out.push({ scenario: label, hasRow, urlAfter, navOk: urlAfter ? urlAfter.includes(expectPath) : false, pageErrors: errs.length })
  } catch (e) {
    out.push({ scenario: label, error: String(e).split('\n')[0], pageErrors: errs.length })
  } finally { await ctx.close() }
}

await navTest('mytasks', 'my-tasks', '/work/my-tasks')
await navTest('absences', 'absences', '/team')

// Scenario 3: ProfileWidgetSuggestions "+" button
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(PROFILE)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2800)
    const suggHeader = await page.evaluate(() => document.body.textContent.includes('Empfohlene Widgets'))
    await page.screenshot({ path: resolve(outDir, '3-suggestions.png') })
    const plus = page.getByRole('button', { name: 'Zum Dashboard hinzufügen' }).first()
    const hasPlus = await plus.isVisible().catch(() => false)
    let toastShown = false, dismissed = false
    if (hasPlus) {
      const cardsBefore = await page.locator('[aria-label="Zum Dashboard hinzufügen"]').count()
      await plus.click({ timeout: 5000 })
      await page.waitForTimeout(1200)
      toastShown = await page.evaluate(() => /hinzugefügt/.test(document.body.textContent || ''))
      const cardsAfter = await page.locator('[aria-label="Zum Dashboard hinzufügen"]').count()
      dismissed = cardsAfter < cardsBefore
      await page.screenshot({ path: resolve(outDir, '4-after-plus.png') })
    }
    out.push({ scenario: 'suggestions-plus', suggHeader, hasPlus, toastShown, dismissed, rawKeys: await rawKeys(page), pageErrors: errs.length })
  } catch (e) {
    out.push({ scenario: 'suggestions-plus', error: String(e).split('\n')[0], pageErrors: errs.length })
  } finally { await ctx.close() }
}

// Scenario 4: widget grid visual — Birthdays (MSW) + CrossModule unread
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(seed(['birthdays', 'cross-module-overview', 'absences', 'my-tasks']))
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2800)
    await page.evaluate(() => { const el = document.querySelector('.layout'); if (el) el.scrollIntoView() })
    await page.waitForTimeout(800)
    const gridText = await page.evaluate(() => document.querySelector('.layout')?.textContent || '')
    await page.screenshot({ path: resolve(outDir, '5-widgets.png') })
    out.push({
      scenario: 'widgets-visual',
      birthdaysRendered: /Geburtstag|in \d+ Tag|heute/i.test(gridText),
      crossModuleRendered: /Heute im Überblick|Nachrichten|Aufgaben/i.test(gridText),
      rawKeys: await rawKeys(page), pageErrors: errs.length,
    })
  } catch (e) {
    out.push({ scenario: 'widgets-visual', error: String(e).split('\n')[0], pageErrors: errs.length })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
