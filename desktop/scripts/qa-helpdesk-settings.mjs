/**
 * QA — helpdesk H-6 settings panel against :5174.
 *  A) Open the Module-Settings overlay on /helpdesk → Helpdesk panel renders
 *     with personal + Geschäftszeiten + Routing-Regeln sections.
 *  B) Personal pref consumption: seed startTab='statistik' → helpdesk opens on
 *     the Statistik tab.
 * Screenshots: .qa-screenshots/helpdesk-settings/
 *
 * Usage: node scripts/qa-helpdesk-settings.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots', 'helpdesk-settings')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const WIPE = `try{localStorage.removeItem('cosmi-helpdesk')}catch(e){}`
const PREF_STATS = `try{localStorage.setItem('cosmi-helpdesk-prefs',JSON.stringify({state:{startTab:'statistik',defaultStatusFilter:'all'},version:0}))}catch(e){}`

const scan = (page) => page.evaluate(() => {
  const all = Array.from(document.querySelectorAll('body *')).filter((n) => n.children.length === 0)
    .map((n) => (n.textContent || '').trim()).filter(Boolean)
  return {
    rawKeys: all.filter((t) => /^(helpdesk|common|shared|moduleSettings|settings)\.[a-zA-Z]/.test(t)).slice(0, 12),
    doubleBrace: all.filter((t) => /\{\{|\}\}/.test(t)).slice(0, 12),
  }
})

const out = []
const browser = await chromium.launch()

// ── A) settings panel render ──────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(WIPE)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3500)
    // open module-settings overlay via sidebar nav item
    await page.getByText('Modul-Einstellungen', { exact: true }).first().click()
    await page.waitForTimeout(1200)
    const body = await page.evaluate(() => document.body.textContent || '')
    await page.screenshot({ path: resolve(outDir, '0-settings-panel.png'), fullPage: true })
    out.push({
      step: 'panel',
      helpdeskTitle: /Helpdesk-Einstellungen/.test(body),
      hasBusinessHours: /Geschäftszeiten/.test(body),
      hasRouting: /Routing-Regeln/.test(body),
      hasPersonal: /Start-Tab/.test(body),
      ...(await scan(page)),
      pageErrors: errs.slice(0, 4),
    })
  } catch (e) { out.push({ step: 'panel', error: String(e).split('\n')[0], pageErrors: errs.slice(0, 4) }) }
  finally { await ctx.close() }
}

// ── B) personal pref consumption ──────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(WIPE); await ctx.addInitScript(PREF_STATS)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3500)
    // Statistik tab active → "Tickets pro Tag" chart heading is visible
    const statsVisible = await page.getByText('Tickets pro Tag').count()
    const activeTab = await page.evaluate(() => {
      const b = Array.from(document.querySelectorAll('button')).find((el) => el.className.includes('tab-accent-active'))
      return b ? (b.textContent || '').trim() : ''
    })
    await page.screenshot({ path: resolve(outDir, '1-pref-stats-tab.png') })
    out.push({ step: 'pref-consume', statsHeadingVisible: statsVisible, activeTab, pageErrors: errs.slice(0, 4) })
  } catch (e) { out.push({ step: 'pref-consume', error: String(e).split('\n')[0], pageErrors: errs.slice(0, 4) }) }
  finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
