import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/module-settings')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const rawRe = /(moduleSettings\.)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, rawRe.source)
}

async function openSettingsOverlay(page) {
  // bottom-left sidebar "Einstellungen" button (sidebar layout)
  await page.locator('a:has-text("Einstellungen"), button:has-text("Einstellungen")').first().click({ timeout: 8000 })
  await page.waitForTimeout(700)
}

const browser = await chromium.launch()
const out = []

// 1) Open from Finance context → Finance preselected + "Aktiv" marker, route unchanged
for (const w of [1440, 820]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1600)
    const routeBefore = await page.evaluate(() => location.hash)
    await openSettingsOverlay(page)
    const groups = await page.evaluate(() =>
      ['Module', 'Cosmi (Allgemein)'].filter((g) => document.body.textContent.includes(g)))
    const hasActive = await page.evaluate(() => document.body.textContent.includes('Aktiv'))
    const dialogOpen = await page.locator('[role="dialog"]').count()
    const routeAfter = await page.evaluate(() => location.hash)
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, `overlay-finance-${w}.png`) })
    out.push({ step: `overlay-finance-${w}`, groups, activeBadge: hasActive, dialogOpen, routeStable: routeBefore === routeAfter, rawKeys: rk, errs: errs.length })
  } catch (e) {
    out.push({ step: `overlay-finance-${w}`, error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

// 2) Switch to Calendar entry inside overlay → scope sections render
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1500)
    await openSettingsOverlay(page)
    // click Calendar entry in the overlay left nav
    await page.locator('[role="dialog"] button:has-text("Kalender")').first().click({ timeout: 8000 })
    await page.waitForTimeout(700)
    const scopeBadges = await page.evaluate(() =>
      ['Persönlich', 'Für alle'].filter((b) => document.body.textContent.includes(b)))
    await page.screenshot({ path: resolve(outDir, `overlay-calendar.png`) })
    out.push({ step: 'overlay-calendar', scopeBadges, errs: errs.length })
    // close via Escape
    await page.keyboard.press('Escape')
    await page.waitForTimeout(400)
    const dialogAfter = await page.locator('[role="dialog"]').count()
    out.push({ step: 'overlay-close', dialogClosed: dialogAfter === 0 })
  } catch (e) {
    out.push({ step: 'overlay-calendar', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
