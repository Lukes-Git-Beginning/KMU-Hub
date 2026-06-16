// QA work W-4: "Stunden abrechnen" -> real draft invoice from MSW time entries.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(work|finanzen|common|shared)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function dialogText(page) {
  return page.evaluate(() => {
    const d = document.querySelector('[role="dialog"]')
    return d ? d.innerText.replace(/\n{2,}/g, '\n').slice(0, 900) : null
  })
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 960 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  await page.goto(`${BASE}/#/work/projects/prj-001`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3500)

  // open "Stunden abrechnen"
  await page.locator('button[title="Stunden abrechnen"]').first().evaluate((el) => el.click())
  await page.waitForTimeout(900)
  out.dialogOpen = !!(await page.locator('[role="dialog"]').count())
  const dt = await dialogText(page)
  out.dialogText = dt
  out.rawKeys = await scanRawKeys(page)
  // count entry rows in the dialog
  out.rowCount = await page.locator('[role="dialog"] div[role="button"]').count()
  out.customerPrefill = await page.locator('[role="dialog"] input[type="text"]').first().inputValue()
  await page.screenshot({ path: resolve(outDir, 'w4-1-dialog.png') })

  // create the invoice
  await page.getByRole('button', { name: /^Rechnung erstellen$/ }).first().evaluate((el) => el.click())
  await page.waitForTimeout(1500)
  out.toast = await page.evaluate(() => {
    const t = document.querySelector('[data-sonner-toast]')
    return t ? t.innerText : null
  })
  out.dialogClosed = !(await page.locator('[role="dialog"]').count())
  await page.screenshot({ path: resolve(outDir, 'w4-2-after-create.png') })

  // verify the draft appears in finanzen invoices
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2500)
  await page.getByRole('button', { name: /^Rechnungen/ }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1500)
  out.invoicesText = await page.evaluate(() => {
    const m = document.body.innerText.match(/RE-2026-\d+/g)
    return m ? [...new Set(m)] : []
  })
  out.hasDraftBadge = (await page.getByText('Entwurf', { exact: false }).count()) > 0
  await page.screenshot({ path: resolve(outDir, 'w4-3-finanzen-invoices.png') })
} catch (err) {
  out.fatal = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
