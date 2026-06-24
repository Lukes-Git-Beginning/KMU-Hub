// QA für S-2: prüft, dass die stateful MSW-Schreib-Ops wirken (Demo-Mode).
// Testet IP-Rule-Add, Session-Terminate und Vault-Reveal über echte Klicks.
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
await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(`
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const results = {}

async function gotoTab(idx) {
  await page.goto(`${FE}/#/admin/security-legacy`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(1400)
  await page.evaluate((i) => { const b = document.querySelectorAll('aside nav button'); if (b[i]) b[i].click() }, idx)
  await page.waitForTimeout(1100)
}
const bodyHas = (re) => page.evaluate((r) => new RegExp(r, 'i').test(document.body.innerText), re.source)

try {
  // ── IP-Rule add (legacy idx 4 = ip-access) ──────────────────────────
  await gotoTab(4)
  const cidrBefore = await page.evaluate(() => (document.body.innerText.match(/\/\d{1,2}\b/g) || []).length)
  // open add dialog (button mit Plus / "Regel")
  await page.evaluate(() => {
    const btns = [...document.querySelectorAll('button')]
    const b = btns.find((x) => /regel|hinzuf|add/i.test(x.textContent || ''))
    if (b) b.click()
  })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, 'sec-act-ip-dialog.png') })
  // fill CIDR + description via Playwright (React-controlled inputs)
  await page.fill('input[placeholder="192.168.1.0/24"]', '172.16.99.0/24').catch(() => {})
  await page.fill('input[placeholder="Office network"]', 'QA-Testregel').catch(() => {})
  await page.waitForTimeout(500)
  await page.getByRole('button', { name: /erstellen|create/i }).click({ timeout: 5000 }).catch(async () => {
    await page.click('text=Erstellen', { timeout: 3000 }).catch(() => {})
  })
  await page.waitForTimeout(1400)
  await page.screenshot({ path: resolve(outDir, 'sec-act-ip-after.png') })
  results.ipRuleAdded = await bodyHas(/172\.16\.99\.0/)

  // ── Vault reveal (legacy idx 2 = vault) ─────────────────────────────
  await gotoTab(2)
  const hiddenBefore = await page.evaluate(() => (document.body.innerText.match(/verborgen|hidden/gi) || []).length)
  await page.evaluate(() => {
    // reveal-Icon = erster Button in der ersten Datenzeile (Eye)
    const row = document.querySelector('tbody tr')
    if (row) { const b = row.querySelector('button'); if (b) b.click() }
  })
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, 'sec-act-vault-reveal.png') })
  results.vaultRevealed = await bodyHas(/demo-secret-/)

  // ── Session terminate (legacy idx 1 = sessions) ─────────────────────
  await gotoTab(1)
  const sessBefore = await page.evaluate(() => document.querySelectorAll('button').length && (document.body.innerText.match(/beenden|terminate/gi) || []).length)
  // click first "Sitzung beenden"
  await page.evaluate(() => {
    const btns = [...document.querySelectorAll('button')]
    const b = btns.find((x) => /sitzung beenden|beenden|terminate/i.test(x.textContent || '') && !/alle/i.test(x.textContent || ''))
    if (b) b.click()
  })
  await page.waitForTimeout(700)
  // confirm in AlertDialog
  await page.evaluate(() => {
    const btns = [...document.querySelectorAll('button')]
    const b = btns.find((x) => /beenden|bestät|confirm|terminate|ja/i.test(x.textContent || '') && x.closest('[role=alertdialog]'))
    if (b) b.click()
  })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'sec-act-session-after.png') })
  const sessAfterCount = await page.evaluate(() => document.querySelectorAll('[class*="border"]').length)
  results.sessionMeineText = await page.evaluate(() => (document.body.innerText.match(/\d+\s*Meine/i) || [''])[0])

  console.log('\n=== S-2 AKTIONS-ERGEBNIS ===')
  console.log('IP-Rule hinzugefügt (172.16.99.0 sichtbar):', results.ipRuleAdded)
  console.log('Vault-Reveal (demo-secret sichtbar):', results.vaultRevealed)
  console.log('Sessions "x Meine" nach Terminate:', results.sessionMeineText)
  console.log('Page errors:', errors.length ? errors.slice(0, 3).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'sec-act-error.png') }).catch(() => {})
} finally {
  await browser.close()
}
