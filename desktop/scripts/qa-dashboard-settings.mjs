import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dashboard-settings')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const rawRe = /(dashboard\.settings\.|moduleSettings\.)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, rawRe.source)
}

// TextReveal renders the greeting as word spans (whitespace may be NBSP) —
// match with \s+ instead of a literal space.
async function greetingVisible(page) {
  return page.evaluate(() => /Guten\s+(Morgen|Tag|Abend)/.test(document.body.innerText))
}

const browser = await chromium.launch()
const out = []

// 1) From / → Dashboard entry preselected, scope groups + sections render
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    // TextReveal animates the greeting word by word — give it time to settle.
    await page.waitForTimeout(3200)
    const greetBefore = await greetingVisible(page)

    await page.locator('a:has-text("Modul-Einstellungen"), button:has-text("Modul-Einstellungen")').first().click({ timeout: 8000 })
    await page.waitForTimeout(800)
    const scopeGroups = await page.evaluate(() =>
      ['Persönlich', 'Für alle'].filter((g) => document.body.textContent.includes(g)))
    const sections = await page.evaluate(() =>
      ['Persönliche Ansicht', 'Team-Standard-Widgets', 'Erlaubte Widgets']
        .filter((s) => document.body.textContent.includes(s)))
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'panel-top.png') })

    await page.locator('text=Erlaubte Widgets').first().scrollIntoViewIfNeeded().catch(() => {})
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'panel-tenant.png') })
    out.push({ step: 'panel', greetBefore, scopeGroups, sections, rawKeys: rk, errs: errs.length })

    // 2) Toggle greeting off → reload → greeting gone
    const dlg = page.locator('[role="dialog"]').first()
    await dlg.locator('text=Begrüßung anzeigen').first().scrollIntoViewIfNeeded()
    await dlg.locator('label:has-text("Begrüßung anzeigen") input[type="checkbox"]').first().click({ timeout: 6000 })
    await page.waitForTimeout(400)
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.reload()
    await page.waitForTimeout(3200)
    const greetAfter = await greetingVisible(page)
    await page.screenshot({ path: resolve(outDir, 'dashboard-no-greeting.png') })
    out.push({ step: 'pref-applies', greetingBefore: greetBefore, greetingAfterDisable: greetAfter, errs: errs.length })
  } catch (e) {
    out.push({ step: 'panel', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

// 3) Narrow viewport @760
{
  const ctx = await browser.newContext({ viewport: { width: 760, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1600)
    await page.locator('button[aria-label="Navigation öffnen"]').first().click({ timeout: 8000 })
    await page.waitForTimeout(500)
    await page.locator('a:has-text("Modul-Einstellungen"), button:has-text("Modul-Einstellungen")').first().click({ timeout: 8000 })
    await page.waitForTimeout(800)
    const sections = await page.evaluate(() =>
      ['Persönliche Ansicht', 'Team-Standard-Widgets'].filter((s) => document.body.textContent.includes(s)))
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'panel-760.png') })
    out.push({ step: 'narrow-760', sections, rawKeys: rk, errs: errs.length })
  } catch (e) {
    out.push({ step: 'narrow-760', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
