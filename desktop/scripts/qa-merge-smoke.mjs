import { chromium } from 'playwright'
const BASE='http://localhost:5173'
const STUB=`const n=()=>Promise.resolve();const h={get:(_,p)=>p==='then'?undefined:new Proxy(n,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(n,h)`
const ONB=`try{const k='cosmi-ui';const r=localStorage.getItem(k);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(k,JSON.stringify(p))}catch(e){}`
const b=await chromium.launch();const c=await b.newContext({viewport:{width:1440,height:950}})
await c.addInitScript(STUB);await c.addInitScript(ONB);const pg=await c.newPage()
const out={}
for(const [name,hash] of [['formulare','#/formulare'],['berichte','#/berichte']]){
  const errs=[];const failed=[]
  pg.on('pageerror',e=>errs.push(String(e)))
  pg.on('requestfailed',r=>{if(r.url().includes('/api/'))failed.push(r.url())})
  await pg.goto(`${BASE}/${hash}`,{waitUntil:'domcontentloaded',timeout:20000}).catch(e=>out[name+'_nav']=String(e))
  await pg.waitForTimeout(3500)
  const txt=await pg.evaluate(()=>document.body.innerText)
  const raw=[...new Set([...txt.matchAll(/\b(formulare|berichte|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map(m=>m[0]))]
  out[name]={loaded:txt.length>200, rawKeys:raw.length, pageErrors:errs.length, failedApi:failed.length}
  pg.removeAllListeners('pageerror');pg.removeAllListeners('requestfailed')
}
console.log(JSON.stringify(out,null,2))
await b.close()
