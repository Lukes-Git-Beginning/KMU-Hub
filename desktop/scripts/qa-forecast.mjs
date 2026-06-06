import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/phase3-qa')
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser=await chromium.launch(); const out=[]
for (const w of [1440, 720]) {
  const ctx=await browser.newContext({viewport:{width:w,height:900}})
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page=await ctx.newPage(); const errs=[]; page.on('pageerror',e=>errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/kontakte/pipeline`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
    // 3rd toggle button = forecast
    await page.locator('div.rounded-md.border > button').nth(2).click(); await page.waitForTimeout(1200)
    const raw = await page.evaluate(()=>Array.from(document.querySelectorAll('*')).filter(n=>/\{?crm\.deals/.test(n.textContent||'')).map(n=>n.textContent.trim()))
    const hasWeighted = await page.getByText(/Gewichtete Prognose/).count()
    await page.screenshot({path:resolve(outDir,`forecast-${w}.png`)})
    out.push({w, rawKeys: raw.slice(0,3), hasWeighted, errs:errs.length})
  } catch(e){ out.push({w, error:String(e).split('\n')[0]}) } finally { await ctx.close() }
}
await browser.close(); console.log(JSON.stringify(out,null,2))
