// QA X-4: dokumente tenant-settings persist. Verifies the settings overlay's
// dokumente panel HYDRATES from the backend. Pre-seed via API:
//   PUT /settings/dokumente/tenant {allowedFileTypeGroups:[documents,images],
//   defaultShareScope:team, onlyOfficeEnabled:false, trashRetentionDays:14}
// Then open Modul-Einstellungen -> Dokumente and screenshot: only documents+
// images toggled on (not all 6), share=team, OnlyOffice off, trash=14 proves
// the store hydrated from the backend rather than its shipped defaults.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots', 'settings-x4')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
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
// Wipe the localStorage copy so any hydrated value can only come from the backend.
const WIPE_LOCAL = `try { localStorage.removeItem('cosmi-dokumente-settings') } catch (e) {}`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
await ctx.addInitScript(WIPE_LOCAL)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))
const page = await ctx.newPage()
const errors = []
let putFired = false
let getFired = false
page.on('pageerror', (e) => errors.push(String(e)))
page.on('request', (req) => {
  const u = req.url()
  if (u.includes('/settings/dokumente/tenant')) {
    if (req.method() === 'PUT') putFired = true
    if (req.method() === 'GET') getFired = true
  }
})

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.locator('input[type=email]').waitFor({ state: 'visible', timeout: 20000 })
  await page.locator('input[type=email]').fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3500)

  // Open Modul-Einstellungen (sidebar bottom)
  await page.getByText(/Modul-Einstellung/i).first().click({ timeout: 10000 })
  await page.waitForTimeout(1200)
  // Pick the Dokumente module from the overlay's left nav (a button/role item).
  const navItem = page.getByRole('button', { name: 'Dokumente', exact: true })
  if (await navItem.count()) {
    await navItem.first().click({ timeout: 8000 })
  } else {
    // Fallback: left-most "Dokumente" (nav column), force past any overlay.
    await page.getByText('Dokumente', { exact: true }).first().click({ timeout: 8000, force: true })
  }
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, '1-dokumente-settings-top.png'), fullPage: false })

  // Scroll the dialog content down to the tenant "Erlaubte Dateitypen" section.
  await page.getByText(/Erlaubte Dateitypen/i).first().scrollIntoViewIfNeeded().catch(() => {})
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '2-dokumente-settings-filetypes.png'), fullPage: false })

  // Read the rendered switches to confirm hydration: documents+images ON, others OFF.
  const switchStates = await page.evaluate(() => {
    const rows = Array.from(document.querySelectorAll('label')).filter((l) => /\.(pdf|png|xls|ppt|mp3|zip)/i.test(l.innerText) || /Dokument|Bild|Tabelle|Präsent|Medien|Archiv|Sonstige/i.test(l.innerText))
    return rows.map((l) => {
      const sw = l.querySelector('[role=switch],button[aria-checked]')
      return { label: l.innerText.split('\n')[0]?.trim(), checked: sw?.getAttribute('aria-checked') }
    }).slice(0, 8)
  })

  const body = await page.evaluate(() => document.body.innerText)
  console.log('\n=== ERGEBNIS X-4 dokumente settings ===')
  console.log('GET /settings/dokumente/tenant gefeuert (Hydration):', getFired)
  console.log('PUT /settings/dokumente/tenant gefeuert (Write-Through):', putFired)
  console.log('Panel sichtbar:', /Dateityp|Aufbewahrung|Freigabe|OnlyOffice/i.test(body))
  console.log('Switch-Zustände (erwartet: Dokumente+Bilder checked, Rest unchecked):')
  for (const s of switchStates) console.log(`  ${s.checked === 'true' ? '☑' : '☐'} ${s.label}`)
  console.log('page errors:', errors.length ? errors.slice(0, 2).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'error.png'), fullPage: false }).catch(() => {})
} finally {
  await browser.close()
}
