/**
 * QA script — Dashboard D-1: Persistenz scharf + Store-Crash-Fix
 *
 * Verifies:
 *  (1) Dashboard loads cleanly with initFromServer() on mount (no pageerror),
 *      and the persistence endpoints no longer 404:
 *        - GET  /dashboard/layout      (load)
 *        - PUT  /dashboard/layout      (add widget → debounced sync)
 *        - DELETE /dashboard/layout    (reset)
 *  (2) Admin role-defaults page (/settings/dashboard): the "Aktuelles als
 *      Standard" button (handleCopyCurrentLayout) no longer crashes
 *      (was: s.layouts undefined → .map of undefined).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dashboard-d1')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const RAW_RE = /(dashboard\.[a-z]|widgets\.[a-z]|settings\.dashboard\.)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim()).slice(0, 12)
  }, RAW_RE.source)
}

function trackDash(page, bucket) {
  page.on('response', (r) => {
    const u = r.url()
    if (/\/api\/v1\/(dashboard\/(layout|defaults)|feature-flags)/.test(u)) {
      bucket.push({ method: r.request().method(), path: u.replace(/^.*\/api\/v1\//, ''), status: r.status() })
    }
  })
}

const browser = await chromium.launch()
const out = []

// ── Scenario 1: load + persistence endpoints ───────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []; const net = []
  page.on('pageerror', (e) => errs.push(String(e)))
  trackDash(page, net)
  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2800)
    const layoutExists = await page.evaluate(() => !!document.querySelector('.layout'))
    await page.screenshot({ path: resolve(outDir, '1-load.png') })

    // edit mode
    const editBtn = page.locator('button:has-text("Anpassen"), button:has-text("Bearbeiten"), button:has-text("Dashboard anpassen")').first()
    if (await editBtn.isVisible().catch(() => false)) {
      await editBtn.click({ timeout: 5000 }); await page.waitForTimeout(600)
    }
    // add a widget → PUT layout (debounced 2s)
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForTimeout(300)
    const addBtn = page.locator('button:has-text("Widget hinzufügen")').first()
    if (await addBtn.isVisible().catch(() => false)) {
      await addBtn.click({ timeout: 5000 }); await page.waitForTimeout(700)
      const item = page.locator('[role="dialog"] button:has(p.font-medium)').first()
      await item.click({ timeout: 4000 }).catch(() => {})
      await page.waitForTimeout(2700)
    }
    await page.screenshot({ path: resolve(outDir, '2-edit-add.png') })

    // reset → DELETE + GET
    const resetBtn = page.locator('button:has-text("Zurücksetzen"), button:has-text("Reset")').first()
    if (await resetBtn.isVisible().catch(() => false)) {
      await resetBtn.click({ timeout: 5000 }); await page.waitForTimeout(1300)
    }
    const rk = await rawKeys(page)
    out.push({
      scenario: '1-load+persist', layoutExists, rawKeys: rk, pageErrors: errs.slice(0, 3),
      dashNet: net, net404: net.filter((n) => n.status >= 400),
    })
  } catch (e) {
    out.push({ scenario: '1-load+persist', error: String(e).split('\n')[0], pageErrors: errs.slice(0, 3), dashNet: net })
  } finally { await ctx.close() }
}

// ── Scenario 2: admin role-defaults page — crash-fix ───────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []; const net = []
  page.on('pageerror', (e) => errs.push(String(e)))
  trackDash(page, net)
  try {
    await page.goto(`${BASE}/#/settings/dashboard`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2200)
    const bodyText = await page.evaluate(() => document.body.textContent || '')
    const hasRoleTabs = /Widget/.test(bodyText) && /(Standard|admin|Administrator)/i.test(bodyText)
    const redirectedHome = await page.evaluate(() => !!document.querySelector('.layout'))
    await page.screenshot({ path: resolve(outDir, '3-admin-defaults.png') })

    let copyClicked = false, crashAfterCopy = false
    const copyBtn = page.locator('button:has-text("Aktuelles"), button:has-text("als Standard")').first()
    if (await copyBtn.isVisible().catch(() => false)) {
      const before = errs.length
      await copyBtn.click({ timeout: 5000 }); await page.waitForTimeout(1600)
      copyClicked = true
      crashAfterCopy = errs.length > before
      await page.screenshot({ path: resolve(outDir, '4-after-copy.png') })
    }
    const rk = await rawKeys(page)
    out.push({
      scenario: '2-admin-defaults', hasRoleTabs, redirectedHome, copyClicked, crashAfterCopy,
      rawKeys: rk, pageErrors: errs.slice(0, 3), dashNet: net, net404: net.filter((n) => n.status >= 400),
    })
  } catch (e) {
    out.push({ scenario: '2-admin-defaults', error: String(e).split('\n')[0], pageErrors: errs.slice(0, 3), dashNet: net })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
