// QA P2 zeiterfassung — manual entry dialog + project/customer attribution on entries.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/ze-p2')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = ['zeiterfassung.', 'profil.zeiterfassung.', 'api.hr.']

async function dismiss(page) {
  for (const name of [/überspringen/i, /skip/i]) {
    const b = page.getByRole('button', { name }).first()
    try { if (await b.isVisible({ timeout: 500 })) { await b.click(); await page.waitForTimeout(300) } } catch (e) {}
  }
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2200)
  await dismiss(page)
  await page.waitForTimeout(1200)

  // 1. Entries with project/customer/billable
  await page.screenshot({ path: resolve(outDir, '01-entries.png') })
  let body = (await page.locator('body').innerText()).toLowerCase()
  results.push({ shot: 'entries', errors: errors.length, rawKeys: RAW.filter((p) => body.includes(p)) })

  // 2. Open manual entry dialog
  await page.getByRole('button', { name: /neuer eintrag/i }).first().click()
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '02-dialog.png') })
  body = (await page.locator('body').innerText()).toLowerCase()
  results.push({ shot: 'dialog', errors: errors.length, rawKeys: RAW.filter((p) => body.includes(p)) })

  // 3. Select a project, then submit
  try {
    await page.getByRole('combobox').first().click()
    await page.waitForTimeout(400)
    await page.getByRole('option').nth(1).click()
    await page.waitForTimeout(300)
  } catch (e) { results.push({ note: 'project select skipped: ' + String(e).slice(0, 80) }) }
  await page.screenshot({ path: resolve(outDir, '03-dialog-filled.png') })

  await page.getByRole('button', { name: /eintrag speichern/i }).first().click()
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, '04-after-submit.png') })
  body = (await page.locator('body').innerText()).toLowerCase()
  results.push({ shot: 'after submit', errors: errors.length, rawKeys: RAW.filter((p) => body.includes(p)) })
} catch (err) {
  results.push({ error: String(err) })
} finally {
  await ctx.close()
}

// Dialog @ small width
const ctx2 = await browser.newContext({ viewport: { width: 500, height: 900 } })
await ctx2.addInitScript(ELECTRON_STUB)
await ctx2.addInitScript(SUPPRESS)
const page2 = await ctx2.newPage()
try {
  await page2.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page2.waitForTimeout(2200)
  await dismiss(page2)
  await page2.getByRole('button', { name: /neuer eintrag/i }).first().click()
  await page2.waitForTimeout(700)
  await page2.screenshot({ path: resolve(outDir, '05-dialog-small.png') })
  results.push({ shot: 'dialog small', ok: true })
} catch (err) { results.push({ shot: 'dialog small', error: String(err).slice(0, 100) }) } finally { await ctx2.close() }

await browser.close()
console.log(JSON.stringify(results, null, 2))
