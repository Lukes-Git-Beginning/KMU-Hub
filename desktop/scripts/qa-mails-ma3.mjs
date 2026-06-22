import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma3')
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

out.switcherVisible = await page.getByRole('button', { name: /Konto wechseln/i }).count()
out.unifiedEntryVisible = await page.getByText(/Alle Eingänge/).count()
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '01-sidebar.png'), fullPage: false })

// Open the switcher dropdown
await page.getByRole('button', { name: /Konto wechseln/i }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.dropdownAccounts = await page.evaluate(() =>
  Array.from(document.querySelectorAll('[role="option"]')).map((o) => o.textContent.trim()).slice(0, 6),
)
await page.screenshot({ path: resolve(outDir, '02-dropdown.png'), fullPage: false })

// Pick the support account
const supportOpt = page.getByRole('option').filter({ hasText: /support@/i }).first()
if (await supportOpt.count()) {
  await supportOpt.click().catch(() => {})
  await page.waitForTimeout(1200)
}
out.supportTickets = await page.evaluate(() => /Ticket #44/.test(document.body.textContent || ''))
await page.screenshot({ path: resolve(outDir, '03-support-account.png'), fullPage: false })

// Open unified inbox via the sidebar entry
await page.getByText(/Alle Eingänge/).first().click().catch(() => {})
await page.waitForTimeout(1200)
// In unified view, do we see messages from multiple accounts (account chips)?
out.unifiedAccountChips = await page.evaluate(() => {
  const txt = document.body.textContent || ''
  return ['info@techvision.de', 'support@techvision.de', 'stefan.vogel@techvision.de'].filter((e) => txt.includes(e)).length
})
out.unifiedRowCount = await page.locator('[class*="cursor-pointer"]').count()
await page.screenshot({ path: resolve(outDir, '04-unified.png'), fullPage: false })

out.errs = errs.length
out.errors = errs.slice(0, 5)
console.log(JSON.stringify(out, null, 2))
await browser.close()
