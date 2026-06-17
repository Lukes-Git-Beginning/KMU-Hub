/**
 * QA — notifications F5: click a notification → expand with actions (open/pin/dismiss).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/notifications-actions')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(notifications\.[a-z])/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('main *, .max-w-3xl *'))
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

try {
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2800)
  const cardCount = await page.locator('.cursor-pointer').count()
  await page.screenshot({ path: resolve(outDir, '0-center.png') })

  // click first notification card → should expand with action bar
  const card = page.locator('.cursor-pointer').first()
  await card.evaluate((el) => el.click()).catch(async () => { await card.click({ timeout: 5000 }) })
  await page.waitForTimeout(600)
  const hasOpen = await page.locator('button:has-text("Öffnen")').count()
  const hasPin = await page.locator('button:has-text("Anpinnen")').count()
  const hasDismiss = await page.locator('button:has-text("Ignorieren")').count()
  await page.screenshot({ path: resolve(outDir, '1-expanded.png') })
  out.push({ check: 'expand', cardCount, openBtn: hasOpen, pinBtn: hasPin, dismissBtn: hasDismiss, rawKeys: await rawKeys(page) })

  // pin it
  if (hasPin > 0) {
    await page.locator('button:has-text("Anpinnen")').first().click({ timeout: 4000 })
    await page.waitForTimeout(500)
    const hasUnpin = await page.locator('button:has-text("Lösen")').count()
    await page.screenshot({ path: resolve(outDir, '2-pinned.png') })
    out.push({ check: 'pin', becameUnpin: hasUnpin > 0 })
  }

  // dismiss: expand first card again, click Ignorieren → count drops
  const before = await page.locator('.cursor-pointer').count()
  const firstCard = page.locator('.cursor-pointer').first()
  await firstCard.evaluate((el) => el.click()).catch(() => {})
  await page.waitForTimeout(400)
  const dismissBtn = page.locator('button:has-text("Ignorieren")').first()
  if (await dismissBtn.isVisible().catch(() => false)) {
    await dismissBtn.click({ timeout: 4000 })
    await page.waitForTimeout(500)
    const after = await page.locator('.cursor-pointer').count()
    out.push({ check: 'dismiss', before, after, dropped: after < before })
  }
  out.push({ pageErrors: errs.slice(0, 3) })
} catch (e) { out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 3) }) }
finally { await ctx.close(); await browser.close() }

console.log(JSON.stringify(out, null, 2))
