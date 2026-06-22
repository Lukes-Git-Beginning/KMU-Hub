import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma4')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(mails|common|shared)\.[a-z]+\.[a-z._]+/i
function findRawKeys(re){const rx=new RegExp(re,'i');return [...new Set(Array.from(document.querySelectorAll('body *')).filter((n)=>n.children.length===0&&rx.test(n.textContent||'')).map((n)=>n.textContent.trim()))].slice(0,12)}
const rowCount = (page) => page.locator('[class*="cursor-pointer"]').count()

const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

await page.goto(`${BASE}/#/mails`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2600)
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
out.allCount = await rowCount(page)
await page.screenshot({ path: resolve(outDir, '01-filterbar.png'), fullPage: false })

// Filter: Ungelesen
await page.getByRole('button', { name: /^Ungelesen$/ }).click().catch(() => {})
await page.waitForTimeout(900)
out.unreadCount = await rowCount(page)
await page.screenshot({ path: resolve(outDir, '02-unread.png'), fullPage: false })

// Filter: Markiert
await page.getByRole('button', { name: /^Markiert$/ }).click().catch(() => {})
await page.waitForTimeout(900)
out.starredCount = await rowCount(page)
await page.screenshot({ path: resolve(outDir, '03-starred.png'), fullPage: false })

// back to Alle
await page.getByRole('button', { name: /^Alle$/ }).click().catch(() => {})
await page.waitForTimeout(700)

// Sort by sender, ascending
const firstSenderBefore = await page.evaluate(() => document.querySelector('[class*="cursor-pointer"]')?.textContent?.slice(0, 30) || '')
await page.getByRole('button', { name: /Sortieren|Sort/i }).first().click().catch(() => {})
await page.waitForTimeout(400)
await page.getByRole('menuitemradio', { name: /Absender/ }).click().catch(() => {})
await page.waitForTimeout(300)
await page.getByRole('menuitemradio', { name: /Aufsteigend|Ascending/ }).click().catch(() => {})
await page.waitForTimeout(800)
const firstSenderAfter = await page.evaluate(() => document.querySelector('[class*="cursor-pointer"]')?.textContent?.slice(0, 30) || '')
out.sortChanged = firstSenderBefore !== firstSenderAfter
out.firstSenderAfterSort = firstSenderAfter
await page.screenshot({ path: resolve(outDir, '04-sorted.png'), fullPage: false })

out.errs = errs.length
out.errors = errs.slice(0, 5)
console.log(JSON.stringify(out, null, 2))
await browser.close()
