// QA quiet-hours UI polish: Cosmi TimePicker (popover) + options hidden when inactive.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/quiet-hours-ui')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, p) => (p === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`
async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(settings|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2000)
  await page.getByText('Benachrichtigungen', { exact: false }).first().click({ timeout: 5000 }).catch((e) => { out.tabErr = String(e).split('\n')[0] })
  await page.waitForTimeout(800)

  // Ensure quiet hours active (toggle on if needed) — switch sits in the "aktiv" row
  const qhSwitch = page.locator('button[role="switch"]').last()
  const isOn = (await qhSwitch.getAttribute('aria-checked')) === 'true'
  if (!isOn) { await qhSwitch.click(); await page.waitForTimeout(400) }

  // 1) TimePicker present (clock button with HH:MM, NOT a native time input)
  out.nativeTimeInputs = await page.locator('input[type="time"]').count()
  out.timePickerButtons = await page.locator('button:has(svg.lucide-clock)').count()
  await page.screenshot({ path: resolve(outDir, '01-active-with-timepicker.png'), fullPage: true })

  // 2) Open the picker popover → hour/minute columns visible
  await page.locator('button:has(svg.lucide-clock)').first().click({ timeout: 4000 }).catch((e) => { out.pickerErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  out.pickerPopoverOpen = await page.getByText(/^23$|^22$|^00$/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '02-picker-popover.png'), fullPage: true })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // 3) Deactivate quiet hours → options (save button, weekdays, pickers) disappear
  await qhSwitch.click()
  await page.waitForTimeout(500)
  // Options collapse (stay in DOM but 0-height/hidden) → isVisible must be false
  out.saveVisibleWhenInactive = await page.getByText(/Ruhezeiten speichern/).first().isVisible().catch(() => false)
  out.timePickerVisibleWhenInactive = await page.locator('button:has(svg.lucide-clock)').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '03-inactive-options-hidden.png'), fullPage: true })

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
