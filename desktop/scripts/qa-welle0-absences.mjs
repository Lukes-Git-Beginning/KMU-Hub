/**
 * QA Welle 0 — Dashboard Absences widget
 * Verifies: Absences widget shows real people (not "0 abwesend") after MSW fix.
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

await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
// Wait for MSW + React Query to settle
await page.waitForTimeout(3500)

await page.screenshot({ path: resolve(outDir, 'welle0-dashboard-full.png'), fullPage: false })

// Grab text from absences widget area
const absenceText = await page.evaluate(() => {
  const allText = Array.from(document.querySelectorAll('*'))
    .filter(n => n.children.length === 0 && /abwesend|Abwesend|absent|Urlaub|Krank|Homeoffice/i.test(n.textContent || ''))
    .map(n => n.textContent.trim())
    .filter(Boolean)
    .slice(0, 20)
  return allText
})

// Check for raw i18n keys
const rawKeys = await page.evaluate(() => {
  return Array.from(document.querySelectorAll('body *'))
    .filter(n => n.children.length === 0 && /dashboard\.[a-z]|widgets\.[a-z]/.test(n.textContent || ''))
    .map(n => n.textContent.trim())
    .slice(0, 10)
})

console.log('Absence-related text found:', JSON.stringify(absenceText, null, 2))
console.log('Raw i18n keys:', JSON.stringify(rawKeys, null, 2))
console.log('Page errors:', errors.length ? errors : 'none')

await browser.close()
console.log('Screenshot saved to scripts/screenshots/welle0-dashboard-full.png')
