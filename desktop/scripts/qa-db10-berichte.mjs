import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db10-berichte')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(document|blocks|berichte|common)\.[a-z]+\.[a-z._]+/i

function findRawKeys(reSource) {
  const rx = new RegExp(reSource, 'i')
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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: 2 })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2600)
await page.getByRole('button', { name: 'Berichte', exact: true }).first().click().catch(() => {})
await page.waitForTimeout(1200)
await page.getByText('Verkaufsbericht Q2 2026').first().click().catch(() => {})
await page.waitForTimeout(2000)

// The report opens in the print reader; the new blocks sit on a later page.
const codeBlock = page.locator('code.hljs').first()
await codeBlock.waitFor({ timeout: 15000 }).catch(() => {})
await codeBlock.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(500)

out.hljsSpans = await page.locator('code.hljs span[class^="hljs-"]').count()
out.tableHeaderCells = await page.locator('table thead th').count()
out.quotePresent = await page.locator('figure:has(blockquote)').count()
out.raw = await page.evaluate(findRawKeys, RAW.source)

const codeCard = page.locator('.report-keep:has(code.hljs)').first()
await codeCard.screenshot({ path: resolve(outDir, 'zoom-code.png') }).catch(() => {})
const tableCard = page.locator('.report-keep:has(table)').first()
await tableCard.scrollIntoViewIfNeeded().catch(() => {})
await tableCard.screenshot({ path: resolve(outDir, 'zoom-table.png') }).catch(() => {})
const quoteCard = page.locator('figure:has(blockquote)').first()
await quoteCard.scrollIntoViewIfNeeded().catch(() => {})
await quoteCard.screenshot({ path: resolve(outDir, 'zoom-quote.png') }).catch(() => {})
await page.screenshot({ path: resolve(outDir, '01-report.png'), fullPage: false })

out.errs = errs.length
out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
