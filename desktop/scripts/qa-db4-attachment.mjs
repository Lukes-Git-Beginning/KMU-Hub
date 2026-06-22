import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db4-attachment')
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

await page.goto(`${BASE}/#/wiki?a=wart-004`, { waitUntil: 'domcontentloaded' })
const card = page.locator('a[download]').first()
await card.waitFor({ timeout: 15000 }).catch(() => {})
await card.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(500)
out.fileName = (await card.textContent().catch(() => '') || '').trim().slice(0, 60)
out.downloadHref = ((await card.getAttribute('download').catch(() => '')) || '')
out.read = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await card.screenshot({ path: resolve(outDir, 'zoom-read-attachment.png') }).catch(() => {})

// Edit mode → file card with name input + Replace.
const editBtn = page.locator('button[title="Bearbeiten"]').first()
await editBtn.click().catch(() => {})
await page.waitForTimeout(1800)
const editCard = page.locator('div.rounded-xl:has(input[placeholder="Dateiname"])').first()
await editCard.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(400)
out.edit = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await editCard.screenshot({ path: resolve(outDir, 'zoom-edit-attachment.png') }).catch(() => {})

out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
