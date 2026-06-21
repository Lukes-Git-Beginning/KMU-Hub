// DND-suppression QA: enable DND before the live arrivals reveal, then assert
// no toast surfaces over the full arrival window. Dev server must be on :5174.
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/notifications')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const has = async (loc) => (await loc.count().catch(() => 0)) > 0

try {
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1800)
  // Open the notifications settings tab (button, not the page heading).
  const tab = page.locator('button', { hasText: 'Benachrichtigungen' }).first()
  if (await has(tab)) { await tab.click(); await page.waitForTimeout(800) }
  // Enable DND quickly — before the 7s/16s arrivals reveal.
  const enable = page.locator('button', { hasText: 'Aktivieren' }).first()
  if (await has(enable)) { await enable.click(); await page.waitForTimeout(600) }
  out.dndActive = await page.evaluate(() => /\bAktiv\b/.test(document.body.innerText))
  await page.screenshot({ path: resolve(outDir, '7-dnd-enabled.png'), fullPage: true })

  // Wait out the entire arrival window; with DND active no NOTIFICATION toast
  // may appear. (The settings "DND aktiviert" success toast is not a
  // notification toast — exclude it by matching the notification action label.)
  let notifToasts = 0
  let anyToasts = 0
  const deadline = Date.now() + 28000
  while (Date.now() < deadline) {
    const counts = await page.evaluate(() => {
      const all = Array.from(document.querySelectorAll('[data-sonner-toast]'))
      const notif = all.filter((t) => /Öffnen|Als gelesen markieren/.test(t.textContent || ''))
      return { any: all.length, notif: notif.length }
    }).catch(() => ({ any: 0, notif: 0 }))
    if (counts.notif > notifToasts) notifToasts = counts.notif
    if (counts.any > anyToasts) anyToasts = counts.any
    await page.waitForTimeout(1500)
  }
  out.notifToastsWhileDND = notifToasts
  out.anyToastsWhileDND = anyToasts
  await page.screenshot({ path: resolve(outDir, '8-dnd-no-toast.png') })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'ERROR-dnd.png') }).catch(() => {})
}
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
