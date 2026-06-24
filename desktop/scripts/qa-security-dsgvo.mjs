// QA für S-3: DSAR-Suche + GDPR-Export-Approve (Demo-Mode).
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
  try {
    const KEY='cosmi-ui'; const raw=localStorage.getItem(KEY); const p=raw?JSON.parse(raw):{state:{},version:0}
    p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(KEY,JSON.stringify(p))
  } catch(e){}
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
  // ── DSAR-Suche (idx 7) ──────────────────────────────────────────────
  await gotoTab(7)
  await page.keyboard.press('Escape').catch(() => {})
  // Target the DSAR card input specifically (not the global Ctrl+K search)
  await page.fill('input[placeholder*="Name"], input[placeholder*="E-Mail"]', 'Lena').catch(() => {})
  await page.keyboard.press('Enter').catch(() => {})
  await page.waitForTimeout(1300)
  await page.screenshot({ path: resolve(outDir, 'sec-dsgvo-dsar.png') })
  res.dsarPerson = await has('Lena Braun')
  res.dsarModule = await has('CRM Kontakte')
  res.dsarEmail = await has('lena.braun')

  // ── GDPR-Export approve (idx 5) ─────────────────────────────────────
  await gotoTab(5)
  await page.keyboard.press('Escape').catch(() => {})
  const pendingBefore = await page.evaluate(() => (document.body.innerText.match(/ausstehend/gi) || []).length)
  // click first "Genehmigen"/"Approve"
  await page.getByRole('button', { name: /genehmigen|approve/i }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1400)
  await page.screenshot({ path: resolve(outDir, 'sec-dsgvo-export-approved.png') })
  res.exportHasNames = await has('Thomas Meier')
  const pendingAfter = await page.evaluate(() => (document.body.innerText.match(/ausstehend/gi) || []).length)
  res.approveWorked = pendingAfter < pendingBefore
  res.downloadVisible = await has('herunterladen|download')

  console.log('\n=== S-3 DSGVO-ERGEBNIS ===')
  console.log('DSAR Person (Lena Braun):', res.dsarPerson, '| Modul (CRM Kontakte):', res.dsarModule, '| E-Mail:', res.dsarEmail)
  console.log('Export zeigt Namen (Thomas Meier):', res.exportHasNames)
  console.log('Approve reduzierte "ausstehend"-Anzahl:', res.approveWorked, `(${pendingBefore}->${pendingAfter})`)
  console.log('Download-Button sichtbar:', res.downloadVisible)
  console.log('Page errors:', errors.length ? errors.slice(0, 3).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'sec-dsgvo-error.png') }).catch(() => {})
} finally {
  await browser.close()
}
