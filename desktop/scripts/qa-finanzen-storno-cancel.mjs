import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

// Storno runs through POST /invoices/{id}/cancel: the dialog shows the issued
// invoice read-only (no reason field, no editable positions), confirming sets
// the invoice to cancelled and a storno credit note appears.
const BASE = process.env.QA_BASE || 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/finanzen-storno-cancel')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(finanzen|common)\.[a-z]+\.[a-z._]+/i

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
page.setDefaultTimeout(8000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
await page.locator('.cosmi-startup').waitFor({ state: 'detached', timeout: 20000 }).catch(() => {})
await page.getByRole('button', { name: /^Rechnungen/ }).first().click().catch(() => {})
await page.waitForTimeout(1200)

// an issued (Versendet) demo invoice; rows are a div grid, so climb from the
// number to the nearest ancestor that holds the row's action button
const invoiceNumber = 'RE-2026-005'
out.invoiceNumber = invoiceNumber
const rowOf = (nr) => page.locator(`xpath=//*[normalize-space(text())="${nr}"]/ancestor::*[descendant::button][1]`)
out.rowText = (await rowOf(invoiceNumber).innerText().catch(() => '')).replace(/\s+/g, ' ').slice(0, 120)
// an overlay intercepts pointer events on the kebab; Radix also opens on Enter
const kebab = rowOf(invoiceNumber).locator('button').last()
await kebab.focus()
await page.keyboard.press('Enter')
await page.waitForTimeout(400)
await page.getByRole('menuitem', { name: /^Stornieren$/ }).click()
await page.waitForTimeout(700)

const dialog = page.getByRole('dialog')
out.dialogTitle = await dialog.getByText(/Rechnung stornieren/).count()
out.notice = await dialog.getByText(/Storno-Gutschrift über den vollen Betrag/).count()
out.reasonField = await dialog.locator('textarea').count()
out.editableInputs = await dialog.locator('input').count()
out.positionsLabel = await dialog.getByText(/^Positionen$/).count()
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '1-dialog.png') })

await dialog.getByRole('button', { name: /^Stornieren$/ }).click()
await page.waitForTimeout(1000)
out.toast = await page.getByText(new RegExp(`Rechnung ${invoiceNumber} storniert`)).count()
// Enter on the kebab also opens the invoice detail, which stays open behind
out.stornoDialogClosed = (await page.getByText(/^Rechnung stornieren$/).count()) === 0
out.statusCancelled = await page.getByText(/^Storniert$/).count()
await page.screenshot({ path: resolve(outDir, '2-cancelled.png') })

await page.keyboard.press('Escape')
await page.waitForTimeout(500)
await page.getByRole('button', { name: /^Gutschriften/ }).first().click().catch(() => {})
await page.waitForTimeout(900)
out.stornoNoteForInvoice = await page.getByText(invoiceNumber).count()
out.stornoBadges = await page.getByText(/^Storno$/).count()
await page.screenshot({ path: resolve(outDir, '3-credit-notes.png') })

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
