import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma1')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(mails|common|shared)\.[a-z]+\.[a-z._]+/i

function findRawKeys(re) {
  const rx = new RegExp(re, 'i')
  return [
    ...new Set(
      Array.from(document.querySelectorAll('body *'))
        .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
        .map((n) => n.textContent.trim()),
    ),
  ].slice(0, 12)
}

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

// 1) Inbox list — verify previews populated, folder icons, unread badges
out.list = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, '01-inbox.png'), fullPage: false })

// Capture the unread badge count of the inbox folder before opening a mail.
const badgeBefore = await page.evaluate(() => {
  const el = document.querySelector('aside .badge-accent')
  return el ? el.textContent?.trim() : null
})
out.unreadBefore = badgeBefore

// 2) Open the first message -> reading pane
const firstRow = page.locator('[class*="cursor-pointer"]').first()
await firstRow.click().catch(() => {})
await page.waitForTimeout(900)
out.detail = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, '02-detail.png'), fullPage: false })

// 3) After opening, unread badge should decrement (stateful read)
await page.waitForTimeout(600)
const badgeAfter = await page.evaluate(() => {
  const el = document.querySelector('aside .badge-accent')
  return el ? el.textContent?.trim() : null
})
out.unreadAfter = badgeAfter
out.statefulRead = badgeBefore !== badgeAfter

// 4) Check for NaN KB in attachments (the old size_bytes bug)
out.hasNaNkb = await page.evaluate(() => document.body.textContent?.includes('NaN') ?? false)

// 5) Star toggle from the detail header, then screenshot
const starBtn = page.locator('header, .border-b').getByRole('button').filter({ hasText: '' })
await page.screenshot({ path: resolve(outDir, '03-detail-full.png'), fullPage: true })

// 6) Search box — type a query and verify list filters
const search = page.getByPlaceholder(/[Ss]uch/).first()
if (await search.count()) {
  await search.fill('Rechnung')
  await page.waitForTimeout(900)
  out.searchCount = await page.locator('[class*="cursor-pointer"]').count()
  await page.screenshot({ path: resolve(outDir, '04-search.png'), fullPage: false })
  await search.fill('')
  await page.waitForTimeout(600)
}

out.errs = errs.length
out.errors = errs.slice(0, 5)
console.log(JSON.stringify(out, null, 2))
await browser.close()
