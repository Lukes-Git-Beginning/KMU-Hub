import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/fixes-qa')
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser=await chromium.launch(); const out=[]
async function ctxPage(){const ctx=await browser.newContext({viewport:{width:1440,height:900}});await ctx.addInitScript(STUB);await ctx.addInitScript(ONB);const page=await ctx.newPage();const errs=[];page.on('pageerror',e=>errs.push(String(e)));return {ctx,page,errs}}

// detect three-dot svg class + open menu
{ const {ctx,page,errs}=await ctxPage()
  await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  const svgClasses = await page.evaluate(()=>[...new Set(Array.from(document.querySelectorAll('svg')).map(s=>s.getAttribute('class')||'').filter(c=>/vertical|ellipsis|more/i.test(c)))])
  // click first three-dot button via DOM
  const opened = await page.evaluate(()=>{const svg=document.querySelector('svg.lucide-ellipsis-vertical, svg.lucide-more-vertical'); const btn=svg&&svg.closest('button'); if(btn){btn.click(); return true} return false})
  await page.waitForTimeout(500)
  const menuItems = await page.getByRole('menuitem').allInnerTexts().catch(()=>[])
  let dlg=0, groups=[]
  const assign = page.getByRole('menuitem',{name:/Zu Gruppe/}).first()
  if(await assign.count()){ await assign.click(); await page.waitForTimeout(500); dlg=await page.getByText(/Gruppen von/).count(); groups=await page.locator('[role="dialog"] label').allInnerTexts().catch(()=>[]); await page.screenshot({path:resolve(outDir,'group-assign.png')}) }
  out.push({step:'group-assign', svgClasses, menuOpened:opened, menuItems, dialogOpen:dlg, groups:groups.map(g=>g.replace(/\s+/g,' ').trim()).slice(0,6), errs:errs.length}); await ctx.close() }

// consent: click an individual row's Erteilen (exact), not "Alle erteilen"
{ const {ctx,page,errs}=await ctxPage()
  await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  await page.getByText('Karl Bauer',{exact:false}).first().click(); await page.waitForTimeout(1000)
  const rowGrant = page.getByRole('button',{name:'Erteilen',exact:true})
  const before = await rowGrant.count()
  await rowGrant.first().scrollIntoViewIfNeeded().catch(()=>{}); await page.waitForTimeout(200)
  await rowGrant.first().click(); await page.waitForTimeout(500)
  const confirmShown = await page.getByRole('button',{name:'Bestätigen',exact:true}).count()
  const after = await page.getByRole('button',{name:'Erteilen',exact:true}).count()
  await page.screenshot({path:resolve(outDir,'consent-grant.png')})
  out.push({step:'consent', erteilenBefore:before, confirmShown, erteilenAfter:after, errs:errs.length}); await ctx.close() }

await browser.close(); console.log(JSON.stringify(out,null,2))
