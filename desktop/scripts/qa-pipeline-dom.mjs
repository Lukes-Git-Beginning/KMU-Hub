import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/phase3-qa')
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser=await chromium.launch()
const ctx=await browser.newContext({viewport:{width:1440,height:900}})
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page=await ctx.newPage(); const errs=[]; page.on('pageerror',e=>errs.push(String(e)))
await page.goto(`${BASE}/#/kontakte/pipeline`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
const tg=page.locator('div.rounded-md.border > button'); await tg.nth(1).click(); await page.waitForTimeout(1200)
const rotting = await page.evaluate(() => Array.from(document.querySelectorAll('*')).filter(n=>/inaktiv/.test(n.textContent||'')&&(n.textContent||'').length<20).map(n=>n.textContent.trim()))
// hover first card → check won/lost buttons appear
const card = page.locator('div[role="button"]').first()
await card.hover(); await page.waitForTimeout(400)
const wonVisible = await page.getByRole('button',{name:'Gewonnen'}).first().isVisible().catch(()=>false)
const lostVisible = await page.getByRole('button',{name:'Verloren'}).first().isVisible().catch(()=>false)
await page.screenshot({path:resolve(outDir,'kanban-hover.png')})
console.log(JSON.stringify({rottingBadges:rotting, rottingCount:rotting.length, wonVisible, lostVisible, errs:errs.length},null,2))
await ctx.close(); await browser.close()
