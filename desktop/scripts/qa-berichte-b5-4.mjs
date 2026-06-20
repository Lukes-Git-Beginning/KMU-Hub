import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b5-4-history')
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

// 1) Open modal — fresh: next-run preview present, no history yet.
await page.getByRole('button', { name: /^Zeitplan$/ }).first().click().catch(() => {})
await page.waitForTimeout(700)
out.nextRunVisible = await page.getByText(/Nächste Ausführung/).count()
out.historyBeforeSave = await page.getByText(/Lauf-Historie/).count() // expect 0 (no schedule yet)
await page.screenshot({ path: resolve(outDir, 'b5-4-new.png') })

// 2) Save -> creates schedule, modal closes.
await page.getByRole('button', { name: /Zeitplan speichern/ }).first().click().catch(() => {})
await page.waitForTimeout(1000)

// 3) Reopen -> history section + send-now now present, no runs yet.
await page.getByRole('button', { name: /^Zeitplan$/ }).first().click().catch(() => {})
await page.waitForTimeout(800)
out.historyAfterSave = await page.getByText(/Lauf-Historie/).count()
out.sendNowVisible = await page.getByRole('button', { name: /Jetzt senden/ }).count()
out.noRunsBefore = await page.getByText(/Noch keine Läufe/).count()
await page.screenshot({ path: resolve(outDir, 'b5-4-existing.png') })

// 4) Send now -> records a run, history populates.
await page.getByRole('button', { name: /Jetzt senden/ }).first().click().catch(() => {})
await page.waitForTimeout(1200)
out.noRunsAfter = await page.getByText(/Noch keine Läufe/).count() // expect 0 now
out.historyRows = await page.locator('li').filter({ hasText: /Erfolg|Fehlgeschlagen|Übersprungen/ }).count()
out.modalRawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, 'b5-4-after-send.png') })

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
