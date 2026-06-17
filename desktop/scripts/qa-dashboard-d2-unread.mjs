/** D-2 recheck: CrossModule unread count now reflects real data (was hardcoded 0). */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dashboard-d2')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const seed = `try{localStorage.setItem('cosmi-dashboard',JSON.stringify({state:{scope:'personal',personalActiveWidgets:['cross-module-overview'],personalLayouts:[{i:'cross-module-overview',x:0,y:0,w:6,h:5,minW:4,minH:3}],teamActiveWidgets:[],teamLayouts:[]},version:2}))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(seed)
const page = await ctx.newPage(); const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2800)
await page.evaluate(() => { const el = document.querySelector('.layout'); if (el) el.scrollIntoView() })
await page.waitForTimeout(600)
const unreadRow = await page.evaluate(() => {
  const rows = Array.from(document.querySelectorAll('.layout a, .layout div'))
  const row = rows.find((r) => /ungelesen/i.test(r.textContent || ''))
  return row ? row.textContent.trim() : null
})
await page.screenshot({ path: resolve(outDir, '6-unread-recheck.png') })
console.log(JSON.stringify({ unreadRow, pageErrors: errs.length }, null, 2))
await browser.close()
