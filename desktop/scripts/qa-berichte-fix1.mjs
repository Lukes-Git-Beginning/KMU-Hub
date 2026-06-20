import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const outDir = resolve('.qa-screenshots/fix1-filter'); await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(berichte|common|shared)\.[a-z]+\.[a-z._]+/i
function findRawKeys(re){const rx=new RegExp(re,'i');return[...new Set(Array.from(document.querySelectorAll('body *')).filter(n=>n.children.length===0&&rx.test(n.textContent||'')).map(n=>n.textContent.trim()))].slice(0,12)}
const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 1100 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const p = await ctx.newPage()
const errs = []; p.on('pageerror', e=>errs.push(String(e)))
await p.goto('http://localhost:5173/#/berichte', { waitUntil: 'domcontentloaded' })
await p.waitForTimeout(2500)
const tab = p.getByRole('button', { name: /^Berichte$/ }).first(); if (await tab.count()) await tab.click().catch(()=>{})
await p.waitForTimeout(1000)
await p.getByRole('button').filter({ hasText: /Helpdesk-Auslastung KW 24/ }).first().click().catch(()=>{})
await p.waitForTimeout(1000)
await p.getByRole('button', { name: /^Bearbeiten$/ }).first().click().catch(()=>{})
await p.waitForTimeout(600)
await p.getByRole('button', { name: /Block einfügen/ }).first().click().catch(()=>{})
await p.waitForTimeout(250)
await p.getByRole('button', { name: /^Diagramm$/ }).first().click().catch(()=>{})
await p.waitForTimeout(400)
await p.getByRole('button', { name: /Grafik konfigurieren/ }).first().click().catch(()=>{})
await p.waitForTimeout(500)
await p.getByRole('button', { name: /^Neue Grafik$/ }).first().click().catch(()=>{})
await p.waitForTimeout(400)
const dialog = p.locator('[role=dialog]')
await dialog.getByRole('button', { name: /Rechnungen/ }).first().click().catch(()=>{})
await p.waitForTimeout(700)
await dialog.getByRole('button', { name: /^Status$/ }).first().click().catch(()=>{}); await p.waitForTimeout(200)
await dialog.getByRole('button', { name: /Netto-Betrag/ }).first().click().catch(()=>{}); await p.waitForTimeout(800)
const filterPresent = await dialog.getByText(/^Filter$/).count()
await dialog.getByRole('button', { name: /Hinzufügen/ }).first().click().catch(()=>{})
await p.waitForTimeout(300)
const fieldSelect = dialog.locator('select').filter({ has: p.locator('option', { hasText: 'Kunde' }) }).first()
await fieldSelect.selectOption({ label: 'Kunde' }).catch(async()=>{ await fieldSelect.selectOption('customer_name').catch(()=>{}) })
await p.waitForTimeout(600)
const datalist = await p.evaluate(() => {
  const dl = document.querySelector('datalist#flt-customer_name')
  return { exists: !!dl, options: dl ? dl.querySelectorAll('option').length : 0, sample: dl ? [...dl.querySelectorAll('option')].slice(0,4).map(o=>o.value) : [] }
})
await p.screenshot({ path: resolve(outDir, 'fix1-filter.png') })
console.log(JSON.stringify({ filterPresent, datalist, rawKeys: await p.evaluate(findRawKeys, RAW.source), errs: errs.length }, null, 2))
await b.close()
