import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/phase6-qa'); await mkdir(outDir,{recursive:true})
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW=/\b(crm|kontakte|leads|common)\.[a-z]+\.[a-z._]+/i
const browser=await chromium.launch()
const out=[]
for (const w of [1440, 760]) {
  const ctx=await browser.newContext({viewport:{width:w,height:900}});await ctx.addInitScript(STUB);await ctx.addInitScript(ONB)
  const page=await ctx.newPage();const errs=[];page.on('pageerror',e=>errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/kontakte/auswertungen`,{waitUntil:'domcontentloaded'});await page.waitForTimeout(2200)
    const svgCount=await page.locator('svg.recharts-surface').count()
    const rk=await page.evaluate((re)=>{const rx=new RegExp(re);return [...new Set(Array.from(document.querySelectorAll('body *')).filter(n=>n.children.length===0&&rx.test(n.textContent||'')).map(n=>n.textContent.trim()))].slice(0,8)},RAW.source)
    // scrollbar hidden? check computed
    const sbHidden=await page.evaluate(()=>getComputedStyle(document.documentElement).scrollbarWidth)
    await page.screenshot({path:resolve(outDir,`reports-${w}.png`)})
    out.push({w, charts:svgCount, rawKeys:rk, scrollbarWidth:sbHidden, errs:errs.length})
  } catch(e){ out.push({w, error:String(e).split('\n')[0]}) } finally { await ctx.close() }
}
await browser.close(); console.log(JSON.stringify(out,null,2))
