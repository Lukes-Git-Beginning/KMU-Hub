import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE = 'http://localhost:5173'
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' }); await page.waitForTimeout(2200)
const markup = await page.evaluate(() => {
  const th = Array.from(document.querySelectorAll('th')).find((t) => /^SLA$/.test((t.textContent || '').trim()))
  if (!th) return { found: false }
  const span = th.querySelector('span')
  return { found: true, hasSpan: !!span, cls: span ? span.className : th.className }
})
const sla = page.locator('th span').filter({ hasText: /^SLA$/ }).first()
const box = await sla.boundingBox().catch(() => null)
if (box) { await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2); await page.waitForTimeout(900) }
const tip = await page.evaluate(() => /Service Level Agreement/.test(document.body.textContent || ''))
if (box) { await page.screenshot({ path: resolve('.qa-screenshots/team-helpdesk-fixes/f8-sla-tooltip.png'), clip: { x: Math.max(0, box.x - 140), y: Math.max(0, box.y - 8), width: 380, height: 170 } }) }
console.log(JSON.stringify({ markup, tooltipShown: tip }, null, 2))
await b.close()
