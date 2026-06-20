import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b2-1-columns')
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

// Open the draft (Helpdesk KW 24) -> opens in read mode, switch to edit.
const card = page.getByRole('button').filter({ hasText: /Helpdesk-Auslastung KW 24/ }).first()
await card.click().catch(() => {})
await page.waitForTimeout(1200)
const editToggle = page.getByRole('button', { name: /^Bearbeiten$/ }).first()
if (await editToggle.count()) await editToggle.click().catch(() => {})
await page.waitForTimeout(800)
out.editor = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-1-editor.png'), fullPage: true })

// Open the row insert picker (shows block types + Layout: 2/3 columns).
const addBtn = page.getByRole('button', { name: /Block einfügen/ }).first()
await addBtn.click().catch(() => {})
await page.waitForTimeout(400)
await page.screenshot({ path: resolve(outDir, 'b2-1-insert.png'), fullPage: true })

// Add a 2-column row.
const twoCol = page.getByRole('button', { name: /^2 Spalten$/ }).first()
await twoCol.click().catch(() => {})
await page.waitForTimeout(500)
out.afterTwoCol = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, 'b2-1-2col-row.png'), fullPage: true })

// Open the row options menu on the newly added (last) row.
const rowMenu = page.getByRole('button', { name: /Zeilen-Optionen/ }).last()
await rowMenu.click({ force: true }).catch(() => {})
await page.waitForTimeout(400)
await page.screenshot({ path: resolve(outDir, 'b2-1-rowmenu.png'), fullPage: true })

// Apply the 60 / 40 width preset.
const preset = page.getByRole('button', { name: /60 \/ 40/ }).first()
await preset.click().catch(() => {})
await page.waitForTimeout(400)
await page.screenshot({ path: resolve(outDir, 'b2-1-width-6040.png'), fullPage: true })

out.errs = errs.length
console.log(JSON.stringify(out, null, 2))
await browser.close()
