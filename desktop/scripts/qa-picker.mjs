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
await page.getByRole('button',{name:/Neuer Deal/}).first().click(); await page.waitForTimeout(800)
// contact field — label "Kontakt"/"Ansprechpartner". Type a partial name.
const contactInput = page.locator('input[placeholder="Thomas Weber"]')
await contactInput.fill('Mar'); await page.waitForTimeout(500)
const suggestions = await page.locator('ul li button').allInnerTexts().catch(()=>[])
await page.screenshot({path:resolve(outDir,'deal-form-picker.png')})
// pick first suggestion
let picked=null
if (suggestions.length){ await page.locator('ul li button').first().click(); await page.waitForTimeout(300); picked=await contactInput.inputValue() }
console.log(JSON.stringify({suggestions, picked, errs:errs.length},null,2))
await ctx.close(); await browser.close()
