import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/phase5-qa'); await mkdir(outDir,{recursive:true})
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const rawRe=/crm\.activities\.(agenda|bucket|toast|view)/
const browser=await chromium.launch(); const out=[]
async function open(page){await page.goto(`${BASE}/#/kontakte/aktivitaeten`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)}
async function toAgenda(page){await page.locator('div.rounded-md.border > button').nth(1).click(); await page.waitForTimeout(1000)}
async function rawKeys(page){return page.evaluate((re)=>{const rx=new RegExp(re);return Array.from(document.querySelectorAll('body *')).filter(n=>n.children.length===0&&rx.test(n.textContent||'')).map(n=>n.textContent.trim()).slice(0,8)},rawRe.source)}

// 1) list view (completed badge works) + 2) agenda render
for (const w of [1440, 600]) {
  const ctx=await browser.newContext({viewport:{width:w,height:900}}); await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page=await ctx.newPage(); const errs=[]; page.on('pageerror',e=>errs.push(String(e)))
  try {
    await open(page)
    const completedBadges = await page.getByText('Erledigt',{exact:true}).count()
    await toAgenda(page)
    const buckets = await page.evaluate(()=>['Überfällig','Heute','Diese Woche','Später','Ohne Termin'].filter(b=>document.body.textContent.includes(b)))
    const rk = await rawKeys(page)
    await page.screenshot({path:resolve(outDir,`agenda-${w}.png`)})
    out.push({step:`view-${w}`, completedBadgesInList:completedBadges, agendaBuckets:buckets, rawKeys:rk, errs:errs.length})
  } catch(e){ out.push({step:`view-${w}`, error:String(e).split('\n')[0]}) } finally { await ctx.close() }
}

// 3) complete in agenda → item leaves
{
  const ctx=await browser.newContext({viewport:{width:1440,height:900}}); await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page=await ctx.newPage(); const errs=[]; page.on('pageerror',e=>errs.push(String(e)))
  try {
    await open(page); await toAgenda(page)
    const before = await page.locator('section .rounded-lg.border').count()
    await page.locator('section button[title]').first().click(); await page.waitForTimeout(1000)
    const toast = await page.evaluate(()=>{const el=Array.from(document.querySelectorAll('*')).find(n=>/erledigt markiert/i.test(n.textContent||'')&&(n.textContent||'').length<40);return el?el.textContent.trim():null})
    const after = await page.locator('section .rounded-lg.border').count()
    out.push({step:'complete', before, after, toast, errs:errs.length})
  } catch(e){ out.push({step:'complete', error:String(e).split('\n')[0]}) } finally { await ctx.close() }
}
await browser.close(); console.log(JSON.stringify(out,null,2))
