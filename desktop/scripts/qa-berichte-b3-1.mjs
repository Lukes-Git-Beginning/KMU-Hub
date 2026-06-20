import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b3-1-lifecycle')
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

// Open the draft (has blocks) -> "Als fertig markieren" available.
await page.getByRole('button').filter({ hasText: /Helpdesk-Auslastung KW 24/ }).first().click().catch(() => {})
await page.waitForTimeout(1200)
out.draft = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b3-1-draft.png') })

// draft -> final
await page.getByRole('button', { name: /Als fertig markieren/ }).first().click().catch(() => {})
await page.waitForTimeout(1100)
await page.screenshot({ path: resolve(outDir, 'b3-1-final.png') })

// final -> released
await page.getByRole('button', { name: /^Freigeben$/ }).first().click().catch(() => {})
await page.waitForTimeout(1200)
out.released = {
  rawKeys: await page.evaluate(findRawKeys, RAW.source),
  errs: errs.length,
  hasReleasedOn: await page.getByText(/Freigegeben am/).count(),
}
await page.screenshot({ path: resolve(outDir, 'b3-1-released.png') })

// open the more menu (Archivieren / Duplizieren)
await page.getByRole('button', { name: /Weitere Aktionen/ }).first().click({ force: true }).catch(() => {})
await page.waitForTimeout(500)
await page.screenshot({ path: resolve(outDir, 'b3-1-menu.png') })

out.errs = errs.length
console.log(JSON.stringify(out, null, 2))
await browser.close()
