import { chromium } from 'playwright'
const BASE='http://localhost:5173'
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser=await chromium.launch()
const ctx=await browser.newContext({viewport:{width:1440,height:900}})
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page=await ctx.newPage(); const errs=[]; page.on('pageerror',e=>errs.push(String(e)))
await page.goto(`${BASE}/#/kontakte/leads`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
const before = await page.locator('div.group').count()
// open last lead's action menu → Qualifizieren
await page.locator('div.group').first().locator('button').last().click(); await page.waitForTimeout(400)
await page.getByText(/Qualifizieren/).first().click(); await page.waitForTimeout(500)
await page.getByRole('button',{name:'Umwandeln'}).click(); await page.waitForTimeout(1000)
const toast = await page.evaluate(()=>{const el=Array.from(document.querySelectorAll('*')).find(n=>/umgewandelt/i.test(n.textContent||'')&&(n.textContent||'').length<50);return el?el.textContent.trim():null})
const after = await page.locator('div.group').count()
// qualified filter count
console.log(JSON.stringify({before, after, toast, errs:errs.length}))
await ctx.close(); await browser.close()
