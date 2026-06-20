import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b2-5-callout-image-table')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(berichte|common|shared)\.[a-z]+\.[a-z._]+/i
const IMG =
  "data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='480' height='180'><rect width='480' height='180' fill='%230f766e'/><text x='240' y='100' font-size='26' fill='white' text-anchor='middle'>Demo-Bild</text></svg>"

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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1400 } })
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

async function insert(name) {
  await page.getByRole('button', { name: /Block einfügen/ }).first().click().catch(() => {})
  await page.waitForTimeout(250)
  await page.getByRole('button', { name }).first().click().catch(() => {})
  await page.waitForTimeout(450)
}

// --- Callout ---
await insert(/^Empfehlung$/)
await page.getByRole('button', { name: /Warnung/ }).first().click().catch(() => {})
await page.getByPlaceholder('Titel (optional)').first().fill('Achtung: Reaktionszeit').catch(() => {})
const rte = page.locator('.ProseMirror').last()
await rte.click().catch(() => {})
await rte.type('Die Erstreaktionszeit lag in KW 24 über dem Zielwert.').catch(() => {})
await page.waitForTimeout(400)
out.callout = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-5-callout.png'), fullPage: true })

// --- Image ---
await insert(/^Bild$/)
await page.getByPlaceholder('Bild-URL').first().fill(IMG).catch(() => {})
await page.getByPlaceholder('Alt-Text').first().fill('Demo Diagramm').catch(() => {})
await page.waitForTimeout(500)
out.image = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-5-image.png'), fullPage: true })

// --- Table (picker mode=table) ---
await insert(/^Tabelle$/)
await page.getByRole('button', { name: /Tabelle konfigurieren/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.tablePickerTitle = await page.locator('[role=dialog] h3').first().textContent().catch(() => null)
const dialog = page.locator('[role=dialog]')
await dialog.getByRole('button').filter({ hasText: /BWA/ }).first().click().catch(() => {})
await page.waitForTimeout(1400)
await page.screenshot({ path: resolve(outDir, 'b2-5-table-picker.png'), fullPage: true })
await dialog.getByRole('button', { name: /^Übernehmen$/ }).first().click().catch(() => {})
await page.waitForTimeout(1600)
out.table = {
  rawKeys: await page.evaluate(findRawKeys, RAW.source),
  errs: errs.length,
  tables: await page.locator('table').count(),
}
await page.screenshot({ path: resolve(outDir, 'b2-5-table.png'), fullPage: true })

// --- Read mode ---
await page.getByRole('button', { name: /^Lesen$/ }).first().click().catch(() => {})
await page.waitForTimeout(1200)
out.read = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-5-read.png'), fullPage: true })

console.log(JSON.stringify(out, null, 2))
await browser.close()
