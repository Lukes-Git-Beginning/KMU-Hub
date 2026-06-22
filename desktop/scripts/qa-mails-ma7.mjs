import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma7')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(mails|common|shared)\.[a-z]+\.[a-z._]+/i
function findRawKeys(re){const rx=new RegExp(re,'i');return [...new Set(Array.from(document.querySelectorAll('body *')).filter((n)=>n.children.length===0&&rx.test(n.textContent||'')).map((n)=>n.textContent.trim()))].slice(0,12)}

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

// Open the first message (Klaus Gruber — seeded linked to a contact)
await page.locator('[class*="cursor-pointer"]').filter({ hasText: /Klaus Gruber/ }).first().click().catch(() => {})
await page.waitForTimeout(900)
out.crmPanelVisible = await page.getByRole('button', { name: /Deal aus Mail/ }).count()
out.activityBtn = await page.getByRole('button', { name: /Aktivität/ }).count()
out.linkBtn = await page.getByRole('button', { name: /Kontakt verknüpfen/ }).count()
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '01-crm-panel.png'), fullPage: false })

// Create deal from mail
await page.getByRole('button', { name: /Deal aus Mail/ }).click().catch(() => {})
await page.waitForTimeout(900)
out.dealToast = await page.evaluate(() => /Deal aus E-Mail erstellt/.test(document.body.textContent || ''))
await page.screenshot({ path: resolve(outDir, '02-deal-toast.png'), fullPage: false })
await page.waitForTimeout(1500) // let toast fade

// Log activity
await page.getByRole('button', { name: /^Aktivität$/ }).click().catch(() => {})
await page.waitForTimeout(900)
out.activityToast = await page.evaluate(() => /protokolliert/.test(document.body.textContent || ''))
await page.screenshot({ path: resolve(outDir, '03-activity-toast.png'), fullPage: false })
await page.waitForTimeout(1500)

// Open contact picker + link
await page.getByRole('button', { name: /Kontakt verknüpfen/ }).click().catch(() => {})
await page.waitForTimeout(500)
out.pickerVisible = await page.getByPlaceholder(/Kontakt suchen/).count()
const firstResult = page.locator('button', { hasText: /@/ }).filter({ hasText: /@/ })
await page.screenshot({ path: resolve(outDir, '04-picker.png'), fullPage: false })

out.errs = errs.length
out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
