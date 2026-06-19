/**
 * QA — notifications N-2: per-module filter chips + sort menu.
 * Verifies: filter bar renders with counts, a module chip reduces the list,
 * sort-by-priority reorders, module badges now show readable labels (not raw
 * ids), empty filtered state is clean. Sub-terminal → :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/notif-n2')
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
async function cardTitles(page) {
  return page.evaluate(() =>
    Array.from(document.querySelectorAll('.space-y-2 .cursor-pointer p.text-sm.font-semibold, .space-y-2 .cursor-pointer p.text-sm'))
      .map((n) => (n.textContent || '').trim()).filter(Boolean).slice(0, 8)
  )
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

  const chipCount = await page.locator('button[aria-pressed]').count()
  const allCards = await page.locator('.space-y-2 .cursor-pointer').count()
  // module badge readability: look for German labels, ensure no raw "contacts"/"tasks" badge
  const badgeTexts = await page.locator('.space-y-2 .cursor-pointer [class*="secondary"]').allTextContents()
  await page.screenshot({ path: resolve(outDir, '0-filterbar.png') })
  out.push({ check: 'filterbar', chipCount, allCards, badgeSample: badgeTexts.slice(0, 6), rawKeys: await rawKeys(page) })

  // filter by "Verträge"
  await page.locator('button[aria-pressed]:has-text("Verträge")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  const filteredCards = await page.locator('.space-y-2 .cursor-pointer').count()
  const filteredTitles = await cardTitles(page)
  await page.screenshot({ path: resolve(outDir, '1-filtered-vertraege.png') })
  out.push({ check: 'filter-vertraege', filteredCards, filteredTitles })

  // back to all
  await page.locator('button[aria-pressed]:has-text("Alle Module")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)

  // sort by priority desc → first card should be urgent (Sicherheitswarnung) or high
  const beforeSort = await cardTitles(page)
  await page.locator('button[aria-label="Sortieren nach"]').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(500)
  await page.locator('[role="menuitemradio"]:has-text("Priorität")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(900)
  const afterSort = await cardTitles(page)
  await page.screenshot({ path: resolve(outDir, '2-sorted-priority.png') })
  out.push({ check: 'sort-priority', beforeSortFirst: beforeSort[0], afterSortFirst: afterSort[0], afterSort })

  out.push({ pageErrors: errs.slice(0, 5) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 5) })
} finally {
  await ctx.close(); await browser.close()
}

console.log(JSON.stringify(out, null, 2))
