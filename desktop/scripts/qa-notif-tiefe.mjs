/**
 * QA — Notifications Demo-Tiefe.
 * Verifies: priority filter chips (filter to a single priority), snooze from the
 * detail modal (dropdown → item → row disappears), and no raw i18n keys.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/notif-tiefe')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []
const rawKeys = (txt) => (txt.match(/\b(notifications|shared|common)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '1-center.png') })

  // 1) Priority filter chips present
  const urgentChip = page.getByRole('button', { name: /^Dringend/ })
  const hasPriorityChips = (await urgentChip.count()) > 0
  const cardsBefore = await page.locator('[role="button"][aria-label]').count()
  out.push({ step: 'priority chips present', hasPriorityChips, pass: hasPriorityChips })

  // 2) Filter to "Dringend" → security alert visible, list narrows
  if (hasPriorityChips) {
    await urgentChip.first().click()
    await page.waitForTimeout(700)
    await page.screenshot({ path: resolve(outDir, '2-filtered-urgent.png') })
    const bodyTxt = await page.evaluate(() => document.body.innerText)
    out.push({ step: 'filter urgent', hasSecurityAlert: /Sicherheitswarnung/.test(bodyTxt), pass: /Sicherheitswarnung/.test(bodyTxt) })
    // reset
    await page.getByRole('button', { name: /Alle Priorit/ }).first().click()
    await page.waitForTimeout(500)
  }

  // 3) Open a row → detail modal with snooze
  await page.locator('[role="button"][aria-label]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, '3-detail.png') })
  const snoozeTrigger = page.getByRole('button', { name: /^Erinnern$/ })
  const hasSnooze = (await snoozeTrigger.count()) > 0
  out.push({ step: 'detail has snooze button', hasSnooze, pass: hasSnooze })

  // 4) Open snooze dropdown → options visible
  if (hasSnooze) {
    await snoozeTrigger.first().click()
    await page.waitForTimeout(500)
    await page.screenshot({ path: resolve(outDir, '4-snooze-menu.png') })
    const menuTxt = await page.evaluate(() => document.querySelector('[role="menu"]')?.textContent || '')
    out.push({
      step: 'snooze options',
      opts: menuTxt.slice(0, 60),
      pass: /In 1 Stunde/.test(menuTxt) && /Morgen früh/.test(menuTxt),
    })
    // 5) Click "In 1 Stunde" → modal closes, row snoozed
    await page.getByRole('menuitem', { name: /In 1 Stunde/ }).click()
    await page.waitForTimeout(900)
    const dialogGone = (await page.locator('[role="dialog"]').count()) === 0
    const cardsAfter = await page.locator('[role="button"][aria-label]').count()
    await page.screenshot({ path: resolve(outDir, '5-after-snooze.png') })
    out.push({ step: 'snooze applied (modal closed, list shrank)', dialogGone, cardsBefore, cardsAfter, pass: dialogGone && cardsAfter < cardsBefore })
  }

  const leaks = rawKeys(await page.evaluate(() => document.body.innerText))
  out.push({ step: 'no raw i18n keys', leaks, pass: leaks.length === 0 })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 8), pass: errs.length === 0 })
await ctx.close(); await b.close()
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== NOTIF-TIEFE QA: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
