import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db3-table')
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

// CRM article carries a pipeline comparison table.
await page.goto(`${BASE}/#/wiki?a=wart-006`, { waitUntil: 'domcontentloaded' })
const readTable = page.locator('.report-keep:has(table)').first()
await readTable.waitFor({ timeout: 15000 }).catch(() => {})
await readTable.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(500)
out.headerCells = await page.locator('table thead th').count()
out.bodyRows = await page.locator('table tbody tr').count()
out.read = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await readTable.screenshot({ path: resolve(outDir, 'zoom-read-table.png') }).catch(() => {})
await page.screenshot({ path: resolve(outDir, '01-read-table.png'), fullPage: false })

// Edit mode → grid of cell inputs + add/remove controls.
const editBtn = page.locator('button[title="Bearbeiten"]').first()
await editBtn.click().catch(() => {})
await page.waitForTimeout(1800)
const editGrid = page.locator('div.rounded-xl:has(input[placeholder]):has(button[aria-label])').filter({ hasText: 'Kopfzeile' }).first()
await editGrid.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(400)
out.edit = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await editGrid.screenshot({ path: resolve(outDir, 'zoom-edit-table.png') }).catch(() => {})
await page.screenshot({ path: resolve(outDir, '02-edit-table.png'), fullPage: false })

out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
