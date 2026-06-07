import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/settings-fundament')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// raw-key detector for our new namespaces
const rawRe = /(settings\.scope\.|settings\.calendar\.section\.|team\.member\.moduleLead\.)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, rawRe.source)
}

const browser = await chromium.launch()
const out = []

async function openCalendarSettings(page) {
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  // click the "Kalender" settings nav button
  await page.getByRole('button', { name: 'Kalender', exact: true }).first().click()
  await page.waitForTimeout(900)
}

async function switchProfile(page, label) {
  // open floating dev profile switcher (bottom-left pill showing current role)
  const trigger = page.locator('button.fixed.bottom-4.left-4')
  await trigger.click({ timeout: 8000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, `_switcher-open.png`) })
  // click the target role profile row (button) inside the panel
  await page.locator('button').filter({ hasText: label }).first().click({ timeout: 8000 })
  await page.waitForTimeout(600)
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(300)
}

// 1) Calendar settings as ADMIN (tenant section unlocked) at full + narrow width
for (const w of [1440, 720]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await openCalendarSettings(page)
    const badges = await page.evaluate(() =>
      ['Persönlich', 'Für alle'].filter((b) => document.body.textContent.includes(b)))
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, `calendar-admin-${w}.png`), fullPage: true })
    out.push({ step: `calendar-admin-${w}`, scopeBadges: badges, rawKeys: rk, errs: errs.length })
  } catch (e) {
    out.push({ step: `calendar-admin-${w}`, error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

// 2) Calendar settings as MEMBER (tenant section LOCKED — lock hint + lock badge)
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await openCalendarSettings(page)
    // switch to a non-lead role in place — the mounted tab re-renders locked
    await switchProfile(page, 'Mitarbeiter')
    await page.waitForTimeout(800)
    const lockHint = await page.evaluate(() =>
      document.body.textContent.includes('nur von der Modulleitung'))
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, `calendar-member-locked.png`), fullPage: true })
    out.push({ step: 'calendar-member-locked', lockHintShown: lockHint, rawKeys: rk, errs: errs.length })
  } catch (e) {
    out.push({ step: 'calendar-member-locked', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

// 3) Team member detail (admin) — module-lead toggle visible
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1800)
    // open first member card to reveal the detail panel
    await page.getByText('Unbekannt', { exact: false }).first().click({ timeout: 8000 }).catch(() => {})
    await page.waitForTimeout(1200)
    // expand "Erweiterte Moduleinstellungen" if collapsed
    await page.getByText('Erweiterte Moduleinstellungen', { exact: false }).first().click().catch(() => {})
    await page.waitForTimeout(500)
    const hasLeadSection = await page.evaluate(() =>
      document.body.textContent.includes('Erweiterte Moduleinstellungen'))
    await page.screenshot({ path: resolve(outDir, `team-modulelead.png`), fullPage: true })
    out.push({ step: 'team-modulelead', leadSectionShown: hasLeadSection, errs: errs.length })
  } catch (e) {
    out.push({ step: 'team-modulelead', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
