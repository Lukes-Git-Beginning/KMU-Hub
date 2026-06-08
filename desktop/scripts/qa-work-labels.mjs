// QA work Phase 3 — Labels/Tags auf Tasks. Chips in Kanban/Liste/Suche/Detail,
// Label-Picker im Detail, Label-Filter in der Suche. Raw-Keys + pageErrors.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUP = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(work|common|moduleSettings|settings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function shot(page, name) { await page.screenshot({ path: resolve(outDir, name), fullPage: true }) }

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const errors = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(SUP)
const page = await ctx.newPage()
page.on('pageerror', (e) => errors.push(String(e).split('\n')[0]))

try {
  // 1. Kanban — label chips on cards
  await page.goto(`${BASE}/#/work/projects/prj-001`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2800)
  // ensure kanban view
  await page.getByRole('button', { name: /Kanban|Board/i }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(1200)
  out.kanbanLabelChips = await page.locator('text=/^Feature$|^Bug$|^Design$|^Blockiert$/').count().catch(() => -1)
  await shot(page, 'labels-kanban.png')
  out.kanbanRawKeys = await scanRawKeys(page)

  // 2. List view — chips in rows
  await page.getByRole('button', { name: /Liste/i }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(1200)
  out.listLabelChips = await page.locator('text=/^Feature$|^Bug$|^Design$|^Blockiert$/').count().catch(() => -1)
  await shot(page, 'labels-list.png')

  // 3. Task detail (tsk-002 → Feature + Blockiert) + picker
  await page.goto(`${BASE}/#/work/projects/prj-001/tasks/tsk-002`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  out.detailHasLabelsLabel = await page.getByText(/^Labels$/).first().isVisible().catch(() => false)
  out.detailChips = await page.locator('text=/^Feature$|^Blockiert$/').count().catch(() => -1)
  await shot(page, 'labels-detail.png')
  // open picker
  await page.getByText(/^Label$/).first().click({ timeout: 3000 }).catch((e) => { out.pickerErr = String(e).split('\n')[0] })
  await page.waitForTimeout(600)
  out.pickerOpts = await page.locator('text=/^Bug$|^Feature$|^Design$|^Blockiert$/').count().catch(() => -1)
  await shot(page, 'labels-picker.png')
  out.detailRawKeys = await scanRawKeys(page)

  // 4. Search — chips + label filter
  await page.goto(`${BASE}/#/work/search`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(1500)
  await page.getByPlaceholder(/Suchen|Aufgaben|search/i).first().fill('a').catch(() => {})
  await page.waitForTimeout(400)
  await page.keyboard.type('rchitektur').catch(() => {})
  await page.waitForTimeout(1200)
  out.searchLabelFilterVisible = await page.getByRole('button', { name: /^Labels$/ }).first().isVisible().catch(() => false)
  await shot(page, 'labels-search.png')
  // open label filter
  await page.getByRole('button', { name: /^Labels$/ }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(500)
  await shot(page, 'labels-search-filter.png')
  out.searchRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.pageErrors = [...new Set(errors)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
