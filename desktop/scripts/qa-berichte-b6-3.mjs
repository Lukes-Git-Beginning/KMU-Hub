import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b6-3-save-doc')
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

await page.getByRole('button', { name: /^Teilen$/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.saveEntry = await page.getByRole('button', { name: /Als PDF in Dokumente/ }).count()
await page.getByRole('button', { name: /Als PDF in Dokumente/ }).first().click().catch(() => {})
await page.waitForTimeout(700)
out.dialogOpen = await page.getByText(/In Dokumente ablegen/).count()
out.dialogRawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, 'b6-3-dialog.png') })

await page.getByRole('button', { name: /^In Dokumente ablegen$/ }).first().click().catch(() => {})
await page.waitForTimeout(1000)
out.toast = await page.getByText(/abgelegt/).count()
await page.screenshot({ path: resolve(outDir, 'b6-3-saved.png') })

// Verify the file actually landed in the "Berichte" folder via the API.
out.filed = await page.evaluate(async () => {
  try {
    const r = await fetch('http://localhost:8080/api/v1/documents/files?folder_id=fld-berichte')
    const j = await r.json()
    const f = (j.files ?? []).find((x) => /Verkaufsbericht Q2 2026/.test(x.name))
    return f
      ? { name: f.name, mime: f.mime_type, tags: (f.tags ?? []).map((t) => t.name), by: f.created_by_name }
      : 'not-found'
  } catch (e) {
    return 'fetch-error: ' + String(e)
  }
})

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
