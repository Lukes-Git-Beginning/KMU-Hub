/**
 * QA — notifications N-1: schema fix + seed upgrade.
 * Verifies: bell unread count > 0, unread cards marked (border-l-primary),
 * priority colours + module badges present, preferences panel opens without
 * crash (array schema) and a toggle survives a refetch.
 * Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/notif-n1')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(notifications\.[a-z]|\{\{)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim()).slice(0, 8)
  }, RAW_RE.source)
}

const browser = await chromium.launch()
const out = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage(); const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })

try {
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)

  // Bell unread badge in the header
  const bellBadge = await page.locator('header button.relative span').first().textContent().catch(() => null)
  // Center subtitle ("X ungelesene Benachrichtigungen")
  const subtitle = await page.locator('.max-w-3xl p.text-muted-foreground').first().textContent().catch(() => null)
  const unreadCards = await page.locator('.border-l-primary').count()
  const moduleBadges = await page.locator('.max-w-3xl .space-y-2 [class*="secondary"]').count()
  await page.screenshot({ path: resolve(outDir, '0-center-all.png') })
  out.push({ check: 'all-tab', bellBadge, subtitle, unreadCards, moduleBadges, rawKeys: await rawKeys(page) })

  // Unread tab
  await page.locator('button:has-text("Ungelesen")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1200)
  const unreadTabCards = await page.locator('.max-w-3xl .space-y-2 > *').count()
  await page.screenshot({ path: resolve(outDir, '1-center-unread.png') })
  out.push({ check: 'unread-tab', cards: unreadTabCards })

  // back to all, open preferences panel
  await page.locator('button:has-text("Alle")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(500)
  await page.locator('button:has-text("Einstellungen")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1500)
  const prefsTitle = await page.locator('text=Benachrichtigungseinstellungen').count()
  const checkboxes = await page.locator('input[type="checkbox"]').count()
  const groupHeaders = await page.locator('h4.capitalize').count()
  await page.screenshot({ path: resolve(outDir, '2-preferences.png'), fullPage: true })
  out.push({ check: 'preferences', prefsTitle, checkboxes, groupHeaders, pageErrorsSoFar: errs.slice(0, 3) })

  // toggle the first checkbox, then re-open panel to confirm it persisted
  if (checkboxes > 0) {
    const first = page.locator('input[type="checkbox"]').first()
    const before = await first.isChecked()
    await first.click({ timeout: 4000 }).catch(async () => { await first.evaluate((el) => el.click()) })
    await page.waitForTimeout(900)
    // close + reopen to force a refetch
    await page.locator('button:has-text("Einstellungen")').first().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(400)
    await page.locator('button:has-text("Einstellungen")').first().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(1200)
    const after = await page.locator('input[type="checkbox"]').first().isChecked().catch(() => null)
    await page.screenshot({ path: resolve(outDir, '3-prefs-after-toggle.png'), fullPage: true })
    out.push({ check: 'toggle-persist', before, after, flipped: before !== after })
  }

  out.push({ pageErrors: errs.slice(0, 5) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 5) })
} finally {
  await ctx.close(); await browser.close()
}

console.log(JSON.stringify(out, null, 2))
