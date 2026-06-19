// QA — berichte B-4: module-settings panel + schedule SortMenu (dev :5173)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/berichte')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const scanRaw = (page) =>
  page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
      .filter(Boolean)
    return {
      rawKeys: [...new Set(all.filter((t) => /^(berichte|moduleSettings|common|shared|settings)\.[a-zA-Z]/.test(t)))].slice(0, 12),
      doubleBrace: [...new Set(all.filter((t) => /\{\{|\}\}/.test(t)))].slice(0, 12),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

try {
  // ── A) Settings overlay on /berichte ──
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.getByText('Modul-Einstellungen', { exact: true }).first().click()
  await page.waitForTimeout(1300)
  const body = await page.evaluate(() => document.body.textContent || '')
  out.settingsTitle = /Berichte-Einstellungen/.test(body)
  out.hasPersonal = /Standard-Format/.test(body)
  out.hasTenant = /Erlaubte Export-Formate/.test(body)
  out.hasDomains = /E-Mail-Domains|zentria\.tech/.test(body)
  Object.assign(out, await scanRaw(page))
  await page.screenshot({ path: resolve(outDir, '5-settings.png'), fullPage: true })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)

  // ── B) SortMenu in schedules ──
  await page.getByRole('button', { name: /^Geplant/ }).first().click()
  await page.waitForTimeout(1000)
  const firstRow = () => page.locator('table tbody tr td:first-child').first().innerText()
  out.firstRowDefault = (await firstRow()).trim()
  out.sortMenuPresent = await page.getByRole('button', { name: /Sortieren/ }).count()
  // open sort menu, pick descending
  await page.getByRole('button', { name: /Sortieren/ }).first().click()
  await page.waitForTimeout(400)
  await page.getByRole('menuitemradio', { name: /Absteigend|Descending/ }).first().click()
  await page.waitForTimeout(700)
  out.firstRowDesc = (await firstRow()).trim()
  out.sortChanged = out.firstRowDefault !== out.firstRowDesc
  await page.screenshot({ path: resolve(outDir, '5b-sort.png'), fullPage: true })
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errs.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
