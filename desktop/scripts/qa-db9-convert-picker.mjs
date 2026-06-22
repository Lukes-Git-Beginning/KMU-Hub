import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db9-convert-picker')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(document|blocks|wiki|common)\.[a-z]+\.[a-z._]+/i

function findRawKeys(reSource) {
  const rx = new RegExp(reSource, 'i')
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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: 2 })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

await page.goto(`${BASE}/#/wiki?a=wart-001`, { waitUntil: 'domcontentloaded' })
await page.getByRole('heading', { name: /Willkommen im Cosmi-Wiki/ }).first().waitFor({ timeout: 15000 }).catch(() => {})
const editBtn = page.locator('button[title="Bearbeiten"]').first()
await editBtn.click().catch(() => {})
await page.waitForTimeout(1600)

// 1) Grouped insert menu.
await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
await page.waitForTimeout(300)
await page.getByRole('button', { name: /Block einfügen/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
out.hasContentGroup = await page.getByText('Inhalt', { exact: true }).count()
out.hasLayoutGroup = await page.getByText('Layout', { exact: true }).count()
out.pickerRaw = await page.evaluate(findRawKeys, RAW.source)
const picker = page.locator('div.rounded-xl:has(> div > span)').filter({ hasText: 'Inhalt' }).first()
await picker.screenshot({ path: resolve(outDir, 'zoom-picker-grouped.png') }).catch(() => {})
await page.screenshot({ path: resolve(outDir, '01-picker.png'), fullPage: false })

// Close the picker.
await page.locator('button[aria-label="Abbrechen"]').first().click().catch(() => {})
await page.waitForTimeout(300)

// 2) Convert: open a block's convert popover, capture targets, then convert
//    the welcome H1 heading into a quote and confirm the text carried over.
await page.evaluate(() => window.scrollTo(0, 0))
await page.waitForTimeout(300)
const h1Input = page.locator('input[value="Willkommen im Cosmi-Wiki"]').first()
const h1Block = h1Input.locator('xpath=ancestor::div[contains(@class,"group/block")][1]')
await h1Block.hover().catch(() => {})
await page.waitForTimeout(200)
const convertBtn = h1Block.locator('button[aria-label="Block umwandeln"]').first()
await convertBtn.click({ force: true }).catch(() => {})
await page.waitForTimeout(400)
out.convertTargets = await page.locator('button:has-text("Zitat"), button:has-text("Hinweis"), button:has-text("Liste")').count()
await page.screenshot({ path: resolve(outDir, '02-convert-popover.png'), fullPage: false })

await page.getByRole('button', { name: /^Zitat$/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
// After conversion the heading input is gone; a quote textarea holds the text.
out.quoteTextareaValue = await page
  .locator('textarea')
  .evaluateAll((els) => els.map((e) => e.value).find((v) => /Willkommen im Cosmi-Wiki/.test(v)) || '')
  .catch(() => '')
out.editRaw = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '03-converted.png'), fullPage: false })

out.errs = errs.length
out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
