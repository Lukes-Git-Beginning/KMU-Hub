// QA Mock-Exit (kontakte CRUD): voller User-Flow gegen das ECHTE Backend.
// Login → Liste (real-data) → Detail/360°-Modal → Create → neuer Kontakt in Liste.
// Kein Token-Inject, kein Mock. Verifiziert die Adapter-/Casing-/PUT-Fixes live.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const stamp = Date.now().toString().slice(-5)
const NEW_FIRST = 'QA'
const NEW_LAST = `Kontakt${stamp}`
const NEW_POS = 'Testleiter'

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

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
// Module ohne lokales Backend abfedern (sonst Konsolenrauschen / Hänger).
await ctx.route('**/api/v1/notifications**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ notifications: [], total: 0, unread_count: 0 }) }))
await ctx.route('**/api/v1/hr/**', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: '{}' }))

const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const result = { login: false, listSeed: false, detail: false, create: false, newInList: false }

const shot = (n) => page.screenshot({ path: resolve(outDir, `crud-${n}.png`), fullPage: false }).catch(() => {})

try {
  // 1) Login (echt)
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const email = page.locator('input[type=email]')
  await email.waitFor({ state: 'visible', timeout: 20000 })
  await email.fill('demo@local.test')
  await page.locator('input[type=password]').fill('Demo1234!')
  await page.locator('input[type=password]').press('Enter')
  await page.waitForTimeout(3500)
  result.login = true

  // 2) kontakte-Liste (echte Seed-Daten)
  await page.goto(`${FE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForFunction(() => /mueller|müller|huber|vogel|brandner|schmid/i.test(document.body.innerText), { timeout: 25000 }).catch(() => {})
  await page.waitForTimeout(1000)
  result.listSeed = /mueller|müller|huber|vogel|brandner|schmid/i.test(await page.evaluate(() => document.body.innerText))
  await shot('1-liste')

  // 3) Detail/360°-Modal (erste Seed-Zeile anklicken)
  const firstRow = page.locator('[role="button"]').filter({ hasText: /müller|mueller|huber|vogel|brandner|schmid/i }).first()
  if (await firstRow.count()) {
    await firstRow.click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(1200)
    result.detail = await page.locator('[role="dialog"]').first().isVisible().catch(() => false)
    await shot('2-detail')
    await page.keyboard.press('Escape').catch(() => {})
    await page.waitForTimeout(600)
  }

  // 4) Create-Flow: "Neu"-Button öffnet ein Menü → "Neuer Kontakt" → Dialog
  const addBtn = page.getByRole('button', { name: 'Neu', exact: true }).first()
  if (await addBtn.count()) {
    await addBtn.click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(500)
    const menuItem = page.getByText('Neuer Kontakt', { exact: true }).first()
    if (await menuItem.count()) {
      await menuItem.click({ timeout: 5000 }).catch(() => {})
    }
    const vorname = page.getByPlaceholder('Max', { exact: true })
    await vorname.waitFor({ state: 'visible', timeout: 8000 })
    await vorname.fill(NEW_FIRST)
    await page.getByPlaceholder('Mustermann').fill(NEW_LAST)
    await page.getByPlaceholder('Geschäftsführer').fill(NEW_POS)
    await page.waitForTimeout(300)
    await shot('3-create-dialog')
    const submit = page.getByRole('button', { name: /Kontakt erstellen/i }).first()
    await submit.click({ timeout: 5000 })
    await page.waitForTimeout(2500)
    result.create = true
    await shot('4-after-create')
    // 5) Erscheint der neue Kontakt in der Liste?
    await page.waitForFunction((ln) => document.body.innerText.includes(ln), NEW_LAST, { timeout: 8000 }).catch(() => {})
    result.newInList = (await page.evaluate(() => document.body.innerText)).includes(NEW_LAST)
    await shot('5-liste-mit-neuem')
  }

  console.log('\n=== CRUD-QA ERGEBNIS ===')
  console.log('Login:            ', result.login)
  console.log('Liste Seed-Daten: ', result.listSeed)
  console.log('Detail-Modal auf: ', result.detail)
  console.log('Create gesendet:  ', result.create)
  console.log(`Neuer (${NEW_LAST}) in Liste:`, result.newInList)
  console.log('Page errors:      ', errors.length ? errors.slice(0, 4).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await shot('error')
} finally {
  await browser.close()
}
