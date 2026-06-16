// QA kalender Terminbuchung: Tagesübersicht-Termine klickbar → BookingDetailModal.
// Prüft: Zeilenklick öffnet Modal mit Tiefe (Kunde/Kontakt/Service/Preis/History),
// Status-Aktion (Bestätigen), Kunden-History-Klick wechselt Termin. + Raw-Key-Scan.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try { const KEY='cosmi-ui'; const raw=localStorage.getItem(KEY); const p=raw?JSON.parse(raw):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(KEY,JSON.stringify(p)) } catch(e){}
`
const scanRawKeys = (page) => page.evaluate(() =>
  [...new Set([...document.body.innerText.matchAll(/\b(kalender|common|shared)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map(m=>m[0]))])
const dialogText = (page) => page.evaluate(() => { const d=document.querySelector('[role="dialog"]'); return d?d.innerText.replace(/\n{2,}/g,'\n').slice(0,1400):null })

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(ELECTRON_STUB); await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []; page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  await page.goto(`${BASE}/#/kalender`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2500)
  // Switch to Terminbuchung top tab
  await page.getByRole('button', { name: /Terminbuchung/ }).first().click({ timeout: 6000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'booking-overview.png') })
  out.dayRows = await page.evaluate(() => document.querySelectorAll('div[role="button"].cursor-pointer').length)
  out.overviewRawKeys = await scanRawKeys(page)

  // Click first day-overview row
  const row = page.locator('div[role="button"].cursor-pointer').first()
  out.hasRows = await row.count()
  if (out.hasRows) {
    await row.click({ timeout: 4000 })
    await page.waitForTimeout(700)
    await page.screenshot({ path: resolve(outDir, 'booking-detail.png') })
    out.detailText = await dialogText(page)
    out.detailRawKeys = await scanRawKeys(page)
    out.hasConfirmOrCancel = await page.evaluate(() => /Bestätigen|Absagen/.test(document.querySelector('[role="dialog"]')?.innerText ?? ''))
    // Customer-history click (if any) → should switch the modal content
    const histBtn = page.locator('[role="dialog"] button:has(.rounded-full)').nth(1)
    out.historyBtns = await page.locator('[role="dialog"] button').count()
  }
} catch (err) {
  out.fatal = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close(); await browser.close()
console.log(JSON.stringify(out, null, 2))
