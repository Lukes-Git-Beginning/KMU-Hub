// QA Screenshot-Run für das security-Modul (DSGVO/Security-Center), Demo-Mode.
// DEV_BYPASS_AUTH setzt automatisch den admin-Profile-User → Hub-Guard erfüllt,
// kein Login nötig. Fährt beide Hubs ab: neuer Admin-Hub (/admin/security) mit
// 7 Sub-Tabs + Legacy-Hub (/admin/security-legacy) mit 9 Tabs.
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
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

async function shot(name) {
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, `security-${name}.png`), fullPage: false })
}
async function jsClick(selector) {
  const ok = await page.evaluate((sel) => {
    const el = document.querySelector(sel)
    if (el) { el.click(); return true }
    return false
  }, selector)
  return ok
}

try {
  // ── Neuer Admin-Hub ───────────────────────────────────────────────
  await page.goto(`${FE}/#/admin/security`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(2500)
  await shot('hub-new-00-default')

  const newTabs = ['audit', 'gdpr', 'sessions', 'ip-whitelist', 'vault', 'privacy', 'ai']
  for (const key of newTabs) {
    const clicked = await jsClick(`#security-tab-${key}`)
    await page.waitForTimeout(900)
    await shot(`hub-new-${key}`)
    if (!clicked) console.log(`  [warn] new-hub tab not found: ${key}`)
  }

  // ── Legacy-Hub — ISOLIERT: vor jedem Tab frisch laden, damit ein
  // gecrashter Tab nicht die Error-Boundary für die Folge-Tabs "klebt".
  const legacyTabs = ['audit', 'sessions', 'vault', 'password-policy', 'ip-access', 'gdpr-exports', 'gdpr-erasure', 'dsar', 'retention']
  const crashReport = {}
  for (let i = 0; i < legacyTabs.length; i++) {
    const key = legacyTabs[i]
    await page.goto(`${FE}/#/admin/security-legacy`, { waitUntil: 'domcontentloaded', timeout: 30000 })
    await page.waitForTimeout(1400)
    // nav-button mit Index i (Reihenfolge entspricht legacyTabs)
    await page.evaluate((idx) => {
      const btns = document.querySelectorAll('aside nav button')
      if (btns[idx]) btns[idx].click()
    }, i)
    await page.waitForTimeout(1100)
    await shot(`iso-${String(i + 1).padStart(2, '0')}-${key}`)
    const crashed = await page.evaluate(() => /schiefgelaufen|went wrong/i.test(document.body.innerText))
    const errMsg = await page.evaluate(() => {
      const m = document.body.innerText.match(/(Cannot read[^\n]+|[^\n]*is not a function[^\n]*)/)
      return m ? m[0].slice(0, 80) : ''
    })
    crashReport[key] = crashed ? `CRASH: ${errMsg}` : 'ok'
  }
  console.log('\n=== ISO-CRASH-REPORT (legacy, isoliert) ===')
  for (const k of legacyTabs) console.log(`  ${k}: ${crashReport[k]}`)

  // Body-Text-Stichprobe auf Raw-Keys
  const bodyText = await page.evaluate(() => document.body.innerText)
  const rawKeyHits = (bodyText.match(/security\.[a-zA-Z.]+|admin\.security\.[a-zA-Z.]+/g) || []).slice(0, 20)
  console.log('\n=== ERGEBNIS ===')
  console.log('Raw-Key-Treffer (sichtbar):', rawKeyHits.length ? rawKeyHits.join(', ') : 'keine')
  console.log('Page errors:', errors.length ? errors.slice(0, 5).join(' | ') : 'keine')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e))
  await page.screenshot({ path: resolve(outDir, 'security-error.png'), fullPage: false }).catch(() => {})
} finally {
  await browser.close()
}
