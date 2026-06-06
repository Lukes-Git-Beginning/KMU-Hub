import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/fixes-qa'); await mkdir(outDir,{recursive:true})
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW=/\b(kontakte|crm)\.[a-z]+\.[a-z._]+/i
const browser=await chromium.launch(); const out=[]
async function ctxPage(w=1440){const ctx=await browser.newContext({viewport:{width:w,height:900}});await ctx.addInitScript(STUB);await ctx.addInitScript(ONB);const page=await ctx.newPage();const errs=[];page.on('pageerror',e=>errs.push(String(e)));return {ctx,page,errs}}
async function rawKeys(page){return page.evaluate((re)=>{const rx=new RegExp(re);return [...new Set(Array.from(document.querySelectorAll('body *')).filter(n=>n.children.length===0&&rx.test(n.textContent||'')).map(n=>n.textContent.trim()))].slice(0,10)},RAW.source)}

// 1) Kontakte toolbar full width + sort dropdown
{ const {ctx,page,errs}=await ctxPage(1440)
  await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  const rk = await rawKeys(page)
  const hasListLabel = await page.getByText('Liste',{exact:true}).count()
  const hasRaster = await page.getByText('Raster',{exact:true}).count()
  const hasNeu = await page.getByRole('button',{name:/Neu/}).count()
  await page.screenshot({path:resolve(outDir,'kontakte-toolbar.png')})
  // open sort dropdown
  await page.getByRole('button',{name:/Sortier|Name/}).first().click().catch(()=>{}); await page.waitForTimeout(500)
  const sortOpts = await page.getByRole('menuitemradio').allInnerTexts().catch(()=>[])
  await page.screenshot({path:resolve(outDir,'kontakte-sortmenu.png')})
  out.push({step:'kontakte-toolbar', rawKeys:rk, hasListLabel, hasRaster, hasNeu, sortOpts, errs:errs.length}); await ctx.close() }

// 2) Group assign dialog via action menu
{ const {ctx,page,errs}=await ctxPage(1440)
  await page.goto(`${BASE}/#/kontakte`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  await page.locator('div.group, button.group').first().hover().catch(()=>{})
  await page.locator('button:has(svg.lucide-ellipsis-vertical), button:has(svg.lucide-more-vertical)').first().click().catch(()=>{}); await page.waitForTimeout(400)
  const assign = page.getByText('Zu Gruppe hinzufügen').first()
  let dlg=0, groupsShown=[]
  if(await assign.count()){ await assign.click(); await page.waitForTimeout(500); dlg=await page.getByText(/Gruppen von/).count(); groupsShown=await page.locator('[role="dialog"] label').allInnerTexts().catch(()=>[]); await page.screenshot({path:resolve(outDir,'group-assign.png')}) }
  out.push({step:'group-assign', dialogOpen:dlg, groups:groupsShown.slice(0,6), errs:errs.length}); await ctx.close() }

// 3) Activities sort labels (full width)
{ const {ctx,page,errs}=await ctxPage(1440)
  await page.goto(`${BASE}/#/kontakte/aktivitaeten`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
  const rk = await rawKeys(page)
  await page.screenshot({path:resolve(outDir,'activities-sort.png')})
  out.push({step:'activities', rawKeys:rk, errs:errs.length}); await ctx.close() }

await browser.close(); console.log(JSON.stringify(out,null,2))
