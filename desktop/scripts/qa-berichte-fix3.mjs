import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const outDir = resolve('.qa-screenshots/fix3-multicol'); await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const p = await ctx.newPage()
const errs = []; p.on('pageerror', e=>errs.push(String(e)))
await p.goto('http://localhost:5173/#/berichte', { waitUntil: 'domcontentloaded' })
await p.waitForTimeout(2500)
const tab = p.getByRole('button', { name: /^Berichte$/ }).first(); if (await tab.count()) await tab.click().catch(()=>{})
await p.waitForTimeout(1000)
await p.getByRole('button').filter({ hasText: /Helpdesk-Auslastung KW 24/ }).first().click().catch(()=>{})
await p.waitForTimeout(1000)
await p.getByRole('button', { name: /^Bearbeiten$/ }).first().click().catch(()=>{})
await p.waitForTimeout(600)
await p.getByRole('button', { name: /Block einfügen/ }).first().click().catch(()=>{})
await p.waitForTimeout(300)
await p.getByRole('button', { name: /^2 Spalten$/ }).first().click().catch(()=>{})
await p.waitForTimeout(400)
await p.getByRole('button').filter({ hasText: /^Block$/ }).nth(0).click().catch(()=>{}); await p.waitForTimeout(200)
await p.getByRole('button', { name: /^Überschrift$/ }).first().click().catch(()=>{}); await p.waitForTimeout(300)
await p.getByRole('button').filter({ hasText: /^Block$/ }).nth(1).click().catch(()=>{}); await p.waitForTimeout(200)
await p.getByRole('button', { name: /^Text$/ }).first().click().catch(()=>{}); await p.waitForTimeout(400)
// LEFT block of the 2-col row = the LAST 'Überschrift' input
const leftHeading = p.getByPlaceholder('Überschrift').last()
await leftHeading.fill('Linke Spalte').catch(()=>{})
await leftHeading.hover().catch(()=>{})
await p.waitForTimeout(400)
// hit-test exactly where the left block's delete sits (top-right of its wrapper)
const result = await p.evaluate(() => {
  const inputs = [...document.querySelectorAll('input[placeholder="Überschrift"]')]
  const last = inputs[inputs.length-1]
  const wrap = last && last.closest("[class~=\"group/block\"]")
  if (!wrap) return { found:false }
  const r = wrap.getBoundingClientRect()
  const x = r.right - 6, y = r.top + 6
  const el = document.elementFromPoint(x, y)
  return { found:true, onDelete: !!el?.closest('[aria-label="Block löschen"]'), tag: el?.tagName, cls: (el?.className||'').toString().slice(0,40) }
})
await p.screenshot({ path: resolve(outDir, 'fix3-2col-left-hover.png') })
console.log(JSON.stringify({ result, errs: errs.length }, null, 2))
await b.close()
