/** Second pass: expand the history (the fourth route) and confirm the state
 *  survives closing and reopening the contact. */
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
const BASE = process.env.QA_BASE || 'http://localhost:5180'
const outDir = resolve('.qa-screenshots/consent-wired')
const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 620, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', e => errors.push(String(e)))
const out = {}
try {
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(3500)
  await page.getByText('Karl Bauer').first().click({ timeout: 8000 })
  await page.waitForTimeout(1500)
  await page.getByText(/DSGVO-Einwilligungen/i).first().scrollIntoViewIfNeeded().catch(()=>{})
  await page.waitForTimeout(600)
  // grant one so there is something to show history for
  const g = page.getByRole('button', { name: /^Erteilen$/ }).first()
  if (await g.count()) { await g.click(); await page.waitForTimeout(400)
    await page.getByRole('button', { name: /^Bestätigen$/ }).first().click(); await page.waitForTimeout(1200) }
  // expand history
  const hist = page.getByRole('button', { name: /Verlauf/ }).first()
  out.historyLinkFound = await hist.count() > 0
  if (out.historyLinkFound) { await hist.click(); await page.waitForTimeout(1200)
    await page.screenshot({ path: resolve(outDir, '05-history.png') }) }
  const txt = await page.evaluate(() => document.body.innerText)
  out.rawKeys = (txt.match(/kontakte\.consent\.[a-zA-Z_.]+/g) || []).slice(0,5)
  out.noHistoryShown = /Kein Verlauf/.test(txt)
  out.countBeforeClose = (txt.match(/(\d+)\s+erteilt/) || [])[1] ?? null
  // close the detail panel and reopen -> state must persist in the demo store
  await page.keyboard.press('Escape'); await page.waitForTimeout(900)
  await page.getByText('Karl Bauer').first().click({ timeout: 8000 }); await page.waitForTimeout(1800)
  await page.getByText(/DSGVO-Einwilligungen/i).first().scrollIntoViewIfNeeded().catch(()=>{})
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '06-reopened.png') })
  const txt2 = await page.evaluate(() => document.body.innerText)
  out.countAfterReopen = (txt2.match(/(\d+)\s+erteilt/) || [])[1] ?? null
  out.errors = errors.slice(0,5)
} catch (e) { out.fatal = String(e).split('\n')[0]; out.errors = errors.slice(0,5)
  await page.screenshot({ path: resolve(outDir,'fatal2.png') }).catch(()=>{}) }
finally { await browser.close() }
console.log(JSON.stringify(out, null, 2))
