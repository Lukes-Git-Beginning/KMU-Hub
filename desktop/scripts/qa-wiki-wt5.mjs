// QA — wiki WT-5: content-walk search + match highlight + ICU plural views (dev :5174)
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
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
      .filter(Boolean)
    return {
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 15),
      doubleBrace: [...new Set(all.filter((t) => /\{\{|\}\}/.test(t)))].slice(0, 10),
      icuLeak: [...new Set(all.filter((t) => /\{count|plural,/.test(t)))].slice(0, 10),
      replacementChar: [...new Set(all.filter((t) => /�/.test(t)))].slice(0, 10),
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

  // ── A) Search by a CONTENT-only term → highlight + snippet ──
  const searchBox = page.getByPlaceholder('Wiki durchsuchen...')
  await searchBox.fill('Wissensbasis')
  await page.waitForTimeout(1200)
  out.searchResultCount = await page.evaluate(() => {
    const m = (document.body.textContent || '').match(/(\d+)\s+Treffer/)
    return m ? Number(m[1]) : -1
  })
  out.hasMark = await page.locator('mark').count()
  out.markText = await page.locator('mark').first().textContent().catch(() => null)
  out.snippetVisible = /Wissensbasis/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'wt5-1-search-content.png'), fullPage: false })

  // ── B) Search by a TITLE term → title highlight ──
  await searchBox.fill('Onboarding')
  await page.waitForTimeout(1000)
  out.titleMarkCount = await page.locator('mark').count()
  await page.screenshot({ path: resolve(outDir, 'wt5-2-search-title.png'), fullPage: false })
  await searchBox.fill('')
  await page.waitForTimeout(500)

  // ── C) ICU plural views render cleanly (no raw ICU syntax) ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  const body = await bodyText()
  out.viewsRendered = /\d+\s+Aufrufe/.test(body)
  out.noIcuLeak = !/\{count|plural,/.test(body)
  Object.assign(out, await scanRaw(page))
  await page.screenshot({ path: resolve(outDir, 'wt5-3-views-plural.png'), fullPage: false })

  // ── D) Narrow ──
  await page.setViewportSize({ width: 820, height: 900 })
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, 'wt5-4-narrow.png'), fullPage: false })
  out.narrow = await scanRaw(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wt5-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
