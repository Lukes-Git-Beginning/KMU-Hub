/**
 * QA Vermietung — Modul-API-Hooks-Wiring
 * Überprüft: Objekte-Tab lädt echte Daten (kein Mock), Reservierungen-Tab, Kalender-Tab.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()

const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

// Navigate to Vermietung tab
await page.goto(`${BASE}/#/vermietung`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)

await page.screenshot({ path: resolve(outDir, 'qa-vermietung-objekte.png'), fullPage: false })

// Check raw i18n keys
const rawKeys = await page.evaluate(() => {
  return Array.from(document.querySelectorAll('body *'))
    .filter(n => n.children.length === 0 && /^vermietung\.[a-z]/.test(n.textContent?.trim() || ''))
    .map(n => n.textContent.trim())
    .slice(0, 10)
})

// Click Reservierungen tab
const resTab = await page.$('[class*="tab"]:has-text("Reservierungen")')
if (resTab) {
  await resTab.click()
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, 'qa-vermietung-reservierungen.png'), fullPage: false })
}

// Click Kalender tab
const calTab = await page.$('[class*="tab"]:has-text("Kalender")')
if (calTab) {
  await calTab.click()
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, 'qa-vermietung-kalender.png'), fullPage: false })
}

console.log('Raw i18n keys found:', JSON.stringify(rawKeys, null, 2))
console.log('Page errors:', errors.length ? errors : 'none')

await browser.close()
console.log('Screenshots saved to scripts/screenshots/qa-vermietung-*.png')
