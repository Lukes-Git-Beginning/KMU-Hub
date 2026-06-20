// QA — wiki W-5: module settings panel + personal pref applies (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/wiki')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const scanRaw = (page) =>
  page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *')).filter((n) => n.children.length === 0).map((n) => (n.textContent || '').trim()).filter(Boolean)
    return {
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared|moduleSettings|settings)\.[a-zA-Z]/.test(t)))].slice(0, 15),
      replacementChar: [...new Set(all.filter((t) => /�/.test(t)))].slice(0, 8),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) Open module settings (preselects Wiki on /wiki) ──
  await page.getByText('Modul-Einstellungen', { exact: true }).first().click()
  await page.waitForTimeout(1500)
  let body = await bodyText()
  out.settingsTitle = /Wiki-Einstellungen/.test(body)
  out.hasPersonal = /Standard-Ansicht/.test(body) && /Lesebreite/.test(body)
  out.hasTenant = /Freigabe & Sichtbarkeit|Standard-Freigabe/.test(body)
  out.hasRbacNote = /serverseitig durchgesetzt/.test(body)
  Object.assign(out, await scanRaw(page))
  await page.screenshot({ path: resolve(outDir, 'w5-1-settings.png'), fullPage: true })

  // ── B) Set a personal pref: switch to flat list view ──
  await page.getByRole('button', { name: 'Liste', exact: true }).first().click()
  await page.waitForTimeout(500)
  // close the settings overlay
  await page.keyboard.press('Escape')
  await page.waitForTimeout(800)
  body = await bodyText()
  // in flat view the sidebar category tree is hidden
  out.flatHidesTree = !/Prozesse & Workflows/.test(body)
  await page.screenshot({ path: resolve(outDir, 'w5-2-flat-applied.png'), fullPage: false })

  // ── C) Persistence: reload and confirm the pref still applies ──
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  body = await bodyText()
  out.flatPersisted = !/Prozesse & Workflows/.test(body)
  await page.screenshot({ path: resolve(outDir, 'w5-3-after-reload.png'), fullPage: false })

  // restore tree view for cleanliness
  await page.getByText('Modul-Einstellungen', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  await page.getByRole('button', { name: 'Baum', exact: true }).first().click()
  await page.waitForTimeout(400)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(600)
  out.treeRestored = /Prozesse & Workflows/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'w5-4-tree-restored.png'), fullPage: false })

  // ── D) Narrow width — open settings at full width, then shrink ──
  await page.getByText('Modul-Einstellungen', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  await page.setViewportSize({ width: 760, height: 900 })
  await page.waitForTimeout(800)
  out.narrowSettingsTitle = /Wiki-Einstellungen/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'w5-5-settings-narrow.png'), fullPage: true })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'w5-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
