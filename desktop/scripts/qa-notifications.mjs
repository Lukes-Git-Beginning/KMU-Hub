// Visual + functional QA for the notifications batch (N-1..N-5) on the REVIEW clone (:5174).
// Run with the dev server already up on :5174. Usage: node scripts/qa-notifications.mjs [phase]
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/notifications')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const scanRaw = (page) =>
  page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
      .filter(Boolean)
    return {
      rawKeys: [...new Set(all.filter((t) => /^(notifications|settings|layout|dashboard|common)\.[a-zA-Z]/.test(t)))].slice(0, 20),
      doubleBrace: [...new Set(all.filter((t) => /\{\{|\}\}/.test(t)))].slice(0, 10),
      replacementChar: [...new Set(all.filter((t) => /�/.test(t)))].slice(0, 10),
      emoji: [...new Set(all.filter((t) => /[\u{1F000}-\u{1FAFF}\u{2600}-\u{27BF}]/u.test(t)))].slice(0, 10),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const has = async (loc) => (await loc.count().catch(() => 0)) > 0

try {
  // ── Notification center ──
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.screenshot({ path: resolve(outDir, '1-center-all.png'), fullPage: true })
  out.centerTitle = await page.evaluate(() => document.querySelector('h1')?.textContent || '')
  out.groupBadge = await page.evaluate(() => /weitere|more|altri|autre/i.test(document.body.innerText))

  // Unread tab
  const unreadTab = page.getByRole('tab').nth(1)
  if (await has(unreadTab)) { await unreadTab.click(); await page.waitForTimeout(800) }
  await page.screenshot({ path: resolve(outDir, '2-center-unread.png'), fullPage: true })

  // Open a row → detail modal
  await page.getByRole('tab').first().click(); await page.waitForTimeout(500)
  const firstCard = page.locator('[role="button"][aria-label]').first()
  if (await has(firstCard)) { await firstCard.click(); await page.waitForTimeout(700) }
  await page.screenshot({ path: resolve(outDir, '3-detail-modal.png') })
  out.detailHasPin = await page.evaluate(() => /Anheften|anpinnen|Pin|Épingler|Fissa/i.test(document.body.innerText))
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)

  // ── Bell popover ──
  const bell = page.locator('button[aria-label*="enachrichtigung"], button[aria-label*="otification"], button[aria-label*="otifiche"]').first()
  if (await has(bell)) { await bell.click(); await page.waitForTimeout(700) }
  await page.screenshot({ path: resolve(outDir, '4-bell-popover.png') })
  await page.keyboard.press('Escape').catch(() => {})

  // ── Sidebar: notifications nav entry present? ──
  out.sidebarNav = await page.evaluate(() => {
    const links = Array.from(document.querySelectorAll('a[href*="notifications"]'))
    return links.map((l) => (l.textContent || '').trim()).filter(Boolean)
  })

  // ── Settings: notifications panel ──
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2500)
  const notifTab = page.getByText(/Benachrichtigung|Notification|Notifiche/i).first()
  if (await has(notifTab)) { await notifTab.click(); await page.waitForTimeout(1200) }
  await page.screenshot({ path: resolve(outDir, '5-settings-notifications.png'), fullPage: true })
  out.mutedList = await page.evaluate(() => {
    const txt = document.body.innerText
    return { hasMuted: /random|Thomas|ERP/i.test(txt) }
  })

  Object.assign(out, await scanRaw(page))
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'ERROR.png') }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
