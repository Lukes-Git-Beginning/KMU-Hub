// QA feedback fixes — stateful clock (in/out works) + settings (no tenant weekly, read-only personal).
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/ze-fix')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

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

  // 1. Clocked-in state (toolbar shows Ausstempeln)
  await page.screenshot({ path: resolve(outDir, '01-clocked-in.png') })

  // 2. Clock out → should become idle (Einstempeln)
  await page.getByRole('button', { name: /^ausstempeln$/i }).first().click()
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, '02-after-clockout.png') })
  const afterOut = (await page.locator('main').innerText()).toLowerCase()
  results.push({ step: 'after clock-out', errors: errors.length, idleVisible: afterOut.includes('einstempeln') })

  // 3. Clock in again → timer restarts
  await page.getByRole('button', { name: /^einstempeln$/i }).first().click()
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, '03-after-clockin.png') })
  const afterIn = (await page.locator('main').innerText()).toLowerCase()
  results.push({ step: 'after clock-in', errors: errors.length, clockedInVisible: afterIn.includes('ausstempeln') })

  // 4. Settings panel (weekly removed from tenant, read-only in personal)
  await page.getByText(/Modul-Einstellungen/i).first().click()
  await page.waitForTimeout(900)
  await page.screenshot({ path: resolve(outDir, '04-settings.png') })
  const settings = (await page.locator('body').innerText()).toLowerCase()
  results.push({ step: 'settings', errors: errors.length, hasReadonlyWeekly: settings.includes('wochensoll'), rawKeys: ['zeiterfassung.', 'moduleSettings.'].filter((p) => settings.includes(p)) })
} catch (err) {
  results.push({ error: String(err).slice(0, 200) })
} finally {
  await ctx.close()
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
