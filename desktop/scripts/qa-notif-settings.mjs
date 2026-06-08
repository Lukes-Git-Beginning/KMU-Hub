/**
 * QA: Settings -> Benachrichtigungen (Notifications Settings Tab).
 *
 * Verifies that after the mock-handler schema fix, the existing tab
 * loads real quiet-hours + DND data, toggles work without page errors,
 * and saving quiet hours persists via the stateful mock.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/notif-settings')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const rawRe = /(notifications|settings)\.[a-zA-Z][a-zA-Z0-9.]+/g
async function scanRawKeys(page) {
  return page.evaluate(({ src, flags }) => {
    const rx = new RegExp(src, flags)
    const text = document.body.innerText
    return [...new Set([...text.matchAll(rx)].map((m) => m[0]))].slice(0, 20)
  }, { src: rawRe.source, flags: rawRe.flags })
}

async function readQHInputs(page) {
  const inputs = await page.locator('input[type="time"]').all()
  const values = []
  for (const inp of inputs) values.push(await inp.inputValue())
  return values
}

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

const out = {}

try {
  // 1) Open settings page
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.screenshot({ path: resolve(outDir, '01-settings-initial.png'), fullPage: true })

  // 2) Click "Benachrichtigungen" tab in left sidebar
  await page.locator('button:has-text("Benachrichtigungen")').first().click({ timeout: 8000 })
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, '02-notifications-tab-initial.png'), fullPage: true })

  // 3) Verify quiet hours inputs are populated (not stuck loading, not empty)
  const initialTimes = await readQHInputs(page)
  out.initialQuietHourValues = initialTimes
  out.qhPopulated = initialTimes.length >= 2 && initialTimes[0] !== '' && initialTimes[1] !== ''

  // 4) Verify DND section visible (Aktivieren button = DND off, present in initial state)
  out.dndVisible = (await page.locator('button:has-text("Aktivieren"), button:has-text("Deaktivieren")').count()) > 0

  // 5) Enable DND (click "Aktivieren")
  const enableDND = page.locator('button:has-text("Aktivieren")').first()
  if (await enableDND.count() > 0) {
    await enableDND.click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(1200)
  }
  await page.screenshot({ path: resolve(outDir, '03-dnd-enabled.png'), fullPage: true })

  // 6) Disable DND (now button should say "Deaktivieren")
  const disableDND = page.locator('button:has-text("Deaktivieren")').first()
  if (await disableDND.count() > 0) {
    await disableDND.click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(1200)
    out.dndToggleWorks = true
  } else {
    out.dndToggleWorks = false
  }
  await page.screenshot({ path: resolve(outDir, '04-dnd-disabled.png'), fullPage: true })

  // 7) Modify quiet hours start time
  const startInput = page.locator('input[type="time"]').first()
  await startInput.fill('23:00').catch(() => {})
  await page.waitForTimeout(300)

  // 8) Click "Ruhezeiten speichern"
  const saveBtn = page.locator('button:has-text("Ruhezeiten speichern")').first()
  if (await saveBtn.count() > 0) {
    await saveBtn.click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(1500)
    out.saveButtonExists = true
  } else {
    out.saveButtonExists = false
  }
  await page.screenshot({ path: resolve(outDir, '05-quiet-hours-saved.png'), fullPage: true })

  // 9) Verify the new value persisted (read inputs again)
  const afterSave = await readQHInputs(page)
  out.afterSaveQuietHourValues = afterSave
  out.persistedStartTime = afterSave[0] === '23:00'

  // 10) Reload + reopen tab, ensure value still 23:00 (stateful mock check)
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2000)
  await page.locator('button:has-text("Benachrichtigungen")').first().click({ timeout: 8000 }).catch(() => {})
  await page.waitForTimeout(1500)
  const afterReload = await readQHInputs(page)
  out.afterReloadQuietHourValues = afterReload
  out.persistedAcrossReload = afterReload[0] === '23:00'
  await page.screenshot({ path: resolve(outDir, '06-after-reload-persisted.png'), fullPage: true })

  // 11) Raw keys scan
  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.pageErrors = errors.slice(0, 10)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
