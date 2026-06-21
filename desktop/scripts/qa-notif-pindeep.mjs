// N-4 QA: pin/dismiss persistence (survives a remount/refetch) + deep links to
// concrete items render without error. Dev server must be on :5174.
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
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const has = async (loc) => (await loc.count().catch(() => 0)) > 0
const firstCardTitle = () =>
  page.evaluate(() => document.querySelector('[role="button"][aria-label]')?.getAttribute('aria-label') || '')

const openCenter = async () => {
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2500)
}

try {
  await openCenter()

  // ── PIN: pin the "Urlaubsantrag" row (mid-list) → it should float to top + persist ──
  const urlaubCard = page.locator('[role="button"][aria-label*="Urlaubsantrag"]').first()
  if (await has(urlaubCard)) { await urlaubCard.click(); await page.waitForTimeout(600) }
  const pinBtn = page.locator('button', { hasText: /Anpinnen|Anheften/ }).first()
  if (await has(pinBtn)) { await pinBtn.click(); await page.waitForTimeout(900) }
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(500)
  out.firstAfterPin = await firstCardTitle()
  await page.screenshot({ path: resolve(outDir, '10-pinned.png'), fullPage: true })

  // Remount the center → pin must survive (server-persisted)
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1200)
  await openCenter()
  out.firstAfterRemount = await firstCardTitle()

  // ── DISMISS: dismiss the current first row → must disappear + stay gone ──
  const before = await firstCardTitle()
  const card = page.locator('[role="button"][aria-label]').first()
  if (await has(card)) { await card.click(); await page.waitForTimeout(500) }
  const dismissBtn = page.locator('button', { hasText: /Ignorieren/ }).first()
  if (await has(dismissBtn)) { await dismissBtn.click(); await page.waitForTimeout(900) }
  const afterDismiss = await firstCardTitle()
  out.dismissChangedFirst = before !== afterDismiss && Boolean(afterDismiss)
  // Remount → dismissed stays gone
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1000)
  await openCenter()
  out.dismissedTitle = before
  out.dismissedStillGone = await page.evaluate((b) => !document.body.innerText.includes(b), before.slice(0, 24))

  // ── DEEP LINKS: each must render without a page error ──
  const links = ['/kontakte/pipeline/dl-016', '/team/member/usr-e6', '/finanzen?invoice=inv-007', '/kommunikation?bereich=team']
  out.deepLinks = {}
  for (const l of links) {
    const errBefore = errs.length
    await page.goto(`${BASE}/#${l}`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2200)
    const bodyLen = await page.evaluate(() => document.body.innerText.trim().length)
    out.deepLinks[l] = { bodyLen, newErrors: errs.length - errBefore }
  }
  await page.screenshot({ path: resolve(outDir, '11-deeplink-last.png') })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'ERROR-n4.png') }).catch(() => {})
}
out.pageErrors = errs.slice(0, 10)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
