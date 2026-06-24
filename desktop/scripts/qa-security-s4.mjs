// QA für S-4: Erasure (echte Preview-API + Legal-Hold) + Retention-Toggle.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const SUPPRESS = `
  try { const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p)) } catch(e){}
`
await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const res = {}
async function gotoTab(idx) {
  await page.goto(`${FE}/#/admin/security-legacy`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(1400)
  await page.evaluate((i) => { const b = document.querySelectorAll('aside nav button'); if (b[i]) b[i].click() }, idx)
  await page.waitForTimeout(1000)
}
const has = (re) => page.evaluate((r) => new RegExp(r, 'i').test(document.body.innerText), re)

try {
  // ── Erasure (idx 6): search → select → preview ──────────────────────
  await gotoTab(6)
  await page.keyboard.press('Escape').catch(() => {})
  await page.fill('input[placeholder*="Mitarbeiter"], input[placeholder*="Name"], input[placeholder*="Benutzer"], input[placeholder*="search"]', 'Max').catch(() => {})
  await page.waitForTimeout(700)
  // click first search result (Max Mustermann)
  await page.evaluate(() => {
    const b = [...document.querySelectorAll('button')].find((x) => /Mustermann|Schmidt|Müller/.test(x.textContent || ''))
    if (b) b.click()
  })
  await page.waitForTimeout(500)
  // click "Vorschau"/preview
  await page.getByRole('button', { name: /vorschau|preview/i }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'sec-s4-erasure-preview.png') })
  res.erasureModules = await has('CRM Kontakte')
  res.erasureLegalHold = await has('147 AO|gesperrt|Aufbewahrungspflicht')
  res.erasureRetainBadge = await has('Aufbewahren|retain|Behalten')

  // ── Retention (idx 8): toggle auto-deletion ─────────────────────────
  await gotoTab(8)
  const pausedBefore = await page.evaluate(() => (document.body.innerText.match(/pausiert/gi) || []).length)
  await page.evaluate(() => {
    const sw = document.querySelector('tbody tr button[role="switch"]')
    if (sw) sw.click()
  })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, 'sec-s4-retention-toggle.png') })
  const pausedAfter = await page.evaluate(() => (document.body.innerText.match(/pausiert/gi) || []).length)
  res.retentionToggle = pausedAfter > pausedBefore

  console.log('\n=== S-4 ERGEBNIS ===')
  console.log('Erasure-Preview (CRM Kontakte sichtbar):', res.erasureModules)
  console.log('Legal-Hold-Hinweis (§147 AO):', res.erasureLegalHold)
  console.log('Retention-Toggle wirkt (pausiert erscheint):', res.retentionToggle, `(${pausedBefore}->${pausedAfter})`)
  console.log('Page errors:', errors.length ? errors.slice(0, 3).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'sec-s4-error.png') }).catch(() => {})
} finally {
  await browser.close()
}
