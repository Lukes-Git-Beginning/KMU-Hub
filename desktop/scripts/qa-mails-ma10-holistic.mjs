import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma10')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(mails|common|shared|moduleSettings)\.[a-z]+\.[a-z._]+/i
function findRawKeys(re){const rx=new RegExp(re,'i');return [...new Set(Array.from(document.querySelectorAll('body *')).filter((n)=>n.children.length===0&&rx.test(n.textContent||'')).map((n)=>n.textContent.trim()))].slice(0,15)}

const browser = await chromium.launch()
const out = { steps: {}, rawKeysAll: [] }
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const scan = async (name) => {
  const rk = await page.evaluate(findRawKeys, RAW.source)
  out.steps[name] = { rawKeys: rk, errs: errs.length }
  out.rawKeysAll.push(...rk)
}

// 1. Inbox
await page.goto(`${BASE}/#/mails`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2600)
await scan('inbox')
await page.screenshot({ path: resolve(outDir, '01-inbox.png'), fullPage: false })

// 2. Thread detail
await page.locator('[class*="cursor-pointer"]').filter({ hasText: /Cosmi-Einf/ }).first().click().catch(() => {})
await page.waitForTimeout(900)
await scan('thread')
await page.screenshot({ path: resolve(outDir, '02-thread.png'), fullPage: false })

// 3. Label filter
await page.locator('aside').getByText('Rechnung', { exact: true }).first().click().catch(() => {})
await page.waitForTimeout(800)
await scan('labelFilter')

// 4. Unified inbox
await page.getByText(/Alle Eingänge/).first().click().catch(() => {})
await page.waitForTimeout(800)
await scan('unified')
await page.screenshot({ path: resolve(outDir, '03-unified.png'), fullPage: false })

// 5. Compose + template dialog
await page.getByRole('button', { name: /Neue E-Mail/ }).first().click().catch(() => {})
await page.waitForTimeout(700)
await page.getByTitle(/Vorlage/i).first().click().catch(() => {})
await page.waitForTimeout(600)
await scan('templateDialog')
await page.screenshot({ path: resolve(outDir, '04-template.png'), fullPage: false })
await page.keyboard.press('Escape')
await page.waitForTimeout(400)

out.uniqueRawKeys = [...new Set(out.rawKeysAll)]
out.totalErrs = errs.length
out.errors = errs.slice(0, 8)
console.log(JSON.stringify(out, null, 2))
await browser.close()
