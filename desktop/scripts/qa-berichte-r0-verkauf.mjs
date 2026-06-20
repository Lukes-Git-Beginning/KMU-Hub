import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/r0-berichte')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2000)
const tab = page.getByRole('button', { name: /^Berichte$/ }).first()
if (await tab.count()) {
  await tab.click().catch(() => {})
  await page.waitForTimeout(1000)
}
const card = page
  .locator('[role="button"]')
  .filter({ hasText: /Verkaufsbericht Q2/ })
  .first()
await card.click().catch(() => {})
await page.waitForTimeout(1500)
await page.screenshot({ path: resolve(outDir, 'editor-verkauf-1440.png') })
console.log(JSON.stringify({ errs: errs.length }, null, 2))
await browser.close()
