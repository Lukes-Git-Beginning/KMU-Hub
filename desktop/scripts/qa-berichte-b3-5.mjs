import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b3-5-print')
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

await page.getByRole('button').filter({ hasText: /Verkaufsbericht Q2/ }).first().click().catch(() => {})
await page.waitForTimeout(3000)

// Screen mode — charts still render with the document palette.
out.screen = {
  rawKeys: await page.evaluate(findRawKeys, RAW.source),
  errs: errs.length,
  charts: await page.locator('svg.recharts-surface').count(),
}
await page.locator('.report-page').nth(1).screenshot({ path: resolve(outDir, 'b3-5-screen-body.png') })

// Print emulation — sheet drops its shadow/min-height and goes full width.
await page.emulateMedia({ media: 'print' })
await page.waitForTimeout(400)
out.print = await page.evaluate(() => {
  const el = document.querySelector('.report-page')
  if (!el) return null
  const s = getComputedStyle(el)
  return { boxShadow: s.boxShadow, breakAfter: s.breakAfter || s.pageBreakAfter, minHeight: s.minHeight }
})

console.log(JSON.stringify(out, null, 2))
await browser.close()
