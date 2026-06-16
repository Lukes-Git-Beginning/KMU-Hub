// QA Booking Pages: panel list, editor (edit), preview, new editor.
// Verifies layout, i18n (no raw keys), MSW wiring, demo seed data visible.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/booking')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
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

function scanRawKeys(page) {
  return page.evaluate(() =>
    [
      ...new Set(
        [...document.body.innerText.matchAll(/\b(kalender\.(booking|external|newBooking))\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map(
          (m) => m[0],
        ),
      ),
    ],
  )
}
const bodyText = (page) => page.evaluate(() => document.body.innerText.replace(/\n{2,}/g, '\n').slice(0, 400))

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  // 1. Navigate to calendar module (Terminbuchung is a tab inside it)
  await page.goto(`${BASE}/#/kalender`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2500)

  // 2. Click Terminbuchung tab
  const terminbuchungBtn = page.locator('button, [role="tab"]').filter({ hasText: /Terminbuchung|Booking/i }).first()
  if (await terminbuchungBtn.count()) {
    await terminbuchungBtn.click()
    await page.waitForTimeout(1500)
  }

  // 3. Screenshot: booking list panel with demo seed
  await page.screenshot({ path: resolve(outDir, 'booking-list.png') })
  out.listRawKeys = await scanRawKeys(page)
  out.listText = await bodyText(page)
  out.hasDemoPage = await page.locator('text=muster-dienstleister').count()
  out.hasNewButton = await page.locator('button').filter({ hasText: /Neue Buchungsseite|New booking/i }).count()

  // 4. Click "Bearbeiten" on the demo page
  const editBtn = page.locator('button').filter({ hasText: /Bearbeiten|Edit/i }).first()
  if (await editBtn.count()) {
    await editBtn.click()
    await page.waitForTimeout(1200)
  }
  await page.screenshot({ path: resolve(outDir, 'booking-editor.png') })
  out.editorRawKeys = await scanRawKeys(page)
  out.editorText = await bodyText(page)
  out.hasCalendarPicker = await page.locator('select').count()
  out.hasSaveButton = await page.locator('button').filter({ hasText: /Änderungen|Save changes/i }).count()

  // 5. Click cancel/back to list
  const cancelBtn = page.locator('button').filter({ hasText: /Zurück zur Übersicht|Back to overview/i }).first()
  if (await cancelBtn.count()) {
    await cancelBtn.click()
    await page.waitForTimeout(800)
  } else {
    const abbrechen = page.locator('button').filter({ hasText: /Abbrechen/i }).first()
    if (await abbrechen.count()) {
      await abbrechen.click()
      await page.waitForTimeout(800)
    }
  }

  // 6. Click "Vorschau" on the demo page
  const previewBtn = page.locator('button').filter({ hasText: /Vorschau|Preview/i }).first()
  if (await previewBtn.count()) {
    await previewBtn.click()
    await page.waitForTimeout(1200)
  }
  await page.screenshot({ path: resolve(outDir, 'booking-preview.png') })
  out.previewRawKeys = await scanRawKeys(page)
  out.previewText = await bodyText(page)
  out.hasCompanyName = await page.locator('text=Zentria Muster GmbH').count()
  out.hasSlugLink = await page.locator('text=muster-dienstleister').count()

  // 7. Back to list, then click "Neue Buchungsseite"
  const backBtn = page.locator('button').filter({ hasText: /Zurück zur Übersicht|Back to overview/i }).first()
  if (await backBtn.count()) {
    await backBtn.click()
    await page.waitForTimeout(800)
  }
  const newPageBtn = page.locator('button').filter({ hasText: /Neue Buchungsseite|New booking page/i }).first()
  if (await newPageBtn.count()) {
    await newPageBtn.click()
    await page.waitForTimeout(1000)
  }
  await page.screenshot({ path: resolve(outDir, 'booking-new.png') })
  // Scroll the bottom of the form into view so the availability-rules editor is visible
  const createBtn = page.locator('button').filter({ hasText: /Buchungsseite erstellen|Create booking/i }).first()
  if (await createBtn.count()) {
    await createBtn.scrollIntoViewIfNeeded()
    await page.waitForTimeout(500)
  }
  await page.screenshot({ path: resolve(outDir, 'booking-new-availability.png') })
  out.newRawKeys = await scanRawKeys(page)
  out.newText = await bodyText(page)
  out.hasSlugInput = await page.locator('input[placeholder*="mein-unternehmen"]').count()
  out.hasCreateButton = await page.locator('button').filter({ hasText: /Buchungsseite erstellen|Create booking/i }).count()
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.pageErrors = errors.slice(0, 5)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
