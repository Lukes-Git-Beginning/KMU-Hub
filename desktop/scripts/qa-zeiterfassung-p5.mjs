// QA P5 zeiterfassung — team view + approval workflow + week submission.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/ze-p5')
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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2200)
  await dismiss(page)

  // 1. Team tab present (admin = lead) → open it
  await page.getByRole('tab', { name: /^team$/i }).first().click()
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, '01-team.png') })
  let body = (await page.locator('body').innerText()).toLowerCase()
  results.push({ shot: 'team', errors: errors.length, rawKeys: RAW.filter((p) => body.includes(p)) })

  // 2. Approve first submitted member
  await page.getByRole('button', { name: /freigeben/i }).first().click()
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, '02-after-approve.png') })
  results.push({ shot: 'after approve', errors: errors.length })

  // 3. Week submit banner
  await page.getByRole('tab', { name: /^woche$/i }).first().click()
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, '03-week-banner.png') })
  body = (await page.locator('body').innerText()).toLowerCase()
  results.push({ shot: 'week banner', errors: errors.length, rawKeys: RAW.filter((p) => body.includes(p)) })

  // 4. Submit week
  await page.getByRole('button', { name: /woche einreichen/i }).first().click()
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, '04-after-submit.png') })
  results.push({ shot: 'after submit', errors: errors.length })
} catch (err) {
  results.push({ error: String(err).slice(0, 200) })
} finally {
  await ctx.close()
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
