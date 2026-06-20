import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b5-2-schedule')
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

// 1) Released report -> "Zeitplan" button active, opens modal.
await page.getByRole('button').filter({ hasText: /Verkaufsbericht Q2 2026/ }).first().click().catch(() => {})
await page.waitForTimeout(1500)
out.scheduleBtnReleased = await page.getByRole('button', { name: /^Zeitplan$/ }).count()
await page.getByRole('button', { name: /^Zeitplan$/ }).first().click().catch(() => {})
await page.waitForTimeout(800)
out.modalOpen = await page.getByText(/Zeitplan einrichten/).count()
out.modalRawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, 'b5-2-modal-weekly.png') })

// 2) Switch rhythm to monthly -> day-of-month select appears.
await page.getByRole('button', { name: /^Monatlich$/ }).first().click().catch(() => {})
await page.waitForTimeout(400)
out.hasDayOfMonth = await page.getByText(/Tag im Monat/).count()
await page.screenshot({ path: resolve(outDir, 'b5-2-modal-monthly.png') })

// 3) Switch to daily -> neither weekday nor day select.
await page.getByRole('button', { name: /^Täglich$/ }).first().click().catch(() => {})
await page.waitForTimeout(400)
await page.screenshot({ path: resolve(outDir, 'b5-2-modal-daily.png') })

// close modal
await page.keyboard.press('Escape').catch(() => {})
await page.waitForTimeout(500)

// 4) Draft report -> back to library, open draft, "Zeitplan" button is disabled.
await page.getByRole('button', { name: /Zurück zur Übersicht/ }).first().click().catch(() => {})
await page.waitForTimeout(1000)
await page.getByRole('button').filter({ hasText: /Helpdesk-Auslastung KW 24/ }).first().click().catch(() => {})
await page.waitForTimeout(1200)
const draftBtn = page.getByRole('button', { name: /^Zeitplan$/ }).first()
out.draftScheduleDisabled = (await draftBtn.count()) ? await draftBtn.isDisabled() : 'no-button'
await page.screenshot({ path: resolve(outDir, 'b5-2-draft-guard.png') })

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
