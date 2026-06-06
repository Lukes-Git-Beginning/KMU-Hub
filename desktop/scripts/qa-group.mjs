import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/final-qa')
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser=await chromium.launch()
const ctx=await browser.newContext({viewport:{width:1440,height:900}});await ctx.addInitScript(STUB);await ctx.addInitScript(ONB)
const page=await ctx.newPage();const errs=[];page.on('pageerror',e=>errs.push(String(e)))
await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'});await page.waitForTimeout(1700)
const row=page.locator('div[role="button"]').filter({hasText:'Karl Bauer'}).first()
await row.hover();await page.waitForTimeout(300)
await row.locator('button[aria-haspopup="menu"]:visible').first().click();await page.waitForTimeout(500)
const items=await page.getByRole('menuitem').allInnerTexts().catch(()=>[])
let dlg=0,rows=[]
const a=page.getByRole('menuitem',{name:/Zu Gruppe/}).first()
if(await a.count()){await a.click();await page.waitForTimeout(500);dlg=await page.getByText(/Gruppen von/).count();rows=await page.locator('[role="dialog"] label').allInnerTexts().catch(()=>[]);await page.screenshot({path:resolve(outDir,'group-assign.png')})}
console.log(JSON.stringify({menuItems:items.map(i=>i.trim()),dialogOpen:dlg,groupRows:rows.map(r=>r.replace(/\s+/g,' ').trim()),errs:errs.length},null,2))
await ctx.close();await browser.close()
