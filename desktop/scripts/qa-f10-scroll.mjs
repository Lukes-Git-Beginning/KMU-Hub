import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE = 'http://localhost:5173'
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 760 } }) // shorter viewport → modal must scroll
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const out = {}
await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' }); await page.waitForTimeout(2200)
await page.locator('table tbody tr').first().click({ timeout: 5000 }).catch(() => {})
await page.waitForTimeout(900)
// find the scrollable body inside the dialog
const before = await page.evaluate(() => {
  const dlg = document.querySelector('[role="dialog"]')
  if (!dlg) return null
  const scroller = Array.from(dlg.querySelectorAll('div')).find((d) => d.scrollHeight > d.clientHeight + 4 && getComputedStyle(d).overflowY === 'auto')
  if (!scroller) return { scrollable: false }
  return { scrollable: true, scrollHeight: scroller.scrollHeight, clientHeight: scroller.clientHeight, scrollTop: scroller.scrollTop }
})
out.before = before
// scroll to bottom
await page.evaluate(() => {
  const dlg = document.querySelector('[role="dialog"]')
  const scroller = Array.from(dlg.querySelectorAll('div')).find((d) => d.scrollHeight > d.clientHeight + 4 && getComputedStyle(d).overflowY === 'auto')
  if (scroller) scroller.scrollTop = scroller.scrollHeight
})
await page.waitForTimeout(500)
const after = await page.evaluate(() => {
  const dlg = document.querySelector('[role="dialog"]')
  const scroller = Array.from(dlg.querySelectorAll('div')).find((d) => d.scrollHeight > d.clientHeight + 4 && getComputedStyle(d).overflowY === 'auto')
  return scroller ? { scrollTop: scroller.scrollTop } : null
})
out.afterScrollTop = after ? after.scrollTop : null
// any visible scrollbar? (Radix ScrollBar would be a div[data-radix-scroll-area-scrollbar])
out.radixScrollbarPresent = await page.evaluate(() => !!document.querySelector('[role="dialog"] [data-radix-scroll-area-scrollbar]'))
await page.screenshot({ path: resolve('.qa-screenshots/team-helpdesk-fixes/f10-scrolled-bottom.png') })
console.log(JSON.stringify(out, null, 2))
await b.close()
