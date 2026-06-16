// QA finanzen P2.5e — Hours→Invoice echter Create end-to-end:
// preview → Button disabled ohne Kunde → Kunde eingeben → enabled → erstellen →
// Dialog schließt → neue Draft-Rechnung erscheint in der Rechnungsliste.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

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
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3000)

  // invoice count before
  await page.getByRole('button', { name: /^Rechnungen|^gebote/ }).first().click({ timeout: 6000 }).catch(() => {})
  await page.waitForTimeout(800)
  out.invoiceCountBefore = await page.locator('div[role="button"].cursor-pointer, tbody tr').count()

  await page.getByRole('button', { name: /Stunden abrechnen/ }).first().click({ timeout: 6000 })
  await page.waitForTimeout(800)
  await page.getByRole('button', { name: /Alle auswählen/ }).first().click({ timeout: 4000 })
  await page.waitForTimeout(300)
  await page.getByRole('button', { name: /^Vorschau/ }).first().click({ timeout: 4000 })
  await page.waitForTimeout(700)

  const createBtn = page.getByRole('button', { name: /^Neue Rechnung$/ }).last()
  out.createBtnFound = await createBtn.count()
  out.disabledNoCustomer = await createBtn.isDisabled().catch((e) => 'err:' + String(e).split('\n')[0])

  const cust = page.locator('[role="dialog"] input[type="text"]').first()
  await cust.fill('Muster Kunde GmbH')
  await page.waitForTimeout(300)
  out.disabledWithCustomer = await createBtn.isDisabled().catch((e) => 'err:' + String(e).split('\n')[0])

  await createBtn.click({ timeout: 4000 })
  await page.waitForTimeout(1500)
  out.dialogClosed = !(await page.locator('[role="dialog"]').count())

  // back on invoices tab — count should be +1, new draft for "Muster Kunde GmbH"
  await page.getByRole('button', { name: /^Rechnungen|^gebote/ }).first().click({ timeout: 6000 }).catch(() => {})
  await page.waitForTimeout(900)
  out.invoiceCountAfter = await page.locator('div[role="button"].cursor-pointer, tbody tr').count()
  out.hasMusterKunde = (await page.evaluate(() => document.body.innerText)).includes('Muster Kunde GmbH')
  await page.screenshot({ path: resolve(outDir, 'p25e-hours-created-invoice.png') })
} catch (err) {
  out.fatal = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
