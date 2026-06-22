import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma8')
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
out.allCount = await rowCount(page)

// Select 3 rows via checkbox
const checks = page.getByRole('button', { name: /Auswählen/ })
for (let i = 0; i < 3; i++) await checks.nth(i).click({ force: true }).catch(() => {})
await page.waitForTimeout(400)
out.bulkBarVisible = await page.evaluate(() => /ausgewählt/.test(document.body.textContent || ''))
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '01-selection.png'), fullPage: false })

// Bulk archive
await page.getByRole('button', { name: /^Archivieren$/ }).first().click().catch(() => {})
await page.waitForTimeout(1000)
out.bulkToast = await page.evaluate(() => /aktualisiert/.test(document.body.textContent || ''))
out.countAfterArchive = await rowCount(page)
out.droppedThree = out.allCount - out.countAfterArchive
await page.screenshot({ path: resolve(outDir, '02-after-archive.png'), fullPage: false })

// Keyboard navigation: open first message, press j, check selection moves
await page.locator('[class*="cursor-pointer"]').first().click().catch(() => {})
await page.waitForTimeout(500)
const subjBefore = await page.evaluate(() => document.querySelector('.flex-1.overflow-y-auto h2')?.textContent || '')
await page.keyboard.press('j')
await page.waitForTimeout(500)
const subjAfter = await page.evaluate(() => document.querySelector('.flex-1.overflow-y-auto h2')?.textContent || '')
out.jKeyMovedSelection = subjBefore !== subjAfter && !!subjAfter
// x toggles selection of current
await page.keyboard.press('x')
await page.waitForTimeout(300)
out.xKeySelected = await page.evaluate(() => /ausgewählt/.test(document.body.textContent || ''))
await page.screenshot({ path: resolve(outDir, '03-keyboard.png'), fullPage: false })

out.errs = errs.length
out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
