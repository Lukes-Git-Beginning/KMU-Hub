import { chromium } from 'playwright'
import { resolve } from 'node:path'
const FE = process.env.QA_FE || 'http://localhost:5174'
const STUB=`const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};const fb=new Proxy(noop,h);const a={getStoredTokens:async()=>null,storeTokens:async()=>{},clearTokens:async()=>{}};window.electronAPI=new Proxy({},{get:(_t,p)=>(p==='auth'?a:p==='then'?undefined:fb[p])})`
const PREP=`try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'de'},version:0}))}catch(e){}`
const b=await chromium.launch();const c=await b.newContext({viewport:{width:1280,height:900},reducedMotion:'reduce'})
await c.addInitScript(STUB);await c.addInitScript(PREP);const pg=await c.newPage()
const errs=[];pg.on('pageerror',e=>errs.push(String(e)))
await pg.goto(`${FE}/#/`,{waitUntil:'domcontentloaded',timeout:30000});await pg.waitForTimeout(2500)
// Click bottom-left "Modul-Einstellungen" settings button
let opened=false
for (const re of [/Modul-Einstellung/i,/Einstellungen/i]) {
  const btn = pg.getByText(re).first()
  try { if (await btn.isVisible({timeout:800})) { await btn.click(); opened=true; break } } catch(e){}
}
await pg.waitForTimeout(1000)
// Click the "Branding" entry in the overlay if present
const hasBranding = await pg.evaluate(()=>!!([...document.querySelectorAll('*')].find(e=>e.children.length===0 && /^Branding$/.test((e.textContent||'').trim()))))
const clicked = await pg.evaluate(()=>{const el=[...document.querySelectorAll('button,a,[role="tab"],li')].find(e=>/^Branding$/.test((e.textContent||'').trim()));if(el){el.click();return true}return false})
await pg.waitForTimeout(900)
await pg.screenshot({path:resolve('.qa-admin-a5','overlay-branding.png')})
console.log(JSON.stringify({opened,hasBranding,clicked,pageerrors:errs.length}))
await b.close()
