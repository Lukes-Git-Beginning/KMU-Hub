import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db6-quote-bookmark')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(document|blocks|wiki|common)\.[a-z]+\.[a-z._]+/i

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

// Quote — welcome article.
await page.goto(`${BASE}/#/wiki?a=wart-001`, { waitUntil: 'domcontentloaded' })
const quote = page.locator('figure:has(blockquote)').first()
await quote.waitFor({ timeout: 15000 }).catch(() => {})
await quote.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(400)
out.quoteText = ((await quote.textContent().catch(() => '')) || '').trim().slice(0, 70)
out.quoteRaw = await page.evaluate(findRawKeys, RAW.source)
await quote.screenshot({ path: resolve(outDir, 'zoom-quote.png') }).catch(() => {})

// Bookmark — DSGVO article.
await page.goto(`${BASE}/#/wiki?a=wart-005`, { waitUntil: 'domcontentloaded' })
const bm = page.locator('a[target="_blank"]:has(svg)').first()
await bm.waitFor({ timeout: 15000 }).catch(() => {})
await bm.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(400)
out.bookmarkHref = ((await bm.getAttribute('href').catch(() => '')) || '').slice(0, 60)
out.bookmarkText = ((await bm.textContent().catch(() => '')) || '').trim().slice(0, 80)
out.bookmarkRaw = await page.evaluate(findRawKeys, RAW.source)
await bm.screenshot({ path: resolve(outDir, 'zoom-bookmark.png') }).catch(() => {})

out.errs = errs.length
out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
