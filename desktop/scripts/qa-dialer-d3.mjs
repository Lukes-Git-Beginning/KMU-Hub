// QA — dialer D-3 contact detail modal: clickable rows + call history (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/dialer-d3')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 940 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(9000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const step = async (name, fn) => {
  try { await fn() } catch (err) { out[`ERR_${name}`] = String(err).split('\n')[0] }
}

await page.goto(`${BASE}/#/dialer/campaigns/dlr-camp-001`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)

// ── Open detail of a contact WITH call history (Hans Müller / cc001) ──
await step('openWithHistory', async () => {
  await page.getByText('Hans Müller', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  const body = await bodyText()
  out.modalCallHistory = /Anruf-Historie/.test(body)
  out.modalLastOutcome = /Letztes Ergebnis/.test(body)
  out.modalHasOutcomes = /Erreicht|Wiedervorlage/.test(body)
  out.modalOpenInCrm = /Im CRM öffnen/.test(body)
  out.modalShowsCompany = /Gruber Maschinenbau AG/.test(body)
  out.modalNoRawKeys = !/dialer\.contactDetail\./.test(body)
  await page.screenshot({ path: resolve(outDir, 'd3-1-detail-history.png'), fullPage: false })
})

// ── Close + open a PENDING contact (no history) → empty state ──
await step('openEmpty', async () => {
  await page.keyboard.press('Escape')
  await page.waitForTimeout(700)
  await page.getByText('Sabine Wagner', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  out.emptyStateShown = /Noch keine Anrufe protokolliert/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'd3-2-detail-empty.png'), fullPage: false })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)
})

out.pageErrors = errs.slice(0, 8)
out.rawKeys = await page.evaluate(() => {
  const all = Array.from(document.querySelectorAll('body *'))
    .filter((n) => n.children.length === 0)
    .map((n) => (n.textContent || '').trim())
  return [...new Set(all.filter((t) => /^(dialer|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 12)
})
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
