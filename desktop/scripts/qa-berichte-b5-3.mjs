import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b5-3-recipients')
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

await page.getByRole('button').filter({ hasText: /Verkaufsbericht Q2 2026/ }).first().click().catch(() => {})
await page.waitForTimeout(1500)
await page.getByRole('button', { name: /^Zeitplan$/ }).first().click().catch(() => {})
await page.waitForTimeout(700)

// Default recipient (CURRENT_USER) chip should be present.
out.defaultChip = await page.getByText(/Stefan Vogel/).count()

// 1) Internal user typeahead.
await page.getByPlaceholder(/Person suchen/).fill('Laura')
await page.waitForTimeout(500)
out.typeaheadMatch = await page.getByText(/Laura Neumann/).count()
await page.screenshot({ path: resolve(outDir, 'b5-3-typeahead.png') })
await page.getByText(/Laura Neumann/).first().click().catch(() => {})
await page.waitForTimeout(400)

// 2) External email.
await page.getByPlaceholder(/Externe E-Mail/).fill('kunde@example.com')
await page.getByRole('button', { name: /^Hinzufügen$/ }).first().click().catch(() => {})
await page.waitForTimeout(400)
out.externalChip = await page.getByText(/kunde@example\.com/).count()

// 3) Invalid email -> error toast, no chip.
await page.getByPlaceholder(/Externe E-Mail/).fill('not-an-email')
await page.getByRole('button', { name: /^Hinzufügen$/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.invalidRejected = (await page.getByText(/^not-an-email$/).count()) === 0

out.modalRawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, 'b5-3-chips.png') })

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
