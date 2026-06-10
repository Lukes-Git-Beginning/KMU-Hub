import { chromium } from 'playwright'
const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUP = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser = await chromium.launch({ headless: false })
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1100 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(SUP)
const page = await ctx.newPage()
await page.goto('http://localhost:5173/#/dokumente', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2800)
await page.getByText('Projektplan_KMU_Hub_v2.pdf').first().click()
await page.waitForTimeout(3500)
await page.screenshot({ path: '.qa-screenshots/dokumente-fb-viewer-headed.png' })
await browser.close()
console.log('ok')
