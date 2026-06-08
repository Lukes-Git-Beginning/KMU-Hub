import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const BASE='http://localhost:5173'
const outDir=resolve('.qa-screenshots')
const STUB=`const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUP=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
async function scanRawKeys(page){const t=await page.evaluate(()=>document.body.innerText);return [...new Set([...t.matchAll(/\b(team|common|shared|moduleSettings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map(m=>m[0]))]}
async function shot(page,n){await page.screenshot({path:resolve(outDir,n),fullPage:true})}
await mkdir(outDir,{recursive:true})
const b=await chromium.launch();const errors=[];const failedApi=[]
const ctx=await b.newContext({viewport:{width:1440,height:1000}})
await ctx.addInitScript(STUB);await ctx.addInitScript(SUP)
const page=await ctx.newPage()
page.on('pageerror',e=>errors.push(String(e).split('\n')[0]))
page.on('requestfailed',r=>{if(r.url().includes('/api/'))failedApi.push(r.url())})
const out={}
try{
  await page.goto(`${BASE}/#/team`,{waitUntil:'domcontentloaded',timeout:20000})
  await page.waitForTimeout(3000)
  // Click first member card heading "Unbekannt"
  await page.getByText('Unbekannt').first().click({timeout:5000}).catch(e=>{out.clickErr=String(e).split('\n')[0]})
  await page.waitForTimeout(1500)
  out.panelOpen = await page.getByText('Mitarbeiterprofil').first().isVisible().catch(()=>false)
  out.expandBtn = await page.getByRole('button',{name:'Erweitern'}).first().isVisible().catch(()=>false)
  await shot(page,'member-panel.png')
  await page.getByRole('button',{name:'Erweitern'}).first().click({timeout:4000}).catch(e=>{out.expandErr=String(e).split('\n')[0]})
  await page.waitForTimeout(1500)
  out.url=page.url()
  out.onMemberPage=page.url().includes('/team/member/')
  out.payrollVisible=await page.getByText(/Lohn-Stammdaten|Steuerklasse|Lohnart/).first().isVisible().catch(()=>false)
  out.backBtn=await page.getByRole('button',{name:/Zurück/}).first().isVisible().catch(()=>false)
  out.pageRawKeys=await scanRawKeys(page)
  await shot(page,'member-fullpage.png')
  // back to team
  await page.getByRole('button',{name:/Zurück/}).first().click({timeout:4000}).catch(()=>{})
  await page.waitForTimeout(1000)
  out.backWorks=page.url().endsWith('/team')||page.url().includes('/#/team')
}catch(err){out.error=String(err).split('\n')[0]}
out.pageErrors=[...new Set(errors)].slice(0,12)
out.failedApiRequests=[...new Set(failedApi)].slice(0,12)
await ctx.close();await b.close()
console.log(JSON.stringify(out,null,2))
