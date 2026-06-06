import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE='http://localhost:5173'; const outDir=resolve('.qa-screenshots/phase4-qa')
import { mkdir } from 'node:fs/promises'; await mkdir(outDir,{recursive:true})
const STUB=`const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser=await chromium.launch(); const out=[]
const rawKeyRe=/leads\.[a-z]|crm\.nav\.leads/
async function rawKeys(page){return page.evaluate((re)=>{const rx=new RegExp(re);return Array.from(document.querySelectorAll('body *')).filter(n=>n.children.length===0&&rx.test(n.textContent||'')).map(n=>n.textContent.trim()).slice(0,8)},rawKeyRe.source)}

// 1) render inbox @ full + small
for (const w of [1440, 520]) {
  const ctx=await browser.newContext({viewport:{width:w,height:900}})
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page=await ctx.newPage(); const errs=[]; page.on('pageerror',e=>errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/kontakte/leads`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
    const rows = await page.locator('div.group').count()
    const rk = await rawKeys(page)
    await page.screenshot({path:resolve(outDir,`leads-inbox-${w}.png`)})
    out.push({step:`render-${w}`, rows, rawKeys:rk, errs:errs.length})
  } catch(e){ out.push({step:`render-${w}`, error:String(e).split('\n')[0]}) } finally { await ctx.close() }
}

// 2) filter + new dialog + convert dialog
{
  const ctx=await browser.newContext({viewport:{width:1440,height:900}})
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page=await ctx.newPage(); const errs=[]; page.on('pageerror',e=>errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/kontakte/leads`,{waitUntil:'domcontentloaded'}); await page.waitForTimeout(1800)
    // filter to "Neu"
    await page.getByRole('button',{name:/^Neu/}).first().click().catch(()=>{}); await page.waitForTimeout(400)
    const afterFilter = await page.locator('div.group').count()
    // open New lead
    await page.getByRole('button',{name:'Neuer Lead'}).first().click(); await page.waitForTimeout(600)
    const newDlg = await page.getByText('Auto-Score (Vorschau)').count()
    await page.screenshot({path:resolve(outDir,'leads-newform.png')})
    await page.keyboard.press('Escape'); await page.waitForTimeout(300)
    // open action menu on first lead → convert
    await page.locator('div.group button[aria-haspopup], div.group button').last().click().catch(()=>{})
    await page.waitForTimeout(500)
    const convertItem = page.getByText(/Qualifizieren/).first()
    let convertDlg=0
    if (await convertItem.count()){ await convertItem.click(); await page.waitForTimeout(600); convertDlg=await page.getByText('Lead umwandeln').count(); await page.screenshot({path:resolve(outDir,'leads-convert.png')}) }
    out.push({step:'interactions', afterFilterNeu:afterFilter, newDialog:newDlg, convertDialog:convertDlg, errs:errs.length})
  } catch(e){ out.push({step:'interactions', error:String(e).split('\n')[0]}) } finally { await ctx.close() }
}
await browser.close(); console.log(JSON.stringify(out,null,2))
