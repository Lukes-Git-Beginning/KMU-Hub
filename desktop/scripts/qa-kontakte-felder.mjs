// G0-6: Das Kontaktformular fuellen, speichern, neu laden — jedes Feld muss
// zurueckkommen. Der Befund war, dass 13 sichtbare Felder beim Speichern
// stillschweigend verschwanden.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5173'
const outDir = resolve('.qa-etappe1')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const PREP = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
  try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}
`

const FIELDS = {
  mobile: '+49 170 1234567',
  department: 'Einkauf',
  street: 'Hauptstraße 1',
  zip: '55131',
  city: 'Mainz',
  website: 'https://example.com',
  linkedin: 'https://linkedin.com/in/jdoe',
  xing: 'https://xing.com/profile/jdoe',
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(PREP)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

const out = { steps: [] }
await page.goto(`${FE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(3200)
await page.screenshot({ path: resolve(outDir, 'kontakte-liste.png'), fullPage: false })

const newBtn = page.getByRole('button', { name: /neuer kontakt|kontakt anlegen|neu/i }).first()
if (await newBtn.count()) {
  await newBtn.click()
  await page.waitForTimeout(1800)
  await page.screenshot({ path: resolve(outDir, 'kontakte-formular.png'), fullPage: false })
  out.steps.push('opened create dialog')

  const byPlaceholder = (ph, value) => page.getByPlaceholder(ph, { exact: false }).first().fill(value)
  await byPlaceholder('Max', 'Erika')
  await byPlaceholder('Mustermann', 'Musterfrau')
  await byPlaceholder('max@firma.de', 'erika@example.com')
  await byPlaceholder('+49 30 123 45 67', '+49 6131 123456')
  await byPlaceholder('+49 170 123 4567', FIELDS.mobile)
  await byPlaceholder('Geschäftsführer', 'Einkaufsleiterin')
  await byPlaceholder('Entwicklung', FIELDS.department)

  // Address / social live behind the disclosure.
  const more = page.getByText(/Adresse, Social Media/i).first()
  if (await more.count()) {
    await more.click()
    await page.waitForTimeout(900)
    await page.screenshot({ path: resolve(outDir, 'kontakte-formular-mehr.png'), fullPage: false })
    const extra = await page.locator('input').evaluateAll((els) =>
      els.map((e) => e.getAttribute('placeholder') || '').filter(Boolean),
    )
    out.disclosurePlaceholders = extra
  }

  await page.screenshot({ path: resolve(outDir, 'kontakte-formular-gefuellt.png'), fullPage: false })

  const submit = page.getByRole('button', { name: /kontakt erstellen/i }).first()
  await submit.click()
  await page.waitForTimeout(2600)
  out.steps.push('submitted')

  // Reload and look the contact up again — the point of G0-6.
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  const search = page.getByPlaceholder(/suchen/i).last()
  if (await search.count()) {
    await search.fill('Musterfrau')
    await page.waitForTimeout(1600)
  }
  await page.screenshot({ path: resolve(outDir, 'kontakte-nach-reload.png'), fullPage: false })

  const row = page.getByText('Erika Musterfrau').first()
  if (await row.count()) {
    await row.click()
    await page.waitForTimeout(2200)
    await page.screenshot({ path: resolve(outDir, 'kontakte-detail.png'), fullPage: false })
    const body = await page.locator('body').innerText()
    out.survived = Object.fromEntries(
      Object.entries(FIELDS).map(([k, v]) => [k, body.includes(v)]),
    )
    out.survived.department = body.includes(FIELDS.department)
  } else {
    out.steps.push('contact not found after reload')
  }
} else {
  out.steps.push('create button not found')
}

out.pageErrors = errors.slice(0, 5)
out.expectedFields = FIELDS
await browser.close()
console.log(JSON.stringify(out, null, 2))
