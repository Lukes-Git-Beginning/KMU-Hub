import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma6')
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
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
out.labelsInSidebar = await page.evaluate(() => ['Wichtig', 'Kunde', 'Rechnung', 'Intern'].filter((n) => document.querySelector('aside')?.textContent?.includes(n)).length)
// label chips on rows present? (Kunde / Rechnung appear in list metadata)
out.labelChipsOnRows = await page.evaluate(() => {
  const rows = Array.from(document.querySelectorAll('[class*="cursor-pointer"]'))
  return rows.some((r) => /Kunde|Rechnung/.test(r.textContent || ''))
})
await page.screenshot({ path: resolve(outDir, '01-labels-sidebar.png'), fullPage: false })

// Filter by label "Kunde"
await page.locator('aside').getByText('Kunde', { exact: true }).first().click().catch(() => {})
await page.waitForTimeout(900)
out.kundeFilterCount = await page.locator('[class*="cursor-pointer"]').count()
await page.screenshot({ path: resolve(outDir, '02-label-filter.png'), fullPage: false })

// Open rules dialog
await page.getByText(/Regeln & Filter/).first().click().catch(() => {})
await page.waitForTimeout(700)
out.rulesDialogOpen = await page.getByText(/Rechnungen markieren/).count()
out.addRuleFormVisible = await page.getByPlaceholder(/Regelname/).count()
await page.screenshot({ path: resolve(outDir, '03-rules.png'), fullPage: false })

// Apply rules
await page.getByRole('button', { name: /Regeln jetzt anwenden/ }).click().catch(() => {})
await page.waitForTimeout(1000)
out.applyToast = await page.evaluate(() => /aktualisiert/.test(document.body.textContent || ''))
await page.screenshot({ path: resolve(outDir, '04-applied.png'), fullPage: false })

out.errs = errs.length
out.errors = errs.slice(0, 5)
console.log(JSON.stringify(out, null, 2))
await browser.close()
