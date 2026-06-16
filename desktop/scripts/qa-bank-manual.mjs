// QA isoliert: manuelle Zuordnung eines offenen Eingangs zu einer Rechnung.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUP = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const dlg = (page) => page.evaluate(() => { const d = document.querySelector('[role="dialog"]'); return d ? d.innerText.replace(/\n{2,}/g,'\n').slice(0,900) : null })

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 960 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(SUP)
const page = await ctx.newPage()
const errors = []; page.on('pageerror', (e) => errors.push(String(e)))
const out = {}
try {
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3000)
  await page.getByRole('button', { name: /^Banking/ }).first().click({ timeout: 6000 })
  await page.waitForTimeout(700)
  await page.getByRole('button', { name: /^Offen \(/ }).first().click({ timeout: 4000 })
  await page.waitForTimeout(500)
  out.manualBtn = await page.locator('button[title="Manuell zuordnen"]').count()
  // open the open-incoming row (Schwarzwald) via the whole-row click
  const row = page.locator('div[role="button"].cursor-pointer', { hasText: 'Schwarzwald' }).first()
  out.rowFound = await row.count()
  await row.click({ timeout: 5000 })
  await page.waitForTimeout(700)
  const dt = await dlg(page)
  out.hasAssignSection = dt ? dt.includes('Rechnung zuordnen') : false
  out.detailText = dt
  await page.screenshot({ path: resolve(outDir, 'bank-manual-open.png') })
  // click first open invoice in the assign list
  const invBtn = page.locator('[role="dialog"] button:has(.font-mono)').first()
  out.invoiceOptions = await page.locator('[role="dialog"] button:has(.font-mono)').count()
  await invBtn.click({ timeout: 4000 })
  await page.waitForTimeout(1000)
  out.dialogClosed = !(await page.locator('[role="dialog"]').count())
  // reopen detail of that tx — should now be matched
  await page.getByRole('button', { name: /^Alle \(/ }).first().click({ timeout: 4000 }).catch(()=>{})
  await page.waitForTimeout(400)
  out.afterCounts = await page.evaluate(() => [...document.querySelectorAll('button')].map(b=>b.textContent.trim()).filter(t=>/^(Zugeordnet|Offen) \(/.test(t)))
  await page.screenshot({ path: resolve(outDir, 'bank-manual-after.png') })
} catch (e) { out.error = String(e).split('\n')[0] }
out.pageErrors = errors.slice(0, 6)
await ctx.close(); await browser.close()
console.log(JSON.stringify(out, null, 2))
