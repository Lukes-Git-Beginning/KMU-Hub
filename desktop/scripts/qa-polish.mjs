import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/polish-qa'); await mkdir(outDir,{recursive:true})
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW=/\b(kontakte|crm|common)\.[a-z]+\.[a-z._]+/i
const browser=await chromium.launch(); const out=[]
async function P(w=1440){const ctx=await browser.newContext({viewport:{width:w,height:900}});await ctx.addInitScript(STUB);await ctx.addInitScript(ONB);const page=await ctx.newPage();const errs=[];page.on('pageerror',e=>errs.push(String(e)));return {ctx,page,errs}}
async function raw(page){return page.evaluate((re)=>{const rx=new RegExp(re);return [...new Set(Array.from(document.querySelectorAll('body *')).filter(n=>n.children.length===0&&rx.test(n.textContent||'')).map(n=>n.textContent.trim()))].slice(0,8)},RAW.source)}

// 1) toolbar + list density + alpha dividers
{ const {ctx,page,errs}=await P(1440)
  await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  const emailsShown = await page.locator('button.group span.truncate').filter({hasText:'@'}).count()
  const letterHeaders = await page.evaluate(()=>Array.from(document.querySelectorAll('.sticky')).map(n=>n.textContent.trim()).filter(t=>t.length===1))
  const rk = await raw(page)
  await page.screenshot({path:resolve(outDir,'kontakte-full.png')})
  out.push({step:'list', emailsShown, letterHeaders:letterHeaders.slice(0,8), rawKeys:rk, errs:errs.length}); await ctx.close() }

// 2) sort menu open (field + direction)
{ const {ctx,page,errs}=await P(1440)
  await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  await page.locator('button:has(svg.lucide-arrow-up-down)').first().click(); await page.waitForTimeout(500)
  const radios = await page.getByRole('menuitemradio').allInnerTexts().catch(()=>[])
  await page.screenshot({path:resolve(outDir,'sortmenu.png')})
  out.push({step:'sortmenu', radios:radios.map(r=>r.trim()), errs:errs.length}); await ctx.close() }

// 3) contact detail modal header
{ const {ctx,page,errs}=await P(1440)
  await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  await page.getByText('Karl Bauer',{exact:false}).first().click(); await page.waitForTimeout(1000)
  const radixClose = await page.locator('[role="dialog"] >> text=Close').count()
  const dlgButtons = await page.locator('[role="dialog"] button').count()
  await page.screenshot({path:resolve(outDir,'modal.png')})
  out.push({step:'modal', radixCloseSrOnly:radixClose, dialogButtons:dlgButtons, errs:errs.length}); await ctx.close() }

// 4) group assign via action menu (real click)
{ const {ctx,page,errs}=await P(1440)
  await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  await page.locator('button:has(svg.lucide-ellipsis-vertical)').first().click(); await page.waitForTimeout(500)
  const items = await page.getByRole('menuitem').allInnerTexts().catch(()=>[])
  let dlg=0, groups=[]
  const a = page.getByRole('menuitem',{name:/Zu Gruppe/}).first()
  if(await a.count()){ await a.click(); await page.waitForTimeout(500); dlg=await page.getByText(/Gruppen von/).count(); groups=await page.locator('[role="dialog"] label').count(); await page.screenshot({path:resolve(outDir,'group-assign.png')}) }
  out.push({step:'group-assign', menuItems:items.map(i=>i.trim()), dialogOpen:dlg, groupRows:groups, errs:errs.length}); await ctx.close() }

await browser.close(); console.log(JSON.stringify(out,null,2))
