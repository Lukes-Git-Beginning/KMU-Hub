import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b3-2-paper')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(berichte|common|shared)\.[a-z]+\.[a-z._]+/i

function findRawKeys(re) {
  const rx = new RegExp(re, 'i')
  return [
    ...new Set(
      Array.from(document.querySelectorAll('body *'))
        .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
        .map((n) => n.textContent.trim()),
    ),
  ].slice(0, 12)
}

const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1200 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2200)
const tab = page.getByRole('button', { name: /^Berichte$/ }).first()
if (await tab.count()) await tab.click().catch(() => {})
await page.waitForTimeout(1000)

// Verkaufsbericht Q2 (released, multi-page: cover + pagebreak + header/footer).
await page.getByRole('button').filter({ hasText: /Verkaufsbericht Q2/ }).first().click().catch(() => {})
await page.waitForTimeout(3000) // chart fetch + render

out.report = {
  rawKeys: await page.evaluate(findRawKeys, RAW.source),
  errs: errs.length,
  pages: await page.locator('.report-page').count(),
  hasPageNumber: await page.getByText(/Seite 2 von 2/).count(),
  charts: await page.locator('svg.recharts-surface').count(),
}
// Element screenshots capture the full sheets even inside the inner scroll area.
const pages = page.locator('.report-page')
await pages.nth(0).screenshot({ path: resolve(outDir, 'b3-2-page1-cover.png') })
await pages.nth(1).screenshot({ path: resolve(outDir, 'b3-2-page2-body.png') })

console.log(JSON.stringify(out, null, 2))
await browser.close()
