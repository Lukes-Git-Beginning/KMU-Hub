// Capture the inactive (dimmed/dashed) module cards by scrolling the license tab.
import { chromium } from 'playwright'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5174'
const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};const fb=new Proxy(noop,h);const a={getStoredTokens:async()=>null,storeTokens:async()=>{},clearTokens:async()=>{}};window.electronAPI=new Proxy({},{get:(_t,p)=>(p==='auth'?a:p==='then'?undefined:fb[p])})`
const PREP = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'de'},version:0}))}catch(e){}`

try {
  const browser = await chromium.launch()
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 }, reducedMotion: 'reduce' })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(PREP)
  const page = await ctx.newPage()
  await page.goto(`${FE}/#/admin/license`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(2800)
  // Scroll the inactive "Werkzeuge"/Tools group into view.
  await page.getByText('Werkzeuge', { exact: true }).scrollIntoViewIfNeeded().catch(() => {})
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve('.qa-admin-a3', 'de-03-inactive.png') })
  await browser.close()
  console.log('ok')
} catch (e) {
  console.error('FAIL', String(e))
  process.exit(1)
}
