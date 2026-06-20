import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b6-4-share-link')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(berichte|common|shared)\.[a-z]+\.[a-z._]+/i

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
const ctx = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
  permissions: ['clipboard-read', 'clipboard-write'],
})
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2200)
const tab = page.getByRole('button', { name: /^Berichte$/ }).first()
if (await tab.count()) await tab.click().catch(() => {})
await page.waitForTimeout(1000)

await page.getByRole('button').filter({ hasText: /Verkaufsbericht Q2 2026/ }).first().click().catch(() => {})
await page.waitForTimeout(1500)

await page.getByRole('button', { name: /^Teilen$/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.linkEntry = await page.getByRole('button', { name: /Externen Link erzeugen/ }).count()
await page.getByRole('button', { name: /Externen Link erzeugen/ }).first().click().catch(() => {})
await page.waitForTimeout(700)
out.dialogOpen = await page.getByText(/Ablauf/).count()
out.dialogRawKeys = await page.evaluate(findRawKeys, RAW.source)
// pick 90 days + enable password
await page.getByRole('button', { name: /^90 Tage$/ }).first().click().catch(() => {})
await page.screenshot({ path: resolve(outDir, 'b6-4-form.png') })

// create the link
await page.getByRole('button', { name: /^Link erzeugen$/ }).first().click().catch(() => {})
await page.waitForTimeout(900)
out.linkShown = await page.getByText(/cosmi:\/\/share\/report/).count()
await page.screenshot({ path: resolve(outDir, 'b6-4-link.png') })

// copy
await page.getByRole('button', { name: /^Kopieren$/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.copiedState = await page.getByText(/^Kopiert$/).count()
out.clipboard = await page.evaluate(() => navigator.clipboard.readText().catch(() => 'n/a'))

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
