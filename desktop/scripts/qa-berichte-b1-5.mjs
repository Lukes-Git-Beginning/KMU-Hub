import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/r0-berichte')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(berichte|common)\.[a-z]+\.[a-z._]+/i

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
await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2200)
const tab = page.getByRole('button', { name: /^Berichte$/ }).first()
if (await tab.count()) await tab.click().catch(() => {})
await page.waitForTimeout(1000)
// Open from library -> should default to READ mode
const card = page.getByRole('button').filter({ hasText: /Monatsbericht Juni/ }).first()
await card.click().catch(() => {})
await page.waitForTimeout(1500)
out.readDefault = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b1-5-read-default.png') })

// Switch to edit, change the title -> save indicator
const editBtn = page.getByRole('button', { name: /^Bearbeiten$/ }).first()
if (await editBtn.count()) await editBtn.click().catch(() => {})
await page.waitForTimeout(500)
const title = page.getByLabel('Berichtstitel')
if (await title.count()) {
  await title.fill('Monatsbericht Juni 2026 — aktualisiert')
  await title.blur()
  await page.waitForTimeout(1300)
}
out.saved = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b1-5-saved.png') })
console.log(JSON.stringify(out, null, 2))
await browser.close()
