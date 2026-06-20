import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b2-3-chart-new')
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

// Open draft, switch to edit, insert a chart block, open picker.
await page.getByRole('button').filter({ hasText: /Helpdesk-Auslastung KW 24/ }).first().click().catch(() => {})
await page.waitForTimeout(1200)
await page.getByRole('button', { name: /^Bearbeiten$/ }).first().click().catch(() => {})
await page.waitForTimeout(700)
await page.getByRole('button', { name: /Block einfügen/ }).first().click().catch(() => {})
await page.waitForTimeout(300)
await page.getByRole('button', { name: /^Diagramm$/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
await page.getByRole('button', { name: /Grafik konfigurieren/ }).first().click().catch(() => {})
await page.waitForTimeout(500)

// Switch to the "Neue Grafik" tab.
await page.getByRole('button', { name: /^Neue Grafik$/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.newTab = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-3-new-empty.png'), fullPage: true })

// Pick the "Rechnungen" (finanzen) source.
const dialog = page.locator('[role=dialog]')
await dialog.getByRole('button', { name: /Rechnungen/ }).first().click().catch(() => {})
await page.waitForTimeout(800)
await page.screenshot({ path: resolve(outDir, 'b2-3-source.png'), fullPage: true })

// Toggle a dimension (Status) and a measure (Netto-Betrag) -> bar chart.
await dialog.getByRole('button', { name: /^Status$/ }).first().click().catch(() => {})
await page.waitForTimeout(300)
await dialog.getByRole('button', { name: /Netto-Betrag/ }).first().click().catch(() => {})
await page.waitForTimeout(1600)
out.afterFields = {
  rawKeys: await page.evaluate(findRawKeys, RAW.source),
  errs: errs.length,
  previewCharts: await page.locator('svg.recharts-surface').count(),
}
await page.screenshot({ path: resolve(outDir, 'b2-3-fields-preview.png'), fullPage: true })

// Save & insert -> the new definition renders in the editor block.
await dialog.getByRole('button', { name: /Speichern & einfügen/ }).first().click().catch(() => {})
await page.waitForTimeout(2200)
out.inserted = {
  rawKeys: await page.evaluate(findRawKeys, RAW.source),
  errs: errs.length,
  charts: await page.locator('svg.recharts-surface').count(),
}
await page.screenshot({ path: resolve(outDir, 'b2-3-inserted.png'), fullPage: true })

console.log(JSON.stringify(out, null, 2))
await browser.close()
