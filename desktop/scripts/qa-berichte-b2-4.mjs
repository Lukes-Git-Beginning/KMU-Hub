import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b2-4-kpi')
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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1200 } })
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

// Open draft, edit mode.
await page.getByRole('button').filter({ hasText: /Helpdesk-Auslastung KW 24/ }).first().click().catch(() => {})
await page.waitForTimeout(1200)
await page.getByRole('button', { name: /^Bearbeiten$/ }).first().click().catch(() => {})
await page.waitForTimeout(700)

// Insert a KPI row (3 columns x KPI).
await page.getByRole('button', { name: /Block einfügen/ }).first().click().catch(() => {})
await page.waitForTimeout(300)
await page.getByRole('button', { name: /Kennzahl-Reihe/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.empty = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-4-kpi-row-empty.png'), fullPage: true })

// Fill the three KPI blocks.
const data = [
  { label: 'Umsatz', value: '142.500', unit: '€', trend: '18', source: 'Finanzen' },
  { label: 'Gewinnrate', value: '32', unit: '%', trend: '4.2', source: 'CRM' },
  { label: 'Neue Deals', value: '37', unit: '', trend: '12', source: 'CRM' },
]
const labels = page.getByPlaceholder('Bezeichnung')
const values = page.getByPlaceholder('Wert')
const units = page.getByPlaceholder('Einheit')
const trends = page.getByPlaceholder('Trend %')
const sources = page.getByPlaceholder('Quelle')
for (let i = 0; i < 3; i += 1) {
  await labels.nth(i).fill(data[i].label).catch(() => {})
  await values.nth(i).fill(data[i].value).catch(() => {})
  if (data[i].unit) await units.nth(i).fill(data[i].unit).catch(() => {})
  await trends.nth(i).fill(data[i].trend).catch(() => {})
  await sources.nth(i).fill(data[i].source).catch(() => {})
}
await page.waitForTimeout(600)
out.filled = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-4-kpi-row-filled.png'), fullPage: true })

// Switch to read mode -> KPI row renders with trend + source.
await page.getByRole('button', { name: /^Lesen$/ }).first().click().catch(() => {})
await page.waitForTimeout(900)
out.read = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-4-kpi-read.png'), fullPage: true })

console.log(JSON.stringify(out, null, 2))
await browser.close()
