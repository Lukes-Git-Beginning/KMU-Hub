import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma2')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(mails|common|shared)\.[a-z]+\.[a-z._]+/i

function findRawKeys(re) {
  const rx = new RegExp(re, 'i')
  return [...new Set(Array.from(document.querySelectorAll('body *')).filter((n) => n.children.length === 0 && rx.test(n.textContent || '')).map((n) => n.textContent.trim()))].slice(0, 12)
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

// Thread count badge visible in the list?
out.listThreadBadge = await page.evaluate(() => {
  const rows = Array.from(document.querySelectorAll('[class*="cursor-pointer"]'))
  const cosmi = rows.find((r) => /Cosmi-Einf/i.test(r.textContent || ''))
  return cosmi ? /\b[2-9]\b/.test(cosmi.textContent || '') : false
})
await page.screenshot({ path: resolve(outDir, '01-list-grouped.png'), fullPage: false })

// Open the Nordwind thread
const row = page.locator('[class*="cursor-pointer"]').filter({ hasText: /Cosmi-Einf/i }).first()
await row.click().catch(() => {})
await page.waitForTimeout(1100)
out.detail = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }

// How many message cards in the thread? (scoped to the thread container)
out.threadCards = await page.locator('[data-testid="mail-thread"] [aria-expanded]').count()
// Thread count chip in header?
out.headerChip = await page.evaluate(() => /Nachricht/.test(document.body.textContent || ''))
await page.screenshot({ path: resolve(outDir, '02-thread.png'), fullPage: true })

// Expand ALL collapsed cards in the thread (so image + quote become visible)
const collapsedCards = page.locator('[data-testid="mail-thread"] [aria-expanded="false"]')
const n = await collapsedCards.count()
for (let i = 0; i < n; i++) {
  // re-query first collapsed each time (DOM updates as we expand)
  const c = page.locator('[data-testid="mail-thread"] [aria-expanded="false"]').first()
  if (await c.count()) {
    await c.click().catch(() => {})
    await page.waitForTimeout(250)
  }
}
await page.waitForTimeout(400)
out.afterExpand = { errs: errs.length }
// Inline image rendered (in the now-expanded em-022)?
out.inlineImg = await page.evaluate(() => !!document.querySelector('[data-testid="mail-thread"] img[src^="data:image/png"]'))
await page.screenshot({ path: resolve(outDir, '03-all-expanded.png'), fullPage: true })

// Toggle quoted history on a sent message (now expanded)
const quoteToggle = page.getByTitle(/Zitierten Verlauf/i).first()
out.hasQuoteToggle = (await quoteToggle.count()) > 0
if (await quoteToggle.count()) {
  await quoteToggle.click().catch(() => {})
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '04-quote.png'), fullPage: true })
}

out.errs = errs.length
out.errors = errs.slice(0, 5)
console.log(JSON.stringify(out, null, 2))
await browser.close()
