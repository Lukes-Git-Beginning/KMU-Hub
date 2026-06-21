// QA — sidebar bottom: user removed, bottom = notifications + module-settings (:5173)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/sidebar')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1400, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(8000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3200)

const aside = page.locator('aside[data-tour="sidebar"]')
out.bottomHasNotifications = await aside.getByText(/Benachrichtigung/i).count()
out.bottomHasModuleSettings = await aside.getByText(/Modul-Einstellung|Einstellung/i).count()
// The old user block showed the user's email + a logout icon in the sidebar.
out.userBlockGone = (await aside.getByText(/@(techvision|zentria|firma)/i).count()) === 0
out.pageErrors = errs.slice(0, 6)
const box = await aside.boundingBox()
if (box) await page.screenshot({ path: resolve(outDir, 'sidebar-bottom.png'), clip: { x: box.x, y: Math.max(0, box.y + box.height - 240), width: box.width + 20, height: 240 } })
await page.screenshot({ path: resolve(outDir, 'sidebar-full.png'), fullPage: false })
console.log(JSON.stringify(out, null, 2))
await ctx.close()
await browser.close()
