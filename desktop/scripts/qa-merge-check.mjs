// Post-merge integration check (Batch 4): notifications + berichte both live
// on one tree (shared i18n + module-settings-registry). Dev :5173.
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/merge-check')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const rawScan = (full) => [...new Set([...full.matchAll(/\b(berichte|notifications|moduleSettings|common|shared)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

try {
  // ── notifications ──
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  const nText = await page.evaluate(() => document.body.innerText)
  out.notif = {}
  out.notif.rendered = nText.length > 200
  out.notif.unreadBadgeNav = await page.locator('nav, aside').locator('text=/^[0-9]+$/').count().catch(() => 0)
  out.notif.sortMenu = await page.getByRole('button', { name: /Sortieren/ }).count().catch(() => 0)
  // click first notification row → DetailModal
  const rows = page.locator('[role="button"]').filter({ hasText: /vor|ago|Uhr|min|Std|: / })
  await rows.first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(800)
  out.notif.detailModal = await page.locator('[role="dialog"]').isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '1-notifications.png'), fullPage: true })
  out.notif.rawKeys = rawScan(nText)
  await page.keyboard.press('Escape').catch(() => {})

  // ── berichte (regression) ──
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  out.berichte = {}
  out.berichte.kpiCards = await page.locator('.text-2xl.font-semibold').count()
  out.berichte.recharts = await page.locator('.recharts-surface').count()
  const bText = await page.evaluate(() => document.body.innerText)
  out.berichte.rawKeys = rawScan(bText)
  await page.screenshot({ path: resolve(outDir, '2-berichte.png'), fullPage: true })
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
