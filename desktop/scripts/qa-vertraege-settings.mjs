import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/vertraege-settings')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const rawRe = /(vertraege\.settings\.|moduleSettings\.)/
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
  await page.locator('a:has-text("Einstellungen"), button:has-text("Einstellungen")').first().click({ timeout: 8000 })
  await page.waitForTimeout(800)
}

const browser = await chromium.launch()
const out = []

// 1) From /vertraege → Verträge entry preselected, both scope groups + sections render
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1600)
    await openSettingsOverlay(page)

    const scopeGroups = await page.evaluate(() =>
      ['Persönlich', 'Für alle'].filter((g) => document.body.textContent.includes(g)))
    const sections = await page.evaluate(() =>
      ['Persönliche Ansicht', 'Vertragsarten', 'Standard-Laufzeiten', 'Nummernkreis', 'Erinnerungs-Standard']
        .filter((s) => document.body.textContent.includes(s)))
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'panel-top.png') })
    out.push({ step: 'panel', scopeGroups, sections, rawKeys: rk, errs: errs.length })

    // scroll the lower tenant sections into view and capture them
    await page.locator('text=Erinnerungs-Standard').first().scrollIntoViewIfNeeded().catch(() => {})
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'panel-tenant.png') })

    // 2) Set personal pref defaultTab = Vorlagen → reload → templates tab active
    const dlg = page.locator('[role="dialog"]').first()
    await dlg.locator('button:has-text("Vorlagen")').first().click({ timeout: 6000 })
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'panel-pref-set.png') })

    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.reload()
    await page.waitForTimeout(1600)
    const prefApplies = await page.evaluate(() =>
      document.body.textContent.includes('Standard-Mietvertrag'))
    await page.screenshot({ path: resolve(outDir, 'module-default-tab.png') })
    out.push({ step: 'pref-applies', defaultTabVorlagen: prefApplies, errs: errs.length })
  } catch (e) {
    out.push({ step: 'panel', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

// 3) Narrow viewport @760 — panel still usable, no overflow break
{
  const ctx = await browser.newContext({ viewport: { width: 760, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1600)
    // Sidebar is off-canvas at this width — open it via the hamburger first.
    await page.locator('button[aria-label="Navigation öffnen"]').first().click({ timeout: 8000 })
    await page.waitForTimeout(500)
    await page.locator('a:has-text("Modul-Einstellungen"), button:has-text("Modul-Einstellungen")').first().click({ timeout: 8000 })
    await page.waitForTimeout(800)
    const sections = await page.evaluate(() =>
      ['Persönliche Ansicht', 'Vertragsarten'].filter((s) => document.body.textContent.includes(s)))
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'panel-760.png') })
    out.push({ step: 'narrow-760', sections, rawKeys: rk, errs: errs.length })
  } catch (e) {
    out.push({ step: 'narrow-760', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
