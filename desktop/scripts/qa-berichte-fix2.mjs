import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const outDir = resolve('.qa-screenshots/fix2-sticky'); await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const p = await ctx.newPage()
await p.goto('http://localhost:5173/#/berichte', { waitUntil: 'domcontentloaded' })
await p.waitForTimeout(2500)
const tab = p.getByRole('button', { name: /^Berichte$/ }).first(); if (await tab.count()) await tab.click().catch(()=>{})
await p.waitForTimeout(1000)
await p.getByRole('button').filter({ hasText: /Verkaufsbericht Q2/ }).first().click().catch(()=>{})
await p.waitForTimeout(2500)
const header = p.getByRole('button', { name: /Zurück zur Übersicht/ })
const before = await header.boundingBox()
// scroll the editor's internal scroll container (ancestor of .report-desk with overflow-y-auto)
await p.evaluate(() => {
  const desk = document.querySelector('.report-desk')
  let el = desk?.parentElement
  while (el && getComputedStyle(el).overflowY !== 'auto') el = el.parentElement
  if (el) el.scrollBy(0, 1500)
})
await p.waitForTimeout(500)
const after = await header.boundingBox()
await p.screenshot({ path: resolve(outDir, 'fix2-scrolled.png') })
console.log(JSON.stringify({ beforeY: before?.y, afterY: after?.y, pinned: before && after && Math.abs(before.y-after.y) < 4 }, null, 2))
await b.close()
