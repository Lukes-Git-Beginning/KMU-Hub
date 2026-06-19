/**
 * QA — notifications N-4: sidebar per-module unread badges + module-settings entry.
 * Verifies: sidebar nav items show the live notification unread counts (e.g.
 * Kontakte 2, Aufgaben 1); the module-settings overlay lists "Benachrichtigungen"
 * and opens the embedded preferences. Sub-terminal → :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/notif-n4')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const out = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage(); const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)

  // read sidebar nav labels + their badge values
  const navBadges = await page.evaluate(() => {
    const out = []
    document.querySelectorAll('nav a').forEach((a) => {
      const labelEl = a.querySelector('span.flex-1') || a.querySelector('span')
      const badgeEl = a.querySelector('span[class*="bg-primary"]')
      if (labelEl && badgeEl) out.push({ label: labelEl.textContent.trim(), badge: badgeEl.textContent.trim() })
    })
    return out
  })
  await page.screenshot({ path: resolve(outDir, '0-sidebar.png') })
  out.push({ check: 'sidebar-badges', navBadges })

  // open the overlay from /notifications → the notifications entry is preselected
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  await page.locator('nav a:has-text("Modul-Einstellungen")').first().click({ timeout: 5000 }).catch(async () => {
    await page.locator('nav a:has-text("Einstellungen")').last().click({ timeout: 5000 }).catch(() => {})
  })
  await page.waitForTimeout(1500)
  const overlayOpen = await page.locator('text=Benachrichtigungen').count()
  await page.screenshot({ path: resolve(outDir, '1-settings-overlay.png') })

  // ensure the notifications entry is selected (overlay left-nav), then read the panel
  await page.locator('button:has-text("Benachrichtigungen")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const sectionTitle = await page.locator('text=Meine Benachrichtigungen').count()
  const matrixPresent = await page.locator('text=Nachrichten').count()
  const dndPresent = await page.locator('text=Bitte nicht stören').count()
  await page.screenshot({ path: resolve(outDir, '2-settings-notifications.png'), fullPage: true })
  out.push({ check: 'settings-entry', overlayOpen, sectionTitle, matrixPresent, dndPresent })

  out.push({ pageErrors: errs.slice(0, 6) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 6) })
} finally {
  await ctx.close(); await browser.close()
}

console.log(JSON.stringify(out, null, 2))
